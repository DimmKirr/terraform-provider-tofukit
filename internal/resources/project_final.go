package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/claude"
	"github.com/tofukit/opentofu-provider-tofukit/internal/scaffolds"
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
	ID                types.String               `tfsdk:"id"`
	Name              types.String               `tfsdk:"name"`
	Description       types.String               `tfsdk:"description"`
	Version           types.String               `tfsdk:"version"`
	Requirements      []schemas.RequirementModel `tfsdk:"requirement"`
	Kits              types.Map                  `tfsdk:"kits"`
	Scaffolds         []schemas.ScaffoldModel    `tfsdk:"scaffold"`
	ExecutionStatus   types.String               `tfsdk:"execution_status"`
	ExecutionStarted  types.String               `tfsdk:"execution_started"`
	ExecutionCompleted types.String               `tfsdk:"execution_completed"`
	ProjectPath       types.String               `tfsdk:"project_path"`
	ExecutionError    types.String               `tfsdk:"execution_error"`
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
			"execution_status": schema.StringAttribute{
				MarkdownDescription: "Status of Claude Code execution (dry_run, pending, running, completed, failed)",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"execution_started": schema.StringAttribute{
				MarkdownDescription: "Timestamp when Claude Code execution started",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"execution_completed": schema.StringAttribute{
				MarkdownDescription: "Timestamp when Claude Code execution completed",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_path": schema.StringAttribute{
				MarkdownDescription: "Path where the project was generated",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"execution_error": schema.StringAttribute{
				MarkdownDescription: "Error message if Claude Code execution failed",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"requirement": schemas.GetRequirementBlock(),
			"scaffold": schemas.GetScaffoldBlock(),
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

	// Get provider configuration
	isDryRun := true
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
		GetOutputPath() string
		GetClaudeHomeDirectory() string
	}); ok {
		isDryRun = provData.GetDryRun()
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
	}

	// Always write the JSON output file first
	outputData := r.buildOutputData(ctx, data)
	r.writeJSONFile(ctx, data, outputData, outputPath)

	// Set initial execution status
	if isDryRun {
		data.ExecutionStatus = types.StringValue("dry_run")
		tflog.Info(ctx, "Project resource created in dry-run mode", map[string]interface{}{
			"project_id": data.ID.ValueString(),
		})
	} else {
		// Execute Claude Code
		data.ExecutionStatus = types.StringValue("pending")
		if err := r.executeClaudeCode(ctx, &data, outputData, outputPath, claudeHomeDir, isDryRun); err != nil {
			// Set error status but don't fail the resource creation
			// This allows users to see the error and potentially retry
			data.ExecutionStatus = types.StringValue("failed")
			data.ExecutionError = types.StringValue(err.Error())
			tflog.Error(ctx, "Claude Code execution failed", map[string]interface{}{
				"project_id": data.ID.ValueString(),
				"error":      err.Error(),
			})
		}
	}


	tflog.Debug(ctx, "About to save state with computed fields", map[string]interface{}{
		"execution_started": data.ExecutionStarted.ValueString(),
		"execution_completed": data.ExecutionCompleted.ValueString(),
		"execution_error": data.ExecutionError.ValueString(),
		"project_path": data.ProjectPath.ValueString(),
		"execution_started_null": data.ExecutionStarted.IsNull(),
		"execution_completed_null": data.ExecutionCompleted.IsNull(),
		"execution_error_null": data.ExecutionError.IsNull(),
		"project_path_null": data.ProjectPath.IsNull(),
		"execution_started_unknown": data.ExecutionStarted.IsUnknown(),
		"execution_completed_unknown": data.ExecutionCompleted.IsUnknown(),
		"execution_error_unknown": data.ExecutionError.IsUnknown(),
		"project_path_unknown": data.ProjectPath.IsUnknown(),
	})

	tflog.Trace(ctx, fmt.Sprintf("created project resource: %s", data.ID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceFinal) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectModelFinal
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get provider configuration
	isDryRun := true
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
		GetOutputPath() string
		GetClaudeHomeDirectory() string
	}); ok {
		isDryRun = provData.GetDryRun()
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
	}

	// If not in dry run mode, check if the generated project still exists
	if !isDryRun && !data.ProjectPath.IsNull() && !data.ProjectPath.IsUnknown() {
		projectPath := data.ProjectPath.ValueString()
		if projectPath != "" {
			executor := claude.NewExecutor(claudeHomeDir, isDryRun)
			if !executor.IsProjectGenerated(ctx, projectPath) {
				// Project no longer exists, update the state
				tflog.Warn(ctx, "Generated project no longer exists", map[string]interface{}{
					"project_id":   data.ID.ValueString(),
					"project_path": projectPath,
				})

				// Check if we should try to reload execution status from metadata
				if status, err := executor.LoadExecutionStatus(ctx, outputPath); err == nil && status != nil {
					// Update state from metadata if available
					data.ExecutionStatus = types.StringValue(status.State)
					data.ExecutionStarted = types.StringValue(status.StartedAt)
					if status.CompletedAt != "" {
						data.ExecutionCompleted = types.StringValue(status.CompletedAt)
					}
					if status.Error != "" {
						data.ExecutionError = types.StringValue(status.Error)
					}
				} else {
					// No metadata available, mark as potentially inconsistent
					data.ExecutionStatus = types.StringValue("unknown")
				}
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceFinal) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectModelFinal
	var state ProjectModelFinal

	// Get the planned data
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the current state to preserve execution fields
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get provider configuration for output path
	outputPath := ".tofukit"
	isDryRun := true
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
		GetOutputPath() string
		GetClaudeHomeDirectory() string
	}); ok {
		outputPath = provData.GetOutputPath()
		isDryRun = provData.GetDryRun()
	}

	// Handle scaffold changes if not in dry-run mode
	if !isDryRun && state.ProjectPath.ValueString() != "" {
		projectDir := state.ProjectPath.ValueString()
		// Convert relative path to absolute
		if !filepath.IsAbs(projectDir) {
			cwd, err := os.Getwd()
			if err == nil {
				projectDir = filepath.Join(cwd, projectDir)
			}
		}

		tflog.Debug(ctx, "Processing scaffold changes", map[string]interface{}{
			"project_dir": projectDir,
			"old_scaffolds": len(state.Scaffolds),
			"new_scaffolds": len(data.Scaffolds),
		})

		scaffoldMgr := scaffolds.NewManager(projectDir)

		// Process scaffold changes (removes deleted scaffolds, creates new ones)
		if err := scaffoldMgr.ProcessScaffoldChanges(ctx, state.Scaffolds, data.Scaffolds); err != nil {
			tflog.Error(ctx, "Failed to process scaffold changes", map[string]interface{}{
				"error": err.Error(),
			})
			// Note: We continue even if scaffold changes fail to maintain state consistency
		}
	}

	// Always write the JSON output file with updated configuration
	outputData := r.buildOutputData(ctx, data)
	r.writeJSONFile(ctx, data, outputData, outputPath)

	// Preserve execution-related fields from the current state
	// These should not change during an update operation
	data.ExecutionStatus = state.ExecutionStatus
	data.ExecutionStarted = state.ExecutionStarted
	data.ExecutionCompleted = state.ExecutionCompleted
	data.ExecutionError = state.ExecutionError
	data.ProjectPath = state.ProjectPath

	tflog.Info(ctx, "Updated project configuration", map[string]interface{}{
		"project_id": data.ID.ValueString(),
		"preserved_status": data.ExecutionStatus.ValueString(),
		"scaffolds_updated": !isDryRun && state.ProjectPath.ValueString() != "",
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProjectResourceFinal) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ProjectModelFinal
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get provider configuration
	isDryRun := true
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
		GetOutputPath() string
		GetClaudeHomeDirectory() string
	}); ok {
		isDryRun = provData.GetDryRun()
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
	}

	// If not in dry run mode and we have a project path, clean it up
	if !isDryRun && !data.ProjectPath.IsNull() && !data.ProjectPath.IsUnknown() {
		projectPath := data.ProjectPath.ValueString()
		if projectPath != "" {
			executor := claude.NewExecutor(claudeHomeDir, isDryRun)
			if err := executor.CleanupProject(ctx, projectPath); err != nil {
				tflog.Warn(ctx, "Failed to cleanup generated project", map[string]interface{}{
					"project_id":   data.ID.ValueString(),
					"project_path": projectPath,
					"error":        err.Error(),
				})
				// Don't fail deletion for cleanup issues
			} else {
				tflog.Info(ctx, "Cleaned up generated project", map[string]interface{}{
					"project_id":   data.ID.ValueString(),
					"project_path": projectPath,
				})
			}
		}
	}

	// Clean up JSON output file
	filename := filepath.Join(outputPath, fmt.Sprintf("project-%s.json", data.Name.ValueString()))
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		tflog.Warn(ctx, "Failed to remove JSON output file", map[string]interface{}{
			"filename": filename,
			"error":    err.Error(),
		})
	}

	// Clean up metadata file
	metadataFile := filepath.Join(outputPath, "claude-execution-metadata.json")
	if err := os.Remove(metadataFile); err != nil && !os.IsNotExist(err) {
		tflog.Warn(ctx, "Failed to remove metadata file", map[string]interface{}{
			"filename": metadataFile,
			"error":    err.Error(),
		})
	}

	// Clean up prompt file
	promptFile := filepath.Join(outputPath, "claude-prompt.txt")
	if err := os.Remove(promptFile); err != nil && !os.IsNotExist(err) {
		tflog.Warn(ctx, "Failed to remove prompt file", map[string]interface{}{
			"filename": promptFile,
			"error":    err.Error(),
		})
	}

	tflog.Info(ctx, "Project resource deleted", map[string]interface{}{
		"project_id": data.ID.ValueString(),
	})
}

