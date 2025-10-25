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
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/files"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
	"github.com/tofukit/opentofu-provider-tofukit/internal/registry"
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
	Stack              types.Dynamic              `tfsdk:"stack"` // Reference to the stack (accepts resource reference or ID string)
	Requirements       []schemas.RequirementModel `tfsdk:"requirement"`
	Kits               types.Dynamic              `tfsdk:"kits"`  // List of kit references or IDs
	Files              types.Map                  `tfsdk:"files"` // Map of files keyed by path
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
	// Model to use for Claude execution (haiku, sonnet, opus)
	Model types.String `tfsdk:"model"`
	// Computed - stores the complete Claude prompt JSON generated during plan phase
	// This is stored in state so apply can use exactly what was planned
	PlannedPromptJSON types.String `tfsdk:"planned_prompt_json"`
	// Computed - SHA256 hash of the configuration for change detection
	PromptHash types.String `tfsdk:"prompt_hash"`
	// Computed - SHA256 hash of actual generated files for drift detection
	OutputHash types.String `tfsdk:"output_hash"`
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
			"stack": schema.DynamicAttribute{
				MarkdownDescription: "Reference to the stack this project uses (e.g., tofukit_stack.python_cli or tofukit_stack.python_cli.id)",
				Optional:            true,
			},
			"kits": schema.DynamicAttribute{
				MarkdownDescription: "List of kit references to include in the project (e.g., [tofukit_language.python, tofukit_framework.click])",
				Optional:            true,
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
			"model": schema.StringAttribute{
				MarkdownDescription: "Model to use for Claude execution. Options: 'haiku' (fast, cheap), 'sonnet' (balanced), 'opus' (most capable). Default: 'sonnet' (Claude CLI default)",
				Optional:            true,
			},
			"planned_prompt_json": schema.StringAttribute{
				MarkdownDescription: "Internal: Complete Claude prompt JSON generated and stored during apply for reference.",
				Computed:            true,
			},
			"prompt_hash": schema.StringAttribute{
				MarkdownDescription: "SHA256 hash of the configuration for change detection",
				Computed:            true,
				// No plan modifiers - hash is computed during Create/Update when all values are known
			},
			"output_hash": schema.StringAttribute{
				MarkdownDescription: "SHA256 hash of actual generated files for drift detection",
				Computed:            true,
			},
			"files": schemas.GetFilesMapAttribute(),
		},
		Blocks: map[string]schema.Block{
			"requirement": schemas.GetRequirementBlock(),
		},
	}
}

// extractIDFromDynamic extracts the ID string from a dynamic attribute that can be either:
// - A string (direct ID)
// - An object with an "id" attribute (resource reference)
func extractIDFromDynamic(ctx context.Context, dynValue types.Dynamic) string {
	if dynValue.IsNull() || dynValue.IsUnknown() {
		return ""
	}

	underlying := dynValue.UnderlyingValue()

	// Try to get as string first
	if strVal, ok := underlying.(types.String); ok && !strVal.IsNull() {
		return strVal.ValueString()
	}

	// Try to get as object and extract .id
	if objVal, ok := underlying.(types.Object); ok && !objVal.IsNull() {
		attrs := objVal.Attributes()
		if idAttr, exists := attrs["id"]; exists {
			if idStr, ok := idAttr.(types.String); ok && !idStr.IsNull() {
				return idStr.ValueString()
			}
		}
	}

	return ""
}

// extractIDsFromDynamicList extracts a list of ID strings from a dynamic attribute that can be either:
// - A list of strings (direct IDs)
// - A list of objects with "id" attributes (resource references)
func extractIDsFromDynamicList(ctx context.Context, dynValue types.Dynamic) []string {
	if dynValue.IsNull() || dynValue.IsUnknown() {
		fmt.Printf("DEBUG extractIDsFromDynamicList: value is null or unknown\n")
		return nil
	}

	var ids []string
	underlying := dynValue.UnderlyingValue()
	fmt.Printf("DEBUG extractIDsFromDynamicList: underlying type = %T\n", underlying)

	// Try to get as list
	if listVal, ok := underlying.(types.List); ok && !listVal.IsNull() {
		fmt.Printf("DEBUG extractIDsFromDynamicList: got list with %d elements\n", len(listVal.Elements()))

		// Iterate through list elements
		for i, elem := range listVal.Elements() {
			fmt.Printf("DEBUG extractIDsFromDynamicList: element[%d] type = %T\n", i, elem)

			// Try as string first
			if strVal, ok := elem.(types.String); ok && !strVal.IsNull() {
				id := strVal.ValueString()
				fmt.Printf("DEBUG extractIDsFromDynamicList: extracted string ID = %s\n", id)
				ids = append(ids, id)
				continue
			}

			// Try as object with .id
			if objVal, ok := elem.(types.Object); ok && !objVal.IsNull() {
				attrs := objVal.Attributes()
				fmt.Printf("DEBUG extractIDsFromDynamicList: object has %d attributes: %v\n", len(attrs), func() []string {
					keys := make([]string, 0, len(attrs))
					for k := range attrs {
						keys = append(keys, k)
					}
					return keys
				}())

				if idAttr, exists := attrs["id"]; exists {
					if idStr, ok := idAttr.(types.String); ok && !idStr.IsNull() {
						id := idStr.ValueString()
						fmt.Printf("DEBUG extractIDsFromDynamicList: extracted object ID = %s\n", id)
						ids = append(ids, id)
					}
				}
			}
		}
	} else {
		fmt.Printf("DEBUG extractIDsFromDynamicList: not a list or list is null\n")
	}

	fmt.Printf("DEBUG extractIDsFromDynamicList: returning %d IDs: %v\n", len(ids), ids)
	return ids
}

