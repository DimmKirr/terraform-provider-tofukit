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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/files"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/openai"
	"github.com/tofukit/opentofu-provider-tofukit/internal/registry"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
	"github.com/tofukit/opentofu-provider-tofukit/internal/uri"
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
	Requirements       []schemas.RequirementModel `tfsdk:"requirements"`
	Kits               types.Dynamic              `tfsdk:"kits"`     // List of kit references or IDs
	Files              types.Map                  `tfsdk:"files"`    // Map of files keyed by path
	Features           types.Dynamic              `tfsdk:"features"` // Map of features keyed by feature name
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
	// Drift detection
	DriftDetected types.Bool `tfsdk:"drift_detected"`
	DriftedFiles  types.List `tfsdk:"drifted_files"`
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
		MarkdownDescription: `Project resource that orchestrates LLM-generated code by composing stacks, features, kits, and files into a complete application.

**Examples:** Flask NYC weather API, Go CLI tool, React dashboard, Django REST service`,
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
				MarkdownDescription: "List of kit references to include in the project (e.g., [tofukit_language.python, tofukit_library.click])",
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
				MarkdownDescription: "Model to use for this project in provider/model format. Examples: 'anthropic/claude-opus-4.5', 'anthropic/claude-sonnet-4.5', 'anthropic/claude-haiku-4.5', 'openai/gpt-5.2', 'openai/gpt-5.1-codex'. Overrides provider-level model setting. If not specified, uses provider's default model.",
				Optional:            true,
			},
			"planned_prompt_json": schema.StringAttribute{
				MarkdownDescription: "Internal: Complete Claude prompt JSON generated and stored during apply for reference.",
				Computed:            true,
				Sensitive:           true,
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
			"drift_detected": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "True if any files have been modified outside Terraform",
			},
			"drifted_files": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of file paths that have drifted from their expected state",
			},
			"files": schemas.GetFilesMapAttribute(),
			"features": schema.DynamicAttribute{
				MarkdownDescription: "Features to implement (capabilities bundled with files, kits, and verifications). Can be inline definitions or references to feature resources.",
				Optional:            true,
			},
			"requirements": schemas.GetRequirementsListAttribute(),
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
		tflog.Debug(ctx, "extractIDsFromDynamicList: value is null or unknown", nil)
		return []string{} // Return empty slice instead of nil
	}

	var ids []string
	underlying := dynValue.UnderlyingValue()
	tflog.Debug(ctx, "extractIDsFromDynamicList: underlying type", map[string]interface{}{
		"type": fmt.Sprintf("%T", underlying),
	})

	// Try to get as list
	if listVal, ok := underlying.(types.List); ok && !listVal.IsNull() {
		tflog.Debug(ctx, "extractIDsFromDynamicList: got list", map[string]interface{}{
			"element_count": len(listVal.Elements()),
		})

		// Iterate through list elements
		for i, elem := range listVal.Elements() {
			tflog.Debug(ctx, "extractIDsFromDynamicList: processing list element", map[string]interface{}{
				"index": i,
				"type":  fmt.Sprintf("%T", elem),
			})

			// Try as string first
			if strVal, ok := elem.(types.String); ok && !strVal.IsNull() {
				id := strVal.ValueString()
				tflog.Debug(ctx, "extractIDsFromDynamicList: extracted string ID", map[string]interface{}{
					"id": id,
				})
				ids = append(ids, id)
				continue
			}

			// Try as object with .id
			if objVal, ok := elem.(types.Object); ok && !objVal.IsNull() {
				attrs := objVal.Attributes()
				attrKeys := make([]string, 0, len(attrs))
				for k := range attrs {
					attrKeys = append(attrKeys, k)
				}
				tflog.Debug(ctx, "extractIDsFromDynamicList: object attributes", map[string]interface{}{
					"attr_count": len(attrs),
					"attr_keys":  attrKeys,
				})

				if idAttr, exists := attrs["id"]; exists {
					if idStr, ok := idAttr.(types.String); ok && !idStr.IsNull() {
						id := idStr.ValueString()
						tflog.Debug(ctx, "extractIDsFromDynamicList: extracted object ID", map[string]interface{}{
							"id": id,
						})
						ids = append(ids, id)
					}
				}
			}
		}
	} else if tupleVal, ok := underlying.(types.Tuple); ok && !tupleVal.IsNull() {
		// Handle Tuple type (Terraform sometimes stores resource lists as tuples)
		tflog.Debug(ctx, "extractIDsFromDynamicList: got tuple", map[string]interface{}{
			"element_count": len(tupleVal.Elements()),
		})

		// Iterate through tuple elements
		for i, elem := range tupleVal.Elements() {
			tflog.Debug(ctx, "extractIDsFromDynamicList: processing tuple element", map[string]interface{}{
				"index": i,
				"type":  fmt.Sprintf("%T", elem),
			})

			// Try as string first
			if strVal, ok := elem.(types.String); ok && !strVal.IsNull() {
				id := strVal.ValueString()
				tflog.Debug(ctx, "extractIDsFromDynamicList: extracted string ID from tuple", map[string]interface{}{
					"id": id,
				})
				ids = append(ids, id)
				continue
			}

			// Try as object with .id
			if objVal, ok := elem.(types.Object); ok && !objVal.IsNull() {
				attrs := objVal.Attributes()
				tflog.Debug(ctx, "extractIDsFromDynamicList: tuple object attributes", map[string]interface{}{
					"attr_count": len(attrs),
				})

				if idAttr, exists := attrs["id"]; exists {
					if idStr, ok := idAttr.(types.String); ok && !idStr.IsNull() {
						id := idStr.ValueString()
						tflog.Debug(ctx, "extractIDsFromDynamicList: extracted object ID from tuple", map[string]interface{}{
							"id": id,
						})
						ids = append(ids, id)
					}
				}
			}
		}
	} else {
		tflog.Warn(ctx, "extractIDsFromDynamicList: not a list/tuple or is null", map[string]interface{}{
			"actual_type": fmt.Sprintf("%T", underlying),
		})

		// NEW: Try to extract as single string ID
		if strVal, ok := underlying.(types.String); ok && !strVal.IsNull() {
			id := strVal.ValueString()
			tflog.Info(ctx, "extractIDsFromDynamicList: extracted single string ID", map[string]interface{}{
				"id": id,
			})
			return []string{id}
		}
	}

	tflog.Debug(ctx, "extractIDsFromDynamicList: returning IDs", map[string]interface{}{
		"count": len(ids),
		"ids":   ids,
	})
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

	// Skip if this is resource destruction (no plan for new state)
	if req.Plan.Raw.IsNull() {
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
	//
	// FIX BUG-003: Unchanged files handled by UseStateForUnknown() PlanModifiers
	// The PlanModifiers on content_hash, file_hash, file_modtime (in schemas/common.go) tell
	// Terraform to preserve state values when plan values are unknown. This prevents unchanged
	// files from showing as modified with "(known after apply)" in plan output.

	tflog.Debug(ctx, "ModifyPlan: File operations detected (map schema ensures accurate plan display)", map[string]interface{}{
		"config_count": len(configFilesList),
		"state_count":  len(stateFilesList),
		"added":        added,
		"removed":      removed,
		"modified":     modified,
		"unchanged":    unchanged,
	})

	// Update the plan
	diags = resp.Plan.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ProjectResourceFinal) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// DEBUG: Log at very start of Create()
	debugFile, _ := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if debugFile != nil {
		fmt.Fprintf(debugFile, "\n=== Create() ENTRY ===\n")
		defer func() {
			fmt.Fprintf(debugFile, "=== Create() EXIT ===\n")
			debugFile.Close()
		}()
	}

	var data ProjectModelFinal

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("project.%s", data.Name.ValueString()))

	tflog.Debug(ctx, "=== CREATE METHOD STARTED ===", map[string]interface{}{
		"project_id":   data.ID.ValueString(),
		"project_name": data.Name.ValueString(),
	})

	// Validate project has at least one content source
	hasFiles := !data.Files.IsNull() && !data.Files.IsUnknown() && len(data.Files.Elements()) > 0
	hasKits := !data.Kits.IsNull() && !data.Kits.IsUnknown()
	hasStack := !data.Stack.IsNull() && !data.Stack.IsUnknown()
	hasRequirements := len(data.Requirements) > 0
	hasFeatures := !data.Features.IsNull() && !data.Features.IsUnknown()

	if !hasFiles && !hasKits && !hasStack && !hasRequirements && !hasFeatures {
		resp.Diagnostics.AddError(
			"Empty Project Configuration",
			fmt.Sprintf(
				"Project '%s' must specify at least one of the following:\n"+
					"  - file {} blocks (explicit files to create)\n"+
					"  - kits (language/framework setup)\n"+
					"  - stack (reference to a stack resource)\n"+
					"  - features (capability bundles with files/kits/requirements)\n"+
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
		"has_features":     hasFeatures,
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
	if debugFile2, err := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil && debugFile2 != nil {
		fmt.Fprintf(debugFile2, "\n=== Create method ===\n")
		fmt.Fprintf(debugFile2, "Create: data.Kits.IsNull()=%v, IsUnknown()=%v\n", data.Kits.IsNull(), data.Kits.IsUnknown())
		debugFile2.Close()
	}

	// Scan for URIs and build resource registry
	stringFields := r.collectStringFields(ctx, data)
	uriScanner := uri.NewScanner()
	foundURIs := uriScanner.ExtractURIs(stringFields)

	var resourceRegistry map[string]interface{}
	if len(foundURIs) > 0 {
		tflog.Info(ctx, "Found resource URIs in project", map[string]interface{}{
			"project_name": data.Name.ValueString(),
			"uri_count":    len(foundURIs),
			"uris":         foundURIs,
		})

		// Get registry from provider data
		if provData, ok := r.ProviderData.(interface{ GetRegistry() *registry.Registry }); ok {
			reg := provData.GetRegistry()
			registryBuilder := uri.NewRegistryBuilder(reg)
			var err error
			resourceRegistry, err = registryBuilder.BuildRegistry(foundURIs)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Resource Registry Build Failed",
					fmt.Sprintf("Failed to build resource registry for URIs: %s\n"+
						"The project will proceed without resource metadata. "+
						"Ensure all referenced resources exist.", err.Error()),
				)
			} else {
				tflog.Debug(ctx, "Built resource registry", map[string]interface{}{
					"registry_entry_count": len(resourceRegistry),
				})
			}
		}
	}

	// Build output data with enriched files
	outputData := r.buildOutputDataWithFiles(ctx, data, enrichedFiles)

	// Build project context for feature introspection
	// DEBUG: Log before buildProjectContext call
	if debugFile3, err3 := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err3 == nil && debugFile3 != nil {
		fmt.Fprintf(debugFile3, "Create(): About to call buildProjectContext\n")
		fmt.Fprintf(debugFile3, "Create(): Features.IsNull()=%v, IsUnknown()=%v\n", data.Features.IsNull(), data.Features.IsUnknown())
		debugFile3.Close()
	}

	projectContext := r.buildProjectContext(ctx, data)

	// DEBUG: Log after buildProjectContext returns
	if debugFile4, err4 := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err4 == nil && debugFile4 != nil {
		if projectContext != nil {
			fmt.Fprintf(debugFile4, "Create(): buildProjectContext returned, has features: %v\n", projectContext["features"] != nil)
			if featuresData, ok := projectContext["features"]; ok {
				fmt.Fprintf(debugFile4, "Create(): features type: %T, count: %d\n", featuresData, len(featuresData.([]map[string]interface{})))
			} else {
				fmt.Fprintf(debugFile4, "Create(): NO FEATURES in returned context!\n")
			}
		} else {
			fmt.Fprintf(debugFile4, "Create(): buildProjectContext returned nil\n")
		}
		debugFile4.Close()
	}

	// Add project context to output data
	if projectContext != nil && len(projectContext) > 0 {
		outputData["_project_context"] = projectContext
		tflog.Debug(ctx, "Added project context to output data", map[string]interface{}{
			"context_keys": func() []string {
				keys := make([]string, 0, len(projectContext))
				for k := range projectContext {
					keys = append(keys, k)
				}
				return keys
			}(),
		})
	}

	// Add resource registry to output data if present
	if resourceRegistry != nil && len(resourceRegistry) > 0 {
		outputData["_resource_registry"] = resourceRegistry
	}

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

		// Compute and store per-file hashes
		if err := r.computeAndStoreFileHashes(ctx, &data, mergedFiles); err != nil {
			tflog.Warn(ctx, "Failed to compute file hashes", map[string]interface{}{
				"error": err.Error(),
			})
			// Don't fail the resource, just skip storing hashes
		}

		tflog.Info(ctx, "Computed and stored file hashes", map[string]interface{}{
			"project_id": data.ID.ValueString(),
			"file_count": len(mergedFiles),
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

	// Initialize drift flags (no drift on fresh creation)
	data.DriftDetected = types.BoolValue(false)
	data.DriftedFiles = types.ListNull(types.StringType)

	tflog.Debug(ctx, "=== CREATE: INITIALIZED DRIFT FLAGS ===", map[string]interface{}{
		"project_id":       data.ID.ValueString(),
		"drift_detected":   data.DriftDetected.ValueBool(),
		"drift_is_null":    data.DriftDetected.IsNull(),
		"drift_is_unknown": data.DriftDetected.IsUnknown(),
	})

	tflog.Trace(ctx, fmt.Sprintf("created project resource: %s", data.ID.ValueString()))

	tflog.Debug(ctx, "=== CREATE: SAVING STATE ===", map[string]interface{}{
		"project_id":     data.ID.ValueString(),
		"drift_detected": data.DriftDetected.ValueBool(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "=== CREATE: ERROR SAVING STATE ===", map[string]interface{}{
			"errors": resp.Diagnostics.Errors(),
		})
	} else {
		tflog.Debug(ctx, "=== CREATE: STATE SAVED SUCCESSFULLY ===", map[string]interface{}{
			"project_id": data.ID.ValueString(),
		})
	}
}

func (r *ProjectResourceFinal) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProjectModelFinal
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "=== READ METHOD STARTED ===", map[string]interface{}{
		"project_id":           data.ID.ValueString(),
		"drift_detected_in":    data.DriftDetected.ValueBoolPointer(),
		"drift_is_null_in":     data.DriftDetected.IsNull(),
		"drift_is_unknown_in":  data.DriftDetected.IsUnknown(),
		"drifted_files_null":   data.DriftedFiles.IsNull(),
		"drifted_files_length": len(data.DriftedFiles.Elements()),
	})

	// Get provider configuration
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	dangerouslySkipPermissions := false
	claudeMaxTurns := 100
	if provData, ok := r.ProviderData.(interface {
		GetOutputPath() string
		GetClaudeHomeDirectory() string
		GetDangerouslySkipPermissions() bool
		GetClaudeMaxTurns() int
	}); ok {
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
		dangerouslySkipPermissions = provData.GetDangerouslySkipPermissions()
		claudeMaxTurns = provData.GetClaudeMaxTurns()
	}

	// Always update output files with current configuration (including files from stacks)
	// This ensures the Claude prompt JSON always has the complete specification
	mergedFiles := r.collectAndMergeFiles(ctx, data)

	// DRIFT DETECTION: Check each file for changes outside Terraform
	projectPath := data.ProjectPath.ValueString()
	driftedFiles := []string{}

	tflog.Debug(ctx, "=== READ: STARTING DRIFT DETECTION ===", map[string]interface{}{
		"project_path":     projectPath,
		"files_is_null":    data.Files.IsNull(),
		"files_elements":   len(data.Files.Elements()),
		"will_check_drift": projectPath != "" && !data.Files.IsNull(),
	})

	if projectPath != "" && !data.Files.IsNull() {
		tflog.Debug(ctx, "=== READ: BEFORE ElementsAs ===", map[string]interface{}{
			"files_elements": len(data.Files.Elements()),
			"files_type":     fmt.Sprintf("%T", data.Files),
		})

		filesMap := make(map[string]schemas.FileModel)
		diags := data.Files.ElementsAs(ctx, &filesMap, false)

		tflog.Debug(ctx, "=== READ: AFTER ElementsAs ===", map[string]interface{}{
			"filesMap_size": len(filesMap),
			"has_diags":     diags.HasError(),
			"diags_errors":  len(diags.Errors()),
		})

		if diags.HasError() {
			errorStrings := []string{}
			for _, diag := range diags.Errors() {
				errorStrings = append(errorStrings, diag.Summary()+": "+diag.Detail())
			}
			tflog.Error(ctx, "=== READ: ElementsAs FAILED ===", map[string]interface{}{
				"error_count":    len(diags.Errors()),
				"error_messages": errorStrings,
			})
		}

		// Log each file in the map
		for key := range filesMap {
			tflog.Debug(ctx, "=== READ: File in filesMap ===", map[string]interface{}{
				"path": key,
			})
		}

		iterationCount := 0
		for path, fileModel := range filesMap {
			iterationCount++
			tflog.Debug(ctx, "=== READ: ITERATING FILE ===", map[string]interface{}{
				"iteration":         iterationCount,
				"path":              path,
				"file_hash_null":    fileModel.FileHash.IsNull(),
				"file_hash_unknown": fileModel.FileHash.IsUnknown(),
				"file_hash_value":   fileModel.FileHash.ValueString(),
				"content_hash_null": fileModel.ContentHash.IsNull(),
				"modtime_null":      fileModel.FileModTime.IsNull(),
			})

			// Skip if no hash stored yet (first run after upgrade)
			if fileModel.FileHash.IsNull() || fileModel.FileHash.IsUnknown() {
				tflog.Warn(ctx, "=== READ: SKIPPING FILE (no hash) ===", map[string]interface{}{
					"path":              path,
					"file_hash_null":    fileModel.FileHash.IsNull(),
					"file_hash_unknown": fileModel.FileHash.IsUnknown(),
				})
				continue
			}

			fullPath := filepath.Join(projectPath, path)

			// Check if file exists
			fileInfo, err := os.Stat(fullPath)
			if os.IsNotExist(err) {
				// File deleted outside Terraform
				tflog.Warn(ctx, "File deleted outside Terraform", map[string]interface{}{
					"project_id": data.ID.ValueString(),
					"path":       path,
				})
				driftedFiles = append(driftedFiles, path)
				continue
			}
			if err != nil {
				tflog.Warn(ctx, "Failed to stat file for drift detection", map[string]interface{}{
					"path":  path,
					"error": err.Error(),
				})
				continue
			}

			// Optimization: Check modification time first
			currentModTime := fileInfo.ModTime().Format(time.RFC3339)
			if !fileModel.FileModTime.IsNull() && currentModTime == fileModel.FileModTime.ValueString() {
				// File unchanged since last check, skip hashing
				tflog.Trace(ctx, "File modification time unchanged, skipping hash", map[string]interface{}{
					"path": path,
				})
				continue
			}

			// Read actual file content
			actualContent, err := os.ReadFile(fullPath)
			if err != nil {
				tflog.Warn(ctx, "Failed to read file for drift detection", map[string]interface{}{
					"path":  path,
					"error": err.Error(),
				})
				continue
			}

			// Calculate current hash (temporary, in-memory)
			currentHash := files.ComputeFileHash(actualContent)

			// Compare with stored hash
			if currentHash != fileModel.FileHash.ValueString() {
				tflog.Warn(ctx, "File content changed outside Terraform", map[string]interface{}{
					"project_id":    data.ID.ValueString(),
					"path":          path,
					"expected_hash": fileModel.FileHash.ValueString()[:8] + "...",
					"current_hash":  currentHash[:8] + "...",
				})
				tflog.Debug(ctx, "=== READ: DRIFT DETECTED FOR FILE ===", map[string]interface{}{
					"path":          path,
					"expected_hash": fileModel.FileHash.ValueString()[:8],
					"current_hash":  currentHash[:8],
				})
				driftedFiles = append(driftedFiles, path)
			}
		}

		tflog.Debug(ctx, "=== READ: DRIFT CHECK COMPLETED ===", map[string]interface{}{
			"files_checked": len(filesMap),
			"drifted_count": len(driftedFiles),
			"drifted_files": driftedFiles,
		})
	}

	// Set computed drift fields and restore files if drift detected
	if len(driftedFiles) > 0 {
		data.DriftDetected = types.BoolValue(true)
		driftList, diags := types.ListValueFrom(ctx, types.StringType, driftedFiles)
		if diags.HasError() {
			tflog.Warn(ctx, "Failed to create drifted files list", map[string]interface{}{
				"errors": diags.Errors(),
			})
		} else {
			data.DriftedFiles = driftList
		}

		tflog.Info(ctx, "Drift detected in project files - initiating auto-restoration", map[string]interface{}{
			"project_id":    data.ID.ValueString(),
			"drifted_count": len(driftedFiles),
			"drifted_files": driftedFiles,
		})

		// AUTO-RESTORE: Execute Claude to restore drifted files
		// Get merged files and enrich with drift instructions
		mergedFiles := r.collectAndMergeFiles(ctx, data)
		enrichedFiles := r.addDriftInstructions(ctx, mergedFiles, driftedFiles)

		// Build output data for Claude
		outputData := r.buildOutputData(ctx, data)

		// Execute Claude to restore files
		tflog.Info(ctx, "Executing Claude to restore drifted files", map[string]interface{}{
			"project_id": data.ID.ValueString(),
		})

		if err := r.executeClaudeCode(ctx, &data, outputData, enrichedFiles, projectPath, claudeHomeDir, false); err != nil {
			tflog.Error(ctx, "Failed to restore drifted files", map[string]interface{}{
				"project_id": data.ID.ValueString(),
				"error":      err.Error(),
			})
			// Don't fail Read(), just log the error and leave drift flag set
		} else {
			// Successfully restored - recompute hashes and clear drift flags
			outputHash := r.computeOutputHash(ctx, projectPath)
			data.OutputHash = types.StringValue(outputHash)

			if err := r.computeAndStoreFileHashes(ctx, &data, mergedFiles); err != nil {
				tflog.Warn(ctx, "Failed to recompute file hashes after restoration", map[string]interface{}{
					"error": err.Error(),
				})
			}

			// Clear drift flags
			data.DriftDetected = types.BoolValue(false)
			data.DriftedFiles = types.ListNull(types.StringType)

			tflog.Info(ctx, "Successfully restored drifted files and cleared drift flags", map[string]interface{}{
				"project_id": data.ID.ValueString(),
			})
		}
	} else {
		data.DriftDetected = types.BoolValue(false)
		data.DriftedFiles = types.ListNull(types.StringType)
		tflog.Debug(ctx, "=== READ: SET DRIFT_DETECTED=FALSE ===", map[string]interface{}{
			"drifted_count": 0,
		})
	}

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
			executor := claude.NewExecutor(claudeHomeDir, dangerouslySkipPermissions, claudeMaxTurns)
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

	// Initialize drift detection fields if not set
	tflog.Debug(ctx, "=== READ: BEFORE DRIFT INIT CHECK ===", map[string]interface{}{
		"drift_detected":   data.DriftDetected.ValueBoolPointer(),
		"drift_is_null":    data.DriftDetected.IsNull(),
		"drift_is_unknown": data.DriftDetected.IsUnknown(),
	})

	if data.DriftDetected.IsNull() || data.DriftDetected.IsUnknown() {
		tflog.Debug(ctx, "=== READ: REINITIALIZING DRIFT_DETECTED TO FALSE ===", map[string]interface{}{
			"was_null":    data.DriftDetected.IsNull(),
			"was_unknown": data.DriftDetected.IsUnknown(),
		})
		data.DriftDetected = types.BoolValue(false)
	}
	if data.DriftedFiles.IsNull() || data.DriftedFiles.IsUnknown() {
		data.DriftedFiles = types.ListNull(types.StringType)
	}

	tflog.Debug(ctx, "=== READ: SAVING STATE ===", map[string]interface{}{
		"project_id":       data.ID.ValueString(),
		"drift_detected":   data.DriftDetected.ValueBool(),
		"drift_is_null":    data.DriftDetected.IsNull(),
		"drift_is_unknown": data.DriftDetected.IsUnknown(),
		"drifted_files":    len(data.DriftedFiles.Elements()),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "=== READ: ERROR SAVING STATE ===", map[string]interface{}{
			"errors": resp.Diagnostics.Errors(),
		})
	} else {
		tflog.Debug(ctx, "=== READ: STATE SAVED SUCCESSFULLY ===", map[string]interface{}{
			"project_id": data.ID.ValueString(),
		})
	}
}

func (r *ProjectResourceFinal) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// DEBUG: Log at very start of Update()
	debugFile, _ := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if debugFile != nil {
		fmt.Fprintf(debugFile, "\n=== Update() ENTRY ===\n")
		defer func() {
			fmt.Fprintf(debugFile, "=== Update() EXIT ===\n")
			debugFile.Close()
		}()
	}

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

	// Check if update triggered by drift vs config change
	isDriftTriggered := state.DriftDetected.ValueBool()

	tflog.Debug(ctx, "=== UPDATE: DRIFT CHECK ===", map[string]interface{}{
		"project_id":           state.ID.ValueString(),
		"isDriftTriggered":     isDriftTriggered,
		"drift_detected_value": state.DriftDetected.ValueBool(),
		"drift_detected_null":  state.DriftDetected.IsNull(),
	})

	if isDriftTriggered {
		tflog.Info(ctx, "Update triggered by drift detection", map[string]interface{}{
			"project_id": state.ID.ValueString(),
		})
	}

	// Get provider configuration
	outputPath := ".tofukit"
	claudeHomeDir := "~/.claude"
	debug := false
	_ = false // dangerouslySkipPermissions - reserved for future use
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
	hasFeatures := !data.Features.IsNull() && !data.Features.IsUnknown()

	if !hasFiles && !hasKits && !hasStack && !hasRequirements && !hasFeatures {
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
		"has_features":     hasFeatures,
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

	// Add drift context if update triggered by drift
	tflog.Debug(ctx, "=== UPDATE: BEFORE DRIFT INSTRUCTIONS ===", map[string]interface{}{
		"isDriftTriggered":      isDriftTriggered,
		"drifted_files_null":    state.DriftedFiles.IsNull(),
		"drifted_files_unknown": state.DriftedFiles.IsUnknown(),
		"enriched_files_count":  len(enrichedFiles),
	})

	if isDriftTriggered {
		var driftedPaths []string
		if !state.DriftedFiles.IsNull() && !state.DriftedFiles.IsUnknown() {
			state.DriftedFiles.ElementsAs(ctx, &driftedPaths, false)

			tflog.Debug(ctx, "=== UPDATE: ADDING DRIFT INSTRUCTIONS ===", map[string]interface{}{
				"drifted_files": driftedPaths,
				"drifted_count": len(driftedPaths),
			})

			enrichedFiles = r.addDriftInstructions(ctx, enrichedFiles, driftedPaths)

			tflog.Debug(ctx, "=== UPDATE: AFTER addDriftInstructions ===", map[string]interface{}{
				"enriched_files_count": len(enrichedFiles),
			})
		}
	}

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

	tflog.Debug(ctx, "=== UPDATE: EXECUTION TRIGGER CHECK ===", map[string]interface{}{
		"promptChanged":    promptChanged,
		"fileSpecChanged":  fileSpecChanged,
		"outputDrifted":    outputDrifted,
		"isDriftTriggered": isDriftTriggered,
		"will_execute":     promptChanged || fileSpecChanged || outputDrifted || isDriftTriggered,
	})

	if promptChanged || fileSpecChanged || outputDrifted || isDriftTriggered {
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
		if isDriftTriggered {
			tflog.Info(ctx, "Drift detected - triggering restoration", map[string]interface{}{
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

		// Scan for URIs and build resource registry
		stringFields := r.collectStringFields(ctx, data)
		uriScanner := uri.NewScanner()
		foundURIs := uriScanner.ExtractURIs(stringFields)

		var resourceRegistry map[string]interface{}
		if len(foundURIs) > 0 {
			tflog.Info(ctx, "Found resource URIs in project", map[string]interface{}{
				"project_id": data.ID.ValueString(),
				"uri_count":  len(foundURIs),
				"uris":       foundURIs,
			})

			// Get registry from provider data
			if provData, ok := r.ProviderData.(interface{ GetRegistry() *registry.Registry }); ok {
				reg := provData.GetRegistry()
				registryBuilder := uri.NewRegistryBuilder(reg)
				var err error
				resourceRegistry, err = registryBuilder.BuildRegistry(foundURIs)
				if err != nil {
					resp.Diagnostics.AddWarning(
						"Resource Registry Build Failed",
						fmt.Sprintf("Failed to build resource registry for URIs: %s\n"+
							"The project will proceed without resource metadata. "+
							"Ensure all referenced resources exist.", err.Error()),
					)
				} else {
					tflog.Debug(ctx, "Built resource registry", map[string]interface{}{
						"registry_entry_count": len(resourceRegistry),
					})
				}
			}
		}

		// Use enriched files (with action-based instructions) for output
		// Build output data with enriched files
		outputData = r.buildOutputDataWithFiles(ctx, data, enrichedFiles)

		// Build project context for feature introspection
		// DEBUG: Log before buildProjectContext call
		if debugFile, _ := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); debugFile != nil {
			fmt.Fprintf(debugFile, "Update(): About to call buildProjectContext\n")
			fmt.Fprintf(debugFile, "Update(): Features.IsNull()=%v, IsUnknown()=%v\n", data.Features.IsNull(), data.Features.IsUnknown())
			debugFile.Close()
		}

		projectContext := r.buildProjectContext(ctx, data)

		// DEBUG: Log after buildProjectContext returns
		if debugFile, _ := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); debugFile != nil {
			if projectContext != nil {
				fmt.Fprintf(debugFile, "Update(): buildProjectContext returned, has features: %v\n", projectContext["features"] != nil)
				if featuresData, ok := projectContext["features"]; ok {
					fmt.Fprintf(debugFile, "Update(): features type: %T\n", featuresData)
				}
			} else {
				fmt.Fprintf(debugFile, "Update(): buildProjectContext returned nil\n")
			}
			debugFile.Close()
		}

		// Add project context to output data
		if projectContext != nil && len(projectContext) > 0 {
			outputData["_project_context"] = projectContext
			tflog.Debug(ctx, "Added project context to output data in Update", map[string]interface{}{
				"context_keys": func() []string {
					keys := make([]string, 0, len(projectContext))
					for k := range projectContext {
						keys = append(keys, k)
					}
					return keys
				}(),
			})
		}

		// Add resource registry to output data if present
		if resourceRegistry != nil && len(resourceRegistry) > 0 {
			outputData["_resource_registry"] = resourceRegistry
		}

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

			// Recompute all file hashes after successful execution
			if err := r.computeAndStoreFileHashes(ctx, &data, mergedFiles); err != nil {
				tflog.Warn(ctx, "Failed to recompute file hashes", map[string]interface{}{
					"error": err.Error(),
				})
			}

			// Clear drift flags
			data.DriftDetected = types.BoolValue(false)
			data.DriftedFiles = types.ListNull(types.StringType)

			tflog.Info(ctx, "Cleared drift flags after successful restoration", map[string]interface{}{
				"project_id": data.ID.ValueString(),
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
	dangerouslySkipPermissions := false
	claudeMaxTurns := 100
	if provData, ok := r.ProviderData.(interface {
		GetOutputPath() string
		GetClaudeHomeDirectory() string
		GetDangerouslySkipPermissions() bool
		GetClaudeMaxTurns() int
	}); ok {
		outputPath = provData.GetOutputPath()
		claudeHomeDir = provData.GetClaudeHomeDirectory()
		dangerouslySkipPermissions = provData.GetDangerouslySkipPermissions()
		claudeMaxTurns = provData.GetClaudeMaxTurns()
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
			executor := claude.NewExecutor(claudeHomeDir, dangerouslySkipPermissions, claudeMaxTurns)
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
		if len(req.Verifications) > 0 {
			verifications := []map[string]string{}
			for _, v := range req.Verifications {
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

	// NEW: Add feature requirements
	featureRequirements := r.convertFeaturesToRequirements(ctx, data)
	for _, req := range featureRequirements {
		reqData := map[string]interface{}{
			"name": req.Name.ValueString(),
		}

		// Add instructions
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
		if len(req.Verifications) > 0 {
			verifications := []map[string]string{}
			for _, v := range req.Verifications {
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

	tflog.Info(ctx, "Added requirements to output", map[string]interface{}{
		"project_requirements": len(data.Requirements),
		"feature_requirements": len(featureRequirements),
		"total_requirements":   len(data.Requirements) + len(featureRequirements),
	})

	// NEW: Collect stack kits, requirements, and feature requirements
	var reg *registry.Registry
	if provData, ok := r.ProviderData.(interface {
		GetRegistry() *registry.Registry
	}); ok {
		reg = provData.GetRegistry()
	}

	var stackKits map[string]interface{}
	var stackKitRequirements []schemas.RequirementModel
	var stackFeatureRequirements []schemas.RequirementModel

	if reg != nil {
		// Collect stack kits and their requirements
		stackKits, stackKitRequirements = r.collectStackKitsAndRequirements(ctx, data, reg)

		// Convert and add stack kit requirements to outputData
		for _, req := range stackKitRequirements {
			reqData := map[string]interface{}{
				"name": req.Name.ValueString(),
			}

			// Add instructions
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
			if len(req.Verifications) > 0 {
				verifications := []map[string]string{}
				for _, v := range req.Verifications {
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

		// Collect stack feature requirements
		stackFeatureRequirements = r.collectStackFeaturesRequirements(ctx, data, reg)

		// Convert and add stack feature requirements to outputData
		for _, req := range stackFeatureRequirements {
			reqData := map[string]interface{}{
				"name": req.Name.ValueString(),
			}

			// Add instructions
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
			if len(req.Verifications) > 0 {
				verifications := []map[string]string{}
				for _, v := range req.Verifications {
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

		// NEW: Collect requirements from kits referenced by features
		// This handles kits like gotask that are referenced by features (e.g., taskfile feature)
		// but not directly in the stack's kits list
		var featureKitIDs []string
		seen := make(map[string]bool)

		// First, mark all kits from stackKits map as already processed to avoid duplicates
		for kitID := range stackKits {
			seen[kitID] = true
		}

		// 1. Collect kit IDs from project features (excluding already-processed stack kits)
		projectFeatureKitIDs := r.collectFeatureKitIDs(ctx, data)
		for _, kitID := range projectFeatureKitIDs {
			if !seen[kitID] {
				featureKitIDs = append(featureKitIDs, kitID)
				seen[kitID] = true
			}
		}

		// 2. Collect kit IDs from stack features
		if !data.Stack.IsNull() && !data.Stack.IsUnknown() {
			stackID := extractIDFromDynamic(ctx, data.Stack)
			if stackData, exists := reg.GetStack(stackID); exists {
				if stack, ok := stackData.(StackResourceModel); ok {
					// Get feature IDs from stack
					stackFeatureIDs := extractIDsFromDynamicList(ctx, stack.Features)

					// For each stack feature, collect its kit IDs
					for _, featureID := range stackFeatureIDs {
						if featureData, exists := reg.GetFeature(featureID); exists {
							if feature, ok := featureData.(FeatureResourceModel); ok {
								if !feature.Kits.IsNull() && !feature.Kits.IsUnknown() {
									kitIDs := extractIDsFromDynamicList(ctx, feature.Kits)
									for _, kitID := range kitIDs {
										if !seen[kitID] {
											featureKitIDs = append(featureKitIDs, kitID)
											seen[kitID] = true
										}
									}
								}
							}
						}
					}
				}
			}
		}

		tflog.Info(ctx, "Collected kit IDs from features", map[string]interface{}{
			"total_feature_kit_count": len(featureKitIDs),
			"kit_ids":                 featureKitIDs,
		})

		// 3. Get requirements from feature-referenced kits
		featureKitRequirements := r.getKitRequirementsFromRegistry(ctx, featureKitIDs, reg)

		// 4. Convert and add feature kit requirements to outputData
		for _, req := range featureKitRequirements {
			reqData := map[string]interface{}{
				"name": req.Name.ValueString(),
			}

			// Add instructions
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
			if len(req.Verifications) > 0 {
				verifications := []map[string]string{}
				for _, v := range req.Verifications {
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

		tflog.Info(ctx, "Added stack requirements to output", map[string]interface{}{
			"stack_kit_requirements":     len(stackKitRequirements),
			"stack_feature_requirements": len(stackFeatureRequirements),
			"feature_kit_requirements":   len(featureKitRequirements),
			"total_stack_requirements":   len(stackKitRequirements) + len(stackFeatureRequirements) + len(featureKitRequirements),
		})
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

	// Collect feature kit IDs and fetch their data from registry
	featureKitIDs := r.collectFeatureKitIDs(ctx, data)
	if debugFile != nil {
		fmt.Fprintf(debugFile, "DEBUG feature kits: collected %d kit IDs: %v\n", len(featureKitIDs), featureKitIDs)
		fmt.Fprintf(debugFile, "DEBUG feature kits: registry exists=%v\n", reg != nil)
	}

	// Fetch feature kits from registry and add to kits map
	if reg != nil && len(featureKitIDs) > 0 {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "DEBUG feature kits: entering fetch loop for %d kits\n", len(featureKitIDs))
		}

		for _, kitID := range featureKitIDs {
			if compData, exists := reg.GetComponent(kitID); exists {
				if debugFile != nil {
					fmt.Fprintf(debugFile, "DEBUG feature kits: FOUND kit %s in registry (type=%T)\n", kitID, compData)
				}

				// Convert ComponentResourceModel to kit data format
				if component, ok := compData.(ComponentResourceModel); ok {
					kitData := map[string]interface{}{
						"id":          component.ID.ValueString(),
						"name":        component.Name.ValueString(),
						"description": component.Description.ValueString(),
						"version":     component.Version.ValueString(),
					}

					// Add requirements if present
					if len(component.Requirements) > 0 {
						requirementsData := []map[string]interface{}{}
						for _, req := range component.Requirements {
							reqData := map[string]interface{}{
								"name": req.Name.ValueString(),
							}

							// Add instructions
							if len(req.Instructions) > 0 {
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
							}

							// Add verifications if present
							if len(req.Verifications) > 0 {
								verifications := []map[string]string{}
								for _, v := range req.Verifications {
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

							requirementsData = append(requirementsData, reqData)
						}
						kitData["requirements"] = requirementsData
					}

					kits[kitID] = kitData
					tflog.Info(ctx, "Added feature kit from registry", map[string]interface{}{
						"kit_id": kitID,
					})

					if debugFile != nil {
						fmt.Fprintf(debugFile, "DEBUG feature kits: Added kit %s to kits map\n", kitID)
					}
				} else {
					tflog.Warn(ctx, "Feature kit is not ComponentResourceModel", map[string]interface{}{
						"kit_id": kitID,
						"type":   fmt.Sprintf("%T", compData),
					})
					if debugFile != nil {
						fmt.Fprintf(debugFile, "DEBUG feature kits: Component %s has wrong type: %T\n", kitID, compData)
					}
				}
			} else {
				tflog.Warn(ctx, "Feature kit not found in registry", map[string]interface{}{
					"kit_id": kitID,
				})
				if debugFile != nil {
					fmt.Fprintf(debugFile, "DEBUG feature kits: NOT FOUND kit %s in registry\n", kitID)
				}
			}
		}
	} else {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "DEBUG feature kits: NOT fetching - reg=%v, kit_count=%d\n", reg != nil, len(featureKitIDs))
		}
	}

	// Merge stack kits into the kits map
	if stackKits != nil {
		for kitID, kitData := range stackKits {
			kits[kitID] = kitData
		}
		tflog.Info(ctx, "Merged stack kits into kits map", map[string]interface{}{
			"stack_kit_count": len(stackKits),
			"total_kit_count": len(kits),
		})
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
	// Check if debug mode and dry_run are enabled
	debug := false
	dryRun := false
	if provData, ok := r.ProviderData.(interface {
		GetDebug() bool
		GetDryRun() bool
	}); ok {
		debug = provData.GetDebug()
		dryRun = provData.GetDryRun()
	}

	// Get system prompt from resource data
	systemPrompt := ""
	if !data.SystemPrompt.IsNull() && !data.SystemPrompt.IsUnknown() {
		systemPrompt = data.SystemPrompt.ValueString()
	}

	// Resolve model: resource override > provider default
	var resolvedModel string
	resourceModel := data.Model.ValueString()

	if resourceModel != "" {
		// Resource model takes precedence
		resolvedModel = resourceModel
	} else if provData, ok := r.ProviderData.(interface{ GetModel() string }); ok {
		// Fall back to provider default model
		resolvedModel = provData.GetModel()
	} else {
		// Ultimate fallback
		resolvedModel = "anthropic/claude-sonnet-4.5"
	}

	// Get executor for resolved model
	var executor llm.LLMExecutor
	if provData, ok := r.ProviderData.(interface {
		GetExecutorForModel(string) (llm.LLMExecutor, error)
	}); ok {
		var err error
		executor, err = provData.GetExecutorForModel(resolvedModel)
		if err != nil {
			return fmt.Errorf("failed to get executor for model '%s': %w", resolvedModel, err)
		}
	} else {
		return fmt.Errorf("provider data does not support GetExecutorForModel()")
	}

	// Configure executor (these methods are part of llm.LLMExecutor interface)
	executor.SetDebug(debug)
	executor.SetOutputPath(outputPath)
	executor.SetSystemPrompt(systemPrompt)

	// Get max retries from provider config (default to 3)
	maxRetries := 3
	if provData, ok := r.ProviderData.(interface{ GetMaxRetries() int }); ok {
		if retries := provData.GetMaxRetries(); retries > 0 {
			maxRetries = retries
		}
	}

	// Check if we have a planned prompt from the plan phase (stored in state)
	var promptJSON string
	var status *llm.ExecutionStatus
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

	// Execute with the prompt (or skip if dry_run is enabled)
	if dryRun {
		// Dry run mode: Skip LLM execution but still write debug files
		tflog.Info(ctx, "Dry run mode enabled - skipping LLM execution")

		// Mock successful execution status
		status = &llm.ExecutionStatus{
			State:       "completed",
			StartedAt:   time.Now().Format(time.RFC3339),
			CompletedAt: time.Now().Format(time.RFC3339),
			ProjectPath: outputPath,
			Error:       "",
		}

		// Create empty verification report (all passed)
		report = &files.VerificationReport{
			AllPassed: true,
		}

		// Write debug files if debug is enabled
		if debug {
			debugDir := filepath.Join(outputPath, ".debug")
			if err := os.MkdirAll(debugDir, 0755); err != nil {
				tflog.Warn(ctx, "Failed to create debug directory", map[string]interface{}{"error": err.Error()})
			} else {
				// Write prompt JSON
				timestamp := time.Now().Format("20060102-150405")
				promptPath := filepath.Join(debugDir, fmt.Sprintf("claude-prompt-attempt1-%s.json", timestamp))
				if err := os.WriteFile(promptPath, []byte(promptJSON), 0644); err != nil {
					tflog.Warn(ctx, "Failed to write prompt JSON", map[string]interface{}{"error": err.Error()})
				} else {
					tflog.Info(ctx, "Wrote prompt JSON for dry run", map[string]interface{}{"path": promptPath})
				}
			}
		}
	} else {
		// Collect ALL verifications for enforcement: file verifications + kit verifications
		// This ensures both file-level and kit-level verifications are checked after Claude execution
		allVerifications := append([]schemas.FileModelWithPath{}, mergedFiles...)

		// Extract and collect kit verifications from the specification
		// Kit verifications ensure declared dependencies (languages, frameworks, tools) are actually available
		if kitsData, ok := outputData["kits"].(map[string]interface{}); ok {
			kitVerifications := r.CollectKitVerifications(ctx, kitsData)
			allVerifications = append(allVerifications, kitVerifications...)

			tflog.Debug(ctx, "Collected verifications for enforcement", map[string]interface{}{
				"file_verifications":  len(mergedFiles),
				"kit_verifications":   len(kitVerifications),
				"total_verifications": len(allVerifications),
			})
		}

		// Execute with ALL verifications enforced
		// Failed verifications will trigger retries (up to maxRetries attempts)
		// If all retries fail, an error is returned causing Terraform apply to fail

		// Check if executor supports ExecuteWithPromptJSON
		// Both Claude and OpenAI executors implement this method
		type claudeAdapterInterface interface {
			GetClaudeExecutor() *claude.Executor
		}
		type openaiAdapterInterface interface {
			GetOpenAIExecutor() *openai.Executor
		}

		// Try Claude adapter first
		if adapter, ok := executor.(claudeAdapterInterface); ok {
			claudeExecutor := adapter.GetClaudeExecutor()
			var claudeStatus *claude.ExecutionStatus
			claudeStatus, report, err = claudeExecutor.ExecuteWithPromptJSON(ctx, promptJSON, outputPath, allVerifications, maxRetries)
			// Convert claude.ExecutionStatus to llm.ExecutionStatus
			if claudeStatus != nil {
				status = &llm.ExecutionStatus{
					State:       claudeStatus.State,
					StartedAt:   claudeStatus.StartedAt,
					CompletedAt: claudeStatus.CompletedAt,
					ProjectPath: claudeStatus.ProjectPath,
					Error:       claudeStatus.Error,
					Output:      claudeStatus.Output,
					Metadata:    claudeStatus.Metadata,
				}
			}
		} else if adapter, ok := executor.(openaiAdapterInterface); ok {
			// OpenAI adapter
			openaiExecutor := adapter.GetOpenAIExecutor()
			status, report, err = openaiExecutor.ExecuteWithPromptJSON(ctx, promptJSON, outputPath, allVerifications, maxRetries)
		} else {
			return fmt.Errorf("executor does not support ExecuteWithPromptJSON (only Claude and OpenAI executors are supported)")
		}
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
			GetDangerouslySkipPermissions() bool
			GetClaudeMaxTurns() int
		}); ok {
			// Validate Claude Code availability
			claudeHomeDir := provData.GetClaudeHomeDirectory()
			dangerouslySkipPermissions := provData.GetDangerouslySkipPermissions()
			claudeMaxTurns := provData.GetClaudeMaxTurns()
			executor := claude.NewExecutor(claudeHomeDir, dangerouslySkipPermissions, claudeMaxTurns)
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
	claudeClient := claude.NewClient("", false, 100) // temp client just for building prompt
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
		for _, ver := range req.Verifications {
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

// getKitRequirementsFromRegistry extracts requirements from kits stored in the registry
func (r *ProjectResourceFinal) getKitRequirementsFromRegistry(ctx context.Context, kitIDs []string, reg *registry.Registry) []schemas.RequirementModel {
	var allRequirements []schemas.RequirementModel

	if reg == nil || len(kitIDs) == 0 {
		return allRequirements
	}

	for _, kitID := range kitIDs {
		if kitData, exists := reg.GetComponent(kitID); exists {
			if kit, ok := kitData.(ComponentResourceModel); ok {
				tflog.Info(ctx, "Extracting requirements from kit", map[string]interface{}{
					"kit_id":            kitID,
					"kit_name":          kit.Name.ValueString(),
					"requirement_count": len(kit.Requirements),
				})
				allRequirements = append(allRequirements, kit.Requirements...)
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

	tflog.Info(ctx, "Collected requirements from kits", map[string]interface{}{
		"kit_count":          len(kitIDs),
		"total_requirements": len(allRequirements),
	})

	return allRequirements
}

// convertListToStringSlice converts a types.List to []types.String
func convertListToStringSlice(ctx context.Context, list types.List) []types.String {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	elements := list.Elements()
	result := make([]types.String, 0, len(elements))
	for _, elem := range elements {
		if strVal, ok := elem.(types.String); ok {
			result = append(result, strVal)
		}
	}
	return result
}

// convertVerificationsList converts a types.List of verification objects to []schemas.VerificationModel
func convertVerificationsList(ctx context.Context, list types.List) []schemas.VerificationModel {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	elements := list.Elements()
	result := make([]schemas.VerificationModel, 0, len(elements))
	for _, elem := range elements {
		if verifObj, ok := elem.(types.Object); ok {
			verifAttrs := verifObj.Attributes()
			verif := schemas.VerificationModel{}
			if cmd, exists := verifAttrs["command"]; exists {
				if cmdStr, ok := cmd.(types.String); ok {
					verif.Command = cmdStr
				}
			}
			if exp, exists := verifAttrs["expect"]; exists {
				if expStr, ok := exp.(types.String); ok {
					verif.Expect = expStr
				}
			}
			result = append(result, verif)
		}
	}
	return result
}

// buildProjectContext creates metadata about the project for introspection by features
// This enables features (like diagram generation) to discover project components without circular dependencies
func (r *ProjectResourceFinal) buildProjectContext(
	ctx context.Context,
	data ProjectModelFinal,
) map[string]interface{} {
	projectContext := map[string]interface{}{
		"project_info": map[string]interface{}{
			"name":        data.Name.ValueString(),
			"description": data.Description.ValueString(),
			"version":     data.Version.ValueString(),
		},
	}

	// Collect features metadata
	// DEBUG logging to file WITH WARNING IF NULL/UNKNOWN
	debugFile, _ := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if debugFile != nil {
		defer debugFile.Close()
		fmt.Fprintf(debugFile, "\n=== buildProjectContext START ===\n")
		fmt.Fprintf(debugFile, "buildProjectContext: Features.IsNull()=%v, IsUnknown()=%v\n", data.Features.IsNull(), data.Features.IsUnknown())

		if data.Features.IsNull() {
			fmt.Fprintf(debugFile, "⚠️  buildProjectContext: data.Features IS NULL - will NOT add features to context!\n")
		}
		if data.Features.IsUnknown() {
			fmt.Fprintf(debugFile, "⚠️  buildProjectContext: data.Features IS UNKNOWN - will NOT add features to context!\n")
		}
	}

	if !data.Features.IsNull() && !data.Features.IsUnknown() {
		underlyingVal := data.Features.UnderlyingValue()

		if debugFile != nil {
			fmt.Fprintf(debugFile, "buildProjectContext: Features type: %T\n", underlyingVal)
		}

		tflog.Debug(ctx, "[DEBUG-BUG-006] buildProjectContext: features underlying type", map[string]interface{}{
			"type": fmt.Sprintf("%T", underlyingVal),
		})
		if featuresMap, ok := underlyingVal.(types.Map); ok {
			tflog.Debug(ctx, "[DEBUG-BUG-006] buildProjectContext: successfully cast to types.Map", map[string]interface{}{
				"num_features": len(featuresMap.Elements()),
			})
			featuresMetadata := []map[string]interface{}{}

			for featureName, featureValue := range featuresMap.Elements() {
				feature, err := r.parseFeature(ctx, featureValue)
				if err != nil {
					tflog.Warn(ctx, "Failed to parse feature for context", map[string]interface{}{
						"feature_name": featureName,
						"error":        err.Error(),
					})
					continue
				}

				if feature == nil {
					continue
				}

				featureMeta := map[string]interface{}{
					"name": featureName,
				}

				// Add prompt from requirements if present
				if len(feature.Requirements) > 0 {
					// Get the prompt from the first requirement's first instruction
					if len(feature.Requirements[0].Instructions) > 0 {
						if !feature.Requirements[0].Instructions[0].Prompt.IsNull() {
							featureMeta["prompt"] = feature.Requirements[0].Instructions[0].Prompt.ValueString()
						}
					}
				}

				// Add description if available (this would come from FeatureResourceModel)
				// For inline features, we don't have a description field in FeatureModel

				// Add feature files if present
				if !feature.Files.IsNull() && !feature.Files.IsUnknown() {
					files := []string{}
					for path := range feature.Files.Elements() {
						files = append(files, path)
					}
					if len(files) > 0 {
						featureMeta["files"] = files
					}
				}

				// Add feature kits if present
				if !feature.Kits.IsNull() && !feature.Kits.IsUnknown() {
					kitIDs := extractIDsFromDynamicList(ctx, feature.Kits)
					if len(kitIDs) > 0 {
						featureMeta["kits"] = kitIDs
					}
				}

				featuresMetadata = append(featuresMetadata, featureMeta)
			}

			if len(featuresMetadata) > 0 {
				projectContext["features"] = featuresMetadata
			}
		} else {
			tflog.Warn(ctx, "[DEBUG-BUG-006] buildProjectContext: features type cast to types.Map FAILED - checking for basetypes.ObjectValue", map[string]interface{}{
				"actual_type": fmt.Sprintf("%T", underlyingVal),
			})

			// Try basetypes.ObjectValue (this is what Dynamic often returns when reading from state)
			if fObj, objOk := underlyingVal.(basetypes.ObjectValue); objOk {
				tflog.Debug(ctx, "[DEBUG-BUG-006] buildProjectContext: successfully cast to basetypes.ObjectValue", map[string]interface{}{
					"num_attributes": len(fObj.Attributes()),
				})
				featuresMetadata := []map[string]interface{}{}

				for featureName, featureValue := range fObj.Attributes() {
					feature, err := r.parseFeature(ctx, featureValue)
					if err != nil {
						tflog.Warn(ctx, "Failed to parse feature for context (ObjectValue)", map[string]interface{}{
							"feature_name": featureName,
							"error":        err.Error(),
						})
						continue
					}

					if feature == nil {
						continue
					}

					featureMeta := map[string]interface{}{
						"name": featureName,
					}

					// Add prompt from requirements if present
					if len(feature.Requirements) > 0 {
						// Get the prompt from the first requirement's first instruction
						if len(feature.Requirements[0].Instructions) > 0 {
							if !feature.Requirements[0].Instructions[0].Prompt.IsNull() {
								featureMeta["prompt"] = feature.Requirements[0].Instructions[0].Prompt.ValueString()
							}
						}
					}

					// Add feature files if present
					if !feature.Files.IsNull() && !feature.Files.IsUnknown() {
						files := []string{}
						for path := range feature.Files.Elements() {
							files = append(files, path)
						}
						if len(files) > 0 {
							featureMeta["files"] = files
						}
					}

					// Add feature kits if present
					if !feature.Kits.IsNull() && !feature.Kits.IsUnknown() {
						kitIDs := extractIDsFromDynamicList(ctx, feature.Kits)
						if len(kitIDs) > 0 {
							featureMeta["kits"] = kitIDs
						}
					}

					featuresMetadata = append(featuresMetadata, featureMeta)
				}

				if len(featuresMetadata) > 0 {
					projectContext["features"] = featuresMetadata
					tflog.Debug(ctx, "[DEBUG-BUG-006] buildProjectContext: added features to project_context from ObjectValue", map[string]interface{}{
						"feature_count": len(featuresMetadata),
					})
				}
			} else {
				// Neither Map nor ObjectValue - return without features
				tflog.Warn(ctx, "[DEBUG-BUG-006] buildProjectContext: features is neither types.Map nor basetypes.ObjectValue", map[string]interface{}{
					"actual_type": fmt.Sprintf("%T", underlyingVal),
				})
			}
		}
	}

	// TODO: Collect integrations metadata (from registry) when integration resource is implemented
	// For now, integrations are not part of the project context

	// Collect kits metadata
	projectKitIDs := extractIDsFromDynamicList(ctx, data.Kits)
	featureKitIDs := r.collectFeatureKitIDs(ctx, data)
	allKitIDs := append(projectKitIDs, featureKitIDs...)

	// Deduplicate kit IDs
	seenKits := make(map[string]bool)
	uniqueKitIDs := []string{}
	for _, id := range allKitIDs {
		if !seenKits[id] {
			uniqueKitIDs = append(uniqueKitIDs, id)
			seenKits[id] = true
		}
	}

	if len(uniqueKitIDs) > 0 {
		kitsMetadata := []map[string]interface{}{}
		for _, kitID := range uniqueKitIDs {
			// Include the kit ID for reference
			// Kits are stored with their full data in the registry
			// We'll just include the ID for now - full kit details can be looked up if needed
			kitsMetadata = append(kitsMetadata, map[string]interface{}{
				"id": kitID,
			})
		}
		projectContext["kits"] = kitsMetadata
	}

	// Collect requirements metadata
	if len(data.Requirements) > 0 {
		requirementsMetadata := []map[string]interface{}{}
		for _, req := range data.Requirements {
			requirementsMetadata = append(requirementsMetadata, map[string]interface{}{
				"name": req.Name.ValueString(),
			})
		}
		projectContext["requirements"] = requirementsMetadata
	}

	tflog.Debug(ctx, "Built project context for introspection", map[string]interface{}{
		"has_features":     projectContext["features"] != nil,
		"has_kits":         projectContext["kits"] != nil,
		"has_requirements": projectContext["requirements"] != nil,
	})

	return projectContext
}

// parseFeature extracts FeatureModel from either resource reference or inline definition
func (r *ProjectResourceFinal) parseFeature(ctx context.Context, featureValue attr.Value) (*schemas.FeatureModel, error) {
	tflog.Debug(ctx, "=== parseFeature START ===", map[string]interface{}{
		"feature_value_type": fmt.Sprintf("%T", featureValue),
	})

	// NEW: Check if this is a String (feature ID from module output)
	if featureStr, ok := featureValue.(types.String); ok {
		if featureStr.IsNull() || featureStr.IsUnknown() {
			tflog.Warn(ctx, "Feature ID is null or unknown", nil)
			return nil, fmt.Errorf("feature ID is null or unknown")
		}

		featureID := featureStr.ValueString()
		tflog.Info(ctx, "Feature is a string ID, looking up in registry", map[string]interface{}{
			"feature_id": featureID,
		})

		// Look up feature in registry
		if provData, ok := r.ProviderData.(interface {
			GetRegistry() *registry.Registry
		}); ok {
			reg := provData.GetRegistry()
			if reg != nil {
				if featureData, exists := reg.GetFeature(featureID); exists {
					tflog.Info(ctx, "Found feature in registry by ID", map[string]interface{}{
						"feature_id": featureID,
					})

					if feature, ok := featureData.(FeatureResourceModel); ok {
						// Convert FeatureResourceModel to FeatureModel
						return &schemas.FeatureModel{
							Requirements:  feature.Requirements,
							Files:         feature.Files,
							Kits:          feature.Kits,
							Verifications: feature.Verifications,
						}, nil
					}
				} else {
					tflog.Warn(ctx, "Feature not found in registry", map[string]interface{}{
						"feature_id": featureID,
					})
					return nil, fmt.Errorf("feature %s not found in registry", featureID)
				}
			}
		}
		return nil, fmt.Errorf("registry not available for feature lookup")
	}

	// Handle both types.Object and basetypes.ObjectValue
	var attrs map[string]attr.Value

	if featureObj, ok := featureValue.(types.Object); ok {
		attrs = featureObj.Attributes()
		tflog.Info(ctx, "Feature is types.Object", map[string]interface{}{
			"attribute_count": len(attrs),
		})
	} else if featureObj, ok := featureValue.(basetypes.ObjectValue); ok {
		attrs = featureObj.Attributes()
		tflog.Info(ctx, "Feature is basetypes.ObjectValue", map[string]interface{}{
			"attribute_count": len(attrs),
		})
	} else {
		err := fmt.Errorf("feature value is neither string, types.Object, nor basetypes.ObjectValue: %T", featureValue)
		tflog.Error(ctx, "parseFeature failed", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}
	tflog.Info(ctx, "Feature object attributes", map[string]interface{}{
		"attribute_count": len(attrs),
		"has_id":          attrs["id"] != nil,
		"has_prompt":      attrs["prompt"] != nil,
		"has_files":       attrs["files"] != nil,
	})

	// Check if this is a resource reference (has an 'id' attribute)
	if idAttr, hasID := attrs["id"]; hasID {
		tflog.Info(ctx, "Feature has ID attribute - checking if it's a resource reference", map[string]interface{}{
			"id_attr_type":    fmt.Sprintf("%T", idAttr),
			"id_attr_is_null": idAttr == nil,
		})

		if idStr, ok := idAttr.(types.String); ok && !idStr.IsNull() && !idStr.IsUnknown() {
			featureID := idStr.ValueString()
			tflog.Info(ctx, "Feature is a resource reference - looking up in registry", map[string]interface{}{
				"feature_id": featureID,
			})

			// Get the registry to look up the feature
			if provData, ok := r.ProviderData.(interface {
				GetRegistry() *registry.Registry
			}); ok {
				reg := provData.GetRegistry()
				tflog.Info(ctx, "Got registry from provider", map[string]interface{}{
					"registry_not_nil": reg != nil,
				})

				if reg != nil {
					// Log all features in registry for debugging
					allFeatures := reg.GetAllFeatures()
					tflog.Info(ctx, "Features in registry", map[string]interface{}{
						"feature_count": len(allFeatures),
						"feature_ids": func() []string {
							ids := make([]string, 0, len(allFeatures))
							for id := range allFeatures {
								ids = append(ids, id)
							}
							return ids
						}(),
					})

					if featureData, exists := reg.GetFeature(featureID); exists {
						tflog.Info(ctx, "Found feature in registry", map[string]interface{}{
							"feature_id":   featureID,
							"feature_type": fmt.Sprintf("%T", featureData),
						})

						if feature, ok := featureData.(FeatureResourceModel); ok {
							tflog.Info(ctx, "Successfully cast to FeatureResourceModel", map[string]interface{}{
								"feature_id":         featureID,
								"feature_name":       feature.Name.ValueString(),
								"has_files":          !feature.Files.IsNull(),
								"files_count":        len(feature.Files.Elements()),
								"has_requirements":   len(feature.Requirements) > 0,
								"requirements_count": len(feature.Requirements),
							})

							// Convert FeatureResourceModel to FeatureModel
							return &schemas.FeatureModel{
								Requirements:  feature.Requirements,
								Files:         feature.Files,
								Kits:          feature.Kits,
								Verifications: feature.Verifications, // Already the correct type
							}, nil
						} else {
							tflog.Warn(ctx, "Feature data is not FeatureResourceModel", map[string]interface{}{
								"feature_id":  featureID,
								"actual_type": fmt.Sprintf("%T", featureData),
							})
						}
					} else {
						tflog.Warn(ctx, "Feature resource not found in registry", map[string]interface{}{
							"feature_id": featureID,
						})
					}
				} else {
					tflog.Warn(ctx, "Registry is nil", nil)
				}
			} else {
				tflog.Warn(ctx, "Provider data doesn't support GetRegistry", map[string]interface{}{
					"provider_data_type": fmt.Sprintf("%T", r.ProviderData),
				})
			}
		} else {
			tflog.Info(ctx, "ID attribute is not a valid string", map[string]interface{}{
				"id_ok":         ok,
				"id_is_null":    idStr.IsNull(),
				"id_is_unknown": idStr.IsUnknown(),
			})
		}
	} else {
		tflog.Info(ctx, "Feature has no ID attribute - treating as inline definition", nil)
	}

	// No ID or registry lookup failed - parse as inline definition
	tflog.Info(ctx, "Parsing as inline feature definition", nil)
	featureModel := &schemas.FeatureModel{}

	// Extract requirements (optional)
	tflog.Info(ctx, "Checking for requirements attribute", map[string]interface{}{
		"has_requirements": attrs["requirements"] != nil,
	})

	if requirementsVal, exists := attrs["requirements"]; exists {
		tflog.Info(ctx, "Requirements attribute found", map[string]interface{}{
			"type": fmt.Sprintf("%T", requirementsVal),
		})

		if requirementsList, ok := requirementsVal.(types.List); ok {
			tflog.Info(ctx, "Successfully cast to types.List", map[string]interface{}{
				"element_count": len(requirementsList.Elements()),
			})
			elements := requirementsList.Elements()
			featureModel.Requirements = make([]schemas.RequirementModel, 0, len(elements))
			for _, elem := range elements {
				if reqObj, ok := elem.(types.Object); ok {
					reqAttrs := reqObj.Attributes()
					req := schemas.RequirementModel{}

					// Extract name
					if nameVal, exists := reqAttrs["name"]; exists {
						if nameStr, ok := nameVal.(types.String); ok {
							req.Name = nameStr
						}
					}

					// Extract instructions
					if instructionsVal, exists := reqAttrs["instructions"]; exists {
						if instrList, ok := instructionsVal.(types.List); ok {
							instrElements := instrList.Elements()
							req.Instructions = make([]schemas.InstructionModel, 0, len(instrElements))
							for _, instrElem := range instrElements {
								if instrObj, ok := instrElem.(types.Object); ok {
									instrAttrs := instrObj.Attributes()
									instr := schemas.InstructionModel{}

									// Extract prompt
									if promptVal, exists := instrAttrs["prompt"]; exists {
										if promptStr, ok := promptVal.(types.String); ok {
											instr.Prompt = promptStr
										}
									}

									// Extract constraints
									if constraintsVal, exists := instrAttrs["constraints"]; exists {
										if constraintsList, ok := constraintsVal.(types.List); ok {
											constraintsElements := constraintsList.Elements()
											instr.Constraints = make([]types.String, 0, len(constraintsElements))
											for _, constraintElem := range constraintsElements {
												if constraintStr, ok := constraintElem.(types.String); ok {
													instr.Constraints = append(instr.Constraints, constraintStr)
												}
											}
										}
									}

									req.Instructions = append(req.Instructions, instr)
								}
							}
						}
					}

					// Extract verifications
					if verificationsVal, exists := reqAttrs["verifications"]; exists {
						if verifsList, ok := verificationsVal.(types.List); ok {
							verifsElements := verifsList.Elements()
							req.Verifications = make([]schemas.VerificationModel, 0, len(verifsElements))
							for _, verifElem := range verifsElements {
								if verifObj, ok := verifElem.(types.Object); ok {
									verifAttrs := verifObj.Attributes()
									verif := schemas.VerificationModel{}

									if commandVal, exists := verifAttrs["command"]; exists {
										if commandStr, ok := commandVal.(types.String); ok {
											verif.Command = commandStr
										}
									}

									if expectVal, exists := verifAttrs["expect"]; exists {
										if expectStr, ok := expectVal.(types.String); ok {
											verif.Expect = expectStr
										}
									}

									req.Verifications = append(req.Verifications, verif)
								}
							}
						}
					}

					featureModel.Requirements = append(featureModel.Requirements, req)
				}
			}
		} else if requirementsTuple, ok := requirementsVal.(basetypes.TupleValue); ok {
			// Handle basetypes.TupleValue (what inline features actually use)
			tflog.Info(ctx, "Successfully cast to basetypes.TupleValue", map[string]interface{}{
				"element_count": len(requirementsTuple.Elements()),
			})
			elements := requirementsTuple.Elements()
			featureModel.Requirements = make([]schemas.RequirementModel, 0, len(elements))
			for _, elem := range elements {
				if reqObj, ok := elem.(types.Object); ok {
					reqAttrs := reqObj.Attributes()
					req := schemas.RequirementModel{}

					// Extract name
					if nameVal, exists := reqAttrs["name"]; exists {
						if nameStr, ok := nameVal.(types.String); ok {
							req.Name = nameStr
						}
					}

					// Extract instructions
					if instructionsVal, exists := reqAttrs["instructions"]; exists {
						// Instructions might also be a Tuple
						var instrElements []attr.Value
						if instrList, ok := instructionsVal.(types.List); ok {
							instrElements = instrList.Elements()
						} else if instrTuple, ok := instructionsVal.(basetypes.TupleValue); ok {
							instrElements = instrTuple.Elements()
						}

						if instrElements != nil {
							req.Instructions = make([]schemas.InstructionModel, 0, len(instrElements))
							for _, instrElem := range instrElements {
								if instrObj, ok := instrElem.(types.Object); ok {
									instrAttrs := instrObj.Attributes()
									instr := schemas.InstructionModel{}

									// Extract prompt
									if promptVal, exists := instrAttrs["prompt"]; exists {
										if promptStr, ok := promptVal.(types.String); ok {
											instr.Prompt = promptStr
										}
									}

									// Extract constraints
									if constraintsVal, exists := instrAttrs["constraints"]; exists {
										var constraintsElements []attr.Value
										if constraintsList, ok := constraintsVal.(types.List); ok {
											constraintsElements = constraintsList.Elements()
										} else if constraintsTuple, ok := constraintsVal.(basetypes.TupleValue); ok {
											constraintsElements = constraintsTuple.Elements()
										}

										if constraintsElements != nil {
											instr.Constraints = make([]types.String, 0, len(constraintsElements))
											for _, constraintElem := range constraintsElements {
												if constraintStr, ok := constraintElem.(types.String); ok {
													instr.Constraints = append(instr.Constraints, constraintStr)
												}
											}
										}
									}

									req.Instructions = append(req.Instructions, instr)
								}
							}
						}
					}

					// Extract verifications
					if verificationsVal, exists := reqAttrs["verifications"]; exists {
						var verifsElements []attr.Value
						if verifsList, ok := verificationsVal.(types.List); ok {
							verifsElements = verifsList.Elements()
						} else if verifsTuple, ok := verificationsVal.(basetypes.TupleValue); ok {
							verifsElements = verifsTuple.Elements()
						}

						if verifsElements != nil {
							req.Verifications = make([]schemas.VerificationModel, 0, len(verifsElements))
							for _, verifElem := range verifsElements {
								if verifObj, ok := verifElem.(types.Object); ok {
									verifAttrs := verifObj.Attributes()
									verif := schemas.VerificationModel{}

									if commandVal, exists := verifAttrs["command"]; exists {
										if commandStr, ok := commandVal.(types.String); ok {
											verif.Command = commandStr
										}
									}

									if expectVal, exists := verifAttrs["expect"]; exists {
										if expectStr, ok := expectVal.(types.String); ok {
											verif.Expect = expectStr
										}
									}

									req.Verifications = append(req.Verifications, verif)
								}
							}
						}
					}

					featureModel.Requirements = append(featureModel.Requirements, req)
				}
			}
		} else {
			tflog.Warn(ctx, "Failed to cast requirements to types.List or basetypes.TupleValue", map[string]interface{}{
				"actual_type": fmt.Sprintf("%T", requirementsVal),
			})
		}
	} else {
		tflog.Warn(ctx, "Requirements attribute NOT found in inline feature", nil)
	}

	// Extract files (optional)
	if filesVal, exists := attrs["files"]; exists {
		tflog.Info(ctx, "Found files attribute in inline feature", map[string]interface{}{
			"files_type": fmt.Sprintf("%T", filesVal),
		})

		// Try types.Map first
		if filesMap, ok := filesVal.(types.Map); ok {
			featureModel.Files = filesMap
			tflog.Info(ctx, "Successfully extracted files from inline feature (types.Map)", map[string]interface{}{
				"file_count": len(filesMap.Elements()),
			})
		} else if filesMapBase, ok := filesVal.(basetypes.MapValue); ok {
			// Handle basetypes.MapValue
			featureModel.Files = types.MapValueMust(filesMapBase.ElementType(ctx), filesMapBase.Elements())
			tflog.Info(ctx, "Successfully extracted files from inline feature (basetypes.MapValue)", map[string]interface{}{
				"file_count": len(filesMapBase.Elements()),
			})
		} else if filesObj, ok := filesVal.(basetypes.ObjectValue); ok {
			// Handle basetypes.ObjectValue - convert to Map
			// Files is a map attribute, so we need to convert the ObjectValue's attributes to a Map
			attrs := filesObj.Attributes()

			// Get element type from the first file value
			var elementType attr.Type = types.ObjectType{AttrTypes: map[string]attr.Type{}}
			for _, fileVal := range attrs {
				if fileObjVal, ok := fileVal.(basetypes.ObjectValue); ok {
					elementType = fileObjVal.Type(ctx)
					break
				}
			}

			featureModel.Files = types.MapValueMust(elementType, attrs)
			tflog.Info(ctx, "Successfully extracted files from inline feature (basetypes.ObjectValue)", map[string]interface{}{
				"file_count": len(attrs),
			})
		} else if filesDyn, ok := filesVal.(types.Dynamic); ok {
			// Try extracting from types.Dynamic wrapper
			underlying := filesDyn.UnderlyingValue()
			tflog.Info(ctx, "Files is Dynamic, extracting underlying value", map[string]interface{}{
				"underlying_type": fmt.Sprintf("%T", underlying),
			})

			if filesMap, ok := underlying.(types.Map); ok {
				featureModel.Files = filesMap
				tflog.Info(ctx, "Successfully extracted files from Dynamic wrapper", map[string]interface{}{
					"file_count": len(filesMap.Elements()),
				})
			} else if filesMapBase, ok := underlying.(basetypes.MapValue); ok {
				featureModel.Files = types.MapValueMust(filesMapBase.ElementType(ctx), filesMapBase.Elements())
				tflog.Info(ctx, "Successfully extracted files from Dynamic->basetypes.MapValue", map[string]interface{}{
					"file_count": len(filesMapBase.Elements()),
				})
			} else {
				tflog.Error(ctx, "Failed to extract files - unknown type after unwrapping Dynamic", map[string]interface{}{
					"files_type":      fmt.Sprintf("%T", filesVal),
					"underlying_type": fmt.Sprintf("%T", underlying),
				})
			}
		} else {
			tflog.Error(ctx, "Failed to cast files to types.Map, basetypes.MapValue, or types.Dynamic", map[string]interface{}{
				"actual_type": fmt.Sprintf("%T", filesVal),
			})
		}
	} else {
		tflog.Info(ctx, "No files attribute in inline feature", nil)
	}

	// Extract kits (optional)
	if kitsVal, exists := attrs["kits"]; exists {
		if kitsDyn, ok := kitsVal.(types.Dynamic); ok {
			featureModel.Kits = kitsDyn
		}
	}

	// Extract verifications (optional)
	if verifsVal, exists := attrs["verifications"]; exists {
		if verifsList, ok := verifsVal.(types.List); ok {
			elements := verifsList.Elements()
			featureModel.Verifications = make([]schemas.VerificationModel, 0, len(elements))
			for _, elem := range elements {
				if verifObj, ok := elem.(types.Object); ok {
					verifAttrs := verifObj.Attributes()
					verif := schemas.VerificationModel{}
					if cmd, exists := verifAttrs["command"]; exists {
						if cmdStr, ok := cmd.(types.String); ok {
							verif.Command = cmdStr
						}
					}
					if exp, exists := verifAttrs["expect"]; exists {
						if expStr, ok := exp.(types.String); ok {
							verif.Expect = expStr
						}
					}
					featureModel.Verifications = append(featureModel.Verifications, verif)
				}
			}
		}
	}

	return featureModel, nil
}

// collectFeaturesFromDynamic extracts feature files from a Dynamic list of feature references
func (r *ProjectResourceFinal) collectFeaturesFromDynamic(ctx context.Context, featuresDynamic types.Dynamic, reg *registry.Registry) []schemas.FileModelWithPath {
	tflog.Debug(ctx, "=== collectFeaturesFromDynamic START ===", map[string]interface{}{
		"is_null":      featuresDynamic.IsNull(),
		"is_unknown":   featuresDynamic.IsUnknown(),
		"has_registry": reg != nil,
	})

	if featuresDynamic.IsNull() || reg == nil {
		tflog.Info(ctx, "Early return: featuresDynamic is null or registry is nil", nil)
		return []schemas.FileModelWithPath{}
	}

	// Extract feature resource references from the list
	underlyingVal := featuresDynamic.UnderlyingValue()
	tflog.Info(ctx, "Underlying value extracted", map[string]interface{}{
		"type": fmt.Sprintf("%T", underlyingVal),
	})

	featuresList, ok := underlyingVal.(types.List)
	if !ok {
		tflog.Warn(ctx, "Failed to cast underlying value to types.List", map[string]interface{}{
			"actual_type": fmt.Sprintf("%T", underlyingVal),
		})
		return []schemas.FileModelWithPath{}
	}

	tflog.Info(ctx, "Features list extracted", map[string]interface{}{
		"element_count": len(featuresList.Elements()),
	})

	var allFiles []schemas.FileModelWithPath
	for i, elem := range featuresList.Elements() {
		tflog.Info(ctx, "Processing feature element", map[string]interface{}{
			"index": i,
			"type":  fmt.Sprintf("%T", elem),
		})

		// Each element should be a feature resource reference (Object with ID)
		featureObj, ok := elem.(types.Object)
		if !ok {
			tflog.Warn(ctx, "Feature element is not types.Object", map[string]interface{}{
				"index":       i,
				"actual_type": fmt.Sprintf("%T", elem),
			})
			continue
		}

		// Extract the feature ID to look up in registry
		attrs := featureObj.Attributes()
		tflog.Info(ctx, "Feature object attributes", map[string]interface{}{
			"index":      i,
			"attr_count": len(attrs),
		})

		if idAttr, exists := attrs["id"]; exists {
			tflog.Info(ctx, "Found id attribute", map[string]interface{}{
				"index":    i,
				"id_type":  fmt.Sprintf("%T", idAttr),
				"id_value": fmt.Sprintf("%+v", idAttr),
			})

			if idStr, ok := idAttr.(types.String); ok && !idStr.IsNull() && !idStr.IsUnknown() {
				featureID := idStr.ValueString()
				tflog.Info(ctx, "Looking up feature in registry", map[string]interface{}{
					"index":      i,
					"feature_id": featureID,
				})

				// Look up feature in registry
				if featureData, exists := reg.GetFeature(featureID); exists {
					tflog.Info(ctx, "Feature found in registry", map[string]interface{}{
						"feature_id": featureID,
					})

					if feature, ok := featureData.(FeatureResourceModel); ok {
						tflog.Info(ctx, "Feature data cast successful", map[string]interface{}{
							"feature_id":    featureID,
							"has_files":     !feature.Files.IsNull(),
							"files_unknown": feature.Files.IsUnknown(),
						})

						// Convert feature files to FileModelWithPath list
						featureFiles := schemas.FilesMapToList(ctx, feature.Files)
						allFiles = append(allFiles, featureFiles...)

						tflog.Info(ctx, "Collected files from feature", map[string]interface{}{
							"feature_id": featureID,
							"file_count": len(featureFiles),
						})
					} else {
						tflog.Warn(ctx, "Failed to cast feature data to FeatureResourceModel", map[string]interface{}{
							"feature_id":  featureID,
							"actual_type": fmt.Sprintf("%T", featureData),
						})
					}
				} else {
					tflog.Warn(ctx, "Feature not found in registry", map[string]interface{}{
						"feature_id": featureID,
					})
				}
			} else {
				tflog.Warn(ctx, "ID attribute is not a valid string", map[string]interface{}{
					"index":      i,
					"is_null":    idStr.IsNull(),
					"is_unknown": idStr.IsUnknown(),
				})
			}
		} else {
			tflog.Warn(ctx, "No id attribute found in feature object", map[string]interface{}{
				"index": i,
			})
		}
	}

	tflog.Debug(ctx, "=== collectFeaturesFromDynamic END ===", map[string]interface{}{
		"total_files": len(allFiles),
	})

	return allFiles
}

// collectFeatureFiles extracts files from all features in the project
func (r *ProjectResourceFinal) collectFeatureFiles(ctx context.Context, data ProjectModelFinal) []schemas.FileModelWithPath {
	tflog.Debug(ctx, "=== collectFeatureFiles START ===", map[string]interface{}{
		"features_is_null":    data.Features.IsNull(),
		"features_is_unknown": data.Features.IsUnknown(),
	})

	// Allow processing even if Unknown - features might be in registry by now
	if data.Features.IsNull() {
		tflog.Info(ctx, "Features is null, returning empty", nil)
		return []schemas.FileModelWithPath{}
	}

	// If features are Unknown, log but continue - we'll try registry lookups
	if data.Features.IsUnknown() {
		tflog.Warn(ctx, "Features attribute is Unknown, will attempt registry lookups", nil)
	}

	// Extract the underlying value from Dynamic
	underlyingVal := data.Features.UnderlyingValue()

	// AGGRESSIVE DEBUG

	tflog.Info(ctx, "Features underlying value", map[string]interface{}{
		"type": fmt.Sprintf("%T", underlyingVal),
	})

	var featuresMap types.Map

	// Try to cast to types.Map first (most common case)
	if fMap, ok := underlyingVal.(types.Map); ok {
		featuresMap = fMap
		tflog.Info(ctx, "Features is types.Map", map[string]interface{}{
			"element_count": len(featuresMap.Elements()),
		})
	} else if fObj, ok := underlyingVal.(basetypes.ObjectValue); ok {
		// Handle basetypes.ObjectValue (the actual underlying type)
		// Handle types.Object - convert it to a map-like structure
		tflog.Info(ctx, "Features is types.Object, converting to map", map[string]interface{}{
			"attr_count": len(fObj.Attributes()),
		})

		// For Object, we need to iterate over attributes directly
		// We'll process them similarly to map elements
		attrs := fObj.Attributes()
		var allFiles []schemas.FileModelWithPath

		for featureName, featureValue := range attrs {
			tflog.Info(ctx, "Processing feature from object", map[string]interface{}{
				"feature_name": featureName,
				"feature_type": fmt.Sprintf("%T", featureValue),
			})

			feature, err := r.parseFeature(ctx, featureValue)
			if err != nil {
				tflog.Warn(ctx, "Failed to parse feature", map[string]interface{}{
					"feature_name": featureName,
					"error":        err.Error(),
				})
				continue
			}

			if feature == nil {
				continue
			}

			tflog.Info(ctx, "Successfully parsed feature from object", map[string]interface{}{
				"feature_name": featureName,
				"has_files":    !feature.Files.IsNull() && !feature.Files.IsUnknown(),
			})

			if feature.Files.IsNull() || feature.Files.IsUnknown() {
				tflog.Info(ctx, "Feature has no files, skipping", map[string]interface{}{
					"feature_name": featureName,
				})
				continue
			}

			// Convert feature files from map to list
			featureFiles := schemas.FilesMapToList(ctx, feature.Files)
			allFiles = append(allFiles, featureFiles...)

			tflog.Info(ctx, "Collected files from feature in object", map[string]interface{}{
				"feature_name": featureName,
				"file_count":   len(featureFiles),
			})
		}

		tflog.Debug(ctx, "=== collectFeatureFiles END (Object path) ===", map[string]interface{}{
			"total_files_collected": len(allFiles),
		})

		return allFiles
	} else {
		tflog.Error(ctx, "Features is neither Map nor Object", map[string]interface{}{
			"type": fmt.Sprintf("%T", underlyingVal),
		})
		return []schemas.FileModelWithPath{}
	}

	tflog.Info(ctx, "Features map details", map[string]interface{}{
		"element_count": len(featuresMap.Elements()),
		"element_type":  featuresMap.ElementType(ctx).String(),
	})

	var allFiles []schemas.FileModelWithPath

	for featureName, featureValue := range featuresMap.Elements() {
		tflog.Info(ctx, "Processing feature from map", map[string]interface{}{
			"feature_name":  featureName,
			"feature_type":  fmt.Sprintf("%T", featureValue),
			"feature_value": fmt.Sprintf("%+v", featureValue),
		})

		feature, err := r.parseFeature(ctx, featureValue)
		if err != nil {
			tflog.Warn(ctx, "Failed to parse feature", map[string]interface{}{
				"feature_name": featureName,
				"error":        err.Error(),
			})
			continue
		}

		tflog.Info(ctx, "Successfully parsed feature", map[string]interface{}{
			"feature_name": featureName,
			"has_files":    !feature.Files.IsNull() && !feature.Files.IsUnknown(),
		})

		if feature.Files.IsNull() || feature.Files.IsUnknown() {
			tflog.Info(ctx, "Feature has no files, skipping", map[string]interface{}{
				"feature_name": featureName,
			})
			continue
		}

		// Convert feature files from map to list
		featureFiles := schemas.FilesMapToList(ctx, feature.Files)
		allFiles = append(allFiles, featureFiles...)

		tflog.Info(ctx, "Collected files from feature", map[string]interface{}{
			"feature_name": featureName,
			"file_count":   len(featureFiles),
		})
	}

	tflog.Debug(ctx, "=== collectFeatureFiles END ===", map[string]interface{}{
		"total_files_collected": len(allFiles),
	})

	return allFiles
}

// convertFeaturesToRequirements transforms features into requirements for Claude
func (r *ProjectResourceFinal) convertFeaturesToRequirements(ctx context.Context, data ProjectModelFinal) []schemas.RequirementModel {
	tflog.Info(ctx, "=== convertFeaturesToRequirements START ===", nil)

	if data.Features.IsNull() || data.Features.IsUnknown() {
		tflog.Info(ctx, "Features is null or unknown, returning empty requirements", nil)
		return []schemas.RequirementModel{}
	}

	// Extract the underlying value from Dynamic
	underlyingVal := data.Features.UnderlyingValue()
	tflog.Info(ctx, "Extracted underlying value from Features", map[string]interface{}{
		"type": fmt.Sprintf("%T", underlyingVal),
	})

	// Try to cast to types.Map (for map structure)
	featuresMap, ok := underlyingVal.(types.Map)
	if !ok {
		tflog.Warn(ctx, "Features is not types.Map - checking for basetypes.ObjectValue", map[string]interface{}{
			"type": fmt.Sprintf("%T", underlyingVal),
		})

		// Try basetypes.ObjectValue (this is what Dynamic often returns)
		if fObj, objOk := underlyingVal.(basetypes.ObjectValue); objOk {
			tflog.Info(ctx, "Features is basetypes.ObjectValue - processing attributes", map[string]interface{}{
				"attribute_count": len(fObj.Attributes()),
			})

			var requirements []schemas.RequirementModel

			for featureName, featureValue := range fObj.Attributes() {
				tflog.Info(ctx, "Processing feature from ObjectValue", map[string]interface{}{
					"feature_name": featureName,
				})

				feature, err := r.parseFeature(ctx, featureValue)
				if err != nil {
					tflog.Warn(ctx, "Failed to parse feature from ObjectValue", map[string]interface{}{
						"feature_name": featureName,
						"error":        err.Error(),
					})
					continue
				}

				tflog.Info(ctx, "Successfully parsed feature", map[string]interface{}{
					"feature_name":      featureName,
					"requirement_count": len(feature.Requirements),
				})

				// Feature already has requirements! Just pass them through
				requirements = append(requirements, feature.Requirements...)

				tflog.Debug(ctx, "Converted feature to requirements", map[string]interface{}{
					"feature_name":      featureName,
					"requirement_count": len(feature.Requirements),
				})
			}

			tflog.Info(ctx, "=== convertFeaturesToRequirements END (ObjectValue path) ===", map[string]interface{}{
				"total_requirements": len(requirements),
			})
			return requirements
		}

		// Neither Map nor ObjectValue - return empty
		tflog.Warn(ctx, "Features is neither types.Map nor basetypes.ObjectValue, returning empty", map[string]interface{}{
			"type": fmt.Sprintf("%T", underlyingVal),
		})
		return []schemas.RequirementModel{}
	}

	tflog.Info(ctx, "Features is types.Map, processing elements", map[string]interface{}{
		"element_count": len(featuresMap.Elements()),
	})

	var requirements []schemas.RequirementModel

	for featureName, featureValue := range featuresMap.Elements() {
		tflog.Info(ctx, "Processing feature from Map", map[string]interface{}{
			"feature_name": featureName,
		})

		feature, err := r.parseFeature(ctx, featureValue)
		if err != nil {
			tflog.Warn(ctx, "Failed to parse feature from Map", map[string]interface{}{
				"feature_name": featureName,
				"error":        err.Error(),
			})
			continue
		}

		tflog.Info(ctx, "Successfully parsed feature", map[string]interface{}{
			"feature_name":      featureName,
			"requirement_count": len(feature.Requirements),
		})

		// Feature already has requirements! Just pass them through
		requirements = append(requirements, feature.Requirements...)

		tflog.Debug(ctx, "Converted feature to requirements", map[string]interface{}{
			"feature_name":      featureName,
			"requirement_count": len(feature.Requirements),
		})
	}

	tflog.Info(ctx, "=== convertFeaturesToRequirements END (Map path) ===", map[string]interface{}{
		"total_requirements": len(requirements),
	})
	return requirements
}

// collectFeatureKitIDs extracts kit IDs from all features
func (r *ProjectResourceFinal) collectFeatureKitIDs(ctx context.Context, data ProjectModelFinal) []string {
	// DEBUG logging
	debugFile, _ := os.OpenFile("/tmp/tofukit-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if debugFile != nil {
		defer debugFile.Close()
		fmt.Fprintf(debugFile, "\n=== collectFeatureKitIDs START ===\n")
	}

	if data.Features.IsNull() || data.Features.IsUnknown() {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "collectFeatureKitIDs: Features is null or unknown\n")
		}
		return []string{}
	}

	// Extract the underlying value from Dynamic
	underlyingVal := data.Features.UnderlyingValue()

	var allKitIDs []string
	seen := make(map[string]bool)

	// Try to cast to types.Map (for map structure)
	if featuresMap, ok := underlyingVal.(types.Map); ok {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "collectFeatureKitIDs: Features is types.Map with %d elements\n", len(featuresMap.Elements()))
		}

		for featureName, featureValue := range featuresMap.Elements() {
			if debugFile != nil {
				fmt.Fprintf(debugFile, "collectFeatureKitIDs: Processing feature '%s' from Map\n", featureName)
			}

			feature, err := r.parseFeature(ctx, featureValue)
			if err != nil {
				if debugFile != nil {
					fmt.Fprintf(debugFile, "collectFeatureKitIDs: Failed to parse feature '%s': %v\n", featureName, err)
				}
				continue
			}

			if debugFile != nil {
				fmt.Fprintf(debugFile, "collectFeatureKitIDs: Feature '%s' kits IsNull=%v IsUnknown=%v\n",
					featureName, feature.Kits.IsNull(), feature.Kits.IsUnknown())
			}

			if feature.Kits.IsNull() || feature.Kits.IsUnknown() {
				if debugFile != nil {
					fmt.Fprintf(debugFile, "collectFeatureKitIDs: Feature '%s' has no kits\n", featureName)
				}
				continue
			}

			kitIDs := extractIDsFromDynamicList(ctx, feature.Kits)
			if debugFile != nil {
				fmt.Fprintf(debugFile, "collectFeatureKitIDs: Feature '%s' has %d kits: %v\n", featureName, len(kitIDs), kitIDs)
			}

			for _, kitID := range kitIDs {
				if !seen[kitID] {
					allKitIDs = append(allKitIDs, kitID)
					seen[kitID] = true
				}
			}

			tflog.Debug(ctx, "Collected kits from feature", map[string]interface{}{
				"feature_name": featureName,
				"kit_count":    len(kitIDs),
			})
		}
	} else if fObj, ok := underlyingVal.(basetypes.ObjectValue); ok {
		// Handle basetypes.ObjectValue (Dynamic's underlying type for objects)
		attrs := fObj.Attributes()
		if debugFile != nil {
			fmt.Fprintf(debugFile, "collectFeatureKitIDs: Features is basetypes.ObjectValue with %d attributes\n", len(attrs))
		}

		for featureName, featureValue := range attrs {
			if debugFile != nil {
				fmt.Fprintf(debugFile, "collectFeatureKitIDs: Processing feature '%s' from ObjectValue\n", featureName)
			}

			feature, err := r.parseFeature(ctx, featureValue)
			if err != nil {
				if debugFile != nil {
					fmt.Fprintf(debugFile, "collectFeatureKitIDs: Failed to parse feature '%s': %v\n", featureName, err)
				}
				continue
			}

			if debugFile != nil {
				fmt.Fprintf(debugFile, "collectFeatureKitIDs: Feature '%s' kits IsNull=%v IsUnknown=%v\n",
					featureName, feature.Kits.IsNull(), feature.Kits.IsUnknown())
			}

			if feature.Kits.IsNull() || feature.Kits.IsUnknown() {
				if debugFile != nil {
					fmt.Fprintf(debugFile, "collectFeatureKitIDs: Feature '%s' has no kits\n", featureName)
				}
				continue
			}

			kitIDs := extractIDsFromDynamicList(ctx, feature.Kits)
			if debugFile != nil {
				fmt.Fprintf(debugFile, "collectFeatureKitIDs: Feature '%s' has %d kits: %v\n", featureName, len(kitIDs), kitIDs)
			}

			for _, kitID := range kitIDs {
				if !seen[kitID] {
					allKitIDs = append(allKitIDs, kitID)
					seen[kitID] = true
				}
			}

			tflog.Debug(ctx, "Collected kits from feature", map[string]interface{}{
				"feature_name": featureName,
				"kit_count":    len(kitIDs),
			})
		}
	} else {
		if debugFile != nil {
			fmt.Fprintf(debugFile, "collectFeatureKitIDs: Features is neither Map nor ObjectValue, type=%T\n", underlyingVal)
		}
		tflog.Warn(ctx, "Features is neither Map nor ObjectValue", map[string]interface{}{
			"type": fmt.Sprintf("%T", underlyingVal),
		})
		return []string{}
	}

	if debugFile != nil {
		fmt.Fprintf(debugFile, "=== collectFeatureKitIDs END: Total %d kits collected ===\n\n", len(allKitIDs))
	}

	return allKitIDs
}

// CollectKitVerifications extracts verification commands from kit requirements for enforcement.
//
// This function enables kit verification enforcement by collecting all verification commands
// from kit requirements and converting them into FileModelWithPath entries that can be
// passed to the verification runner alongside file verifications.
//
// Key behaviors:
//   - Iterates through all kits in the outputData map
//   - Extracts requirements from each kit
//   - Collects verification commands from each requirement
//   - Returns verifications with pseudo-paths in format: "kit:{kitName}:{reqName}:{idx}"
//   - Handles both "verification" (singular) and "verifications" (plural) field names
//   - Supports both map[string]interface{} and map[string]string value types
//
// The returned verifications are merged with file verifications in executeClaudeCode()
// and enforced by ExecuteWithPromptJSON(). Failed verifications cause provider errors
// and Terraform apply failures.
//
// Note: This is exported for testing purposes
func (r *ProjectResourceFinal) CollectKitVerifications(
	ctx context.Context,
	kits map[string]interface{},
) []schemas.FileModelWithPath {
	var verifications []schemas.FileModelWithPath

	if kits == nil {
		return verifications
	}

	for kitID, kitData := range kits {
		kitMap, ok := kitData.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract kit name for error messages
		kitName := kitID
		if name, ok := kitMap["name"].(string); ok {
			kitName = name
		}

		// Get requirements array
		requirements, ok := kitMap["requirements"].([]interface{})
		if !ok {
			continue
		}

		// Iterate through requirements
		for _, reqInterface := range requirements {
			reqMap, ok := reqInterface.(map[string]interface{})
			if !ok {
				continue
			}

			reqName, _ := reqMap["name"].(string)

			// Get verifications array (try both field names for compatibility)
			// "verifications" (plural) - used by project kits from Terraform state
			// "verification" (singular) - used by registry kits
			verificationsData, ok := reqMap["verifications"].([]interface{})
			if !ok {
				verificationsData, ok = reqMap["verification"].([]interface{})
				if !ok {
					continue
				}
			}

			// Convert each verification to FileModelWithPath for consistency
			for idx, verifyInterface := range verificationsData {
				// Extract command and expect from verification entry
				// Handle two possible types from different serialization paths:
				// 1. map[string]interface{} - from outputData/prompt JSON (most common)
				// 2. map[string]string - from registry serialization (fallback)
				var command, expect string
				if verifyMap, ok := verifyInterface.(map[string]interface{}); ok {
					if cmd, ok := verifyMap["command"].(string); ok {
						command = cmd
					}
					if exp, ok := verifyMap["expect"].(string); ok {
						expect = exp
					}
				} else if verifyMap, ok := verifyInterface.(map[string]string); ok {
					command = verifyMap["command"]
					expect = verifyMap["expect"]
				} else {
					// Unknown type - skip this verification
					continue
				}

				if command == "" {
					continue
				}

				// Create a pseudo-file entry to carry verification info
				// Path format: "kit:{kitName}:{reqName}:{idx}"
				pseudoPath := fmt.Sprintf("kit:%s:%s:%d", kitName, reqName, idx)

				verification := schemas.FileModelWithPath{
					Path: pseudoPath,
					Verifications: []schemas.VerificationModel{
						{
							Command: types.StringValue(command),
							Expect:  types.StringValue(expect),
						},
					},
				}

				verifications = append(verifications, verification)
			}
		}
	}

	tflog.Debug(ctx, "Collected kit verifications", map[string]interface{}{
		"count": len(verifications),
	})

	return verifications
}

// collectStackKitsAndRequirements collects kits and their requirements from the referenced stack
func (r *ProjectResourceFinal) collectStackKitsAndRequirements(ctx context.Context, data ProjectModelFinal, reg *registry.Registry) (map[string]interface{}, []schemas.RequirementModel) {
	kitsMap := make(map[string]interface{})
	var requirements []schemas.RequirementModel

	if reg == nil {
		return kitsMap, requirements
	}

	// Get stack ID from project
	stackID := extractIDFromDynamic(ctx, data.Stack)
	if stackID == "" {
		return kitsMap, requirements
	}

	// Get stack from registry
	stackData, exists := reg.GetStack(stackID)
	if !exists {
		return kitsMap, requirements
	}

	stack, ok := stackData.(StackResourceModel)
	if !ok {
		return kitsMap, requirements
	}

	// Extract kits from stack - they're embedded directly, not registry references
	if stack.Kits.IsNull() || stack.Kits.IsUnknown() {
		return kitsMap, requirements
	}

	// Get underlying value from Dynamic
	underlyingVal := stack.Kits.UnderlyingValue()

	// Extract elements (could be List or Tuple)
	var kitElements []attr.Value
	if listVal, ok := underlyingVal.(types.List); ok && !listVal.IsNull() {
		kitElements = listVal.Elements()
	} else if tupleVal, ok := underlyingVal.(types.Tuple); ok && !tupleVal.IsNull() {
		kitElements = tupleVal.Elements()
	}

	tflog.Info(ctx, "Collecting kits and requirements from stack", map[string]interface{}{
		"stack_id":  stackID,
		"kit_count": len(kitElements),
	})

	// For each kit embedded in the stack
	for _, kitElem := range kitElements {
		kitObj, ok := kitElem.(types.Object)
		if !ok || kitObj.IsNull() {
			continue
		}

		attrs := kitObj.Attributes()

		// Extract basic kit info
		var kitID, kitName, kitDesc, kitVersion string
		if idAttr, exists := attrs["id"]; exists {
			if idStr, ok := idAttr.(types.String); ok && !idStr.IsNull() {
				kitID = idStr.ValueString()
			}
		}
		if nameAttr, exists := attrs["name"]; exists {
			if nameStr, ok := nameAttr.(types.String); ok && !nameStr.IsNull() {
				kitName = nameStr.ValueString()
			}
		}
		if descAttr, exists := attrs["description"]; exists {
			if descStr, ok := descAttr.(types.String); ok && !descStr.IsNull() {
				kitDesc = descStr.ValueString()
			}
		}
		if verAttr, exists := attrs["version"]; exists {
			if verStr, ok := verAttr.(types.String); ok && !verStr.IsNull() {
				kitVersion = verStr.ValueString()
			}
		}

		if kitID != "" {
			kitData := map[string]interface{}{
				"id":          kitID,
				"name":        kitName,
				"description": kitDesc,
				"version":     kitVersion,
			}
			kitsMap[kitID] = kitData

			// Extract requirements from the kit
			if reqsAttr, exists := attrs["requirements"]; exists {
				if reqsList, ok := reqsAttr.(types.List); ok && !reqsList.IsNull() {
					for _, reqElem := range reqsList.Elements() {
						if reqObj, ok := reqElem.(types.Object); ok && !reqObj.IsNull() {
							reqAttrs := reqObj.Attributes()

							// Build RequirementModel from embedded data
							var reqName types.String
							var reqInstructions []schemas.InstructionModel
							var reqVerifications []schemas.VerificationModel

							if nameAttr, exists := reqAttrs["name"]; exists {
								if nameStr, ok := nameAttr.(types.String); ok {
									reqName = nameStr
								}
							}

							// Extract instructions
							if instAttr, exists := reqAttrs["instructions"]; exists {
								if instList, ok := instAttr.(types.List); ok && !instList.IsNull() {
									for _, instElem := range instList.Elements() {
										if instObj, ok := instElem.(types.Object); ok && !instObj.IsNull() {
											instAttrs := instObj.Attributes()
											var prompt types.String
											var constraints []types.String

											if promptAttr, exists := instAttrs["prompt"]; exists {
												if promptStr, ok := promptAttr.(types.String); ok {
													prompt = promptStr
												}
											}

											if consAttr, exists := instAttrs["constraints"]; exists {
												if consList, ok := consAttr.(types.List); ok && !consList.IsNull() {
													for _, consElem := range consList.Elements() {
														if consStr, ok := consElem.(types.String); ok {
															constraints = append(constraints, consStr)
														}
													}
												}
											}

											reqInstructions = append(reqInstructions, schemas.InstructionModel{
												Prompt:      prompt,
												Constraints: constraints,
											})
										}
									}
								}
							}

							// Extract verifications
							if verifAttr, exists := reqAttrs["verifications"]; exists {
								if verifList, ok := verifAttr.(types.List); ok && !verifList.IsNull() {
									for _, verifElem := range verifList.Elements() {
										if verifObj, ok := verifElem.(types.Object); ok && !verifObj.IsNull() {
											verifAttrs := verifObj.Attributes()
											var cmd, expect types.String

											if cmdAttr, exists := verifAttrs["command"]; exists {
												if cmdStr, ok := cmdAttr.(types.String); ok {
													cmd = cmdStr
												}
											}
											if expAttr, exists := verifAttrs["expect"]; exists {
												if expStr, ok := expAttr.(types.String); ok {
													expect = expStr
												}
											}

											reqVerifications = append(reqVerifications, schemas.VerificationModel{
												Command: cmd,
												Expect:  expect,
											})
										}
									}
								}
							}

							requirements = append(requirements, schemas.RequirementModel{
								Name:          reqName,
								Instructions:  reqInstructions,
								Verifications: reqVerifications,
							})
						}
					}
				}
			}

			tflog.Debug(ctx, "Collected embedded kit from stack", map[string]interface{}{
				"kit_id":            kitID,
				"requirement_count": len(requirements),
			})
		}
	}

	tflog.Info(ctx, "Collected stack kits and requirements", map[string]interface{}{
		"kit_count":         len(kitsMap),
		"requirement_count": len(requirements),
	})

	return kitsMap, requirements
}

// collectStackFeaturesRequirements collects requirements from stack features
func (r *ProjectResourceFinal) collectStackFeaturesRequirements(ctx context.Context, data ProjectModelFinal, reg *registry.Registry) []schemas.RequirementModel {
	var requirements []schemas.RequirementModel

	if reg == nil {
		return requirements
	}

	// Get stack ID from project
	stackID := extractIDFromDynamic(ctx, data.Stack)
	if stackID == "" {
		return requirements
	}

	// Get stack from registry
	stackData, exists := reg.GetStack(stackID)
	if !exists {
		return requirements
	}

	stack, ok := stackData.(StackResourceModel)
	if !ok {
		return requirements
	}

	// Extract feature IDs from stack
	stackFeatureIDs := extractIDsFromDynamicList(ctx, stack.Features)

	tflog.Info(ctx, "Collecting requirements from stack features", map[string]interface{}{
		"stack_id":      stackID,
		"feature_count": len(stackFeatureIDs),
	})

	// For each feature in the stack, convert to requirement
	for _, featureID := range stackFeatureIDs {
		featureData, exists := reg.GetFeature(featureID)
		if !exists {
			continue
		}

		feature, ok := featureData.(FeatureResourceModel)
		if !ok {
			continue
		}

		// Feature already has requirements! Just pass them through
		requirements = append(requirements, feature.Requirements...)

		tflog.Debug(ctx, "Collected stack feature requirements", map[string]interface{}{
			"feature_id":        featureID,
			"requirement_count": len(feature.Requirements),
		})
	}

	tflog.Info(ctx, "Collected stack feature requirements", map[string]interface{}{
		"requirement_count": len(requirements),
	})

	return requirements
}

// collectAndMergeFiles collects files from all sources and merges them with proper precedence
func (r *ProjectResourceFinal) collectAndMergeFiles(ctx context.Context, data ProjectModelFinal) []schemas.FileModelWithPath {
	// Debug to file since stdout isn't captured
	debugFile, _ := os.Create("/tmp/tofukit-collectAndMergeFiles-debug.txt")
	if debugFile != nil {
		debugFile.WriteString(fmt.Sprintf("collectAndMergeFiles called\n"))
		debugFile.WriteString(fmt.Sprintf("Features.IsNull() = %v, IsUnknown() = %v\n", data.Features.IsNull(), data.Features.IsUnknown()))
		debugFile.Close()
	}

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
						"stack_id":            stackID,
						"file_count":          len(stackFiles),
						"stack_name":          stack.Name.ValueString(),
						"features_is_null":    stack.Features.IsNull(),
						"features_is_unknown": stack.Features.IsUnknown(),
						"features_value":      fmt.Sprintf("%+v", stack.Features),
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

					// 3. NEW: Add files from features within the primary stack
					// Debug: Write stack.Features to file for inspection
					debugPath := "/tmp/stack-features-debug.txt"
					debugContent := fmt.Sprintf("stack.Features IsNull: %v\nstack.Features IsUnknown: %v\nstack.Features Value: %+v\n",
						stack.Features.IsNull(), stack.Features.IsUnknown(), stack.Features)
					os.WriteFile(debugPath, []byte(debugContent), 0644)

					// Debug: Check underlying type
					underlying := stack.Features.UnderlyingValue()
					debugContent2 := fmt.Sprintf("\nUnderlying type: %T\nUnderlying value: %+v\n", underlying, underlying)
					f, _ := os.OpenFile(debugPath, os.O_APPEND|os.O_WRONLY, 0644)
					if f != nil {
						f.WriteString(debugContent2)
						f.Close()
					}

					stackFeatureIDs := extractIDsFromDynamicList(ctx, stack.Features)
					if len(stackFeatureIDs) > 0 {
						var stackFeatureFiles []schemas.FileModelWithPath
						for _, featureID := range stackFeatureIDs {
							if featureData, exists := reg.GetFeature(featureID); exists {
								if feature, ok := featureData.(FeatureResourceModel); ok {
									featureFiles := schemas.FilesMapToList(ctx, feature.Files)
									stackFeatureFiles = append(stackFeatureFiles, featureFiles...)
									tflog.Info(ctx, "Collected files from stack feature", map[string]interface{}{
										"feature_id": featureID,
										"file_count": len(featureFiles),
									})
								}
							}
						}

						if len(stackFeatureFiles) > 0 {
							merger.AddFiles(ctx, stackFeatureFiles, files.SourceFeature)
							tflog.Info(ctx, "Added files from stack features", map[string]interface{}{
								"stack_id":            stackID,
								"feature_count":       len(stackFeatureIDs),
								"stack_feature_files": len(stackFeatureFiles),
							})
						}
					}
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
	// Also include kits from features
	projectKitIDs := extractIDsFromDynamicList(ctx, data.Kits)
	featureKitIDs := r.collectFeatureKitIDs(ctx, data)
	allKitIDs := append(projectKitIDs, featureKitIDs...)

	// Deduplicate kit IDs
	seenKits := make(map[string]bool)
	uniqueKitIDs := []string{}
	for _, id := range allKitIDs {
		if !seenKits[id] {
			uniqueKitIDs = append(uniqueKitIDs, id)
			seenKits[id] = true
		}
	}

	if reg != nil && len(uniqueKitIDs) > 0 {
		projectKitFiles := r.getKitFilesFromRegistry(ctx, uniqueKitIDs, reg)
		merger.AddFiles(ctx, projectKitFiles, files.SourceProject)
		tflog.Info(ctx, "Added files from project and feature kits", map[string]interface{}{
			"project_kit_count": len(projectKitIDs),
			"feature_kit_count": len(featureKitIDs),
			"total_kit_count":   len(uniqueKitIDs),
			"kit_file_count":    len(projectKitFiles),
		})
	}

	// 4. NEW: Add feature files (higher precedence than kits)
	featureFiles := r.collectFeatureFiles(ctx, data)
	if len(featureFiles) > 0 {
		merger.AddFiles(ctx, featureFiles, files.SourceFeature)
		tflog.Info(ctx, "Added files from features", map[string]interface{}{
			"feature_file_count": len(featureFiles),
		})
	}

	// 5. Finally, add project's own files (highest precedence)
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

// collectStringFields collects all string fields from the project for URI scanning
// This includes: description, requirements, files, features, etc.
func (r *ProjectResourceFinal) collectStringFields(ctx context.Context, data ProjectModelFinal) map[string]string {
	fields := make(map[string]string)

	// Project-level description
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		fields["description"] = data.Description.ValueString()
	}

	// Requirements - instructions and constraints
	for i, req := range data.Requirements {
		for j, instr := range req.Instructions {
			if !instr.Prompt.IsNull() && !instr.Prompt.IsUnknown() {
				fields[fmt.Sprintf("req_%d_instr_%d_prompt", i, j)] = instr.Prompt.ValueString()
			}
			for k, constraint := range instr.Constraints {
				if !constraint.IsNull() && !constraint.IsUnknown() {
					fields[fmt.Sprintf("req_%d_instr_%d_constraint_%d", i, j, k)] = constraint.ValueString()
				}
			}
		}
	}

	// Files - content and instructions
	if !data.Files.IsNull() && !data.Files.IsUnknown() {
		for path, fileAttr := range data.Files.Elements() {
			fileObj, ok := fileAttr.(types.Object)
			if !ok {
				continue
			}

			// Extract file model
			var fileModel schemas.FileModel
			diags := fileObj.As(ctx, &fileModel, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				continue
			}

			// File content
			if !fileModel.Content.IsNull() && !fileModel.Content.IsUnknown() {
				fields[fmt.Sprintf("file_%s_content", path)] = fileModel.Content.ValueString()
			}

			// File instructions
			for i, instr := range fileModel.Instructions {
				if !instr.Prompt.IsNull() && !instr.Prompt.IsUnknown() {
					fields[fmt.Sprintf("file_%s_instr_%d_prompt", path, i)] = instr.Prompt.ValueString()
				}
			}
		}
	}

	// Features - simplified for now, just the prompt
	if !data.Features.IsNull() && !data.Features.IsUnknown() {
		featuresUnderlyingVal := data.Features.UnderlyingValue()
		if featuresMap, ok := featuresUnderlyingVal.(types.Map); ok {
			for featureName, featureAttr := range featuresMap.Elements() {
				featureObj, ok := featureAttr.(types.Object)
				if !ok {
					continue
				}

				// Extract feature model
				var featureModel schemas.FeatureModel
				diags := featureObj.As(ctx, &featureModel, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					continue
				}

				// Feature requirements - instructions and constraints
				for reqIdx, req := range featureModel.Requirements {
					for instrIdx, instr := range req.Instructions {
						// Instruction prompt
						if !instr.Prompt.IsNull() && !instr.Prompt.IsUnknown() {
							fields[fmt.Sprintf("feature_%s_req_%d_instr_%d_prompt", featureName, reqIdx, instrIdx)] = instr.Prompt.ValueString()
						}
						// Instruction constraints
						for constraintIdx, constraint := range instr.Constraints {
							if !constraint.IsNull() && !constraint.IsUnknown() {
								fields[fmt.Sprintf("feature_%s_req_%d_instr_%d_constraint_%d", featureName, reqIdx, instrIdx, constraintIdx)] = constraint.ValueString()
							}
						}
					}
				}
			}
		}
	}

	tflog.Debug(ctx, "Collected string fields for URI scanning", map[string]interface{}{
		"field_count": len(fields),
	})

	return fields
}

// computeAndStoreFileHashes calculates and stores hashes for all files
// Called after successful Claude execution to establish baseline
func (r *ProjectResourceFinal) computeAndStoreFileHashes(
	ctx context.Context,
	data *ProjectModelFinal,
	mergedFiles []schemas.FileModelWithPath,
) error {
	projectPath := data.ProjectPath.ValueString()

	tflog.Debug(ctx, "=== computeAndStoreFileHashes STARTED ===", map[string]interface{}{
		"project_path":     projectPath,
		"merged_files":     len(mergedFiles),
		"files_is_null":    data.Files.IsNull(),
		"files_is_unknown": data.Files.IsUnknown(),
	})

	// If Files is null or unknown, nothing to update
	if data.Files.IsNull() || data.Files.IsUnknown() {
		tflog.Debug(ctx, "=== computeAndStoreFileHashes: Files is null/unknown, skipping ===", nil)
		return nil
	}

	// Extract current files map as attr.Value elements (not Go structs)
	filesElements := data.Files.Elements()
	newFilesMap := make(map[string]attr.Value)

	tflog.Debug(ctx, "=== computeAndStoreFileHashes: Processing files ===", map[string]interface{}{
		"files_elements_count": len(filesElements),
	})

	// Build a map of merged files for quick lookup
	mergedFilesMap := make(map[string]schemas.FileModelWithPath)
	for _, f := range mergedFiles {
		mergedFilesMap[f.Path] = f
	}

	// Process each file in the current Files map
	for path, fileAttr := range filesElements {
		fileObj, ok := fileAttr.(types.Object)
		if !ok {
			// Not an object, keep as-is
			newFilesMap[path] = fileAttr
			continue
		}

		// Get file attributes
		attrs := fileObj.Attributes()

		// Check if this file was merged and exists on disk
		if mergedFile, exists := mergedFilesMap[path]; exists {
			fullPath := filepath.Join(projectPath, path)

			// Read actual file content
			actualContent, err := os.ReadFile(fullPath)
			if err != nil {
				if os.IsNotExist(err) {
					// File wasn't created - set hash fields to null
					tflog.Debug(ctx, "File not found, setting hash fields to null", map[string]interface{}{
						"path": path,
					})
					attrs["content_hash"] = types.StringNull()
					attrs["file_hash"] = types.StringNull()
					attrs["file_modtime"] = types.StringNull()
				} else {
					return fmt.Errorf("failed to read file %s: %w", path, err)
				}
			} else {
				// Compute hashes using the merged file model (for content_hash)
				fileModel := schemas.FileModel{
					Content:       mergedFile.Content,
					Instructions:  mergedFile.Instructions,
					Verifications: mergedFile.Verifications,
				}
				contentHash := files.ComputeContentHash(fileModel)
				fileHash := files.ComputeFileHash(actualContent)

				// Store modification time
				fileInfo, err := os.Stat(fullPath)
				modTime := ""
				if err == nil {
					modTime = fileInfo.ModTime().Format(time.RFC3339)
				}

				tflog.Debug(ctx, "Computed file hashes", map[string]interface{}{
					"path":         path,
					"content_hash": contentHash[:8] + "...",
					"file_hash":    fileHash[:8] + "...",
				})

				tflog.Debug(ctx, "=== computeAndStoreFileHashes: Setting hashes for file ===", map[string]interface{}{
					"path":         path,
					"content_hash": contentHash[:8],
					"file_hash":    fileHash[:8],
					"modtime":      modTime,
				})

				// Update hash fields in attributes
				attrs["content_hash"] = types.StringValue(contentHash)
				attrs["file_hash"] = types.StringValue(fileHash)
				attrs["file_modtime"] = types.StringValue(modTime)
			}
		} else {
			// File not in merged list, set hash fields to null
			attrs["content_hash"] = types.StringNull()
			attrs["file_hash"] = types.StringNull()
			attrs["file_modtime"] = types.StringNull()
		}

		// Create new object with updated attributes
		newFileObj, diags := types.ObjectValue(fileObj.AttributeTypes(ctx), attrs)
		if diags.HasError() {
			return fmt.Errorf("failed to create file object for %s: %s", path, diags.Errors())
		}
		newFilesMap[path] = newFileObj
	}

	// Build FileModel AttrTypes based on the schema
	fileModelAttrTypes := map[string]attr.Type{
		"content":       types.StringType,
		"instructions":  types.ListType{ElemType: types.ObjectType{AttrTypes: schemas.InstructionModelType()}},
		"verifications": types.ListType{ElemType: types.ObjectType{AttrTypes: schemas.VerificationModelType()}},
		"content_hash":  types.StringType,
		"file_hash":     types.StringType,
		"file_modtime":  types.StringType,
		// Resource metadata attributes (optional, present when referencing tofukit_file)
		"id":          types.StringType,
		"name":        types.StringType,
		"link":        types.StringType,
		"description": types.StringType,
	}

	// Create new map
	newMap, diags := types.MapValue(types.ObjectType{AttrTypes: fileModelAttrTypes}, newFilesMap)
	if diags.HasError() {
		return fmt.Errorf("failed to convert files map: %s", diags.Errors())
	}

	tflog.Debug(ctx, "=== computeAndStoreFileHashes: Creating new Files map ===", map[string]interface{}{
		"new_map_elements": len(newFilesMap),
		"has_diags":        diags.HasError(),
	})

	data.Files = newMap

	tflog.Debug(ctx, "=== computeAndStoreFileHashes COMPLETED ===", map[string]interface{}{
		"files_elements": len(data.Files.Elements()),
		"files_is_null":  data.Files.IsNull(),
	})

	return nil
}

// addDriftInstructions adds drift warning instructions to files that changed outside Terraform
func (r *ProjectResourceFinal) addDriftInstructions(
	ctx context.Context,
	files []schemas.FileModelWithPath,
	driftedPaths []string,
) []schemas.FileModelWithPath {
	tflog.Debug(ctx, "=== addDriftInstructions STARTED ===", map[string]interface{}{
		"files_count":   len(files),
		"drifted_paths": driftedPaths,
		"drifted_count": len(driftedPaths),
	})

	// Build drift map for O(1) lookup
	driftMap := make(map[string]bool)
	for _, path := range driftedPaths {
		driftMap[path] = true
	}

	// Add drift warnings to affected files
	modifiedCount := 0
	for i, file := range files {
		tflog.Debug(ctx, "=== addDriftInstructions: Checking file ===", map[string]interface{}{
			"file_path":      file.Path,
			"is_drifted":     driftMap[file.Path],
			"existing_instr": len(file.Instructions),
		})

		if driftMap[file.Path] {
			driftInstruction := schemas.InstructionModel{
				Prompt: types.StringValue(fmt.Sprintf(
					"⚠️ DRIFT DETECTED: File '%s' was modified outside Terraform. "+
						"Restore it to match the specification below. "+
						"Ignore any manual edits that were made.",
					file.Path,
				)),
				Constraints: []types.String{
					types.StringValue("Restore file to match Terraform specification exactly"),
					types.StringValue("Ignore manual edits made outside Terraform"),
					types.StringValue("Ensure the restored file matches the original intent"),
				},
			}

			tflog.Debug(ctx, "=== addDriftInstructions: Adding drift instruction ===", map[string]interface{}{
				"path":         file.Path,
				"drift_prompt": driftInstruction.Prompt.ValueString(),
			})

			// Prepend drift instruction (highest priority)
			files[i].Instructions = append(
				[]schemas.InstructionModel{driftInstruction},
				file.Instructions...,
			)
			modifiedCount++

			tflog.Debug(ctx, "=== addDriftInstructions: After adding instruction ===", map[string]interface{}{
				"path":            file.Path,
				"new_instr_count": len(files[i].Instructions),
			})
		}
	}

	tflog.Debug(ctx, "=== addDriftInstructions COMPLETED ===", map[string]interface{}{
		"files_modified": modifiedCount,
	})

	return files
}

// keysOfMap returns the keys of a map[string]interface{} as a slice
func keysOfMap(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
