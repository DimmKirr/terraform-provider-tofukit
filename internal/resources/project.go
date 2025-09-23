package resources

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

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
	"github.com/tofukit/opentofu-provider-tofukit/internal/registry"
	"github.com/tofukit/opentofu-provider-tofukit/internal/files"
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
	ID                 types.String               `tfsdk:"id"`
	Name               types.String               `tfsdk:"name"`
	Description        types.String               `tfsdk:"description"`
	Version            types.String               `tfsdk:"version"`
	StackRef           types.String               `tfsdk:"stack_ref"` // Reference to the primary stack this project instantiates
	Requirements       []schemas.RequirementModel `tfsdk:"requirement"`
	Kits               types.Map                  `tfsdk:"kits"`  // Additional kits to customize the stack
	Files              []schemas.FileModel        `tfsdk:"file"`  // Project-specific file overrides
	ExecutionStatus    types.String               `tfsdk:"execution_status"`
	ExecutionStarted   types.String               `tfsdk:"execution_started"`
	ExecutionCompleted types.String               `tfsdk:"execution_completed"`
	ProjectPath        types.String               `tfsdk:"project_path"`
	ExecutionError     types.String               `tfsdk:"execution_error"`
	// State tracking for file changes
	FileHash    types.String `tfsdk:"file_hash"`
	LastApplied types.String `tfsdk:"last_applied"`
	// Custom system prompt for Claude
	SystemPrompt types.String `tfsdk:"system_prompt"`
}

