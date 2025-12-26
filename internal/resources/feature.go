package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	Model         types.String                `tfsdk:"model"`
	Type          types.String                `tfsdk:"type"`
	Description   types.String                `tfsdk:"description"`
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
			"model": schema.StringAttribute{
				MarkdownDescription: "Model to use for this feature in provider/model format (e.g., 'anthropic/claude-haiku-4.5', 'openai/gpt-5.2'). Overrides provider-level model setting. If not specified, uses provider's default model.",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: `Feature type classification:

- **product**: End-user capabilities that deliver direct value - features customers actively use and would pay for
  - *SaaS/Web Apps:* Login, search, dark mode, checkout, in-app help, public API docs, user dashboard
  - *Open Source/CLI:* Installation, CLI commands, usage examples (developers are the users)
  - *Documentation:* In-app help, public API docs, website user guides (customer-facing)

- **platform**: Developer-facing infrastructure and architectural patterns that enable product features
  - *Infrastructure:* CI/CD, Docker, logging, monitoring, 12 Factor architecture
  - *Design Systems:* Design tokens, component libraries, style guides (e.g., button styles, color systems)
  - *Developer Tools:* Code formatters (Black, Prettier), linters, pre-commit hooks
  - *Documentation:* README.md for SaaS repos, architecture diagrams, CONTRIBUTING.md, setup guides (for developers working on codebase)

- **methodology**: Development process approaches and practices (TDD, BDD, Agile, Scrum, pair programming, trunk-based development)`,
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("product", "platform", "methodology"),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of what this feature does",
				Optional:            true,
			},
			"requirements": schemas.GetRequiredRequirementsListAttribute(), // Required for features!
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

	data.ID = types.StringValue(fmt.Sprintf("feature.%s", data.Name.ValueString()))
	data.Link = types.StringValue(fmt.Sprintf("tofukit://feature/%s", data.Name.ValueString()))

	tflog.Info(ctx, "Created feature resource", map[string]interface{}{
		"feature_id":        data.ID.ValueString(),
		"name":              data.Name.ValueString(),
		"link":              data.Link.ValueString(),
		"has_requirements":  len(data.Requirements),
		"has_files":         !data.Files.IsNull() && len(data.Files.Elements()) > 0,
		"has_kits":          !data.Kits.IsNull(),
		"has_verifications": len(data.Verifications) > 0,
	})

	// Initialize hash fields to null for registry-only resources (no actual files created)
	if !data.Files.IsNull() {
		data.Files = r.initializeFileHashFields(ctx, data.Files)
	}

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

	// Preserve computed fields from state
	data.ID = state.ID
	data.Link = types.StringValue(fmt.Sprintf("tofukit://feature/%s", data.Name.ValueString()))

	tflog.Info(ctx, "Updated feature resource", map[string]interface{}{
		"feature_id":        data.ID.ValueString(),
		"name":              data.Name.ValueString(),
		"link":              data.Link.ValueString(),
		"has_requirements":  len(data.Requirements),
		"has_files":         !data.Files.IsNull() && len(data.Files.Elements()) > 0,
		"has_kits":          !data.Kits.IsNull(),
		"has_verifications": len(data.Verifications) > 0,
	})

	// Initialize hash fields to null for registry-only resources (no actual files created)
	if !data.Files.IsNull() {
		data.Files = r.initializeFileHashFields(ctx, data.Files)
	}

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

// initializeFileHashFields sets hash fields to null for registry-only resources
// These resources don't create actual files, so hash fields should be null
func (r *FeatureResource) initializeFileHashFields(ctx context.Context, filesMap types.Map) types.Map {
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
