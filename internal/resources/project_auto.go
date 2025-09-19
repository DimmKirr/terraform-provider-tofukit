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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// NewProjectResourceAuto creates the auto-calculating project resource
func NewProjectResourceAuto() resource.Resource {
	return &ProjectResourceAuto{}
}

type ProjectResourceAuto struct {
	BaseComponent
}

// ProjectModelAuto has computed kits and depends_on_refs
type ProjectModelAuto struct {
	ID           types.String               `tfsdk:"id"`
	Name         types.String               `tfsdk:"name"`
	Description  types.String               `tfsdk:"description"`
	Version      types.String               `tfsdk:"version"`
	Requirements []schemas.RequirementModel `tfsdk:"requirement"`

	// These are COMPUTED attributes - automatically calculated
	DependsOnRefs types.List `tfsdk:"depends_on_refs"`
	Kits          types.Map  `tfsdk:"kits"`
}

func (r *ProjectResourceAuto) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *ProjectResourceAuto) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Project resource that automatically tracks all dependencies and their kits",
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
				MarkdownDescription: "Automatically computed list of all dependencies (transitive closure)",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"kits": schema.MapAttribute{
				MarkdownDescription: "Automatically collected kits from all dependencies with their full details",
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

func (r *ProjectResourceAuto) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProjectModelAuto

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("project.%s", data.Name.ValueString()))

	// For now, initialize empty computed attributes
	// In a real implementation, these would be populated from Terraform's dependency graph
	// or through a custom provider mechanism that tracks resource creation order

	// Initialize empty list for depends_on_refs
	emptyList, _ := types.ListValue(types.StringType, []attr.Value{})
	data.DependsOnRefs = emptyList

	// Initialize empty map for kits
	emptyMap, _ := types.MapValue(
		types.ObjectType{
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
		map[string]attr.Value{},
	)
	data.Kits = emptyMap

	// Write output file if configured
	r.writeOutputFile(ctx, data)

	tflog.Trace(ctx, fmt.Sprintf("created project resource: %s", data.ID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceAuto) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectModelAuto
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// In a real implementation, we would recalculate dependencies here
	// based on the current state of all resources

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceAuto) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectModelAuto
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write output file if configured
	r.writeOutputFile(ctx, data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceAuto) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Nothing to do on delete
}

// writeOutputFile writes the complete project context to a JSON file
func (r *ProjectResourceAuto) writeOutputFile(ctx context.Context, data ProjectModelAuto) {
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
			"name":        data.Name.ValueString(),
			"description": data.Description.ValueString(),
			"version":     data.Version.ValueString(),
		},
		"requirements": []map[string]interface{}{},
		"depends_on":   []string{},
		"kits":         map[string]interface{}{},
	}

	// Add project requirements
	for _, req := range data.Requirements {
		reqData := map[string]interface{}{
			"name": req.Name.ValueString(),
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

	// Add computed dependencies if available
	if !data.DependsOnRefs.IsNull() && !data.DependsOnRefs.IsUnknown() {
		deps := []string{}
		for _, dep := range data.DependsOnRefs.Elements() {
			if depStr, ok := dep.(types.String); ok && !depStr.IsNull() {
				deps = append(deps, depStr.ValueString())
			}
		}
		outputData["depends_on"] = deps
	}

	// Add kits if available
	if !data.Kits.IsNull() && !data.Kits.IsUnknown() {
		kitsMap := data.Kits.Elements()
		kitsOutput := make(map[string]interface{})

		for key, kitValue := range kitsMap {
			if kitObj, ok := kitValue.(types.Object); ok {
				kitAttrs := kitObj.Attributes()
				kitData := r.extractKitData(kitAttrs)
				if kitData != nil {
					kitsOutput[key] = kitData
				}
			}
		}
		outputData["kits"] = kitsOutput
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

	tflog.Info(ctx, fmt.Sprintf("Wrote project output to %s", filename))
}

// extractKitData extracts kit data from attributes
func (r *ProjectResourceAuto) extractKitData(attrs map[string]attr.Value) map[string]interface{} {
	kitData := map[string]interface{}{}

	// Extract basic fields
	if id, ok := attrs["id"].(types.String); ok && !id.IsNull() {
		kitData["id"] = id.ValueString()
	}
	if t, ok := attrs["type"].(types.String); ok && !t.IsNull() {
		kitData["type"] = t.ValueString()
	}
	if name, ok := attrs["name"].(types.String); ok && !name.IsNull() {
		kitData["name"] = name.ValueString()
	}
	if desc, ok := attrs["description"].(types.String); ok && !desc.IsNull() {
		kitData["description"] = desc.ValueString()
	}
	if ver, ok := attrs["version"].(types.String); ok && !ver.IsNull() {
		kitData["version"] = ver.ValueString()
	}

	// Extract requirements if present
	if reqList, ok := attrs["requirements"].(types.List); ok && !reqList.IsNull() {
		requirements := []map[string]interface{}{}

		for _, reqElem := range reqList.Elements() {
			if reqObj, ok := reqElem.(types.Object); ok {
				reqAttrs := reqObj.Attributes()
				reqData := map[string]interface{}{}

				if name, ok := reqAttrs["name"].(types.String); ok && !name.IsNull() {
					reqData["name"] = name.ValueString()
				}

				// Extract instructions
				if instList, ok := reqAttrs["instructions"].(types.List); ok && !instList.IsNull() {
					instructions := []string{}
					for _, inst := range instList.Elements() {
						if instStr, ok := inst.(types.String); ok && !instStr.IsNull() {
							instructions = append(instructions, instStr.ValueString())
						}
					}
					reqData["instructions"] = instructions
				}

				// Extract verification
				if verObj, ok := reqAttrs["verification"].(types.Object); ok && !verObj.IsNull() {
					verAttrs := verObj.Attributes()
					verification := map[string]string{}

					if cmd, ok := verAttrs["command"].(types.String); ok && !cmd.IsNull() {
						verification["command"] = cmd.ValueString()
					}
					if exp, ok := verAttrs["expect"].(types.String); ok && !exp.IsNull() {
						verification["expect"] = exp.ValueString()
					}

					reqData["verification"] = verification
				}

				requirements = append(requirements, reqData)
			}
		}

		kitData["requirements"] = requirements
	}

	return kitData
}