func (r *ProjectResourceFinal) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Configure adds the provider configured data to the resource
func (r *ProjectResourceFinal) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.BaseComponent.Configure(ctx, req, resp)
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
			"stack_ref": schema.StringAttribute{
				MarkdownDescription: "Reference to the primary stack this project instantiates (e.g., 'stack.django_rest_api')",
				Optional:            true,
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
									"verification": types.ListType{
										ElemType: types.ObjectType{
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
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"execution_status": schema.StringAttribute{
				MarkdownDescription: "Status of Claude Code execution (pending, running, completed, failed)",
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
				Optional:            true,
			},
			"file_hash": schema.StringAttribute{
				MarkdownDescription: "Hash of file configuration for change detection",
				Computed:            true,
			},
			"last_applied": schema.StringAttribute{
				MarkdownDescription: "Timestamp when files were last successfully applied",
				Computed:            true,
			},
			"system_prompt": schema.StringAttribute{
				MarkdownDescription: "Custom system prompt for Claude. If not set, uses the default system prompt for a senior software architect with 30+ years of experience.",
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"requirement": schemas.GetRequirementBlock(),
			"file":        schemas.GetFileBlock(),
		},
	}
}

func (r *ProjectResourceFinal) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	fmt.Printf("🔧 DEBUG: ProjectResourceFinal.Create called\n")
	var data ProjectModelFinal

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		fmt.Printf("🔧 DEBUG: ProjectResourceFinal.Create failed with diagnostics error\n")
		return
	}

	fmt.Printf("🔧 DEBUG: ProjectResourceFinal.Create - setting ID for project: %s\n", data.Name.ValueString())
	data.ID = types.StringValue(fmt.Sprintf("project.%s", data.Name.ValueString()))

	// Get provider configuration
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	debug := false
	if provData, ok := r.ProviderData.(interface {
		GetOutputPath() string
		GetClaudeHomeDirectory() string
		GetDebug() bool
	}); ok {
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
		debug = provData.GetDebug()
	}

	// Initialize all computed fields to ensure they're never unknown
	data.ExecutionStarted = types.StringValue("")
	data.ExecutionCompleted = types.StringValue("")
	data.ExecutionError = types.StringValue("")
	data.ProjectPath = types.StringValue(filepath.Join(outputPath, data.Name.ValueString()))
	data.FileHash = types.StringValue(r.computeFileHash(data.Files))
	// Initialize LastApplied with current time since files will be applied
	data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))

	// Execute Claude Code - Claude will create all files based on files
	data.ExecutionStatus = types.StringValue("pending")
	data.ExecutionStarted = types.StringValue(time.Now().Format(time.RFC3339))

	// Collect files FIRST
	mergedFiles := r.collectAndMergeFiles(ctx, data)

	// Build output data AFTER collecting files to ensure they're included
	outputData := r.buildOutputData(ctx, data)
	r.writeJSONFile(ctx, data, outputData, outputPath)

	// Write debug files if debug mode is enabled
	if debug {
		r.writeDebugFiles(ctx, data, outputData, outputPath)
	}

	if err := r.executeClaudeCode(ctx, &data, outputData, mergedFiles, outputPath, claudeHomeDir, false); err != nil {
		// Set error status and fail the resource creation
		data.ExecutionStatus = types.StringValue("failed")
		data.ExecutionError = types.StringValue(err.Error())
		// Ensure completed time is set even on failure
		if data.ExecutionCompleted.IsNull() || data.ExecutionCompleted.ValueString() == "" {
			data.ExecutionCompleted = types.StringValue(time.Now().Format(time.RFC3339))
		}
		tflog.Error(ctx, "Claude Code execution failed", map[string]interface{}{
			"project_id": data.ID.ValueString(),
			"error":      err.Error(),
		})
		// Fail the resource creation with a clear error message
		resp.Diagnostics.AddError(
			"Claude Code Execution Failed",
			fmt.Sprintf("Failed to execute Claude Code for project '%s': %s\n\n"+
				"Check the debug files in %s/.debug/ for more details.",
			data.Name.ValueString(), err.Error(), outputPath),
		)
		return
	} else {
		// Ensure LastApplied is set on successful execution
		if data.LastApplied.IsNull() || data.LastApplied.IsUnknown() {
			data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
		}
	}

	// Final safety check: Ensure LastApplied is NEVER unknown or null before saving
	if data.LastApplied.IsNull() || data.LastApplied.IsUnknown() {
		tflog.Warn(ctx, "LastApplied was null/unknown at end of Create, setting to current time", map[string]interface{}{
			"project_id": data.ID.ValueString(),
		})
		data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
	}

	tflog.Debug(ctx, "About to save state with computed fields", map[string]interface{}{
		"execution_started":           data.ExecutionStarted.ValueString(),
		"execution_completed":         data.ExecutionCompleted.ValueString(),
		"execution_error":             data.ExecutionError.ValueString(),
		"project_path":                data.ProjectPath.ValueString(),
		"execution_started_null":      data.ExecutionStarted.IsNull(),
		"execution_completed_null":    data.ExecutionCompleted.IsNull(),
		"execution_error_null":        data.ExecutionError.IsNull(),
		"project_path_null":           data.ProjectPath.IsNull(),
		"execution_started_unknown":   data.ExecutionStarted.IsUnknown(),
		"execution_completed_unknown": data.ExecutionCompleted.IsUnknown(),
		"execution_error_unknown":     data.ExecutionError.IsUnknown(),
		"project_path_unknown":        data.ProjectPath.IsUnknown(),
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
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	if provData, ok := r.ProviderData.(interface {
		GetOutputPath() string
		GetClaudeHomeDirectory() string
	}); ok {
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
	}

	// Always update output files with current configuration (including files from stacks)
	// This ensures the Claude prompt JSON always has the complete specification
	mergedFiles := r.collectAndMergeFiles(ctx, data)
	outputData := r.buildOutputData(ctx, data)
	r.writeJSONFile(ctx, data, outputData, outputPath)

	// Check for debug mode
	debug := false
	if provData, ok := r.ProviderData.(interface {
		GetDebug() bool
	}); ok {
		debug = provData.GetDebug()
	}

	// Write debug files if debug mode is enabled
	if debug {
		r.writeDebugFiles(ctx, data, outputData, outputPath)
	}

	tflog.Info(ctx, "Updated output files during Read", map[string]interface{}{
		"project_id":       data.ID.ValueString(),
		"merged_files": len(mergedFiles),
	})

	// Check if the generated project still exists
	if !data.ProjectPath.IsNull() && !data.ProjectPath.IsUnknown() {
		projectPath := data.ProjectPath.ValueString()
		if projectPath != "" {
			executor := claude.NewExecutor(claudeHomeDir)
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

	// Get provider configuration
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	debug := false
	if provData, ok := r.ProviderData.(interface {
		GetOutputPath() string
		GetClaudeHomeDirectory() string
		GetDebug() bool
	}); ok {
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
		debug = provData.GetDebug()
	}

	// Detect file changes using fingerprinting
	fileChanged := r.detectFileChanges(ctx, state.Files, data.Files)

	// Preserve ID and project path from existing state
	data.ID = state.ID
	data.ProjectPath = state.ProjectPath

	// Initially preserve the file hash from state - will be updated only on successful apply
	data.FileHash = state.FileHash

	if fileChanged {
		tflog.Info(ctx, "File changes detected, triggering re-execution", map[string]interface{}{
			"project_id":    data.ID.ValueString(),
			"old_files": len(state.Files),
			"new_files": len(data.Files),
		})

		// Always execute Claude Code for file changes
		// Execute Claude Code with the updated specification
		// Preserve existing timestamps - they represent when the project was first executed
		// Only update status to reflect the re-execution
		data.ExecutionStatus = types.StringValue("pending")
		// IMPORTANT: Preserve timestamps from state to avoid provider inconsistency
		data.ExecutionStarted = state.ExecutionStarted
		data.ExecutionCompleted = state.ExecutionCompleted

		// Collect merged files from all sources
		mergedFiles := r.collectAndMergeFiles(ctx, data)

		// Build output data AFTER collecting files to ensure they're included
		outputData := r.buildOutputData(ctx, data)
		r.writeJSONFile(ctx, data, outputData, outputPath)

		// Write debug files if debug mode is enabled
		if debug {
			r.writeDebugFiles(ctx, data, outputData, outputPath)
		}

		if err := r.executeClaudeCode(ctx, &data, outputData, mergedFiles, outputPath, claudeHomeDir, true); err != nil {
			// Set error status and fail the resource update
			data.ExecutionStatus = types.StringValue("failed")
			data.ExecutionError = types.StringValue(err.Error())
			// Ensure completed time is set even on failure
			if data.ExecutionCompleted.IsNull() || data.ExecutionCompleted.ValueString() == "" {
				data.ExecutionCompleted = types.StringValue(time.Now().Format(time.RFC3339))
			}
			// Don't use state.LastApplied here - we want to fail the update
			data.LastApplied = state.LastApplied
			// Ensure LastApplied is never unknown
			if data.LastApplied.IsNull() || data.LastApplied.IsUnknown() {
				data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
			}
			tflog.Error(ctx, "Claude Code execution failed during update", map[string]interface{}{
				"project_id": data.ID.ValueString(),
				"error":      err.Error(),
			})
			resp.Diagnostics.AddError(
				"Claude Code Execution Failed",
				fmt.Sprintf("Failed to execute Claude Code for project %s: %s", data.Name.ValueString(), err.Error()),
			)
			return
		} else {
			// Only update hash and LastApplied on successful execution
			data.FileHash = types.StringValue(r.computeFileHash(data.Files))
			data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
			// Preserve the original timestamps even on successful re-execution
			data.ExecutionStarted = state.ExecutionStarted
			data.ExecutionCompleted = state.ExecutionCompleted
		}

		// No file changes detected, preserve existing execution state
		tflog.Info(ctx, "No file changes detected, preserving execution state", map[string]interface{}{
			"project_id":     data.ID.ValueString(),
			"file_count": len(data.Files),
		})

		// Still need to update output files with current configuration (including files from stacks)
		// Collect merged files to ensure stack files are included
		mergedFiles = r.collectAndMergeFiles(ctx, data)

		// Build output data with all files
		outputData = r.buildOutputData(ctx, data)
		r.writeJSONFile(ctx, data, outputData, outputPath)

		// Write debug files if debug mode is enabled
		if debug {
			r.writeDebugFiles(ctx, data, outputData, outputPath)
		}

		tflog.Info(ctx, "Updated output files with merged files", map[string]interface{}{
			"merged_file_count": len(mergedFiles),
		})

		// Preserve all execution-related fields from the current state
		data.ExecutionStatus = state.ExecutionStatus
		data.ExecutionStarted = state.ExecutionStarted
		data.ExecutionCompleted = state.ExecutionCompleted
		data.ExecutionError = state.ExecutionError
		data.LastApplied = state.LastApplied
		// Ensure LastApplied is never unknown or null
		if data.LastApplied.IsNull() || data.LastApplied.IsUnknown() {
			data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
		}
	}

	// Final safety check: Ensure LastApplied is NEVER unknown or null before saving
	if data.LastApplied.IsNull() || data.LastApplied.IsUnknown() {
		tflog.Warn(ctx, "LastApplied was null/unknown at end of Update, setting to current time", map[string]interface{}{
			"project_id": data.ID.ValueString(),
		})
		data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
	}

	tflog.Info(ctx, "Updated project configuration", map[string]interface{}{
		"project_id":       data.ID.ValueString(),
		"file_changed": fileChanged,
		"execution_status": data.ExecutionStatus.ValueString(),
		"last_applied":     data.LastApplied.ValueString(),
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
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	if provData, ok := r.ProviderData.(interface {
		GetOutputPath() string
		GetClaudeHomeDirectory() string
	}); ok {
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
	}

	// Clean up files and project directory
	if !data.ProjectPath.IsNull() && !data.ProjectPath.IsUnknown() {
		projectPath := data.ProjectPath.ValueString()
		if projectPath != "" {
			// Remove file files
			if len(data.Files) > 0 {
				fileManager := files.NewManager(projectPath)
				fileManager.RemoveAllFiles(ctx, data.Files)
				tflog.Info(ctx, "Removed file files", map[string]interface{}{
					"project_path": projectPath,
					"count":        len(data.Files),
				})
			}

			// Clean up any generated project
			executor := claude.NewExecutor(claudeHomeDir)
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

		// Add verifications if present
		if len(req.Verification) > 0 {
			verifications := []map[string]string{}
			for _, v := range req.Verification {
				verif := map[string]string{
					"command": v.Command.ValueString(),
				}
				if !v.Expect.IsNull() && !v.Expect.IsUnknown() {
					verif["expect"] = v.Expect.ValueString()
				}
				verifications = append(verifications, verif)
			}
			reqData["verification"] = verifications
		}

		outputData["requirements"] = append(outputData["requirements"].([]map[string]interface{}), reqData)
	}

	// Add merged files from all sources
	mergedFiles := r.collectAndMergeFiles(ctx, data)
	tflog.Info(ctx, "Collected merged files for output", map[string]interface{}{
		"count":        len(mergedFiles),
		"project_name": data.Name.ValueString(),
	})
	files := []map[string]interface{}{}
	for _, file := range mergedFiles {
		fileData := map[string]interface{}{
			"path": file.Path.ValueString(),
		}

		// Debug log
		tflog.Debug(ctx, "Processing file for output", map[string]interface{}{
			"path":               file.Path.ValueString(),
			"has_verifications":  len(file.Verification) > 0,
			"verification_count": len(file.Verification),
		})

		if !file.Content.IsNull() && !file.Content.IsUnknown() {
			fileData["content"] = file.Content.ValueString()
		}

		if !file.Generate.IsNull() && !file.Generate.IsUnknown() {
			fileData["generate"] = file.Generate.ValueBool()
		}

		if file.Instructions != nil && len(file.Instructions) > 0 {
			instructions := make([]string, len(file.Instructions))
			for i, inst := range file.Instructions {
				instructions[i] = inst.ValueString()
			}
			fileData["instructions"] = instructions
		}

		// Add verifications if present
		if len(file.Verification) > 0 {
			verifications := []map[string]interface{}{}
			for _, v := range file.Verification {
				verif := map[string]interface{}{
					"command": v.Command.ValueString(),
				}
				if !v.Expect.IsNull() && !v.Expect.IsUnknown() {
					verif["expect"] = v.Expect.ValueString()
				}
				verifications = append(verifications, verif)
			}
			fileData["verification"] = verifications
		}

		files = append(files, fileData)
	}
	// Always include files, even if empty, for consistency
	outputData["files"] = files

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

							// Get verification list
							if verList, ok := reqAttrs["verification"].(types.List); ok && !verList.IsNull() {
								verifications := []map[string]string{}
								for _, verElem := range verList.Elements() {
									if verObj, ok := verElem.(types.Object); ok && !verObj.IsNull() {
										verAttrs := verObj.Attributes()
										verification := make(map[string]string)

										if cmd, ok := verAttrs["command"].(types.String); ok && !cmd.IsNull() {
											verification["command"] = cmd.ValueString()
										}
										if exp, ok := verAttrs["expect"].(types.String); ok && !exp.IsNull() {
											verification["expect"] = exp.ValueString()
										}

										if len(verification) > 0 {
											verifications = append(verifications, verification)
										}
									}
								}
								if len(verifications) > 0 {
									reqData["verification"] = verifications
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
	// This function is now deprecated - writeDebugFiles handles JSON output
	// Keeping empty for backward compatibility but all writing is done in writeDebugFiles
	tflog.Debug(ctx, "writeJSONFile called but delegating to writeDebugFiles")
}

// executeClaudeCode executes Claude Code with the project specification
func (r *ProjectResourceFinal) executeClaudeCode(ctx context.Context, data *ProjectModelFinal, outputData map[string]interface{}, mergedFiles []schemas.FileModel, outputPath string, claudeHomeDir string, preserveTimestamps bool) error {
	// Check if debug mode is enabled
	debug := false
	if provData, ok := r.ProviderData.(interface {
		GetDebug() bool
	}); ok {
		debug = provData.GetDebug()
	}

	// Get system prompt from resource data
	systemPrompt := ""
	if !data.SystemPrompt.IsNull() && !data.SystemPrompt.IsUnknown() {
		systemPrompt = data.SystemPrompt.ValueString()
	}

	executor := claude.NewExecutor(claudeHomeDir)
	executor.SetDebug(debug)
	executor.SetOutputPath(outputPath)
	executor.SetSystemPrompt(systemPrompt)

	// Write debug files if debug mode is enabled
	if debug {
		r.writeDebugFiles(ctx, *data, outputData, outputPath)
	}

	// Execute Claude Code
	status, err := executor.Execute(ctx, outputData, outputPath)
	if err != nil {
		// Even on error, ensure computed fields are set to avoid "unknown value" errors
		// These fields were already initialized to empty strings, but make sure they stay that way
		return fmt.Errorf("Claude Code execution failed: %w", err)
	}

	// Update the model with execution results
	data.ExecutionStatus = types.StringValue(status.State)

	// Only update timestamps if not preserving them (i.e., during initial creation)
	if !preserveTimestamps {
		// Set timestamps from the execution status
		data.ExecutionStarted = types.StringValue(status.StartedAt)
		data.ExecutionCompleted = types.StringValue(status.CompletedAt)
	}
	// Always update project path
	data.ProjectPath = types.StringValue(status.ProjectPath)

	if status.Error != "" {
		data.ExecutionError = types.StringValue(status.Error)
		return fmt.Errorf("Claude Code execution failed: %s", status.Error)
	}

	// Clear any previous error with empty string
	data.ExecutionError = types.StringValue("")

	// Verify that Claude created all merged file files correctly
	if len(mergedFiles) > 0 && status.ProjectPath != "" {
		fileManager := files.NewManager(status.ProjectPath)
		if err := fileManager.VerifyFiles(ctx, mergedFiles); err != nil {
			// Log the verification failure but don't fail the operation
			// Claude might have created the files but with slightly different content
			tflog.Warn(ctx, "Merged file verification failed after Claude execution", map[string]interface{}{
				"error":        err.Error(),
				"project_path": status.ProjectPath,
			})
			// Store the verification error for visibility
			data.ExecutionError = types.StringValue(fmt.Sprintf("Warning: %v", err))
		} else {
			tflog.Info(ctx, "All merged files verified successfully after Claude execution", map[string]interface{}{
				"merged_count":  len(mergedFiles),
				"project_count": len(data.Files),
				"project_path":  status.ProjectPath,
			})
			// Clear any previous execution error since verification succeeded
			data.ExecutionError = types.StringValue("")

			// Run verification commands if files have them
			if err := fileManager.RunVerifications(ctx, mergedFiles); err != nil {
				// Verification commands failed - this is a hard error
				tflog.Error(ctx, "File verification commands failed", map[string]interface{}{
					"error":        err.Error(),
					"project_path": status.ProjectPath,
				})
				data.ExecutionError = types.StringValue(fmt.Sprintf("Verification failed: %v", err))
				data.ExecutionStatus = types.StringValue("verification_failed")

				// Return error to fail the operation
				return fmt.Errorf("file verification commands failed: %w", err)
			}
		}
	}

	// LastApplied was already set during initialization

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
		GetClaudeHomeDirectory() string
	}); ok {
		// Validate Claude Code availability
		claudeHomeDir := provData.GetClaudeHomeDirectory()
		executor := claude.NewExecutor(claudeHomeDir)
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

// writeDebugFiles writes debug information when debug mode is enabled
func (r *ProjectResourceFinal) writeDebugFiles(ctx context.Context, data ProjectModelFinal, outputData map[string]interface{}, outputPath string) {
	// Create .debug directory for all debug files
	debugDir := filepath.Join(outputPath, ".debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		tflog.Warn(ctx, "Failed to create .debug directory", map[string]interface{}{
			"error": err.Error(),
			"path":  debugDir,
		})
		return
	}

	// Write project-{name}-{timestamp}.json - the full project specification
	projectName := data.Name.ValueString()
	timestamp := time.Now().Unix()
	projectSpecPath := filepath.Join(debugDir, fmt.Sprintf("project-%s-%d.json", projectName, timestamp))
	specJSON, err := json.MarshalIndent(outputData, "", "  ")
	if err == nil {
		if writeErr := os.WriteFile(projectSpecPath, specJSON, 0644); writeErr == nil {
			tflog.Debug(ctx, "Wrote project specification", map[string]interface{}{
				"path":      projectSpecPath,
				"size":      len(specJSON),
				"timestamp": timestamp,
			})
		}
	}

	// Create prompt files for debugging
	projectSpec := outputData

	// Extract custom system prompt if set
	customSystemPrompt := ""
	if !data.SystemPrompt.IsNull() && !data.SystemPrompt.IsUnknown() {
		customSystemPrompt = data.SystemPrompt.ValueString()
	}

	// Build structured prompt using the new template approach
	promptObj := claude.BuildProjectPrompt(projectSpec, customSystemPrompt)

	// Write claude-prompt.md using the template-based approach
	promptMdPath := filepath.Join(debugDir, fmt.Sprintf("claude-prompt-%d.md", timestamp))
	mdContent := promptObj.ToMarkdown()
	if writeErr := os.WriteFile(promptMdPath, []byte(mdContent), 0644); writeErr == nil {
		tflog.Debug(ctx, "Wrote prompt markdown file", map[string]interface{}{
			"path": promptMdPath,
			"size": len(mdContent),
		})
	}

	// Build the actual JSON prompt that will be sent to Claude
	claudeClient := claude.NewClient("") // temp client just for building prompt
	jsonPrompt, _ := claudeClient.BuildPrompt(projectSpec)

	// Write claude-prompt.json - the actual JSON prompt that gets sent to Claude
	promptJSONPath := filepath.Join(debugDir, fmt.Sprintf("claude-prompt-%d.json", timestamp))
	// Pretty-print the JSON for readability in debug
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, []byte(jsonPrompt), "", "  ")
	if writeErr := os.WriteFile(promptJSONPath, prettyJSON.Bytes(), 0644); writeErr == nil {
		tflog.Debug(ctx, "Wrote prompt JSON file", map[string]interface{}{
			"path": promptJSONPath,
			"size": prettyJSON.Len(),
		})
	}

	// Write initial entry to claude-log.jsonl - this will contain all prompts and responses
	logPath := filepath.Join(debugDir, fmt.Sprintf("claude-log-%d.jsonl", timestamp))
	logEntry := map[string]interface{}{
		"type":              "prompt",
		"timestamp":         time.Now().Format(time.RFC3339),
		"content":           jsonPrompt, // Now logging the JSON prompt
		"format":            "json",
		"system_prompt":     customSystemPrompt,
		"working_directory": outputPath,
		"metadata": map[string]interface{}{
			"project_name":    data.Name.ValueString(),
			"project_version": data.Version.ValueString(),
		},
	}
	if logBytes, err := json.Marshal(logEntry); err == nil {
		if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			logFile.Write(append(logBytes, '\n'))
			logFile.Close()
			tflog.Debug(ctx, "Created Claude log file", map[string]interface{}{
				"path": logPath,
			})
		}
	}

	// Note: During actual Claude execution, additional files may be created:
	// - claude-execution-metadata.json: Execution status and results
	// - Additional logs from the Claude SDK
}

// computeFileHash creates a fingerprint of file configuration for change detection
func (r *ProjectResourceFinal) computeFileHash(files []schemas.FileModel) string {
	if len(files) == 0 {
		return ""
	}

	// Create a deterministic representation of files
	type fileEntry struct {
		Path         string   `json:"path"`
		Content      string   `json:"content"`
		Generate     bool     `json:"generate,omitempty"`
		Instructions []string `json:"instructions,omitempty"`
	}

	var entries []fileEntry
	for _, file := range files {
		entry := fileEntry{
			Path:    file.Path.ValueString(),
			Content: file.Content.ValueString(),
		}

		// Include optional fields if they're set
		if !file.Generate.IsNull() && !file.Generate.IsUnknown() {
			entry.Generate = file.Generate.ValueBool()
		}
		if file.Instructions != nil && len(file.Instructions) > 0 {
			instructions := make([]string, len(file.Instructions))
			for i, inst := range file.Instructions {
				instructions[i] = inst.ValueString()
			}
			entry.Instructions = instructions
		}

		entries = append(entries, entry)
	}

	// Sort entries by path for deterministic hash
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	// Create JSON representation and hash it
	jsonData, err := json.Marshal(entries)
	if err != nil {
		// Fallback to simple string concatenation if JSON marshal fails
		var parts []string
		for _, entry := range entries {
			parts = append(parts, fmt.Sprintf("%s:%s", entry.Path, entry.Content))
		}
		jsonData = []byte(fmt.Sprintf("%v", parts))
	}

	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// detectFileChanges determines if files have meaningfully changed
func (r *ProjectResourceFinal) detectFileChanges(ctx context.Context, oldFiles, newFiles []schemas.FileModel) bool {
	oldHash := r.computeFileHash(oldFiles)
	newHash := r.computeFileHash(newFiles)

	changed := oldHash != newHash

	tflog.Debug(ctx, "File change detection", map[string]interface{}{
		"old_hash":  oldHash,
		"new_hash":  newHash,
		"changed":   changed,
		"old_count": len(oldFiles),
		"new_count": len(newFiles),
	})

	return changed
}

// collectAndMergeFiles collects files from all sources and merges them with proper precedence
func (r *ProjectResourceFinal) collectAndMergeFiles(ctx context.Context, data ProjectModelFinal) []schemas.FileModel {
	merger := files.NewMerger()

	// Get the registry to access stacks
	var reg *registry.Registry
	if provData, ok := r.ProviderData.(interface {
		GetRegistry() *registry.Registry
	}); ok {
		reg = provData.GetRegistry()
		tflog.Info(ctx, "Got registry from provider", map[string]interface{}{
			"registry_not_nil": reg != nil,
		})
	} else {
		tflog.Warn(ctx, "Failed to get registry from provider", map[string]interface{}{
			"provider_type": fmt.Sprintf("%T", r.ProviderData),
		})
	}

	if reg != nil {
		// Log all stacks in the registry for debugging
		allStacks := reg.GetAllStacks()
		tflog.Info(ctx, "Stacks in registry before processing", map[string]interface{}{
			"stack_count": len(allStacks),
			"stack_ids": func() []string {
				ids := make([]string, 0, len(allStacks))
				for id := range allStacks {
					ids = append(ids, id)
				}
				return ids
			}(),
		})

		// If a specific stack is referenced, use only that stack
		if !data.StackRef.IsNull() && !data.StackRef.IsUnknown() {
			stackRef := data.StackRef.ValueString()

			tflog.Info(ctx, "Looking for specific stack", map[string]interface{}{
				"stack_ref": stackRef,
			})

			// Get the primary stack this project instantiates
			if stackData, exists := reg.GetStack(stackRef); exists {
				if stack, ok := stackData.(StackResourceModel); ok {
					tflog.Info(ctx, "Found and processing primary stack", map[string]interface{}{
						"stack_ref":      stackRef,
						"file_count": len(stack.Files),
						"stack_name":     stack.Name.ValueString(),
					})

					// Log the actual files with verification details
					for i, file := range stack.Files {
						logData := map[string]interface{}{
							"index":               i,
							"path":                file.Path.ValueString(),
							"has_verifications":   len(file.Verification) > 0,
							"verification_count":  len(file.Verification),
						}

						if len(file.Verification) > 0 {
							for j, v := range file.Verification {
								logData[fmt.Sprintf("verification_%d_command", j)] = v.Command.ValueString()
								if !v.Expect.IsNull() {
									logData[fmt.Sprintf("verification_%d_expect", j)] = v.Expect.ValueString()
								}
							}
						}

						tflog.Info(ctx, "Stack file retrieved from registry", logData)
					}

					// TODO: First add files from kits within the primary stack (lowest precedence)
					// This requires stacks to have their own kit references
					// For now, we'll just note this is where stack's kits would be processed

					// Add the primary stack's own files (higher precedence than stack's kits)
					merger.AddFiles(ctx, stack.Files, files.SourceStack)
				} else {
					tflog.Warn(ctx, "Stack data is not StackResourceModel", map[string]interface{}{
						"stack_ref":   stackRef,
						"actual_type": fmt.Sprintf("%T", stackData),
					})
				}
			} else {
				tflog.Warn(ctx, "Referenced stack not found in registry", map[string]interface{}{
					"stack_ref": stackRef,
					"available_stacks": func() []string {
						ids := make([]string, 0, len(allStacks))
						for id := range allStacks {
							ids = append(ids, id)
						}
						return ids
					}(),
				})
			}
		} else {
			// No explicit stack reference, collect from ALL stacks
			// This is a fallback for backward compatibility
			allStacks := reg.GetAllStacks()
			tflog.Info(ctx, "No stack_ref specified, processing all stacks", map[string]interface{}{
				"stack_count": len(allStacks),
			})

			for stackID, stackData := range allStacks {
				tflog.Info(ctx, "Checking stack data", map[string]interface{}{
					"stack_id": stackID,
					"type":     fmt.Sprintf("%T", stackData),
				})

				if stack, ok := stackData.(StackResourceModel); ok {
					tflog.Info(ctx, "Processing stack files", map[string]interface{}{
						"stack_id":       stackID,
						"file_count": len(stack.Files),
						"stack_name":     stack.Name.ValueString(),
					})

					// Add stack's files
					merger.AddFiles(ctx, stack.Files, files.SourceStack)
				} else {
					tflog.Warn(ctx, "Stack data is not StackResourceModel", map[string]interface{}{
						"stack_id":    stackID,
						"actual_type": fmt.Sprintf("%T", stackData),
					})
				}
			}
		}
	}

	// TODO: Add files from project's additional kits
	// These are kits added directly to the project to customize the stack
	// This would process data.Kits and collect their files
	// merger.AddFiles(ctx, kitFiles, files.SourceProject)

	// Finally, add project's own files (highest precedence)
	merger.AddFiles(ctx, data.Files, files.SourceProject)

	// Get the final merged list
	mergedFiles := merger.GetMergedFiles()

	tflog.Info(ctx, "Merged files from all sources", map[string]interface{}{
		"total_count":   len(mergedFiles),
		"stack_ref":     data.StackRef.ValueString(),
		"project_count": len(data.Files),
	})

	return mergedFiles
}
