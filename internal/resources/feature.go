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

var _ resource.Resource = &FeatureResource{}

func NewFeatureResource() resource.Resource {
	return &FeatureResource{
		BaseComponent: BaseComponent{Kind: "feature"},
	}
}

type FeatureResource struct {
	BaseComponent
}

type FeatureResourceModel struct {
	ID            types.String                `tfsdk:"id"`
	Name          types.String                `tfsdk:"name"`
	Link          types.String                `tfsdk:"link"`
	Description   types.String                `tfsdk:"description"`
	Prompt        types.String                `tfsdk:"prompt"`       // What the feature does (LLM-facing requirement)
	Constraints   types.List                  `tfsdk:"constraints"`  // Implementation constraints (what NOT to do)
	Requirements  []schemas.RequirementModel  `tfsdk:"requirements"` // Now uses common type!
	Files         types.Map                   `tfsdk:"files"`
	Kits          types.Dynamic               `tfsdk:"kits"`
	Verifications []schemas.VerificationModel `tfsdk:"verifications"`
}

func (r *FeatureResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_feature"
}

// Configure adds the provider configured data to the resource
func (r *FeatureResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.BaseComponent.Configure(ctx, req, resp)
}

func (r *FeatureResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Feature resource for tofukit - enables reusable feature definitions (capabilities) across projects",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier (format: feature.<name>)",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Unique name for this feature resource (used for registry lookup)",
				Required:            true,
			},
			"link": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URI link to this resource for cross-referencing (e.g., tofukit://feature/name)",
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of what this feature does",
				Optional:            true,
			},
			"prompt": schema.StringAttribute{
				MarkdownDescription: "What the feature does (LLM-facing requirement)",
				Required:            true,
			},
			"constraints": schema.ListAttribute{
				MarkdownDescription: "Implementation constraints (what NOT to do)",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"requirements": schemas.GetRequirementsListAttribute(), // Now uses common type!
			"files":        schemas.GetFilesMapAttribute(),
			"kits": schema.DynamicAttribute{
				MarkdownDescription: "Kit dependencies for this feature",
				Optional:            true,
			},
			"verifications": schema.ListNestedAttribute{
				MarkdownDescription: "Verification commands for this feature",
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
		},
	}
}

// ModifyPlan sets the ID during plan phase since it's predictable from name
func (r *FeatureResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip if this is a delete operation
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan FeatureResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If ID is unknown and name is known, compute the ID now
	if plan.ID.IsUnknown() && !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		plan.ID = types.StringValue(fmt.Sprintf("feature.%s", plan.Name.ValueString()))
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

		tflog.Info(ctx, "ModifyPlan: Set feature ID during plan", map[string]interface{}{
			"feature_id": plan.ID.ValueString(),
			"name":       plan.Name.ValueString(),
		})
	}
}

func (r *FeatureResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FeatureResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate: at least one requirement must be specified
	if len(data.Requirements) == 0 {
		resp.Diagnostics.AddError(
			"Invalid Feature Resource Configuration",
			"Feature resource must have at least one 'requirement' that describes what the feature does.",
		)
		return
	}

	// Validate: at least one of files, kits, requirements, or verifications
	hasFiles := !data.Files.IsNull() && len(data.Files.Elements()) > 0
	hasKits := !data.Kits.IsNull()
	hasRequirements := len(data.Requirements) > 0
	hasVerifications := len(data.Verifications) > 0

	if !hasFiles && !hasKits && !hasRequirements && !hasVerifications {
		resp.Diagnostics.AddError(
			"Invalid Feature Resource Configuration",
			"Feature resource must have at least one of: 'files', 'kits', or 'verifications'.",
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("feature.%s", data.Name.ValueString()))
	data.Link = types.StringValue(fmt.Sprintf("tofukit://feature/%s", data.Name.ValueString()))

	tflog.Info(ctx, "Created feature resource", map[string]interface{}{
		"feature_id":        data.ID.ValueString(),
		"name":              data.Name.ValueString(),
		"link":              data.Link.ValueString(),
		"has_requirements":  len(data.Requirements),
		"has_files":         hasFiles,
		"has_kits":          hasKits,
		"has_verifications": hasVerifications,
	})

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FeatureResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FeatureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save to registry so it's available for other resources
	// This is important when resources already exist in state
	tflog.Info(ctx, "Feature Read: Saving to registry", map[string]interface{}{
		"feature_id": data.ID.ValueString(),
		"name":       data.Name.ValueString(),
	})

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FeatureResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FeatureResourceModel
	var state FeatureResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate: at least one requirement must be specified
	if len(data.Requirements) == 0 {
		resp.Diagnostics.AddError(
			"Invalid Feature Resource Configuration",
			"Feature resource must have at least one 'requirement' that describes what the feature does.",
		)
		return
	}

	// Validate: at least one of files, kits, requirements, or verifications
	hasFiles := !data.Files.IsNull() && len(data.Files.Elements()) > 0
	hasKits := !data.Kits.IsNull()
	hasRequirements := len(data.Requirements) > 0
	hasVerifications := len(data.Verifications) > 0

	if !hasFiles && !hasKits && !hasRequirements && !hasVerifications {
		resp.Diagnostics.AddError(
			"Invalid Feature Resource Configuration",
			"Feature resource must have at least one of: 'files', 'kits', or 'verifications'.",
		)
		return
	}

	// Preserve computed fields from state
	data.ID = state.ID
	data.Link = types.StringValue(fmt.Sprintf("tofukit://feature/%s", data.Name.ValueString()))

	tflog.Info(ctx, "Updated feature resource", map[string]interface{}{
		"feature_id":        data.ID.ValueString(),
		"name":              data.Name.ValueString(),
		"link":              data.Link.ValueString(),
		"has_requirements":  len(data.Requirements),
		"has_files":         hasFiles,
		"has_kits":          hasKits,
		"has_verifications": hasVerifications,
	})

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *FeatureResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FeatureResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting feature resource from registry", map[string]interface{}{
		"feature_id": data.ID.ValueString(),
		"name":       data.Name.ValueString(),
	})

	r.RemoveFromRegistry(ctx, data.ID.ValueString())
}
