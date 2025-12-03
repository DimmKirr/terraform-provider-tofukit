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

// ComponentResourceModel is the common model for all component types
type ComponentResourceModel struct {
	ID           types.String               `tfsdk:"id"`
	Name         types.String               `tfsdk:"name"`
	Link         types.String               `tfsdk:"link"`
	Description  types.String               `tfsdk:"description"`
	Version      types.String               `tfsdk:"version"`
	Requirements []schemas.RequirementModel `tfsdk:"requirements"`
	Files        types.Map                  `tfsdk:"files"` // Files that this kit provides (map keyed by path)
}

// Generic component resource that can be used for all component types
type ComponentResource struct {
	BaseComponent
}

func (r *ComponentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.Kind
}

func (r *ComponentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// Get base attributes and add files and requirements
	attributes := schemas.GetBaseComponentAttributes()
	attributes["files"] = schemas.GetFilesMapAttribute()
	attributes["requirements"] = schemas.GetRequirementsListAttribute()

	// Generate kind-specific descriptions with examples
	var description string
	switch r.Kind {
	case "library":
		description = `Library resource for representing code dependencies and packages.

**Examples:** pytest (Python testing), requests (HTTP client), gorm (Go ORM), cobra (Go CLI), viper (Go config), React (JavaScript UI), lodash (JavaScript utils)`
	case "tool":
		description = `Tool resource for representing development tools and CLI utilities.

**Examples:** uv (Python package manager), npm (Node package manager), Docker, kubectl, terraform, git, make, gcc, protoc`
	case "language":
		description = `Language resource for representing programming languages and runtimes.

**Examples:** Python 3.12, Go 1.23, Node.js 20, Java 21, Rust 1.75, Ruby 3.3`
	default:
		description = fmt.Sprintf("%s component for tofukit", r.Kind)
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: description,
		Attributes:          attributes,
	}
}

func (r *ComponentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ComponentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate ID from kind and name
	data.ID = types.StringValue(fmt.Sprintf("%s.%s", r.Kind, data.Name.ValueString()))
	// Generate link URI: tofukit://kit/<kind>/<name>
	data.Link = types.StringValue(fmt.Sprintf("tofukit://kit/%s/%s", r.Kind, data.Name.ValueString()))

	tflog.Trace(ctx, fmt.Sprintf("created %s resource: %s (link: %s)", r.Kind, data.ID.ValueString(), data.Link.ValueString()))

	// Initialize hash fields to null for registry-only resources (no actual files created)
	if !data.Files.IsNull() {
		data.Files = r.initializeFileHashFields(ctx, data.Files)
	}

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ComponentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ComponentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ComponentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ComponentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Recompute link in case name changed
	data.Link = types.StringValue(fmt.Sprintf("tofukit://kit/%s/%s", r.Kind, data.Name.ValueString()))

	// Initialize hash fields to null for registry-only resources (no actual files created)
	if !data.Files.IsNull() {
		data.Files = r.initializeFileHashFields(ctx, data.Files)
	}

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ComponentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ComponentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.RemoveFromRegistry(ctx, data.ID.ValueString())
}

func (r *ComponentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := ComponentResourceModel{
		ID:   types.StringValue(fmt.Sprintf("%s.%s", r.Kind, req.ID)),
		Name: types.StringValue(req.ID),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Now create specific resources for each kind

func NewLibraryResource() resource.Resource {
	return &ComponentResource{
		BaseComponent: BaseComponent{Kind: "library"},
	}
}

func NewToolResource() resource.Resource {
	return &ComponentResource{
		BaseComponent: BaseComponent{Kind: "tool"},
	}
}

// NewIntegrationResource moved to integration.go - using dedicated implementation instead of generic ComponentResource

// initializeFileHashFields sets hash fields to null for registry-only resources
// These resources don't create actual files, so hash fields should be null
func (r *ComponentResource) initializeFileHashFields(ctx context.Context, filesMap types.Map) types.Map {
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
