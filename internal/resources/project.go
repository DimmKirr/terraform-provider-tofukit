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
	ID                 types.String               `tfsdk:"id"`
	Name               types.String               `tfsdk:"name"`
	Description        types.String               `tfsdk:"description"`
	Version            types.String               `tfsdk:"version"`
	StackRef           types.String               `tfsdk:"stack_ref"` // Reference to the primary stack this project instantiates
	Requirements       []schemas.RequirementModel `tfsdk:"requirement"`
	Kits               types.Map                  `tfsdk:"kits"`     // Additional kits to customize the stack
	Scaffolds          []schemas.ScaffoldModel    `tfsdk:"scaffold"` // Project-specific overrides
	ExecutionStatus    types.String               `tfsdk:"execution_status"`
	ExecutionStarted   types.String               `tfsdk:"execution_started"`
	ExecutionCompleted types.String               `tfsdk:"execution_completed"`
	ProjectPath        types.String               `tfsdk:"project_path"`
	ExecutionError     types.String               `tfsdk:"execution_error"`
	// State tracking for scaffold changes
	ScaffoldHash types.String `tfsdk:"scaffold_hash"`
	LastApplied  types.String `tfsdk:"last_applied"`
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
				Optional:            true,
			},
			"scaffold_hash": schema.StringAttribute{
				MarkdownDescription: "Hash of scaffold configuration for change detection",
				Computed:            true,
			},
			"last_applied": schema.StringAttribute{
				MarkdownDescription: "Timestamp when scaffolds were last successfully applied",
				Computed:            true,
			},
			"system_prompt": schema.StringAttribute{
				MarkdownDescription: "Custom system prompt for Claude. If not set, uses the default system prompt for a senior software architect with 30+ years of experience.",
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"requirement": schemas.GetRequirementBlock(),
			"scaffold":    schemas.GetScaffoldBlock(),
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
	debug := false
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
		GetOutputPath() string
		GetClaudeHomeDirectory() string
		GetDebug() bool
	}); ok {
		isDryRun = provData.GetDryRun()
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
		debug = provData.GetDebug()
	}

	// Initialize all computed fields to ensure they're never unknown
	data.ExecutionStarted = types.StringValue("")
	data.ExecutionCompleted = types.StringValue("")
	data.ExecutionError = types.StringValue("")
	data.ProjectPath = types.StringValue(filepath.Join(outputPath, data.Name.ValueString()))
	data.ScaffoldHash = types.StringValue(r.computeScaffoldHash(data.Scaffolds))
	// Initialize LastApplied with current time since scaffolds will be applied
	data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))

	// Set initial execution status
	if isDryRun {
		data.ExecutionStatus = types.StringValue("dry_run")

		// In dry-run mode, we create merged scaffolds directly as a preview
		// This helps users see what files would be created
		mergedScaffolds := r.collectAndMergeScaffolds(ctx, data)

		tflog.Info(ctx, "After collectAndMergeScaffolds in dry-run", map[string]interface{}{
			"merged_count": len(mergedScaffolds),
			"project_name": data.Name.ValueString(),
		})

		// Build output data AFTER collecting scaffolds to ensure they're included
		outputData := r.buildOutputData(ctx, data)
		r.writeJSONFile(ctx, data, outputData, outputPath)

		if len(mergedScaffolds) > 0 {
			projectPath := data.ProjectPath.ValueString()
			scaffoldManager := scaffolds.NewManager(projectPath)
			for _, scaffold := range mergedScaffolds {
				// In dry-run mode, only write scaffolds that have content
				// Skip scaffolds with generate=true and no content (they need Claude)
				if !scaffold.Generate.IsNull() && scaffold.Generate.ValueBool() &&
					(scaffold.Content.IsNull() || scaffold.Content.ValueString() == "") {
					tflog.Info(ctx, "Skipping scaffold in dry-run (needs generation)", map[string]interface{}{
						"path": scaffold.Path.ValueString(),
					})
					continue
				}
				if err := scaffoldManager.WriteScaffold(ctx, scaffold); err != nil {
					tflog.Warn(ctx, "Failed to write scaffold preview", map[string]interface{}{
						"path":  scaffold.Path.ValueString(),
						"error": err.Error(),
					})
				}
			}
			tflog.Info(ctx, "Created merged scaffold preview files (dry-run)", map[string]interface{}{
				"project_path":  projectPath,
				"merged_count":  len(mergedScaffolds),
				"project_count": len(data.Scaffolds),
			})
		}

		// Write debug files even in dry-run mode if debug is enabled
		if debug {
			r.writeDebugFiles(ctx, data, outputData, outputPath)
		}

		tflog.Info(ctx, "Project resource created in dry-run mode", map[string]interface{}{
			"project_id": data.ID.ValueString(),
		})
	} else {
		// Execute Claude Code - Claude will create all files based on scaffolds
		data.ExecutionStatus = types.StringValue("pending")
		data.ExecutionStarted = types.StringValue(time.Now().Format(time.RFC3339))

		// Collect scaffolds FIRST
		mergedScaffolds := r.collectAndMergeScaffolds(ctx, data)

		// Build output data AFTER collecting scaffolds to ensure they're included
		outputData := r.buildOutputData(ctx, data)
		r.writeJSONFile(ctx, data, outputData, outputPath)

		// Write debug files if debug mode is enabled
		if debug {
			r.writeDebugFiles(ctx, data, outputData, outputPath)
		}

		if err := r.executeClaudeCode(ctx, &data, outputData, mergedScaffolds, outputPath, claudeHomeDir, isDryRun, false); err != nil {
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

	// Always update output files with current configuration (including scaffolds from stacks)
	// This ensures the Claude prompt JSON always has the complete specification
	mergedScaffolds := r.collectAndMergeScaffolds(ctx, data)
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
		"merged_scaffolds": len(mergedScaffolds),
	})

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

	// Get provider configuration
	outputPath := ".tofukit"
	isDryRun := true
	claudeHomeDir := "~/.claude"
	debug := false
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
		GetOutputPath() string
		GetClaudeHomeDirectory() string
		GetDebug() bool
	}); ok {
		outputPath = provData.GetOutputPath()
		isDryRun = provData.GetDryRun()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
		debug = provData.GetDebug()
	}

	// Detect scaffold changes using fingerprinting
	scaffoldChanged := r.detectScaffoldChanges(ctx, state.Scaffolds, data.Scaffolds)

	// Preserve ID and project path from existing state
	data.ID = state.ID
	data.ProjectPath = state.ProjectPath

	// Initially preserve the scaffold hash from state - will be updated only on successful apply
	data.ScaffoldHash = state.ScaffoldHash

	if scaffoldChanged {
		tflog.Info(ctx, "Scaffold changes detected, triggering re-execution", map[string]interface{}{
			"project_id":    data.ID.ValueString(),
			"old_scaffolds": len(state.Scaffolds),
			"new_scaffolds": len(data.Scaffolds),
			"is_dry_run":    isDryRun,
		})

		if isDryRun {
			// In dry-run mode, update scaffold preview files directly with merged scaffolds
			projectDir := state.ProjectPath.ValueString()
			if projectDir != "" {
				// Convert relative path to absolute
				if !filepath.IsAbs(projectDir) {
					if cwd, err := os.Getwd(); err == nil {
						projectDir = filepath.Join(cwd, projectDir)
					}
				}

				scaffoldMgr := scaffolds.NewManager(projectDir)

				// Get merged scaffolds from all sources
				oldMergedScaffolds := r.collectAndMergeScaffolds(ctx, state)
				newMergedScaffolds := r.collectAndMergeScaffolds(ctx, data)

				// Build output data AFTER collecting scaffolds to ensure they're included
				outputData := r.buildOutputData(ctx, data)
				r.writeJSONFile(ctx, data, outputData, outputPath)

				// Write debug files if debug mode is enabled
				if debug {
					r.writeDebugFiles(ctx, data, outputData, outputPath)
				}

				// Process scaffold changes (removes deleted scaffolds, creates new ones)
				if err := scaffoldMgr.ProcessScaffoldChanges(ctx, oldMergedScaffolds, newMergedScaffolds); err != nil {
					tflog.Error(ctx, "Failed to process scaffold preview changes", map[string]interface{}{
						"error": err.Error(),
					})
					// Set error but don't fail the operation in dry-run mode
					data.ExecutionError = types.StringValue(fmt.Sprintf("Scaffold update failed: %v", err))
					// Preserve LastApplied from state on error
					data.LastApplied = state.LastApplied
					// Ensure LastApplied is never unknown
					if data.LastApplied.IsNull() || data.LastApplied.IsUnknown() {
						data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
					}
				} else {
					// Clear any previous errors and update timestamp and hash
					data.ExecutionError = types.StringValue("")
					data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
					data.ScaffoldHash = types.StringValue(r.computeScaffoldHash(data.Scaffolds))
				}

				// Preserve dry-run status and timestamps from state
				data.ExecutionStatus = state.ExecutionStatus
				data.ExecutionStarted = state.ExecutionStarted
				data.ExecutionCompleted = state.ExecutionCompleted
			}
		} else {
			// In execution mode, trigger Claude Code with the updated specification
			// Preserve existing timestamps - they represent when the project was first executed
			// Only update status to reflect the re-execution
			data.ExecutionStatus = types.StringValue("pending")
			// IMPORTANT: Preserve timestamps from state to avoid provider inconsistency
			data.ExecutionStarted = state.ExecutionStarted
			data.ExecutionCompleted = state.ExecutionCompleted
			// Clear any previous error since we're re-executing
			data.ExecutionError = types.StringValue("")

			// Execute Claude Code with the updated specification and merged scaffolds
			mergedScaffolds := r.collectAndMergeScaffolds(ctx, data)

			// Build output data AFTER collecting scaffolds to ensure they're included
			outputData := r.buildOutputData(ctx, data)
			r.writeJSONFile(ctx, data, outputData, outputPath)

			// Write debug files if debug mode is enabled
			if debug {
				r.writeDebugFiles(ctx, data, outputData, outputPath)
			}

			if err := r.executeClaudeCode(ctx, &data, outputData, mergedScaffolds, outputPath, claudeHomeDir, isDryRun, true); err != nil {
				// Set error status and fail the resource update
				data.ExecutionStatus = types.StringValue("failed")
				data.ExecutionError = types.StringValue(err.Error())
				// Preserve the original completed timestamp and LastApplied
				data.ExecutionCompleted = state.ExecutionCompleted
				data.LastApplied = state.LastApplied
				// Ensure LastApplied is never unknown
				if data.LastApplied.IsNull() || data.LastApplied.IsUnknown() {
					data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
				}
				tflog.Error(ctx, "Claude Code execution failed during update", map[string]interface{}{
					"project_id": data.ID.ValueString(),
					"error":      err.Error(),
				})
				// Fail the resource update with a clear error message
				resp.Diagnostics.AddError(
					"Claude Code Execution Failed During Update",
					fmt.Sprintf("Failed to execute Claude Code for project '%s' update: %s\n\n"+
						"Check the debug files in %s/.debug/ for more details.",
						data.Name.ValueString(), err.Error(), outputPath),
				)
				return
			} else {
				// Only update hash and LastApplied on successful execution
				data.ScaffoldHash = types.StringValue(r.computeScaffoldHash(data.Scaffolds))
				data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))
				// Preserve the original timestamps even on successful re-execution
				data.ExecutionStarted = state.ExecutionStarted
				data.ExecutionCompleted = state.ExecutionCompleted
			}
		}
	} else {
		// No scaffold changes detected, preserve existing execution state
		tflog.Info(ctx, "No scaffold changes detected, preserving execution state", map[string]interface{}{
			"project_id":     data.ID.ValueString(),
			"scaffold_count": len(data.Scaffolds),
		})

		// Still need to update output files with current configuration (including scaffolds from stacks)
		// Collect merged scaffolds to ensure stack scaffolds are included
		mergedScaffolds := r.collectAndMergeScaffolds(ctx, data)

		// Build output data with all scaffolds
		outputData := r.buildOutputData(ctx, data)
		r.writeJSONFile(ctx, data, outputData, outputPath)

		// Write debug files if debug mode is enabled
		if debug {
			r.writeDebugFiles(ctx, data, outputData, outputPath)
		}

		tflog.Info(ctx, "Updated output files with merged scaffolds", map[string]interface{}{
			"merged_scaffold_count": len(mergedScaffolds),
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
		"scaffold_changed": scaffoldChanged,
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

	// Clean up scaffolds and project directory
	if !data.ProjectPath.IsNull() && !data.ProjectPath.IsUnknown() {
		projectPath := data.ProjectPath.ValueString()
		if projectPath != "" {
			// Remove scaffold files
			if len(data.Scaffolds) > 0 {
				scaffoldManager := scaffolds.NewManager(projectPath)
				scaffoldManager.RemoveAllScaffolds(ctx, data.Scaffolds)
				tflog.Info(ctx, "Removed scaffold files", map[string]interface{}{
					"project_path": projectPath,
					"count":        len(data.Scaffolds),
				})
			}

			// If not in dry run mode, also clean up any generated project
			if !isDryRun {
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

	// Add merged scaffolds from all sources
	mergedScaffolds := r.collectAndMergeScaffolds(ctx, data)
	tflog.Info(ctx, "Collected merged scaffolds for output", map[string]interface{}{
		"count":        len(mergedScaffolds),
		"project_name": data.Name.ValueString(),
	})
	scaffolds := []map[string]interface{}{}
	for _, scaffold := range mergedScaffolds {
		scaffoldData := map[string]interface{}{
			"path": scaffold.Path.ValueString(),
		}

		// Debug log
		tflog.Debug(ctx, "Processing scaffold for output", map[string]interface{}{
			"path":             scaffold.Path.ValueString(),
			"has_verification": scaffold.Verification != nil,
		})

		if !scaffold.Content.IsNull() && !scaffold.Content.IsUnknown() {
			scaffoldData["content"] = scaffold.Content.ValueString()
		}

		if !scaffold.Generate.IsNull() && !scaffold.Generate.IsUnknown() {
			scaffoldData["generate"] = scaffold.Generate.ValueBool()
		}

		if scaffold.Instructions != nil && len(scaffold.Instructions) > 0 {
			instructions := make([]string, len(scaffold.Instructions))
			for i, inst := range scaffold.Instructions {
				instructions[i] = inst.ValueString()
			}
			scaffoldData["instructions"] = instructions
		}

		// Add verification if present
		if scaffold.Verification != nil {
			verification := map[string]interface{}{
				"command": scaffold.Verification.Command.ValueString(),
			}
			if !scaffold.Verification.Expect.IsNull() && !scaffold.Verification.Expect.IsUnknown() {
				verification["expect"] = scaffold.Verification.Expect.ValueString()
			}
			scaffoldData["verification"] = verification
		}

		scaffolds = append(scaffolds, scaffoldData)
	}
	// Always include scaffolds, even if empty, for consistency
	outputData["scaffolds"] = scaffolds

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
	// This function is now deprecated - writeDebugFiles handles JSON output
	// Keeping empty for backward compatibility but all writing is done in writeDebugFiles
	tflog.Debug(ctx, "writeJSONFile called but delegating to writeDebugFiles")
}

// executeClaudeCode executes Claude Code with the project specification
func (r *ProjectResourceFinal) executeClaudeCode(ctx context.Context, data *ProjectModelFinal, outputData map[string]interface{}, mergedScaffolds []schemas.ScaffoldModel, outputPath string, claudeHomeDir string, dryRun bool, preserveTimestamps bool) error {
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

	executor := claude.NewExecutor(claudeHomeDir, dryRun)
	executor.SetDebug(debug)
	executor.SetOutputPath(outputPath)
	executor.SetSystemPrompt(systemPrompt)

	// Write debug files if debug mode is enabled (even when not in dry_run)
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

	// Verify that Claude created all merged scaffold files correctly
	if len(mergedScaffolds) > 0 && status.ProjectPath != "" {
		scaffoldManager := scaffolds.NewManager(status.ProjectPath)
		if err := scaffoldManager.VerifyScaffolds(ctx, mergedScaffolds); err != nil {
			// Log the verification failure but don't fail the operation
			// Claude might have created the files but with slightly different content
			tflog.Warn(ctx, "Merged scaffold verification failed after Claude execution", map[string]interface{}{
				"error":        err.Error(),
				"project_path": status.ProjectPath,
			})
			// Store the verification error for visibility
			data.ExecutionError = types.StringValue(fmt.Sprintf("Warning: %v", err))
		} else {
			tflog.Info(ctx, "All merged scaffolds verified successfully after Claude execution", map[string]interface{}{
				"merged_count":  len(mergedScaffolds),
				"project_count": len(data.Scaffolds),
				"project_path":  status.ProjectPath,
			})
			// Clear any previous execution error since verification succeeded
			data.ExecutionError = types.StringValue("")
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
	claudeClient := claude.NewClient("", false) // temp client just for building prompt
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

// computeScaffoldHash creates a fingerprint of scaffold configuration for change detection
func (r *ProjectResourceFinal) computeScaffoldHash(scaffolds []schemas.ScaffoldModel) string {
	if len(scaffolds) == 0 {
		return ""
	}

	// Create a deterministic representation of scaffolds
	type scaffoldEntry struct {
		Path         string   `json:"path"`
		Content      string   `json:"content"`
		Generate     bool     `json:"generate,omitempty"`
		Instructions []string `json:"instructions,omitempty"`
	}

	var entries []scaffoldEntry
	for _, scaffold := range scaffolds {
		entry := scaffoldEntry{
			Path:    scaffold.Path.ValueString(),
			Content: scaffold.Content.ValueString(),
		}

		// Include optional fields if they're set
		if !scaffold.Generate.IsNull() && !scaffold.Generate.IsUnknown() {
			entry.Generate = scaffold.Generate.ValueBool()
		}
		if scaffold.Instructions != nil && len(scaffold.Instructions) > 0 {
			instructions := make([]string, len(scaffold.Instructions))
			for i, inst := range scaffold.Instructions {
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

// detectScaffoldChanges determines if scaffolds have meaningfully changed
func (r *ProjectResourceFinal) detectScaffoldChanges(ctx context.Context, oldScaffolds, newScaffolds []schemas.ScaffoldModel) bool {
	oldHash := r.computeScaffoldHash(oldScaffolds)
	newHash := r.computeScaffoldHash(newScaffolds)

	changed := oldHash != newHash

	tflog.Debug(ctx, "Scaffold change detection", map[string]interface{}{
		"old_hash":  oldHash,
		"new_hash":  newHash,
		"changed":   changed,
		"old_count": len(oldScaffolds),
		"new_count": len(newScaffolds),
	})

	return changed
}

// collectAndMergeScaffolds collects scaffolds from all sources and merges them with proper precedence
func (r *ProjectResourceFinal) collectAndMergeScaffolds(ctx context.Context, data ProjectModelFinal) []schemas.ScaffoldModel {
	merger := scaffolds.NewMerger()

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
						"scaffold_count": len(stack.Scaffolds),
						"stack_name":     stack.Name.ValueString(),
					})

					// Log the actual scaffolds with verification details
					for i, scaffold := range stack.Scaffolds {
						logData := map[string]interface{}{
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

						tflog.Info(ctx, "Stack scaffold retrieved from registry", logData)
					}

					// TODO: First add scaffolds from kits within the primary stack (lowest precedence)
					// This requires stacks to have their own kit references
					// For now, we'll just note this is where stack's kits would be processed

					// Add the primary stack's own scaffolds (higher precedence than stack's kits)
					merger.AddScaffolds(ctx, stack.Scaffolds, scaffolds.SourceStack)
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
					tflog.Info(ctx, "Processing stack scaffolds", map[string]interface{}{
						"stack_id":       stackID,
						"scaffold_count": len(stack.Scaffolds),
						"stack_name":     stack.Name.ValueString(),
					})

					// Add stack's scaffolds
					merger.AddScaffolds(ctx, stack.Scaffolds, scaffolds.SourceStack)
				} else {
					tflog.Warn(ctx, "Stack data is not StackResourceModel", map[string]interface{}{
						"stack_id":    stackID,
						"actual_type": fmt.Sprintf("%T", stackData),
					})
				}
			}
		}
	}

	// TODO: Add scaffolds from project's additional kits
	// These are kits added directly to the project to customize the stack
	// This would process data.Kits and collect their scaffolds
	// merger.AddScaffolds(ctx, kitScaffolds, scaffolds.SourceProject)

	// Finally, add project's own scaffolds (highest precedence)
	merger.AddScaffolds(ctx, data.Scaffolds, scaffolds.SourceProject)

	// Get the final merged list
	mergedScaffolds := merger.GetMergedScaffolds()

	tflog.Info(ctx, "Merged scaffolds from all sources", map[string]interface{}{
		"total_count":   len(mergedScaffolds),
		"stack_ref":     data.StackRef.ValueString(),
		"project_count": len(data.Scaffolds),
	})

	return mergedScaffolds
}
