package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// NewProjectResourceSimplified creates a simplified project resource
func NewProjectResourceSimplified() resource.Resource {
	return &ProjectResourceSimplified{}
}

type ProjectResourceSimplified struct {
	BaseComponent
}

// ProjectModelSimplified includes a kits attribute that's manually populated
type ProjectModelSimplified struct {
	ID            types.String               `tfsdk:"id"`
	Name          types.String               `tfsdk:"name"`
	Description   types.String               `tfsdk:"description"`
	Version       types.String               `tfsdk:"version"`
	DependsOnRefs []types.String             `tfsdk:"depends_on_refs"`
	Requirements  []schemas.RequirementModel `tfsdk:"requirement"`
	Kits          types.Map                  `tfsdk:"kits"`
}

func (r *ProjectResourceSimplified) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResourceSimplified) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Project resource for tofukit that aggregates all component kits",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the project",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the project",
				Optional:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of the project",
				Required:            true,
			},
			"depends_on_refs": schema.ListAttribute{
				MarkdownDescription: "List of resource references this project depends on (e.g., '@tofukit_language.python')",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"kits": schema.MapAttribute{
				MarkdownDescription: "Map of all kits with their details. Keys are kit IDs, values contain kit information",
				Optional:            true,
				Computed:            true,
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
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"requirement": schemas.GetRequirementBlock(),
		},
	}
}

func (r *ProjectResourceSimplified) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectModelSimplified

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("project.%s", data.Name.ValueString()))

	// Write output file if configured
	r.writeOutputFile(ctx, data)

	tflog.Trace(ctx, fmt.Sprintf("created project resource: %s", data.ID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceSimplified) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectModelSimplified
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceSimplified) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectModelSimplified
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write output file if configured
	r.writeOutputFile(ctx, data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceSimplified) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Nothing to do on delete
}

// writeOutputFile writes the project data to an output file if configured
func (r *ProjectResourceSimplified) writeOutputFile(ctx context.Context, data ProjectModelSimplified) {
	// Get output configuration from provider data
	outputPath := ".tofukit"
	outputFormat := "json"

	if provData, ok := r.ProviderData.(interface {
		GetOutputPath() string
		GetOutputFormat() string
	}); ok {
		outputPath = provData.GetOutputPath()
		outputFormat = provData.GetOutputFormat()
	}

	if outputPath == "" {
		return
	}

	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to create output directory: %v", err))
		return
	}

	// Prepare output data
	outputData := map[string]interface{}{
		"project": map[string]interface{}{
			"name":        data.Name.ValueString(),
			"description": data.Description.ValueString(),
			"version":     data.Version.ValueString(),
		},
		"depends_on": []string{},
		"kits":       map[string]interface{}{},
	}

	// Add dependencies
	for _, ref := range data.DependsOnRefs {
		if !ref.IsNull() && !ref.IsUnknown() {
			outputData["depends_on"] = append(outputData["depends_on"].([]string), ref.ValueString())
		}
	}

	// Add kits if present
	if !data.Kits.IsNull() && !data.Kits.IsUnknown() {
		kitsMap := data.Kits.Elements()
		kitsOutput := make(map[string]interface{})
		for key, kitValue := range kitsMap {
			if kitObj, ok := kitValue.(types.Object); ok {
				kitAttrs := kitObj.Attributes()
				kitData := map[string]interface{}{
					"id":          r.getStringValue(kitAttrs["id"]),
					"type":        r.getStringValue(kitAttrs["type"]),
					"name":        r.getStringValue(kitAttrs["name"]),
					"description": r.getStringValue(kitAttrs["description"]),
					"version":     r.getStringValue(kitAttrs["version"]),
				}

				// Add requirements if present
				if reqList, ok := kitAttrs["requirements"].(types.List); ok && !reqList.IsNull() {
					requirements := []map[string]interface{}{}
					for _, reqElem := range reqList.Elements() {
						if reqObj, ok := reqElem.(types.Object); ok {
							reqAttrs := reqObj.Attributes()
							reqData := map[string]interface{}{
								"name": r.getStringValue(reqAttrs["name"]),
							}

							// Add instructions
							if instList, ok := reqAttrs["instructions"].(types.List); ok && !instList.IsNull() {
								instructions := []string{}
								for _, inst := range instList.Elements() {
									if instStr, ok := inst.(types.String); ok {
										instructions = append(instructions, instStr.ValueString())
									}
								}
								reqData["instructions"] = instructions
							}

							// Add verification
							if verObj, ok := reqAttrs["verification"].(types.Object); ok && !verObj.IsNull() {
								verAttrs := verObj.Attributes()
								reqData["verification"] = map[string]string{
									"command": r.getStringValue(verAttrs["command"]),
									"expect":  r.getStringValue(verAttrs["expect"]),
								}
							}

							requirements = append(requirements, reqData)
						}
					}
					kitData["requirements"] = requirements
				}

				kitsOutput[key] = kitData
			}
		}
		outputData["kits"] = kitsOutput
	}

	// Add project requirements
	requirements := []map[string]interface{}{}
	for _, req := range data.Requirements {
		reqData := map[string]interface{}{
			"name": req.Name.ValueString(),
		}

		// Add instructions
		instructions := []string{}
		for _, inst := range req.Instructions {
			instructions = append(instructions, inst.ValueString())
		}
		reqData["instructions"] = instructions

		// Add verification
		if req.Verification != nil {
			reqData["verification"] = map[string]string{
				"command": req.Verification.Command.ValueString(),
				"expect":  req.Verification.Expect.ValueString(),
			}
		}

		requirements = append(requirements, reqData)
	}
	outputData["requirements"] = requirements

	// Write file based on format
	filename := filepath.Join(outputPath, fmt.Sprintf("project-%s.%s", data.Name.ValueString(), outputFormat))

	switch outputFormat {
	case "json":
		data, err := json.MarshalIndent(outputData, "", "  ")
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to marshal JSON: %v", err))
			return
		}
		if err := os.WriteFile(filename, data, 0644); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to write file: %v", err))
			return
		}
	default:
		tflog.Warn(ctx, fmt.Sprintf("Unsupported output format: %s", outputFormat))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Wrote project output to %s", filename))
}

func (r *ProjectResourceSimplified) getStringValue(v attr.Value) string {
	if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
		return s.ValueString()
	}
	return ""
}