// buildOutputData creates the output data structure from project model
func (r *ProjectResourceFinal) buildOutputData(ctx context.Context, data ProjectModelFinal) map[string]interface{} {

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

	// Add scaffolds
	scaffolds := []map[string]interface{}{}
	for _, scaffold := range data.Scaffolds {
		scaffoldData := map[string]interface{}{
			"path": scaffold.Path.ValueString(),
		}

		if !scaffold.Content.IsNull() && !scaffold.Content.IsUnknown() {
			scaffoldData["content"] = scaffold.Content.ValueString()
		}

		if !scaffold.Generate.IsNull() && !scaffold.Generate.IsUnknown() {
			scaffoldData["generate"] = scaffold.Generate.ValueBool()
		}

		if !scaffold.Template.IsNull() && !scaffold.Template.IsUnknown() {
			scaffoldData["template"] = scaffold.Template.ValueString()
		}

		scaffolds = append(scaffolds, scaffoldData)
	}
	if len(scaffolds) > 0 {
		outputData["scaffolds"] = scaffolds
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

	return outputData
}

// writeJSONFile writes the output data to a JSON file
func (r *ProjectResourceFinal) writeJSONFile(ctx context.Context, data ProjectModelFinal, outputData map[string]interface{}, outputPath string) {
	// Get output format from provider data
	outputFormat := "json"
	if provData, ok := r.ProviderData.(interface {
		GetOutputFormat() string
	}); ok {
		outputFormat = provData.GetOutputFormat()
	}

	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to create output directory: %v", err))
		return
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

// executeClaudeCode executes Claude Code with the project specification
func (r *ProjectResourceFinal) executeClaudeCode(ctx context.Context, data *ProjectModelFinal, outputData map[string]interface{}, outputPath string, claudeHomeDir string, dryRun bool) error {
	executor := claude.NewExecutor(claudeHomeDir, dryRun)

	// Execute Claude Code
	status, err := executor.Execute(ctx, outputData, outputPath)
	if err != nil {
		// Even on error, ensure computed fields are set to avoid "unknown value" errors
		// These fields were already initialized to empty strings, but make sure they stay that way
		return fmt.Errorf("Claude Code execution failed: %w", err)
	}

	// Update the model with execution results
	data.ExecutionStatus = types.StringValue(status.State)

	// Always set string values (even empty) to ensure fields are known
	data.ExecutionStarted = types.StringValue(status.StartedAt)
	data.ExecutionCompleted = types.StringValue(status.CompletedAt)
	data.ProjectPath = types.StringValue(status.ProjectPath)

	if status.Error != "" {
		data.ExecutionError = types.StringValue(status.Error)
		return fmt.Errorf("Claude Code execution failed: %s", status.Error)
	}

	// Clear any previous error with empty string
	data.ExecutionError = types.StringValue("")

	tflog.Info(ctx, "Claude Code execution completed successfully", map[string]interface{}{
		"project_id":   data.ID.ValueString(),
		"project_path": status.ProjectPath,
		"state":        status.State,
	})

	return nil
}

// ValidateConfig validates the project configuration
func (r *ProjectResourceFinal) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data ProjectModelFinal
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate project name
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		name := data.Name.ValueString()
		if name == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("name"),
				"Invalid project name",
				"Project name cannot be empty",
			)
		}
		// Check for valid project name format
		if len(name) > 100 {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("name"),
				"Long project name",
				"Project name is very long and may cause issues with file system paths",
			)
		}
	}

	// Get provider configuration to check Claude Code availability if not in dry run
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
		GetClaudeHomeDirectory() string
	}); ok && !provData.GetDryRun() {
		// In execution mode, validate Claude Code availability
		claudeHomeDir := provData.GetClaudeHomeDirectory()
		executor := claude.NewExecutor(claudeHomeDir, false)
		if err := executor.Validate(ctx); err != nil {
			resp.Diagnostics.AddWarning(
				"Claude Code Validation",
				fmt.Sprintf("Claude Code CLI may not be available: %v. "+
					"This will cause execution to fail. "+
					"Ensure Claude Code CLI is installed and accessible.", err),
			)
		}
	}
}
