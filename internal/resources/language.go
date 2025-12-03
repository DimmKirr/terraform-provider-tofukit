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

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &LanguageResource{}
var _ resource.ResourceWithImportState = &LanguageResource{}

func NewLanguageResource() resource.Resource {
	return &LanguageResource{
		BaseComponent: BaseComponent{Kind: "language"},
	}
}

// LanguageResource defines the resource implementation
type LanguageResource struct {
	BaseComponent
}

// LanguageResourceModel describes the resource data model
type LanguageResourceModel struct {
	ID           types.String               `tfsdk:"id"`
	Name         types.String               `tfsdk:"name"`
	Link         types.String               `tfsdk:"link"`
	Description  types.String               `tfsdk:"description"`
	Version      types.String               `tfsdk:"version"`
	Requirements []schemas.RequirementModel `tfsdk:"requirements"`
}

func (r *LanguageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_language"
}

func (r *LanguageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := schemas.GetBaseComponentAttributes()
	attrs["requirements"] = schemas.GetRequirementsListAttribute()

	resp.Schema = schema.Schema{
		MarkdownDescription: `Language resource for representing programming languages and runtimes.

**Examples:** Python 3.12, Go 1.23, Node.js 20, Java 21, Rust 1.75, Ruby 3.3`,
		Attributes: attrs,
	}
}

func (r *LanguageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	fmt.Printf("🔧 DEBUG: LanguageResource.Create called\n")
	var data LanguageResourceModel

	// Read OpenTofu plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		fmt.Printf("🔧 DEBUG: LanguageResource.Create failed with diagnostics error\n")
		return
	}

	// Generate ID from name
	fmt.Printf("🔧 DEBUG: LanguageResource.Create - setting ID for: %s\n", data.Name.ValueString())
	data.ID = types.StringValue(fmt.Sprintf("language.%s", data.Name.ValueString()))
	data.Link = types.StringValue(fmt.Sprintf("tofukit://kit/language/%s", data.Name.ValueString()))

	// Log the creation
	tflog.Trace(ctx, fmt.Sprintf("created language resource: %s (link: %s)", data.ID.ValueString(), data.Link.ValueString()))

	// Save to registry
	fmt.Printf("🔧 DEBUG: LanguageResource.Create - saving to registry: %s\n", data.ID.ValueString())
	r.SaveToRegistry(ctx, data.ID.ValueString(), data)

	// Save data into OpenTofu state
	fmt.Printf("🔧 DEBUG: LanguageResource.Create - saving to state\n")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	fmt.Printf("🔧 DEBUG: LanguageResource.Create completed for: %s\n", data.Name.ValueString())
}

func (r *LanguageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LanguageResourceModel

	// Read OpenTofu prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// In a real provider, we might check if the resource still exists
	// For now, we just return the state as-is

	// Save updated data into OpenTofu state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LanguageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LanguageResourceModel

	// Read OpenTofu plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Recompute link in case name changed
	data.Link = types.StringValue(fmt.Sprintf("tofukit://kit/language/%s", data.Name.ValueString()))

	// Update the registry
	r.SaveToRegistry(ctx, data.ID.ValueString(), data)

	// Save updated data into OpenTofu state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LanguageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LanguageResourceModel

	// Read OpenTofu prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Remove from registry
	r.RemoveFromRegistry(ctx, data.ID.ValueString())
}

func (r *LanguageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by name
	data := LanguageResourceModel{
		ID:   types.StringValue(fmt.Sprintf("language.%s", req.ID)),
		Name: types.StringValue(req.ID),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
