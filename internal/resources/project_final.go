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

// NewProjectResourceFinal creates the final project resource
func NewProjectResourceFinal() resource.Resource {
	return &ProjectResourceFinal{}
}

type ProjectResourceFinal struct {
	BaseComponent
}

// ProjectModelFinal accepts kits as input
type ProjectModelFinal struct {
	ID           types.String               `tfsdk:"id"`
	Name         types.String               `tfsdk:"name"`
	Description  types.String               `tfsdk:"description"`
	Version      types.String               `tfsdk:"version"`
	Requirements []schemas.RequirementModel `tfsdk:"requirement"`
	Kits         types.Map                  `tfsdk:"kits"`
}

func (r *ProjectResourceFinal) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResourceFinal) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Project resource that aggregates all component kits",
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
			"kits": schema.MapAttribute{
				MarkdownDescription: "Map of all component kits with their details",
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

func (r *ProjectResourceFinal) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectModelFinal

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("project.%s", data.Name.ValueString()))

	// Write output file with all kits
	r.writeOutputFile(ctx, data)

	tflog.Trace(ctx, fmt.Sprintf("created project resource: %s", data.ID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceFinal) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectModelFinal
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceFinal) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectModelFinal
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write output file with all kits
	r.writeOutputFile(ctx, data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceFinal) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Nothing to do on delete
}

// writeOutputFile writes the complete project context to a JSON file
func (r *ProjectResourceFinal) writeOutputFile(ctx context.Context, data ProjectModelFinal) {
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

	// Prepare output data structure
	outputData := map[string]interface{}{
		"project": map[string]interface{}{
			"name":         data.Name.ValueString(),
			"description":  data.Description.ValueString(),
			"version":      data.Version.ValueString(),
			"dependencies": []string{}, // Will be populated from kits
		},
		"requirements": []map[string]interface{}{},
		"kits":         map[string]interface{}{},
	}

	// Add project requirements
	for _, req := range data.Requirements {
		reqData := map[string]interface{}{
			"name": req.Name.ValueString(),
		}

		// Add priority if set
		if !req.Priority.IsNull() && !req.Priority.IsUnknown() {
			reqData["priority"] = req.Priority.ValueInt64()
		}

		// Add instructions
		instructions := []string{}
		for _, inst := range req.Instructions {
			if !inst.IsNull() && !inst.IsUnknown() {
				instructions = append(instructions, inst.ValueString())
			}
		}
		reqData["instructions"] = instructions

		// Add verification if present
		if req.Verification != nil {
			reqData["verification"] = map[string]string{
				"command": req.Verification.Command.ValueString(),
				"expect":  req.Verification.Expect.ValueString(),
			}
		}

		outputData["requirements"] = append(outputData["requirements"].([]map[string]interface{}), reqData)
	}

	// Add kits if available
	if !data.Kits.IsNull() && !data.Kits.IsUnknown() {
		kitsMap := data.Kits.Elements()
		kitsOutput := make(map[string]interface{})
		dependencies := []string{}

		for key, kitValue := range kitsMap {
			if kitObj, ok := kitValue.(types.Object); ok && !kitObj.IsNull() {
				kitAttrs := kitObj.Attributes()
				kitData := make(map[string]interface{})

				// Extract basic fields
				if id, ok := kitAttrs["id"].(types.String); ok && !id.IsNull() {
					kitData["id"] = id.ValueString()
				}
				if t, ok := kitAttrs["type"].(types.String); ok && !t.IsNull() {
					kitData["type"] = t.ValueString()
				}
				if name, ok := kitAttrs["name"].(types.String); ok && !name.IsNull() {
					kitData["name"] = name.ValueString()
				}
				if desc, ok := kitAttrs["description"].(types.String); ok && !desc.IsNull() {
					kitData["description"] = desc.ValueString()
				}
				if ver, ok := kitAttrs["version"].(types.String); ok && !ver.IsNull() {
					kitData["version"] = ver.ValueString()
				}

				// Extract requirements
				if reqList, ok := kitAttrs["requirements"].(types.List); ok && !reqList.IsNull() {
					requirements := []map[string]interface{}{}

					for _, reqElem := range reqList.Elements() {
						if reqObj, ok := reqElem.(types.Object); ok && !reqObj.IsNull() {
							reqAttrs := reqObj.Attributes()
							reqData := make(map[string]interface{})

							// Get name
							if name, ok := reqAttrs["name"].(types.String); ok && !name.IsNull() {
								reqData["name"] = name.ValueString()
							}

							// Get priority if set
							if priority, ok := reqAttrs["priority"].(types.Int64); ok && !priority.IsNull() {
								reqData["priority"] = priority.ValueInt64()
							}

							// Get instructions
							if instList, ok := reqAttrs["instructions"].(types.List); ok && !instList.IsNull() {
								instructions := []string{}
								for _, inst := range instList.Elements() {
									if instStr, ok := inst.(types.String); ok && !instStr.IsNull() {
										instructions = append(instructions, instStr.ValueString())
									}
								}
								reqData["instructions"] = instructions
							}

							// Get verification
							if verObj, ok := reqAttrs["verification"].(types.Object); ok && !verObj.IsNull() {
								verAttrs := verObj.Attributes()
								verification := make(map[string]string)

								if cmd, ok := verAttrs["command"].(types.String); ok && !cmd.IsNull() {
									verification["command"] = cmd.ValueString()
								}
								if exp, ok := verAttrs["expect"].(types.String); ok && !exp.IsNull() {
									verification["expect"] = exp.ValueString()
								}

								if len(verification) > 0 {
									reqData["verification"] = verification
								}
							}

							if len(reqData) > 0 {
								requirements = append(requirements, reqData)
							}
						}
					}

					if len(requirements) > 0 {
						kitData["requirements"] = requirements
					}
				}

				if len(kitData) > 0 {
					kitsOutput[key] = kitData
					// Add to dependencies list with @kit prefix
					dependencies = append(dependencies, fmt.Sprintf("@kit.%s", key))
				}
			}
		}
		outputData["kits"] = kitsOutput

		// Update project dependencies
		if projectData, ok := outputData["project"].(map[string]interface{}); ok {
			projectData["dependencies"] = dependencies
		}
	}

	// Write the JSON file
	filename := filepath.Join(outputPath, fmt.Sprintf("project-%s.%s", data.Name.ValueString(), outputFormat))

	jsonData, err := json.MarshalIndent(outputData, "", "  ")
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to marshal JSON: %v", err))
		return
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to write file: %v", err))
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Wrote project output to %s with %d kits", filename, len(outputData["kits"].(map[string]interface{}))))
}
