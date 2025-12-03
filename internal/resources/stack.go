package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	Link        types.String  `tfsdk:"link"`
	Description types.String  `tfsdk:"description"`
	Kits        types.Dynamic `tfsdk:"kits"`     // Kits that compose this stack (list of kit references)
	Features    types.Dynamic `tfsdk:"features"` // Features that compose this stack (list of feature references)
	Files       types.Map     `tfsdk:"files"`    // Stack's own files (map keyed by path)
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
		MarkdownDescription: `Stack resource for reusable collections of files, kits, and features that can be composed into projects.

**Examples:** Python Flask app stack, Go CLI app stack, React frontend stack, Node.js API stack, Django REST API stack`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the stack",
				Required:            true,
			},
			"link": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URI link to this resource for cross-referencing (e.g., tofukit://stack/name)",
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the stack",
				Optional:            true,
			},
			"kits": schema.DynamicAttribute{
				MarkdownDescription: "List of kit references to include in the stack (e.g., [tofukit_language.python, tofukit_library.click])",
				Optional:            true,
			},
			"features": schema.DynamicAttribute{
				MarkdownDescription: "List of feature references to include in the stack (e.g., [tofukit_feature.readme, tofukit_feature.version_command])",
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
	data.Link = types.StringValue(fmt.Sprintf("tofukit://stack/%s", data.Name.ValueString()))

	// Stacks no longer create files directly - they only contribute files to the project
	// The project resource will handle merging and writing all files with proper precedence

	fileCount := 0
	if !data.Files.IsNull() && !data.Files.IsUnknown() {
		fileCount = len(data.Files.Elements())
	}

	tflog.Info(ctx, "Created stack resource", map[string]interface{}{
		"stack_id":   data.ID.ValueString(),
		"link":       data.Link.ValueString(),
		"file_count": fileCount,
	})

	// Initialize hash fields to null for registry-only resources (no actual files created)
	if !data.Files.IsNull() {
		data.Files = r.initializeFileHashFields(ctx, data.Files)
	}

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
	data.Link = types.StringValue(fmt.Sprintf("tofukit://stack/%s", data.Name.ValueString()))

	// Stacks no longer manage files directly - just save updated file configuration
	// The project resource will handle merging and applying changes

	// Initialize hash fields to null for registry-only resources (no actual files created)
	if !data.Files.IsNull() {
		data.Files = r.initializeFileHashFields(ctx, data.Files)
	}

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

// initializeFileHashFields sets hash fields to null for registry-only resources
// These resources don't create actual files, so hash fields should be null
func (r *StackResource) initializeFileHashFields(ctx context.Context, filesMap types.Map) types.Map {
	if filesMap.IsNull() || filesMap.IsUnknown() {
		return filesMap
	}

	// Extract files map
	filesElements := filesMap.Elements()
	newFilesMap := make(map[string]attr.Value)

	for path, fileValue := range filesElements {
		fileObj, ok := fileValue.(types.Object)
		if !ok {
			newFilesMap[path] = fileValue
			continue
		}

		// Get file attributes
		attrs := fileObj.Attributes()

		// Set hash fields to null
		attrs["content_hash"] = types.StringNull()
		attrs["file_hash"] = types.StringNull()
		attrs["file_modtime"] = types.StringNull()

		// Create new object with updated attributes
		newFileObj, _ := types.ObjectValue(fileObj.AttributeTypes(ctx), attrs)
		newFilesMap[path] = newFileObj
	}

	// Create new map
	newMap, _ := types.MapValue(filesMap.ElementType(ctx), newFilesMap)
	return newMap
}
