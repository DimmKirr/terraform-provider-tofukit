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

// ComponentResourceModel is the common model for all component types
type ComponentResourceModel struct {
	ID            types.String               `tfsdk:"id"`
	Name          types.String               `tfsdk:"name"`
	Description   types.String               `tfsdk:"description"`
	Version       types.String               `tfsdk:"version"`
	DependsOnRefs []types.String             `tfsdk:"depends_on_refs"`
	Requirements  []schemas.RequirementModel `tfsdk:"requirement"`
}

// Generic component resource that can be used for all component types
type ComponentResource struct {
	BaseComponent
}

func (r *ComponentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.Kind
}

func (r *ComponentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("%s component for tofukit", r.Kind),

		Attributes: schemas.GetBaseComponentAttributes(),

		Blocks: map[string]schema.Block{
			"requirement": schemas.GetRequirementBlock(),
		},
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

	tflog.Trace(ctx, fmt.Sprintf("created %s resource: %s", r.Kind, data.ID.ValueString()))
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

func NewFrameworkResource() resource.Resource {
	return &ComponentResource{
		BaseComponent: BaseComponent{Kind: "framework"},
	}
}

func NewToolResource() resource.Resource {
	return &ComponentResource{
		BaseComponent: BaseComponent{Kind: "tool"},
	}
}

func NewMethodologyResource() resource.Resource {
	return &ComponentResource{
		BaseComponent: BaseComponent{Kind: "methodology"},
	}
}

func NewStyleResource() resource.Resource {
	return &ComponentResource{
		BaseComponent: BaseComponent{Kind: "style"},
	}
}

func NewInfrastructureResource() resource.Resource {
	return &ComponentResource{
		BaseComponent: BaseComponent{Kind: "infrastructure"},
	}
}

func NewIntegrationResource() resource.Resource {
	return &ComponentResource{
		BaseComponent: BaseComponent{Kind: "integration"},
	}
}
