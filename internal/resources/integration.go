package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// IntegrationResourceModel represents the integration resource model
type IntegrationResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Link        types.String `tfsdk:"link"`
	Type        types.String `tfsdk:"type"`
	Description types.String `tfsdk:"description"`
	BaseURL     types.String `tfsdk:"base_url"`
	DocsURL     types.String `tfsdk:"docs_url"`
	Version     types.String `tfsdk:"version"`
	Metadata    types.Map    `tfsdk:"metadata"`
}

// IntegrationResource is the resource implementation for external service integrations
type IntegrationResource struct {
	BaseComponent
}

// Ensure IntegrationResource implements the Resource interface
var _ resource.Resource = &IntegrationResource{}

// NewIntegrationResource creates a new integration resource
func NewIntegrationResource() resource.Resource {
	return &IntegrationResource{
		BaseComponent: BaseComponent{Kind: "integration"},
	}
}

func (r *IntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

// Configure adds the provider configured data to the resource
func (r *IntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.BaseComponent.Configure(ctx, req, resp)
}

func (r *IntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Integration resource for representing external API/service dependencies",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier (format: integration.<name>)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique name of the integration",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"link": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URI link to this resource for cross-referencing (format: tofukit://integration/<name>)",
			},
			"type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Integration type: rest_api, graphql_api, grpc_api, webhook, websocket, saas, cloud_service",
				Validators: []validator.String{
					stringvalidator.OneOf("rest_api", "graphql_api", "grpc_api", "webhook", "websocket", "saas", "cloud_service"),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Description of the integration",
			},
			"base_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Base URL for the API endpoint",
			},
			"docs_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "URL to integration documentation",
			},
			"version": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "API version (e.g., 'v1', '2.0')",
			},
			"metadata": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Additional key-value metadata (auth type, rate limits, etc.)",
			},
		},
	}
}

func (r *IntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IntegrationResourceModel

	// Read plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Compute ID: integration.<name>
	data.ID = types.StringValue(fmt.Sprintf("integration.%s", data.Name.ValueString()))

	// Compute link: tofukit://integration/<name>
	data.Link = types.StringValue(fmt.Sprintf("tofukit://integration/%s", data.Name.ValueString()))

	tflog.Trace(ctx, fmt.Sprintf("created integration resource: %s (link: %s)", data.ID.ValueString(), data.Link.ValueString()))

	// Save to global registry
	r.SaveToRegistry(ctx, data.Name.ValueString(), &data)

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IntegrationResourceModel

	// Read current state
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Integration is a registry-only resource, no external state to refresh
	// Just save the current state back
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IntegrationResourceModel

	// Read plan data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Recompute link in case name changed (though name requires replace)
	data.Link = types.StringValue(fmt.Sprintf("tofukit://integration/%s", data.Name.ValueString()))

	tflog.Trace(ctx, fmt.Sprintf("updated integration resource: %s", data.ID.ValueString()))

	// Update registry
	r.SaveToRegistry(ctx, data.Name.ValueString(), &data)

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IntegrationResourceModel

	// Read current state
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("deleting integration resource: %s", data.ID.ValueString()))

	// Remove from registry
	r.RemoveFromRegistry(ctx, data.Name.ValueString())
}