// ModifyPlan is called during the plan phase
// We don't compute PromptHash here because it may include references to other resources
// that are Unknown during plan. Instead, we compute it during Create/Update when all
// values are known, which prevents "inconsistent final plan" errors.
func (r *ProjectResourceFinal) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// No hash computation in ModifyPlan - it happens during Create/Update
	// This prevents inconsistencies when resource references are Unknown during plan
	tflog.Debug(ctx, "ModifyPlan: Hash computation deferred to apply phase")

	// FIX: Correct file list tracking by path instead of position
	// This prevents Terraform from thinking a file was renamed when we remove a middle file

	// Skip if this is resource creation (no state yet)
	if req.State.Raw.IsNull() {
		return
	}

	var state, plan, config ProjectModelFinal

	// Get current state
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get planned changes
	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get config
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert maps to lists for processing
	stateFilesList := schemas.FilesMapToList(ctx, state.Files)
	configFilesList := schemas.FilesMapToList(ctx, config.Files)

	// If no files in either state or config, nothing to fix
	if len(stateFilesList) == 0 && len(configFilesList) == 0 {
		return
	}

	tflog.Debug(ctx, "ModifyPlan: Analyzing file changes (map schema eliminates position-based issues)", map[string]interface{}{
		"state_files":  len(stateFilesList),
		"config_files": len(configFilesList),
	})

	// Build map of state files by path
	stateFilesByPath := make(map[string]schemas.FileModelWithPath)
	for _, file := range stateFilesList {
		stateFilesByPath[file.Path] = file
		tflog.Trace(ctx, "State file", map[string]interface{}{"path": file.Path})
	}

	// Build map of config files by path
	configFilesByPath := make(map[string]schemas.FileModelWithPath)
	for _, file := range configFilesList {
		configFilesByPath[file.Path] = file
		tflog.Trace(ctx, "Config file", map[string]interface{}{"path": file.Path})
	}

	// Determine operations using path-based comparison
	added := []string{}
	removed := []string{}
	modified := []string{}
	unchanged := []string{}

	// Find removed files (in state but not in config)
	for path := range stateFilesByPath {
		if _, existsInConfig := configFilesByPath[path]; !existsInConfig {
			removed = append(removed, path)
		}
	}

	// Find added, modified, and unchanged files
	for path, configFile := range configFilesByPath {
		if stateFile, existsInState := stateFilesByPath[path]; existsInState {
			// File exists in both - check if modified
			stateContent := stateFile.Content.ValueString()
			configContent := configFile.Content.ValueString()

			if stateContent != configContent {
				modified = append(modified, path)
			} else {
				unchanged = append(unchanged, path)
			}
		} else {
			// New file
			added = append(added, path)
		}
	}

	tflog.Info(ctx, "ModifyPlan: File operations detected", map[string]interface{}{
		"added":     added,
		"removed":   removed,
		"modified":  modified,
		"unchanged": unchanged,
	})

	// NOTE: With map schema, files are keyed by path, eliminating position-based comparison issues.
	// Terraform's MapAttribute compares by key (path), not position, so plan display is now accurate.

	tflog.Debug(ctx, "ModifyPlan: File operations detected (map schema ensures accurate plan display)", map[string]interface{}{
		"config_count": len(configFilesList),
		"state_count":  len(stateFilesList),
		"added":        added,
		"removed":      removed,
		"modified":     modified,
	})

	// Update the plan
	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
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

	// Validate project has at least one content source
	hasFiles := !data.Files.IsNull() && !data.Files.IsUnknown() && len(data.Files.Elements()) > 0
	hasKits := !data.Kits.IsNull() && !data.Kits.IsUnknown()
	hasStack := !data.Stack.IsNull() && !data.Stack.IsUnknown()
	hasRequirements := len(data.Requirements) > 0

	if !hasFiles && !hasKits && !hasStack && !hasRequirements {
		resp.Diagnostics.AddError(
			"Empty Project Configuration",
			fmt.Sprintf(
				"Project '%s' must specify at least one of the following:\n"+
					"  - file {} blocks (explicit files to create)\n"+
					"  - kits (language/framework setup)\n"+
					"  - stack (reference to a stack resource)\n"+
					"  - requirement {} blocks (features to implement)\n\n"+
					"Example - Bootstrap a new project using requirements:\n"+
					"  requirement {\n"+
					"    name = \"project-structure\"\n"+
					"    instruction {\n"+
					"      prompt = \"Create a %s\"\n"+
					"    }\n"+
					"  }",
				data.Name.ValueString(),
				data.Description.ValueString(),
			),
		)
		return
	}

	tflog.Info(ctx, "Project validation passed", map[string]interface{}{
		"project_name":     data.Name.ValueString(),
		"has_files":        hasFiles,
		"has_kits":         hasKits,
		"has_stack":        hasStack,
		"has_requirements": hasRequirements,
	})

	// Get provider configuration
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	if provData, ok := r.ProviderData.(interface {
		GetOutputPath() string
		GetClaudeHomeDirectory() string
		GetDebug() bool
	}); ok {
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
		_ = provData.GetDebug() // debug is handled in executeClaudeCode
	}

	// Initialize all computed fields to ensure they're never unknown
	data.ExecutionStarted = types.StringValue("")
	data.ExecutionCompleted = types.StringValue("")
	data.ExecutionError = types.StringValue("")
	data.ProjectPath = types.StringValue(outputPath) // Files created directly in output_path, not in subdirectory

	// Collect merged files FIRST to compute correct hash
	mergedFiles := r.collectAndMergeFiles(ctx, data)

	// Compute config hash for change detection (now that all dependencies are resolved)
	configHash := r.computeConfigHash(ctx, data)
	data.PromptHash = types.StringValue(configHash)
	tflog.Info(ctx, "Create: Computed config hash", map[string]interface{}{
		"project_name": data.Name.ValueString(),
		"config_hash":  configHash,
	})

	// Use merged files for hash to track ALL files (including from stacks)
	data.FileHash = types.StringValue(r.computeFileHash(mergedFiles))
	// Initialize LastApplied with current time since files will be applied
	data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))

	// For Create, all files are "add" operations (no old state)
	// Enrich files with action-based instructions
	enrichedFiles := files.EnrichFilesWithInstructions([]schemas.FileModelWithPath{}, mergedFiles)
	tflog.Info(ctx, "Enriched files with action-based instructions for Create", map[string]interface{}{
		"project_name": data.Name.ValueString(),
		"file_count":   len(enrichedFiles),
	})

	// Execute Claude Code - Claude will create all files based on files
	data.ExecutionStatus = types.StringValue("pending")
	data.ExecutionStarted = types.StringValue(time.Now().Format(time.RFC3339))

	// DEBUG: Check kits in Create
	debugFile, _ := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if debugFile != nil {
		fmt.Fprintf(debugFile, "\n=== Create method ===\n")
		fmt.Fprintf(debugFile, "Create: data.Kits.IsNull()=%v, IsUnknown()=%v\n", data.Kits.IsNull(), data.Kits.IsUnknown())
		debugFile.Close()
	}

	// Build output data with enriched files
	outputData := r.buildOutputDataWithFiles(ctx, data, enrichedFiles)

	r.writeJSONFile(ctx, data, outputData, outputPath)

	// Note: Debug files are written during ModifyPlan (plan phase)
	// Prompt is regenerated deterministically during Create (apply phase)

	if err := r.executeClaudeCode(ctx, &data, outputData, enrichedFiles, outputPath, claudeHomeDir, false); err != nil {
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

		// Compute output hash after successful execution
		// This captures ALL files Claude created (including untracked ones)
		outputHash := r.computeOutputHash(ctx, data.ProjectPath.ValueString())
		data.OutputHash = types.StringValue(outputHash)
		tflog.Info(ctx, "Computed output hash after successful execution", map[string]interface{}{
			"output_hash":  outputHash,
			"project_path": data.ProjectPath.ValueString(),
		})

		// NOTE: We don't store enrichedFiles (with action-based instructions) in state
		// data.Files keeps the original user configuration
		// Action-based instructions are computed fresh each time from the diff

		// Generate and store the prompt that was actually used
		// This happens after apply when all dependencies are resolved
		promptJSON, err := r.generatePromptJSON(ctx, outputData, data.SystemPrompt.ValueString())
		if err != nil {
			tflog.Warn(ctx, "Failed to generate prompt JSON for state storage", map[string]interface{}{
				"error": err.Error(),
			})
			// Don't fail the resource, just skip storing the prompt
		} else {
			data.PlannedPromptJSON = types.StringValue(promptJSON)
			tflog.Info(ctx, "Stored prompt in state after successful execution", map[string]interface{}{
				"prompt_size": len(promptJSON),
			})
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

	// DEBUG: Check kits in Read
	debugFile, _ := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if debugFile != nil {
		fmt.Fprintf(debugFile, "\n=== Read method ===\n")
		fmt.Fprintf(debugFile, "Read: data.Kits.IsNull()=%v, IsUnknown()=%v\n", data.Kits.IsNull(), data.Kits.IsUnknown())
		debugFile.Close()
	}

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
		"project_id":   data.ID.ValueString(),
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

	// Validate project has at least one content source
	hasFiles := !data.Files.IsNull() && !data.Files.IsUnknown() && len(data.Files.Elements()) > 0
	hasKits := !data.Kits.IsNull() && !data.Kits.IsUnknown()
	hasStack := !data.Stack.IsNull() && !data.Stack.IsUnknown()
	hasRequirements := len(data.Requirements) > 0

	if !hasFiles && !hasKits && !hasStack && !hasRequirements {
		resp.Diagnostics.AddError(
			"Empty Project Configuration",
			fmt.Sprintf(
				"Project '%s' must specify at least one of the following:\n"+
					"  - file {} blocks (explicit files to create)\n"+
					"  - kits (language/framework setup)\n"+
					"  - stack (reference to a stack resource)\n"+
					"  - requirement {} blocks (features to implement)\n\n"+
					"Cannot update to an empty project configuration.",
				data.Name.ValueString(),
			),
		)
		return
	}

	tflog.Info(ctx, "Project validation passed", map[string]interface{}{
		"project_name":     data.Name.ValueString(),
		"has_files":        hasFiles,
		"has_kits":         hasKits,
		"has_stack":        hasStack,
		"has_requirements": hasRequirements,
	})

	// Compute new config hash from planned data (all dependencies are resolved now)
	newConfigHash := r.computeConfigHash(ctx, data)
	data.PromptHash = types.StringValue(newConfigHash)
	tflog.Info(ctx, "Update: Computed new config hash", map[string]interface{}{
		"project_id": state.ID.ValueString(),
		"old_hash":   state.PromptHash.ValueString(),
		"new_hash":   newConfigHash,
	})

	// Detect prompt changes using hash comparison (includes all changes: files, requirements, system prompt, etc)
	promptChanged := !state.PromptHash.Equal(data.PromptHash)

	// Collect merged files early to detect file specification changes (including from stacks)
	oldMergedFiles := r.collectAndMergeFiles(ctx, state)
	mergedFiles := r.collectAndMergeFiles(ctx, data)
	currentFileHash := r.computeFileHash(mergedFiles)

	// Enrich files with action-based instructions
	enrichedFiles := files.EnrichFilesWithInstructions(oldMergedFiles, mergedFiles)
	tflog.Info(ctx, "Enriched files with action-based instructions", map[string]interface{}{
		"project_id": state.ID.ValueString(),
		"file_count": len(enrichedFiles),
	})

	// Detect file specification changes (files from project, stacks, kits)
	fileSpecChanged := state.FileHash.IsNull() || state.FileHash.ValueString() != currentFileHash

	// Detect output drift (files modified/deleted on disk, or LLM non-determinism)
	outputDrifted := false
	if !state.OutputHash.IsNull() && !state.OutputHash.IsUnknown() && state.OutputHash.ValueString() != "" {
		currentOutputHash := r.computeOutputHash(ctx, state.ProjectPath.ValueString())
		outputDrifted = (currentOutputHash != state.OutputHash.ValueString())
		if outputDrifted {
			tflog.Warn(ctx, "Output drift detected - files changed outside Terraform", map[string]interface{}{
				"project_id":    state.ID.ValueString(),
				"expected_hash": state.OutputHash.ValueString(),
				"current_hash":  currentOutputHash,
			})
		}
	}

	// Preserve ID and project path from existing state
	data.ID = state.ID
	data.ProjectPath = state.ProjectPath

	// Declare variables outside if/else to make them available in both branches
	var outputData map[string]interface{}

	if promptChanged || fileSpecChanged || outputDrifted {
		if promptChanged {
			tflog.Info(ctx, "Config changed - triggering re-execution", map[string]interface{}{
				"project_id":      data.ID.ValueString(),
				"old_prompt_hash": state.PromptHash.ValueString(),
				"new_prompt_hash": data.PromptHash.ValueString(),
			})
		}
		if fileSpecChanged {
			tflog.Info(ctx, "File specification changed - triggering re-execution", map[string]interface{}{
				"project_id":    data.ID.ValueString(),
				"old_file_hash": state.FileHash.ValueString(),
				"new_file_hash": currentFileHash,
			})
		}
		if outputDrifted {
			tflog.Info(ctx, "Output drift detected - triggering re-execution", map[string]interface{}{
				"project_id": data.ID.ValueString(),
			})
		}

		// Always execute Claude Code for prompt changes (files, requirements, system prompt, etc)
		// Execute Claude Code with the updated specification
		// Preserve existing timestamps - they represent when the project was first executed
		// Only update status to reflect the re-execution
		data.ExecutionStatus = types.StringValue("pending")
		// IMPORTANT: Preserve timestamps from state to avoid provider inconsistency
		data.ExecutionStarted = state.ExecutionStarted
		data.ExecutionCompleted = state.ExecutionCompleted

		// Use enriched files (with action-based instructions) for output
		// Build output data with enriched files
		outputData = r.buildOutputDataWithFiles(ctx, data, enrichedFiles)

		r.writeJSONFile(ctx, data, outputData, outputPath)

		// Note: Debug files are written during ModifyPlan (plan phase)
		// Prompt is regenerated deterministically during Update (apply phase)

		if err := r.executeClaudeCode(ctx, &data, outputData, enrichedFiles, outputPath, claudeHomeDir, true); err != nil {
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
			// Use the currentFileHash we computed earlier
			data.FileHash = types.StringValue(currentFileHash)
			data.LastApplied = types.StringValue(time.Now().Format(time.RFC3339))

			// Compute output hash after successful execution
			outputHash := r.computeOutputHash(ctx, data.ProjectPath.ValueString())
			data.OutputHash = types.StringValue(outputHash)
			tflog.Info(ctx, "Computed output hash after successful update", map[string]interface{}{
				"output_hash": outputHash,
			})

			// NOTE: We don't store enrichedFiles (with action-based instructions) in state
			// data.Files keeps the original user configuration
			// Action-based instructions are computed fresh each time from the diff

			// Generate and store the prompt that was actually used
			// This happens after apply when all dependencies are resolved
			promptJSON, err := r.generatePromptJSON(ctx, outputData, data.SystemPrompt.ValueString())
			if err != nil {
				tflog.Warn(ctx, "Failed to generate prompt JSON for state storage", map[string]interface{}{
					"error": err.Error(),
				})
				// Don't fail the resource, just skip storing the prompt
			} else {
				data.PlannedPromptJSON = types.StringValue(promptJSON)
				tflog.Info(ctx, "Stored prompt in state after successful update", map[string]interface{}{
					"prompt_size": len(promptJSON),
				})
			}

			// Preserve the original timestamps even on successful re-execution
			data.ExecutionStarted = state.ExecutionStarted
			data.ExecutionCompleted = state.ExecutionCompleted
		}

	} else {
		// No changes detected, preserve existing execution state
		tflog.Info(ctx, "No changes detected, preserving execution state", map[string]interface{}{
			"project_id":  data.ID.ValueString(),
			"prompt_hash": data.PromptHash.ValueString(),
			"file_hash":   currentFileHash,
		})

		// mergedFiles already collected earlier for hash comparison

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
		data.OutputHash = state.OutputHash
		data.PlannedPromptJSON = state.PlannedPromptJSON // Preserve prompt from state
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
		"prompt_changed":   promptChanged,
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
			filesList := schemas.FilesMapToList(ctx, data.Files)
			if len(filesList) > 0 {
				fileManager := files.NewManager(projectPath)
				fileManager.RemoveAllFiles(ctx, filesList)
				tflog.Info(ctx, "Removed file files", map[string]interface{}{
					"project_path": projectPath,
					"count":        len(filesList),
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
	// Collect merged files and use them
	mergedFiles := r.collectAndMergeFiles(ctx, data)
	return r.buildOutputDataWithFiles(ctx, data, mergedFiles)
}

func (r *ProjectResourceFinal) buildOutputDataWithFiles(ctx context.Context, data ProjectModelFinal, filesToUse []schemas.FileModelWithPath) map[string]interface{} {

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

		// Add instructions (new structure with prompt and constraints)
		instructions := []map[string]interface{}{}
		for _, inst := range req.Instructions {
			instData := map[string]interface{}{
				"prompt": inst.Prompt.ValueString(),
			}

			// Add constraints if present
			if inst.Constraints != nil && len(inst.Constraints) > 0 {
				constraints := []string{}
				for _, c := range inst.Constraints {
					if !c.IsNull() && !c.IsUnknown() {
						constraints = append(constraints, c.ValueString())
					}
				}
				if len(constraints) > 0 {
					instData["constraints"] = constraints
				}
			}

			instructions = append(instructions, instData)
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

	// Use provided files (may be enriched with action-based instructions)
	mergedFiles := filesToUse
	tflog.Info(ctx, "Collected merged files for output", map[string]interface{}{
		"count":        len(mergedFiles),
		"project_name": data.Name.ValueString(),
	})
	files := map[string]interface{}{}
	for _, file := range mergedFiles {
		fileData := map[string]interface{}{}

		// Debug log
		tflog.Debug(ctx, "Processing file for output", map[string]interface{}{
			"path":               file.Path,
			"has_verifications":  len(file.Verifications) > 0,
			"verification_count": len(file.Verifications),
		})

		if !file.Content.IsNull() && !file.Content.IsUnknown() {
			fileData["content"] = file.Content.ValueString()
		}

		if file.Instructions != nil && len(file.Instructions) > 0 {
			instructions := []map[string]interface{}{}
			for _, inst := range file.Instructions {
				instData := map[string]interface{}{
					"prompt": inst.Prompt.ValueString(),
				}

				// Add constraints if present
				if inst.Constraints != nil && len(inst.Constraints) > 0 {
					constraints := []string{}
					for _, c := range inst.Constraints {
						if !c.IsNull() && !c.IsUnknown() {
							constraints = append(constraints, c.ValueString())
						}
					}
					if len(constraints) > 0 {
						instData["constraints"] = constraints
					}
				}

				instructions = append(instructions, instData)
			}
			fileData["instructions"] = instructions
		}

		// Add verifications if present
		if len(file.Verifications) > 0 {
			verifications := []map[string]interface{}{}
			for _, v := range file.Verifications {
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

		files[file.Path] = fileData
	}
	// Always include files, even if empty, for consistency
	outputData["files"] = files

	// Add kits directly from the Dynamic attribute (they're already fully populated in state)
	kits := map[string]interface{}{}

	// DEBUG: Write to file
	debugFile, _ := os.Create("/tmp/tofukit-debug.log")
	if debugFile != nil {
		fmt.Fprintf(debugFile, "DEBUG buildOutputData: data.Kits.IsNull()=%v, IsUnknown()=%v\n", data.Kits.IsNull(), data.Kits.IsUnknown())
		defer debugFile.Close()
	}

	if !data.Kits.IsNull() && !data.Kits.IsUnknown() {
		// The kits are already stored as full objects in the state, not just IDs
		// Convert the Dynamic value to JSON-serializable format
		underlying := data.Kits.UnderlyingValue()
		if debugFile != nil {
			fmt.Fprintf(debugFile, "DEBUG buildOutputData: underlying type=%T\n", underlying)
		}

		// The underlying value can be either a List or a Tuple depending on the context
		var elements []attr.Value

		if listVal, ok := underlying.(types.List); ok && !listVal.IsNull() {
			elements = listVal.Elements()
			if debugFile != nil {
				fmt.Fprintf(debugFile, "DEBUG buildOutputData: Got list with %d elements\n", len(elements))
			}
		} else if tupleVal, ok := underlying.(types.Tuple); ok && !tupleVal.IsNull() {
			elements = tupleVal.Elements()
			if debugFile != nil {
				fmt.Fprintf(debugFile, "DEBUG buildOutputData: Got tuple with %d elements\n", len(elements))
			}
		}

		if len(elements) > 0 {
			for i, elem := range elements {
				if debugFile != nil {
					fmt.Fprintf(debugFile, "DEBUG buildOutputData: element[%d] type=%T\n", i, elem)
				}

				if objVal, ok := elem.(types.Object); ok && !objVal.IsNull() {
					attrs := objVal.Attributes()
					if debugFile != nil {
						fmt.Fprintf(debugFile, "DEBUG buildOutputData: element[%d] has %d attributes\n", i, len(attrs))
					}

					// Extract ID to use as map key
					if idAttr, exists := attrs["id"]; exists {
						if debugFile != nil {
							fmt.Fprintf(debugFile, "DEBUG buildOutputData: element[%d] has id attribute\n", i)
						}

						if idStr, ok := idAttr.(types.String); ok && !idStr.IsNull() {
							kitID := idStr.ValueString()
							if debugFile != nil {
								fmt.Fprintf(debugFile, "DEBUG buildOutputData: extracted kit ID=%s\n", kitID)
							}

							// Build kit data map from all attributes
							kitData := map[string]interface{}{}
							for key, val := range attrs {
								// Convert each attribute to a JSON-serializable value
								switch v := val.(type) {
								case types.String:
									if !v.IsNull() {
										kitData[key] = v.ValueString()
									}
								case types.List:
									// Handle lists (like requirements)
									if !v.IsNull() {
										listItems := []interface{}{}
										for _, listElem := range v.Elements() {
											// Recursively handle list elements
											if objElem, ok := listElem.(types.Object); ok && !objElem.IsNull() {
												itemMap := map[string]interface{}{}
												for k, v2 := range objElem.Attributes() {
													if strVal, ok := v2.(types.String); ok && !strVal.IsNull() {
														itemMap[k] = strVal.ValueString()
													}
													// Handle nested lists in requirements (instructions, verifications)
													if listVal2, ok := v2.(types.List); ok && !listVal2.IsNull() {
														nestedList := []interface{}{}
														for _, nestedElem := range listVal2.Elements() {
															if strVal, ok := nestedElem.(types.String); ok && !strVal.IsNull() {
																nestedList = append(nestedList, strVal.ValueString())
															} else if objVal3, ok := nestedElem.(types.Object); ok && !objVal3.IsNull() {
																nestedMap := map[string]interface{}{}
																for k3, v3 := range objVal3.Attributes() {
																	if strVal3, ok := v3.(types.String); ok && !strVal3.IsNull() {
																		nestedMap[k3] = strVal3.ValueString()
																	} else if listVal3, ok := v3.(types.List); ok && !listVal3.IsNull() {
																		// Handle lists within nested objects (e.g., constraints in instructions)
																		deepList := []string{}
																		for _, deepElem := range listVal3.Elements() {
																			if strVal4, ok := deepElem.(types.String); ok && !strVal4.IsNull() {
																				deepList = append(deepList, strVal4.ValueString())
																			}
																		}
																		nestedMap[k3] = deepList
																	}
																}
																nestedList = append(nestedList, nestedMap)
															}
														}
														itemMap[k] = nestedList
													}
												}
												listItems = append(listItems, itemMap)
											}
										}
										kitData[key] = listItems
									}
								}
							}

							if debugFile != nil {
								fmt.Fprintf(debugFile, "DEBUG buildOutputData: adding kit %s with %d fields\n", kitID, len(kitData))
							}

							kits[kitID] = kitData
							tflog.Debug(ctx, "Added kit from state", map[string]interface{}{
								"kit_id": kitID,
							})
						}
					}
				}
			}
		}
	}

	if debugFile != nil {
		fmt.Fprintf(debugFile, "DEBUG buildOutputData: final kits map has %d entries\n", len(kits))
	}

	outputData["kits"] = kits

	return outputData
}

// writeJSONFile writes the output data to a JSON file
func (r *ProjectResourceFinal) writeJSONFile(ctx context.Context, data ProjectModelFinal, outputData map[string]interface{}, outputPath string) {
	// This function is now deprecated - writeDebugFiles handles JSON output
	// Keeping empty for backward compatibility but all writing is done in writeDebugFiles
	tflog.Debug(ctx, "writeJSONFile called but delegating to writeDebugFiles")
}

// executeClaudeCode executes Claude Code with the project specification
// If PlannedPromptJSON is available (from plan phase), it uses that exact prompt
// Otherwise, it builds the prompt from outputData (legacy path)
func (r *ProjectResourceFinal) executeClaudeCode(ctx context.Context, data *ProjectModelFinal, outputData map[string]interface{}, mergedFiles []schemas.FileModelWithPath, outputPath string, claudeHomeDir string, preserveTimestamps bool) error {
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

	// Get model from resource data (defaults to sonnet if not specified)
	model := "sonnet"
	if !data.Model.IsNull() && !data.Model.IsUnknown() {
		model = data.Model.ValueString()
	}

	executor := claude.NewExecutor(claudeHomeDir)
	executor.SetDebug(debug)
	executor.SetOutputPath(outputPath)
	executor.SetSystemPrompt(systemPrompt)
	executor.SetModel(model)

	// Get max retries from provider config (default to 3)
	maxRetries := 3
	if provData, ok := r.ProviderData.(interface{ GetMaxRetries() int }); ok {
		if retries := provData.GetMaxRetries(); retries > 0 {
			maxRetries = retries
		}
	}

	// Check if we have a planned prompt from the plan phase (stored in state)
	var promptJSON string
	var status *claude.ExecutionStatus
	var report *files.VerificationReport
	var err error

	if !data.PlannedPromptJSON.IsNull() && !data.PlannedPromptJSON.IsUnknown() && data.PlannedPromptJSON.ValueString() != "" {
		// Use the planned prompt from plan phase (stored in state)
		promptJSON = data.PlannedPromptJSON.ValueString()
		tflog.Info(ctx, "Using planned prompt from state", map[string]interface{}{
			"prompt_size": len(promptJSON),
		})
	} else {
		// Fallback: Generate prompt from outputData if not in state
		promptJSON, err = r.generatePromptJSON(ctx, outputData, data.SystemPrompt.ValueString())
		if err != nil {
			return fmt.Errorf("failed to generate Claude prompt: %w", err)
		}
		tflog.Warn(ctx, "No planned prompt in state, generated from outputData", map[string]interface{}{
			"prompt_size": len(promptJSON),
		})
	}

	// Execute with the prompt
	status, report, err = executor.ExecuteWithPromptJSON(ctx, promptJSON, outputPath, mergedFiles, maxRetries)
	if err != nil {
		// Execution or verification failed
		data.ExecutionStatus = types.StringValue("failed")
		if report != nil && !report.AllPassed {
			// Verification failed after retries
			data.ExecutionError = types.StringValue(fmt.Sprintf("Verification failed after %d attempts:\n%s", maxRetries, report.GetFailureSummary()))
			data.ExecutionStatus = types.StringValue("verification_failed")
		} else {
			// Execution failed
			data.ExecutionError = types.StringValue(err.Error())
		}
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

	// Clear any previous error - verification passed!
	data.ExecutionError = types.StringValue("")

	if report != nil {
		tflog.Info(ctx, "All verifications passed", map[string]interface{}{
			"passed_count": report.PassedCount,
			"total_count":  report.PassedCount + report.FailedCount,
		})
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
	// Note: ProviderData might be nil during validation phase
	if r.ProviderData != nil {
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
}

// generatePromptJSON generates the Claude prompt JSON from the output data
// This is extracted into a helper so it can be called during both plan and apply phases
func (r *ProjectResourceFinal) generatePromptJSON(ctx context.Context, outputData map[string]interface{}, systemPrompt string) (string, error) {
	// Use the BuildProjectPrompt from claude package to generate the prompt
	prompt := claude.BuildProjectPrompt(outputData, systemPrompt)

	// Convert to JSON
	promptJSON, err := prompt.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to convert prompt to JSON: %w", err)
	}

	return promptJSON, nil
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

	// Generate timestamp for this debug session
	timestamp := time.Now().Format("20060102-150405")

	// Write project-{name}-{timestamp}.json - the full project specification
	projectName := data.Name.ValueString()
	projectSpecPath := filepath.Join(debugDir, fmt.Sprintf("project-%s-%s.json", projectName, timestamp))
	specJSON, err := json.MarshalIndent(outputData, "", "  ")
	if err == nil {
		if writeErr := os.WriteFile(projectSpecPath, specJSON, 0644); writeErr == nil {
			tflog.Debug(ctx, "Wrote project specification", map[string]interface{}{
				"path": projectSpecPath,
				"size": len(specJSON),
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

	// Write claude-prompt-{timestamp}.md using the template-based approach
	promptMdPath := filepath.Join(debugDir, fmt.Sprintf("claude-prompt-%s.md", timestamp))
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

	// Write claude-prompt-{timestamp}.json - the actual JSON prompt
	promptJSONPath := filepath.Join(debugDir, fmt.Sprintf("claude-prompt-%s.json", timestamp))
	// Pretty-print the JSON for readability in debug
	var prettyJSON bytes.Buffer
	json.Indent(&prettyJSON, []byte(jsonPrompt), "", "  ")
	if writeErr := os.WriteFile(promptJSONPath, prettyJSON.Bytes(), 0644); writeErr == nil {
		tflog.Debug(ctx, "Wrote prompt JSON file", map[string]interface{}{
			"path": promptJSONPath,
			"size": prettyJSON.Len(),
		})
	}

	// Write initial entry to claude-log-{timestamp}.jsonl
	logPath := filepath.Join(debugDir, fmt.Sprintf("claude-log-%s.jsonl", timestamp))
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
func (r *ProjectResourceFinal) computeFileHash(files []schemas.FileModelWithPath) string {
	if len(files) == 0 {
		return ""
	}

	// Create a deterministic representation of files
	type instructionEntry struct {
		Prompt      string   `json:"prompt"`
		Constraints []string `json:"constraints,omitempty"`
	}
	type fileEntry struct {
		Path         string             `json:"path"`
		Content      string             `json:"content"`
		Instructions []instructionEntry `json:"instructions,omitempty"`
	}

	var entries []fileEntry
	for _, file := range files {
		entry := fileEntry{
			Path:    file.Path,
			Content: file.Content.ValueString(),
		}

		if file.Instructions != nil && len(file.Instructions) > 0 {
			instructions := []instructionEntry{}
			for _, inst := range file.Instructions {
				instEntry := instructionEntry{
					Prompt: inst.Prompt.ValueString(),
				}

				// Add constraints if present
				if inst.Constraints != nil && len(inst.Constraints) > 0 {
					constraints := []string{}
					for _, c := range inst.Constraints {
						if !c.IsNull() && !c.IsUnknown() {
							constraints = append(constraints, c.ValueString())
						}
					}
					if len(constraints) > 0 {
						instEntry.Constraints = constraints
					}
				}

				instructions = append(instructions, instEntry)
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

// computeConfigHash creates a hash of the user's configuration (not runtime state)
// This ensures the hash is deterministic and doesn't change when dependencies are created
func (r *ProjectResourceFinal) computeConfigHash(ctx context.Context, data ProjectModelFinal) string {
	var parts []string

	// 1. Hash files from config (not merged, not from registry)
	filesList := schemas.FilesMapToList(ctx, data.Files)
	for _, file := range filesList {
		parts = append(parts, "file:"+file.Path)
		if !file.Content.IsNull() && !file.Content.IsUnknown() {
			parts = append(parts, "content:"+file.Content.ValueString())
		}
		if file.Instructions != nil {
			for _, inst := range file.Instructions {
				parts = append(parts, "file_instruction_prompt:"+inst.Prompt.ValueString())
				if inst.Constraints != nil {
					for _, c := range inst.Constraints {
						if !c.IsNull() && !c.IsUnknown() {
							parts = append(parts, "file_instruction_constraint:"+c.ValueString())
						}
					}
				}
			}
		}
	}

	// 2. Hash requirements
	for _, req := range data.Requirements {
		parts = append(parts, "requirement:"+req.Name.ValueString())
		for _, inst := range req.Instructions {
			parts = append(parts, "instruction_prompt:"+inst.Prompt.ValueString())
			if inst.Constraints != nil {
				for _, c := range inst.Constraints {
					if !c.IsNull() && !c.IsUnknown() {
						parts = append(parts, "instruction_constraint:"+c.ValueString())
					}
				}
			}
		}
		for _, ver := range req.Verification {
			parts = append(parts, "verification:"+ver.Command.ValueString())
			if !ver.Expect.IsNull() && !ver.Expect.IsUnknown() {
				parts = append(parts, "expect:"+ver.Expect.ValueString())
			}
		}
	}

	// 3. Hash kit IDs (just IDs, not full kit data from registry)
	kitIDs := extractIDsFromDynamicList(ctx, data.Kits)
	for _, kitID := range kitIDs {
		parts = append(parts, "kit:"+kitID)
	}

	// 4. Hash system prompt
	if !data.SystemPrompt.IsNull() && !data.SystemPrompt.IsUnknown() {
		parts = append(parts, "system_prompt:"+data.SystemPrompt.ValueString())
	}

	// 5. Hash stack reference (name only, not contents)
	stackID := extractIDFromDynamic(ctx, data.Stack)
	if stackID != "" {
		parts = append(parts, "stack:"+stackID)
	}

	// Sort for determinism
	sort.Strings(parts)

	// Hash the combined config
	combined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// computeOutputHash creates a hash of all actual files in the project directory
// This detects drift when files are manually modified/deleted or when LLM creates different files
func (r *ProjectResourceFinal) computeOutputHash(ctx context.Context, projectPath string) string {
	if projectPath == "" {
		return ""
	}

	// Check if project path exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return ""
	}

	type fileEntry struct {
		Path string `json:"path"`
		Hash string `json:"hash"`
	}

	var entries []fileEntry

	// Walk all files in project directory
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and debug directory
		if info.IsDir() {
			if strings.Contains(path, ".debug") {
				return filepath.SkipDir
			}
			return nil
		}

		// Compute relative path
		relPath, err := filepath.Rel(projectPath, path)
		if err != nil {
			relPath = path
		}

		// Hash file contents
		fileData, err := os.ReadFile(path)
		if err != nil {
			tflog.Warn(ctx, "Failed to read file for output hash", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			return nil // Skip file but continue walking
		}

		fileHash := sha256.Sum256(fileData)
		entries = append(entries, fileEntry{
			Path: relPath,
			Hash: hex.EncodeToString(fileHash[:]),
		})

		return nil
	})

	if err != nil {
		tflog.Warn(ctx, "Failed to walk project directory for output hash", map[string]interface{}{
			"path":  projectPath,
			"error": err.Error(),
		})
		return ""
	}

	if len(entries) == 0 {
		return ""
	}

	// Sort entries by path for deterministic hash
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	// Create JSON representation and hash it
	jsonData, err := json.Marshal(entries)
	if err != nil {
		tflog.Warn(ctx, "Failed to marshal file entries for output hash", map[string]interface{}{
			"error": err.Error(),
		})
		return ""
	}

	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// detectFileChanges determines if files have meaningfully changed
func (r *ProjectResourceFinal) detectFileChanges(ctx context.Context, oldFiles, newFiles []schemas.FileModelWithPath) bool {
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

// getKitFilesFromRegistry fetches kit data from registry and extracts their files
func (r *ProjectResourceFinal) getKitFilesFromRegistry(ctx context.Context, kitIDs []string, reg *registry.Registry) []schemas.FileModelWithPath {
	var allFiles []schemas.FileModelWithPath

	if reg == nil || len(kitIDs) == 0 {
		return allFiles
	}

	for _, kitID := range kitIDs {
		if kitData, exists := reg.GetComponent(kitID); exists {
			if kit, ok := kitData.(ComponentResourceModel); ok {
				fileCount := 0
				if !kit.Files.IsNull() && !kit.Files.IsUnknown() {
					fileCount = len(kit.Files.Elements())
				}
				tflog.Info(ctx, "Extracting files from kit", map[string]interface{}{
					"kit_id":     kitID,
					"kit_name":   kit.Name.ValueString(),
					"file_count": fileCount,
				})
				// Convert map to list
				kitFiles := schemas.FilesMapToList(ctx, kit.Files)
				allFiles = append(allFiles, kitFiles...)
			} else {
				tflog.Warn(ctx, "Kit data is not ComponentResourceModel", map[string]interface{}{
					"kit_id":      kitID,
					"actual_type": fmt.Sprintf("%T", kitData),
				})
			}
		} else {
			tflog.Warn(ctx, "Kit not found in registry", map[string]interface{}{
				"kit_id": kitID,
			})
		}
	}

	tflog.Info(ctx, "Collected files from kits", map[string]interface{}{
		"kit_count":   len(kitIDs),
		"total_files": len(allFiles),
	})

	return allFiles
}

// collectAndMergeFiles collects files from all sources and merges them with proper precedence
func (r *ProjectResourceFinal) collectAndMergeFiles(ctx context.Context, data ProjectModelFinal) []schemas.FileModelWithPath {
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
		stackID := extractIDFromDynamic(ctx, data.Stack)
		if stackID != "" {
			tflog.Info(ctx, "Looking for specific stack", map[string]interface{}{
				"stack_id": stackID,
			})

			// Get the primary stack this project instantiates
			if stackData, exists := reg.GetStack(stackID); exists {
				if stack, ok := stackData.(StackResourceModel); ok {
					// Convert stack files from map to list
					stackFiles := schemas.FilesMapToList(ctx, stack.Files)

					tflog.Info(ctx, "Found and processing primary stack", map[string]interface{}{
						"stack_id":   stackID,
						"file_count": len(stackFiles),
						"stack_name": stack.Name.ValueString(),
					})

					// 1. First add files from kits within the primary stack (lowest precedence)
					stackKitIDs := extractIDsFromDynamicList(ctx, stack.Kits)
					if len(stackKitIDs) > 0 {
						stackKitFiles := r.getKitFilesFromRegistry(ctx, stackKitIDs, reg)
						merger.AddFiles(ctx, stackKitFiles, files.SourceStack)
						tflog.Info(ctx, "Added files from stack kits", map[string]interface{}{
							"stack_id":       stackID,
							"kit_count":      len(stackKitIDs),
							"kit_file_count": len(stackKitFiles),
						})
					}

					// 2. Add the primary stack's own files (higher precedence than stack's kits)
					merger.AddFiles(ctx, stackFiles, files.SourceStack)
				} else {
					tflog.Warn(ctx, "Stack data is not StackResourceModel", map[string]interface{}{
						"stack_id":    stackID,
						"actual_type": fmt.Sprintf("%T", stackData),
					})
				}
			} else {
				tflog.Warn(ctx, "Referenced stack not found in registry", map[string]interface{}{
					"stack_id": stackID,
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
					// Convert stack files from map to list
					stackFiles := schemas.FilesMapToList(ctx, stack.Files)

					tflog.Info(ctx, "Processing stack files", map[string]interface{}{
						"stack_id":   stackID,
						"file_count": len(stackFiles),
						"stack_name": stack.Name.ValueString(),
					})

					// Add stack's files
					merger.AddFiles(ctx, stackFiles, files.SourceStack)
				} else {
					tflog.Warn(ctx, "Stack data is not StackResourceModel", map[string]interface{}{
						"stack_id":    stackID,
						"actual_type": fmt.Sprintf("%T", stackData),
					})
				}
			}
		}
	}

	// 3. Add files from project's additional kits (override stack kits)
	projectKitIDs := extractIDsFromDynamicList(ctx, data.Kits)
	if reg != nil && len(projectKitIDs) > 0 {
		projectKitFiles := r.getKitFilesFromRegistry(ctx, projectKitIDs, reg)
		merger.AddFiles(ctx, projectKitFiles, files.SourceProject)
		tflog.Info(ctx, "Added files from project kits", map[string]interface{}{
			"kit_count":      len(projectKitIDs),
			"kit_file_count": len(projectKitFiles),
		})
	}

	// 4. Finally, add project's own files (highest precedence)
	projectFiles := schemas.FilesMapToList(ctx, data.Files)
	if len(projectFiles) > 0 {
		merger.AddFiles(ctx, projectFiles, files.SourceProject)
	}

	// Get the final merged list
	mergedFiles := merger.GetMergedFiles()

	// Build log attributes safely
	logAttrs := map[string]interface{}{
		"total_count":   len(mergedFiles),
		"project_count": len(projectFiles),
	}

	// Only add stack if it's not null
	stackID := extractIDFromDynamic(ctx, data.Stack)
	if stackID != "" {
		logAttrs["stack_id"] = stackID
	}

	tflog.Info(ctx, "Merged files from all sources", logAttrs)

	return mergedFiles
}
