package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

var _ resource.Resource = &StackResource{}

func NewStackResource() resource.Resource {
	return &StackResource{
		BaseComponent: BaseComponent{Kind: "stack"},
	}
}

type StackResource struct {
	BaseComponent
}

type StackResourceModel struct {
	ID          types.String  `tfsdk:"id"`
	Name        types.String  `tfsdk:"name"`
	Description types.String  `tfsdk:"description"`
	Kits        types.Dynamic `tfsdk:"kits"`  // Kits that compose this stack (list of kit references)
	Files       types.Map     `tfsdk:"files"` // Stack's own files (map keyed by path)
	// Removed OutputPath - stacks now contribute files to project, not separate directories
}

func (r *StackResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stack"
}

// Configure adds the provider configured data to the resource
func (r *StackResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.BaseComponent.Configure(ctx, req, resp)
}

func (r *StackResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Stack resource for tofukit",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the stack",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the stack",
				Optional:            true,
			},
			"kits": schema.DynamicAttribute{
				MarkdownDescription: "List of kit references to include in the stack (e.g., [tofukit_language.python, tofukit_framework.click])",
				Optional:            true,
			},
			"files": schemas.GetFilesMapAttribute(),
		},
	}
}

func (r *StackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StackResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("stack.%s", data.Name.ValueString()))

	// Stacks no longer create files directly - they only contribute files to the project
	// The project resource will handle merging and writing all files with proper precedence

	fileCount := 0
	if !data.Files.IsNull() && !data.Files.IsUnknown() {
		fileCount = len(data.Files.Elements())
	}

	tflog.Info(ctx, "Created stack resource", map[string]interface{}{
		"stack_id":   data.ID.ValueString(),
		"file_count": fileCount,
	})
	r.SaveToRegistry(ctx, data.ID.ValueString(), data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save to registry so it's available for other resources
	// This is important when resources already exist in state
	fileCount := 0
	if !data.Files.IsNull() && !data.Files.IsUnknown() {
		fileCount = len(data.Files.Elements())
	}

	tflog.Info(ctx, "Stack Read: Saving to registry", map[string]interface{}{
		"stack_id":   data.ID.ValueString(),
		"file_count": fileCount,
	})

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data StackResourceModel
	var state StackResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve computed fields from state
	data.ID = state.ID

	// Stacks no longer manage files directly - just save updated file configuration
	// The project resource will handle merging and applying changes

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StackResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Stacks no longer manage files directly - just remove from registry
	// The project resource will handle cleaning up files when it's deleted

	r.RemoveFromRegistry(ctx, data.ID.ValueString())
}
