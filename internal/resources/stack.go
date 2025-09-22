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
	ID          types.String            `tfsdk:"id"`
	Name        types.String            `tfsdk:"name"`
	Description types.String            `tfsdk:"description"`
	Kits        types.Map               `tfsdk:"kits"`     // Kits that compose this stack
	Scaffolds   []schemas.ScaffoldModel `tfsdk:"scaffold"` // Stack's own scaffolds
	// Removed OutputPath - stacks now contribute scaffolds to project, not separate directories
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
			"kits": schema.MapAttribute{
				MarkdownDescription: "Map of kits that compose this stack",
				Optional:            true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":          types.StringType,
						"type":        types.StringType,
						"name":        types.StringType,
						"description": types.StringType,
						"version":     types.StringType,
						"requirements": types.ListType{
							ElemType: types.ObjectType{
								AttrTypes: map[string]attr.Type{
									"name": types.StringType,
									"instructions": types.ListType{
										ElemType: types.StringType,
									},
									"verification": types.ObjectType{
										AttrTypes: map[string]attr.Type{
											"command": types.StringType,
											"expect":  types.StringType,
										},
									},
								},
							},
						},
					},
				},
			},
		},

		Blocks: map[string]schema.Block{
			"scaffold": schemas.GetScaffoldBlock(),
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

	// Stacks no longer create files directly - they only contribute scaffolds to the project
	// The project resource will handle merging and writing all scaffolds with proper precedence

	// Debug: Log scaffolds in Create
	for i, scaffold := range data.Scaffolds {
		logData := map[string]interface{}{
			"stack_id":         data.ID.ValueString(),
			"index":            i,
			"path":             scaffold.Path.ValueString(),
			"has_verification": scaffold.Verification != nil,
		}

		if scaffold.Verification != nil {
			logData["verification_command"] = scaffold.Verification.Command.ValueString()
			if !scaffold.Verification.Expect.IsNull() {
				logData["verification_expect"] = scaffold.Verification.Expect.ValueString()
			}
		}

		tflog.Info(ctx, "Stack scaffold in Create", logData)
	}

	tflog.Trace(ctx, fmt.Sprintf("created stack resource: %s", data.ID.ValueString()))
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
	tflog.Info(ctx, "Stack Read: Saving to registry", map[string]interface{}{
		"stack_id":       data.ID.ValueString(),
		"scaffold_count": len(data.Scaffolds),
	})

	// Debug: Print scaffolds with verification details
	for i, scaffold := range data.Scaffolds {
		logData := map[string]interface{}{
			"stack_id":         data.ID.ValueString(),
			"index":            i,
			"path":             scaffold.Path.ValueString(),
			"content_length":   len(scaffold.Content.ValueString()),
			"has_verification": scaffold.Verification != nil,
		}

		if scaffold.Verification != nil {
			logData["verification_command"] = scaffold.Verification.Command.ValueString()
			if !scaffold.Verification.Expect.IsNull() {
				logData["verification_expect"] = scaffold.Verification.Expect.ValueString()
			}
		}

		tflog.Info(ctx, "Stack scaffold in Read", logData)
	}

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

	// Stacks no longer manage files directly - just save updated scaffold configuration
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
