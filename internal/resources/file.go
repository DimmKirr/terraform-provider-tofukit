package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/models"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

var _ resource.Resource = &FileResource{}

func NewFileResource() resource.Resource {
	return &FileResource{
		BaseComponent: BaseComponent{Kind: "file"},
	}
}

type FileResource struct {
	BaseComponent
}

type FileResourceModel struct {
	ID            types.String                `tfsdk:"id"`
	Name          types.String                `tfsdk:"name"`
	Path          types.String                `tfsdk:"path"`
	Link          types.String                `tfsdk:"link"`
	Model         types.String                `tfsdk:"model"`
	Description   types.String                `tfsdk:"description"`
	Content       types.String                `tfsdk:"content"`
	Instructions  []schemas.InstructionModel  `tfsdk:"instructions"`
	Verifications []schemas.VerificationModel `tfsdk:"verifications"`
	Image         *schemas.ImageModel         `tfsdk:"image"`
}

func (r *FileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

// Configure adds the provider configured data to the resource
func (r *FileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.BaseComponent.Configure(ctx, req, resp)
}

func (r *FileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `File resource for reusable file definitions that can be shared across projects.

**Examples:** .gitignore, LICENSE, .editorconfig, Dockerfile, Makefile, pyproject.toml, go.mod`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Unique identifier for this file resource, used in tofukit:// URIs for cross-referencing (e.g., 'logo', 'app-config'). This is the linkable identity, not the filesystem path.",
				Required:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "Filesystem path where the file will be created (e.g., 'assets/logo.png', 'config/app.yaml'). Defaults to `name` if not specified. Must be filesystem-compliant across platforms.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					ValidFileName(),
				},
			},
			"link": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URI link to this resource for cross-referencing (e.g., tofukit://file/logo)",
			},
			"model": schema.StringAttribute{
				MarkdownDescription: "Model to use for this file in provider/model format (e.g., 'anthropic/claude-sonnet-4.5', 'openai/gpt-5-image'). Overrides provider-level model setting. If not specified, uses provider's default model.",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of what this file does",
				Optional:            true,
			},
			"content": schema.StringAttribute{
				MarkdownDescription: "Static content of the file (mutually exclusive with instructions)",
				Optional:            true,
			},
			"instructions": schema.ListNestedAttribute{
				MarkdownDescription: "Instructions for generating the file content (mutually exclusive with content)",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"prompt": schema.StringAttribute{
							MarkdownDescription: "LLM-facing instruction describing what to generate",
							Required:            true,
						},
						"constraints": schema.ListAttribute{
							MarkdownDescription: "Constraints that must be respected (what NOT to do)",
							Optional:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
			"verifications": schema.ListNestedAttribute{
				MarkdownDescription: "Verification commands for this file",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"command": schema.StringAttribute{
							MarkdownDescription: "Command to run for verification",
							Required:            true,
						},
						"expect": schema.StringAttribute{
							MarkdownDescription: "Expected output or pattern",
							Optional:            true,
						},
					},
				},
			},
			"image": schema.SingleNestedAttribute{
				MarkdownDescription: "Image generation configuration. Only applicable when using an image generation model (e.g., openai/gpt-image-1-mini, openai/dall-e-3).",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"size": schema.StringAttribute{
						MarkdownDescription: "Image resolution in NNNNxNNNN format (e.g., '1024x1024', '1792x1024'). Default varies by model.",
						Optional:            true,
						Validators: []validator.String{
							ImageSizeValidator(),
						},
					},
					"quality": schema.StringAttribute{
						MarkdownDescription: "Image quality level. Valid values depend on model: 'low'/'medium'/'high' for gpt-image-1*, 'standard'/'hd' for dall-e-3.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate: content and instructions are mutually exclusive
	hasContent := !data.Content.IsNull() && !data.Content.IsUnknown() && data.Content.ValueString() != ""
	hasInstructions := len(data.Instructions) > 0

	if hasContent && hasInstructions {
		resp.Diagnostics.AddError(
			"Invalid File Resource Configuration",
			"File resource cannot have both 'content' and 'instructions'. They are mutually exclusive.",
		)
		return
	}

	if !hasContent && !hasInstructions {
		resp.Diagnostics.AddError(
			"Invalid File Resource Configuration",
			"File resource must have either 'content' or 'instructions'.",
		)
		return
	}

	// Validate image quality if image block is present
	if data.Image != nil && !data.Image.Quality.IsNull() && !data.Image.Quality.IsUnknown() {
		quality := data.Image.Quality.ValueString()
		modelSlug := data.Model.ValueString()

		if modelSlug != "" {
			qualityOptions := models.GetQualityOptions(modelSlug)
			if len(qualityOptions) == 0 {
				// Model doesn't support quality parameter
				resp.Diagnostics.AddError(
					"Invalid Image Quality Configuration",
					fmt.Sprintf("Model %q does not support image quality parameter. Remove the 'quality' attribute from the image block.", modelSlug),
				)
				return
			}

			if !models.IsValidQuality(modelSlug, quality) {
				resp.Diagnostics.AddError(
					"Invalid Image Quality Value",
					fmt.Sprintf("Quality %q is not valid for model %q. Valid options: %s", quality, modelSlug, strings.Join(qualityOptions, ", ")),
				)
				return
			}
		}
	}

	// Set path to name if not specified (backwards compatibility)
	if data.Path.IsNull() || data.Path.IsUnknown() || data.Path.ValueString() == "" {
		data.Path = data.Name
	}

	// ID and Link use name (identity), not path (filesystem location)
	data.ID = types.StringValue(fmt.Sprintf("file.%s", data.Name.ValueString()))
	data.Link = types.StringValue(fmt.Sprintf("tofukit://file/%s", data.Name.ValueString()))

	tflog.Info(ctx, "Created file resource", map[string]interface{}{
		"file_id":          data.ID.ValueString(),
		"name":             data.Name.ValueString(),
		"path":             data.Path.ValueString(),
		"link":             data.Link.ValueString(),
		"has_content":      hasContent,
		"has_instructions": hasInstructions,
	})

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save to registry so it's available for other resources
	// This is important when resources already exist in state
	tflog.Info(ctx, "File Read: Saving to registry", map[string]interface{}{
		"file_id": data.ID.ValueString(),
		"name":    data.Name.ValueString(),
	})

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FileResourceModel
	var state FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate: content and instructions are mutually exclusive
	hasContent := !data.Content.IsNull() && !data.Content.IsUnknown() && data.Content.ValueString() != ""
	hasInstructions := len(data.Instructions) > 0

	if hasContent && hasInstructions {
		resp.Diagnostics.AddError(
			"Invalid File Resource Configuration",
			"File resource cannot have both 'content' and 'instructions'. They are mutually exclusive.",
		)
		return
	}

	if !hasContent && !hasInstructions {
		resp.Diagnostics.AddError(
			"Invalid File Resource Configuration",
			"File resource must have either 'content' or 'instructions'.",
		)
		return
	}

	// Validate image quality if image block is present
	if data.Image != nil && !data.Image.Quality.IsNull() && !data.Image.Quality.IsUnknown() {
		quality := data.Image.Quality.ValueString()
		modelSlug := data.Model.ValueString()

		if modelSlug != "" {
			qualityOptions := models.GetQualityOptions(modelSlug)
			if len(qualityOptions) == 0 {
				// Model doesn't support quality parameter
				resp.Diagnostics.AddError(
					"Invalid Image Quality Configuration",
					fmt.Sprintf("Model %q does not support image quality parameter. Remove the 'quality' attribute from the image block.", modelSlug),
				)
				return
			}

			if !models.IsValidQuality(modelSlug, quality) {
				resp.Diagnostics.AddError(
					"Invalid Image Quality Value",
					fmt.Sprintf("Quality %q is not valid for model %q. Valid options: %s", quality, modelSlug, strings.Join(qualityOptions, ", ")),
				)
				return
			}
		}
	}

	// Set path to name if not specified (backwards compatibility)
	if data.Path.IsNull() || data.Path.IsUnknown() || data.Path.ValueString() == "" {
		data.Path = data.Name
	}

	// Preserve computed fields from state (ID uses name for identity)
	data.ID = state.ID
	data.Link = types.StringValue(fmt.Sprintf("tofukit://file/%s", data.Name.ValueString()))

	tflog.Info(ctx, "Updated file resource", map[string]interface{}{
		"file_id":          data.ID.ValueString(),
		"name":             data.Name.ValueString(),
		"path":             data.Path.ValueString(),
		"link":             data.Link.ValueString(),
		"has_content":      hasContent,
		"has_instructions": hasInstructions,
	})

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting file resource from registry", map[string]interface{}{
		"file_id": data.ID.ValueString(),
		"name":    data.Name.ValueString(),
	})

	r.RemoveFromRegistry(ctx, data.ID.ValueString())
}
