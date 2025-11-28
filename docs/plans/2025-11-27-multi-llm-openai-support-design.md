# Multi-LLM Support with OpenAI Integration - Design Document

**Date:** 2025-11-27
**Status:** Design
**Author:** Claude Code

## Overview

This design adds multi-LLM support to the TofuKit provider, enabling users to choose between Claude and OpenAI (ChatGPT) models on a per-resource or per-query basis. The architecture uses a model naming convention that encodes both the LLM provider and specific model (e.g., `claude-sonnet`, `openai-gpt-4o`), with runtime routing to the appropriate executor.

## Goals and Requirements

### Primary Goals
1. Add OpenAI/ChatGPT support using official `github.com/openai/openai-go` SDK
2. Enable per-resource and per-query model selection
3. Support multiple LLMs simultaneously in a single configuration
4. Validate LLM availability and API keys early (at provider startup)
5. Provide clear migration path from current single-LLM architecture

### Requirements
- **Model Naming Convention**: `{provider}-{api-model}` format (e.g., `claude-haiku`, `openai-gpt-4o`)
- **Provider Configuration**: Flat structure with LLM-specific fields (`claude_max_turns`, `openai_api_key`)
- **Environment Variable Support**: `OPENAI_API_KEY` env var overrides provider field
- **Early Validation**: Fail at provider startup if configuration is invalid
- **Full OpenAI Implementation**: Complete project generation support (not just Query)
- **No Backward Compatibility**: Breaking change with clear migration path
- **Keep Gemini Stub**: Preserve for future implementation

## Architecture

### Current Architecture (Before Changes)

```
Provider Config:
  llm = "claude" | "openai" | "gemini"  ← Single LLM selection
  api_key = "..."                        ← Generic API key

Provider.Configure():
  → Creates single LLMExecutor based on 'llm' field
  → All resources use same LLM

Query Data Source:
  model = "haiku" | "sonnet"             ← Claude-specific model names
```

**Limitations:**
- Can only use one LLM per provider instance
- Model names are Claude-specific
- No way to mix Claude and OpenAI in same config

### New Architecture (After Changes)

```
Provider Config:
  model = "claude-sonnet"                ← NEW: Global default model
  claude_home_directory = "~/.claude"
  claude_max_turns = 100
  openai_api_key = "..."                 ← NEW: LLM-specific config

Provider.Configure():
  → Creates multiple executors (Claude, OpenAI, Gemini)
  → Validates each configured LLM
  → Stores all in ProviderData

Resource/Query Execution:
  model = "openai-gpt-4o"                ← Parse prefix
  → Extract provider: "openai"
  → Extract API model: "gpt-4o"
  → Route to OpenAIExecutor
  → Execute with API model suffix
```

**Benefits:**
- Multiple LLMs in same config
- Clear model naming convention
- Early validation of all LLMs
- Flexible per-resource model selection

## Provider Configuration Changes

### Schema Modifications

**File:** `internal/provider/provider.go`

#### New Fields

```go
type TofukitProviderModel struct {
    // NEW: Global default model
    Model types.String `tfsdk:"model"`

    // NEW: OpenAI-specific configuration
    OpenAIAPIKey types.String `tfsdk:"openai_api_key"`

    // EXISTING: Claude-specific (unchanged)
    ClaudeHomeDirectory        types.String `tfsdk:"claude_home_directory"`
    ClaudeMaxTurns             types.Int64  `tfsdk:"claude_max_turns"`
    DangerouslySkipPermissions types.Bool   `tfsdk:"dangerously_skip_permissions"`

    // EXISTING: General configs (unchanged)
    OutputFormat types.String `tfsdk:"output_format"`
    OutputPath   types.String `tfsdk:"output_path"`
    Debug        types.Bool   `tfsdk:"debug"`
    MaxRetries   types.Int64  `tfsdk:"max_retries"`
    DryRun       types.Bool   `tfsdk:"dry_run"`
}
```

#### Removed Fields

```go
// REMOVED: No longer needed with model prefix parsing
LLM    types.String `tfsdk:"llm"`

// REMOVED: Replaced with LLM-specific fields
APIKey types.String `tfsdk:"api_key"`
```

#### Schema Definition

```go
func (p *TofukitProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "model": schema.StringAttribute{
                MarkdownDescription: "Default model for all resources and queries. Format: {provider}-{model}. Examples: claude-sonnet, claude-haiku, openai-gpt-4o, openai-o1. Default: claude-sonnet",
                Optional:            true,
            },
            "openai_api_key": schema.StringAttribute{
                MarkdownDescription: "OpenAI API key (can also be set via OPENAI_API_KEY environment variable, which takes precedence). Required for using openai-* models.",
                Optional:            true,
                Sensitive:           true,
            },
            // ... existing fields
        },
    }
}
```

### Example Configuration

```hcl
provider "tofukit" {
  # Global default model (NEW)
  model = "claude-sonnet"

  # Claude-specific configs
  claude_home_directory = "~/.claude"
  claude_max_turns = 100
  dangerously_skip_permissions = true

  # OpenAI-specific configs (NEW)
  openai_api_key = "sk-..."  # or use OPENAI_API_KEY env var

  # General configs
  debug = true
  max_retries = 3
  output_path = "./output"
}

# Query with Claude
data "tofukit_query" "claude_query" {
  model = "claude-haiku"  # Override provider default
  instruction { prompt = "..." }
}

# Query with OpenAI
data "tofukit_query" "openai_query" {
  model = "openai-gpt-4o"  # Override provider default
  instruction { prompt = "..." }
}

# Project uses provider default (claude-sonnet)
resource "tofukit_project" "app" {
  name = "my-app"
  # Uses model = "claude-sonnet" from provider
}
```

## Executor Creation and Routing

### ProviderData Structure

**File:** `internal/provider/provider.go`

```go
type ProviderData struct {
    // Existing fields
    OutputFormat string
    OutputPath   string
    Debug        bool
    MaxRetries   int
    DryRun       bool
    Registry     *registry.Registry

    // NEW: Multiple executors
    DefaultModel   string            // e.g., "claude-sonnet"
    ClaudeExecutor llm.LLMExecutor   // nil if Claude not available
    OpenAIExecutor llm.LLMExecutor   // nil if OpenAI not configured
    GeminiExecutor llm.LLMExecutor   // nil if Gemini not configured (stub)

    // Claude-specific (moved from general)
    ClaudeHomeDirectory        string
    ClaudeMaxTurns             int
    DangerouslySkipPermissions bool
}
```

### Configure() Method Logic

**File:** `internal/provider/provider.go`

```go
func (p *TofukitProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
    var data TofukitProviderModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 1. Read configuration values
    defaultModel := "claude-sonnet"  // Default
    if !data.Model.IsNull() {
        defaultModel = data.Model.ValueString()
    }

    claudeHomeDir := "~/.claude"
    if !data.ClaudeHomeDirectory.IsNull() {
        claudeHomeDir = data.ClaudeHomeDirectory.ValueString()
    }

    claudeMaxTurns := 100
    if !data.ClaudeMaxTurns.IsNull() {
        claudeMaxTurns = int(data.ClaudeMaxTurns.ValueInt64())
    }

    // 2. Get OpenAI API key (env var overrides provider field)
    openaiKey := os.Getenv("OPENAI_API_KEY")
    if openaiKey == "" && !data.OpenAIAPIKey.IsNull() {
        openaiKey = data.OpenAIAPIKey.ValueString()
    }

    // 3. Create executors based on available configuration
    var claudeExecutor llm.LLMExecutor
    var openaiExecutor llm.LLMExecutor
    var geminiExecutor llm.LLMExecutor  // Keep stub

    // Claude: Always attempt to create (CLI might be available)
    if err := validateUnbufferAvailable(); err == nil {
        claudeExecutor = newClaudeAdapter(claude.NewExecutor(
            claudeHomeDir,
            dangerouslySkipPermissions,
            claudeMaxTurns,
        ))
        if err := claudeExecutor.Validate(ctx); err != nil {
            tflog.Warn(ctx, "Claude CLI not available", map[string]interface{}{
                "error": err.Error(),
            })
            claudeExecutor = nil
        }
    }

    // OpenAI: Create if API key exists
    if openaiKey != "" {
        openaiExecutor = openai.NewExecutor(openaiKey)
        if err := openaiExecutor.Validate(ctx); err != nil {
            resp.Diagnostics.AddError(
                "OpenAI Configuration Invalid",
                fmt.Sprintf("OpenAI API key is invalid or OpenAI service is unavailable: %v", err),
            )
            return  // Fail if explicitly configured but invalid
        }
        tflog.Info(ctx, "OpenAI executor created successfully")
    }

    // Gemini: Keep stub (not implemented yet)
    // geminiExecutor remains nil

    // 4. Validate default model has available executor
    if err := validateModelHasExecutor(defaultModel, claudeExecutor, openaiExecutor, geminiExecutor); err != nil {
        resp.Diagnostics.AddError(
            "Invalid Default Model",
            fmt.Sprintf("Default model '%s' cannot be used: %v", defaultModel, err),
        )
        return
    }

    // 5. Configure all executors
    if claudeExecutor != nil {
        claudeExecutor.SetDebug(debug)
        claudeExecutor.SetOutputPath(outputPath)
    }
    if openaiExecutor != nil {
        openaiExecutor.SetDebug(debug)
        openaiExecutor.SetOutputPath(outputPath)
    }

    // 6. Create and store ProviderData
    providerData := &ProviderData{
        OutputFormat:               outputFormat,
        OutputPath:                 outputPath,
        Debug:                      debug,
        MaxRetries:                 maxRetries,
        DryRun:                     dryRun,
        DefaultModel:               defaultModel,
        ClaudeExecutor:             claudeExecutor,
        OpenAIExecutor:             openaiExecutor,
        GeminiExecutor:             geminiExecutor,
        Registry:                   registry.New(),
        ClaudeHomeDirectory:        claudeHomeDir,
        ClaudeMaxTurns:             claudeMaxTurns,
        DangerouslySkipPermissions: dangerouslySkipPermissions,
    }

    resp.DataSourceData = providerData
    resp.ResourceData = providerData
}

// Helper function to validate model has executor
func validateModelHasExecutor(model string, claude, openai, gemini llm.LLMExecutor) error {
    provider, _, err := ParseModelPrefix(model)
    if err != nil {
        return err
    }

    switch provider {
    case "claude":
        if claude == nil {
            return fmt.Errorf("Claude CLI not available (check /usr/bin/claude installation)")
        }
    case "openai":
        if openai == nil {
            return fmt.Errorf("OpenAI not configured (set openai_api_key or OPENAI_API_KEY)")
        }
    case "gemini":
        return fmt.Errorf("Gemini not yet implemented")
    default:
        return fmt.Errorf("unknown provider '%s' (supported: claude, openai)", provider)
    }

    return nil
}
```

### Validation Strategy

| Configuration | Behavior |
|--------------|----------|
| Claude CLI not found | Warn, set `ClaudeExecutor = nil` (allow OpenAI-only usage) |
| OpenAI key explicitly set but invalid | **Error immediately** (fail provider initialization) |
| OpenAI key empty/missing | OK, set `OpenAIExecutor = nil` (allow Claude-only usage) |
| Default model requires missing executor | **Error immediately** (e.g., `model = "openai-gpt-4o"` but no API key) |
| Both executors nil | OK if no resources/queries created (edge case) |

## Model Parsing and Routing

### New Helper Functions

**File:** `internal/provider/model_router.go` (new file)

```go
package provider

import (
    "fmt"
    "strings"

    "github.com/tofukit/opentofu-provider-tofukit/internal/llm"
)

// ParseModelPrefix extracts LLM provider and API model from model string
// Examples:
//   "claude-haiku" → "claude", "haiku", nil
//   "openai-gpt-4o" → "openai", "gpt-4o", nil
//   "openai-gpt-4-turbo" → "openai", "gpt-4-turbo", nil (handles multi-dash)
//   "invalid" → "", "", error
func ParseModelPrefix(model string) (provider string, apiModel string, err error) {
    if model == "" {
        return "", "", fmt.Errorf("model cannot be empty")
    }

    parts := strings.SplitN(model, "-", 2)
    if len(parts) != 2 {
        return "", "", fmt.Errorf("invalid model format: '%s' (expected format: {provider}-{model}, e.g., claude-sonnet, openai-gpt-4o)", model)
    }

    provider = parts[0]
    apiModel = parts[1]

    // Validate provider is known
    validProviders := []string{"claude", "openai", "gemini"}
    isValid := false
    for _, vp := range validProviders {
        if provider == vp {
            isValid = true
            break
        }
    }

    if !isValid {
        return "", "", fmt.Errorf("unknown provider '%s' in model '%s' (supported providers: %s)",
            provider, model, strings.Join(validProviders, ", "))
    }

    return provider, apiModel, nil
}

// GetExecutorForModel returns the appropriate executor for a model string
// and the API-specific model suffix to pass to that executor.
//
// Examples:
//   "claude-haiku" → ClaudeExecutor, "haiku", nil
//   "openai-gpt-4o" → OpenAIExecutor, "gpt-4o", nil
//
// Returns error if:
//   - Model format is invalid
//   - Provider is unknown
//   - Required executor is not available (not configured)
func GetExecutorForModel(model string, pd *ProviderData) (llm.LLMExecutor, string, error) {
    provider, apiModel, err := ParseModelPrefix(model)
    if err != nil {
        return nil, "", err
    }

    switch provider {
    case "claude":
        if pd.ClaudeExecutor == nil {
            return nil, "", fmt.Errorf("Claude not available (check that /usr/bin/claude CLI is installed and accessible)")
        }
        return pd.ClaudeExecutor, apiModel, nil

    case "openai":
        if pd.OpenAIExecutor == nil {
            return nil, "", fmt.Errorf("OpenAI not configured (set openai_api_key in provider block or OPENAI_API_KEY environment variable)")
        }
        return pd.OpenAIExecutor, apiModel, nil

    case "gemini":
        if pd.GeminiExecutor == nil {
            return nil, "", fmt.Errorf("Gemini not yet implemented (coming soon)")
        }
        return pd.GeminiExecutor, apiModel, nil

    default:
        // Should never reach here due to ParseModelPrefix validation
        return nil, "", fmt.Errorf("unknown model provider: %s", provider)
    }
}
```

### Usage in Resources and Data Sources

**Example: Query Data Source** (`internal/datasources/query.go`)

```go
func (d *QueryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var data QueryDataSourceModel
    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    providerData := d.getProviderData(req.ProviderData)

    // Determine model to use (query override or provider default)
    model := providerData.DefaultModel
    if !data.Model.IsNull() {
        model = data.Model.ValueString()
    }

    // Route to appropriate executor
    executor, apiModel, err := GetExecutorForModel(model, providerData)
    if err != nil {
        resp.Diagnostics.AddError(
            "Model Configuration Error",
            fmt.Sprintf("Cannot use model '%s': %v", model, err),
        )
        return
    }

    tflog.Info(ctx, "Executing query", map[string]interface{}{
        "model":     model,
        "api_model": apiModel,
    })

    // Execute with API-specific model suffix
    result, err := executor.Query(ctx, queryInstructions, apiModel)
    if err != nil {
        resp.Diagnostics.AddError("Query Failed", err.Error())
        return
    }

    // ... process result
}
```

**Example: Project Resource** (`internal/resources/project.go`)

```go
func (r *ProjectResource) executeClaudeCode(ctx context.Context, ...) error {
    providerData := r.getProviderData()

    // Project always uses provider default model
    model := providerData.DefaultModel

    // Route to appropriate executor
    executor, apiModel, err := GetExecutorForModel(model, providerData)
    if err != nil {
        return fmt.Errorf("model configuration error: %v", err)
    }

    tflog.Info(ctx, "Executing project generation", map[string]interface{}{
        "model":     model,
        "api_model": apiModel,
        "project":   projectName,
    })

    // Execute project generation
    status, err := executor.Execute(ctx, projectSpec, outputDir)
    // ... handle result
}
```

## OpenAI Implementation

### Executor Structure

**File:** `internal/llm/openai/executor.go`

Replace the existing stub with full implementation:

```go
package openai

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/hashicorp/terraform-plugin-log/tflog"
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
    "github.com/tofukit/opentofu-provider-tofukit/internal/llm"
)

type Executor struct {
    client     *openai.Client
    apiKey     string
    debug      bool
    outputPath string
    systemPrompt string
}

func NewExecutor(apiKey string) *Executor {
    return &Executor{
        client: openai.NewClient(option.WithAPIKey(apiKey)),
        apiKey: apiKey,
        debug:  false,
    }
}

func (e *Executor) SetDebug(debug bool) {
    e.debug = debug
}

func (e *Executor) SetOutputPath(outputPath string) {
    e.outputPath = outputPath
}

func (e *Executor) SetSystemPrompt(systemPrompt string) {
    e.systemPrompt = systemPrompt
}

func (e *Executor) Validate(ctx context.Context) error {
    if e.apiKey == "" {
        return fmt.Errorf("API key is empty")
    }

    // Test API key with minimal request (list models)
    _, err := e.client.Models.Get(ctx, "gpt-4o")
    if err != nil {
        return fmt.Errorf("API key validation failed: %v", err)
    }

    return nil
}
```

### Query Implementation

```go
func (e *Executor) Query(ctx context.Context, instructions []string, model string) (string, error) {
    tflog.Info(ctx, "Executing OpenAI query", map[string]interface{}{
        "model":             model,
        "instruction_count": len(instructions),
    })

    prompt := strings.Join(instructions, "\n")

    messages := []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage(prompt),
    }

    // Add system prompt if configured
    if e.systemPrompt != "" {
        messages = append([]openai.ChatCompletionMessageParamUnion{
            openai.SystemMessage(e.systemPrompt),
        }, messages...)
    }

    resp, err := e.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model:    openai.F(model),
        Messages: openai.F(messages),
    })
    if err != nil {
        return "", fmt.Errorf("OpenAI API error: %v", err)
    }

    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("no response from OpenAI")
    }

    tflog.Info(ctx, "OpenAI query completed", map[string]interface{}{
        "model":          model,
        "response_chars": len(resp.Choices[0].Message.Content),
        "finish_reason":  resp.Choices[0].FinishReason,
    })

    return resp.Choices[0].Message.Content, nil
}
```

### Execute Implementation (Project Generation)

**Strategy:** OpenAI doesn't have an interactive CLI like Claude Code, so we need to implement the full file generation workflow ourselves.

**Approach:**
1. Build structured prompt from project spec (similar to Claude)
2. Send to OpenAI API with clear instructions about file structure
3. Parse response into file operations
4. Create/modify files on filesystem
5. Run verifications and retry if needed

```go
func (e *Executor) Execute(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*llm.ExecutionStatus, error) {
    tflog.Info(ctx, "Starting OpenAI project execution", map[string]interface{}{
        "output_dir": outputDir,
    })

    status := &llm.ExecutionStatus{
        State:       "running",
        StartedAt:   time.Now().Format(time.RFC3339),
        ProjectPath: outputDir,
        Metadata:    make(map[string]string),
    }

    // Extract project metadata
    if project, ok := projectSpec["project"].(map[string]interface{}); ok {
        if name, ok := project["name"].(string); ok {
            status.Metadata["project_name"] = name
        }
    }

    // Ensure output directory exists
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        status.State = "failed"
        status.Error = fmt.Sprintf("Failed to create output directory: %v", err)
        return status, err
    }

    // Build prompt from project spec
    prompt, err := e.buildProjectPrompt(projectSpec)
    if err != nil {
        status.State = "failed"
        status.Error = fmt.Sprintf("Failed to build prompt: %v", err)
        return status, err
    }

    // Save prompt for debugging
    if e.debug {
        debugDir := filepath.Join(outputDir, ".debug")
        os.MkdirAll(debugDir, 0755)
        promptPath := filepath.Join(debugDir, "openai-prompt.json")
        os.WriteFile(promptPath, []byte(prompt), 0644)
    }

    // Execute with OpenAI (default model: gpt-4o)
    model := "gpt-4o"
    if modelOverride, ok := projectSpec["_model"].(string); ok {
        model = modelOverride
    }

    response, err := e.executePrompt(ctx, prompt, model)
    if err != nil {
        status.State = "failed"
        status.Error = fmt.Sprintf("OpenAI execution failed: %v", err)
        return status, err
    }

    // Parse response and create files
    if err := e.processResponse(ctx, response, outputDir, projectSpec); err != nil {
        status.State = "failed"
        status.Error = fmt.Sprintf("Failed to process response: %v", err)
        return status, err
    }

    status.State = "completed"
    status.CompletedAt = time.Now().Format(time.RFC3339)
    status.Output = response

    tflog.Info(ctx, "OpenAI project execution completed")
    return status, nil
}
```

### Helper Functions for Project Generation

```go
// buildProjectPrompt converts project spec to OpenAI prompt
func (e *Executor) buildProjectPrompt(projectSpec map[string]interface{}) (string, error) {
    // Use same prompt structure as Claude
    // Format: JSON with clear instructions about file generation

    promptData := map[string]interface{}{
        "instruction": "You are a code generation assistant. Generate files exactly as specified in the project specification below. Output a JSON array of file operations.",
        "project":     projectSpec,
        "output_format": map[string]string{
            "type":        "json",
            "description": "Array of {action, path, content} objects",
        },
    }

    promptJSON, err := json.MarshalIndent(promptData, "", "  ")
    if err != nil {
        return "", err
    }

    return string(promptJSON), nil
}

// executePrompt sends prompt to OpenAI API
func (e *Executor) executePrompt(ctx context.Context, prompt string, model string) (string, error) {
    messages := []openai.ChatCompletionMessageParamUnion{
        openai.UserMessage(prompt),
    }

    if e.systemPrompt != "" {
        messages = append([]openai.ChatCompletionMessageParamUnion{
            openai.SystemMessage(e.systemPrompt),
        }, messages...)
    }

    resp, err := e.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model:    openai.F(model),
        Messages: openai.F(messages),
        ResponseFormat: openai.F(openai.ChatCompletionNewParamsResponseFormatJSONObject{
            Type: openai.F(openai.ChatCompletionNewParamsResponseFormatTypeJSONObject),
        }),
    })
    if err != nil {
        return "", err
    }

    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("no response from OpenAI")
    }

    return resp.Choices[0].Message.Content, nil
}

// processResponse parses OpenAI response and creates files
func (e *Executor) processResponse(ctx context.Context, response string, outputDir string, projectSpec map[string]interface{}) error {
    // Parse JSON response into file operations
    var fileOps []struct {
        Action  string `json:"action"`  // "create", "modify", "delete"
        Path    string `json:"path"`
        Content string `json:"content"`
    }

    if err := json.Unmarshal([]byte(response), &fileOps); err != nil {
        return fmt.Errorf("failed to parse OpenAI response: %v", err)
    }

    // Execute file operations
    for _, op := range fileOps {
        filePath := filepath.Join(outputDir, op.Path)

        switch op.Action {
        case "create", "modify":
            // Ensure parent directory exists
            if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
                return fmt.Errorf("failed to create directory for %s: %v", op.Path, err)
            }

            // Write file
            if err := os.WriteFile(filePath, []byte(op.Content), 0644); err != nil {
                return fmt.Errorf("failed to write file %s: %v", op.Path, err)
            }

            tflog.Debug(ctx, "File operation completed", map[string]interface{}{
                "action": op.Action,
                "path":   op.Path,
            })

        case "delete":
            if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
                return fmt.Errorf("failed to delete file %s: %v", op.Path, err)
            }

        default:
            tflog.Warn(ctx, "Unknown file action", map[string]interface{}{
                "action": op.Action,
                "path":   op.Path,
            })
        }
    }

    return nil
}
```

### Retry and Verification Support

```go
func (e *Executor) RetryExecution(ctx context.Context, projectSpec map[string]interface{}, outputDir string, maxRetries int) (*llm.ExecutionStatus, error) {
    var lastErr error
    var status *llm.ExecutionStatus

    for attempt := 0; attempt <= maxRetries; attempt++ {
        if attempt > 0 {
            tflog.Info(ctx, "Retrying OpenAI execution", map[string]interface{}{
                "attempt":     attempt,
                "max_retries": maxRetries,
            })
        }

        status, lastErr = e.Execute(ctx, projectSpec, outputDir)
        if lastErr == nil && status.State == "completed" {
            return status, nil
        }

        if attempt < maxRetries {
            time.Sleep(time.Second * time.Duration(attempt+1)) // Exponential backoff
        }
    }

    return status, fmt.Errorf("execution failed after %d retries: %v", maxRetries, lastErr)
}
```

### Additional Methods

```go
func (e *Executor) IsProjectGenerated(ctx context.Context, projectPath string) bool {
    // Check if output directory has any files
    entries, err := os.ReadDir(projectPath)
    if err != nil {
        return false
    }

    // Consider generated if directory has files (excluding .debug)
    for _, entry := range entries {
        if entry.Name() != ".debug" {
            return true
        }
    }

    return false
}

func (e *Executor) CleanupProject(ctx context.Context, projectPath string) error {
    return os.RemoveAll(projectPath)
}
```

## Migration Path

### Overview

This is a **breaking change** with no backward compatibility. All existing configurations must be migrated to use the new model naming convention.

### Migration Commands

Users can automate most of the migration using sed commands:

```bash
#!/bin/bash
# migrate-to-multi-llm.sh - Automated migration script

echo "Migrating TofuKit configurations to multi-LLM format..."

# 1. Update provider block: llm field → model field
echo "1. Updating provider block..."
find . -name "*.tofu" -type f -exec sed -i.bak 's/llm = "claude"/model = "claude-sonnet"/g' {} \;
find . -name "*.tofu" -type f -exec sed -i.bak 's/llm = "openai"/model = "openai-gpt-4o"/g' {} \;
find . -name "*.tofu" -type f -exec sed -i.bak 's/llm = "gemini"/model = "gemini-pro"/g' {} \;

# 2. Rename api_key to openai_api_key (manual review recommended)
echo "2. Renaming api_key field (review changes before applying)..."
# Uncomment to execute:
# find . -name "*.tofu" -type f -exec sed -i.bak 's/api_key = /openai_api_key = /g' {} \;

# 3. Update model references in query data sources
echo "3. Updating query model references..."
find . -name "*.tofu" -type f -exec sed -i.bak 's/model = "haiku"/model = "claude-haiku"/g' {} \;
find . -name "*.tofu" -type f -exec sed -i.bak 's/model = "sonnet"/model = "claude-sonnet"/g' {} \;
find . -name "*.tofu" -type f -exec sed -i.bak 's/model = "opus"/model = "claude-opus"/g' {} \;
find . -name "*.tofu" -type f -exec sed -i.bak 's/model = "gpt-4"/model = "openai-gpt-4"/g' {} \;
find . -name "*.tofu" -type f -exec sed -i.bak 's/model = "gpt-4o"/model = "openai-gpt-4o"/g' {} \;

echo "Migration complete! Review changes and run 'terraform validate'"
echo "Backup files saved with .bak extension"
```

### Manual Steps

After running the automated migration:

1. **Review `api_key` changes**: The `api_key` field was renamed to `openai_api_key`. Review these changes manually to ensure they refer to OpenAI (not other services).

2. **Add default model**: If not already present, add `model = "claude-sonnet"` to provider block.

3. **Remove backup files**: After verifying changes, remove `.bak` files:
   ```bash
   find . -name "*.tofu.bak" -delete
   ```

4. **Validate configuration**:
   ```bash
   terraform init
   terraform validate
   ```

5. **Test with plan**:
   ```bash
   terraform plan
   ```

### Example: Before and After

**Before:**
```hcl
provider "tofukit" {
  llm = "claude"
  api_key = "sk-..."  # Generic API key
  claude_home_directory = "~/.claude"
}

data "tofukit_query" "test" {
  model = "haiku"  # Claude-specific name
  instruction { prompt = "..." }
}
```

**After:**
```hcl
provider "tofukit" {
  model = "claude-sonnet"  # NEW: Explicit default
  openai_api_key = "sk-..."  # LLM-specific
  claude_home_directory = "~/.claude"
}

data "tofukit_query" "test" {
  model = "claude-haiku"  # NEW: Prefixed format
  instruction { prompt = "..." }
}
```

### Breaking Changes Summary

| Old Field/Value | New Field/Value | Migration |
|----------------|-----------------|-----------|
| `llm = "claude"` | `model = "claude-sonnet"` | Automated |
| `llm = "openai"` | `model = "openai-gpt-4o"` | Automated |
| `api_key = "..."` | `openai_api_key = "..."` | **Manual review required** |
| `model = "haiku"` | `model = "claude-haiku"` | Automated |
| `model = "sonnet"` | `model = "claude-sonnet"` | Automated |
| `model = "opus"` | `model = "claude-opus"` | Automated |

## Testing Strategy

### Test Organization

**New Test Files:**
- `test/provider_llm_claude_test.go` - Claude-specific integration tests
- `test/provider_llm_openai_test.go` - OpenAI-specific integration tests
- `test/provider_model_routing_test.go` - Unit tests for model parsing/routing

**New Test Configs:**
- `test/testdata/configs/llm-claude-hello-world.tofu` - Simple Claude project (1 file)
- `test/testdata/configs/llm-openai-hello-world.tofu` - Simple OpenAI project (1 file)
- `test/testdata/configs/llm-query-claude.tofu` - Claude query test
- `test/testdata/configs/llm-query-openai.tofu` - OpenAI query test
- `test/testdata/configs/llm-mixed-models.tofu` - Multiple queries with different models

### Unit Tests

**File:** `test/provider_model_routing_test.go`

```go
func TestProviderModelRouting_ParsePrefixValidFormatsSuccess(t *testing.T) {
    tests := []struct {
        name         string
        input        string
        wantProvider string
        wantModel    string
    }{
        {"Claude Haiku", "claude-haiku", "claude", "haiku"},
        {"Claude Sonnet", "claude-sonnet", "claude", "sonnet"},
        {"Claude Opus", "claude-opus", "claude", "opus"},
        {"OpenAI GPT-4o", "openai-gpt-4o", "openai", "gpt-4o"},
        {"OpenAI GPT-4 Turbo", "openai-gpt-4-turbo", "openai", "gpt-4-turbo"},
        {"OpenAI O1", "openai-o1", "openai", "o1"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            provider, model, err := ParseModelPrefix(tt.input)
            require.NoError(t, err)
            assert.Equal(t, tt.wantProvider, provider)
            assert.Equal(t, tt.wantModel, model)
        })
    }
}

func TestProviderModelRouting_ParsePrefixInvalidFormatsError(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"No dash", "invalid"},
        {"Empty string", ""},
        {"Only prefix", "claude"},
        {"Only suffix", "-haiku"},
        {"Unknown provider", "unknown-model"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, _, err := ParseModelPrefix(tt.input)
            assert.Error(t, err)
        })
    }
}

func TestProviderModelRouting_GetExecutorClaudeSuccess(t *testing.T) {
    // Create mock ProviderData with Claude executor
    pd := &ProviderData{
        ClaudeExecutor: &mockExecutor{name: "claude"},
        OpenAIExecutor: nil,
    }

    executor, apiModel, err := GetExecutorForModel("claude-sonnet", pd)
    require.NoError(t, err)
    assert.NotNil(t, executor)
    assert.Equal(t, "sonnet", apiModel)
}

func TestProviderModelRouting_GetExecutorMissingExecutorError(t *testing.T) {
    // ProviderData with no OpenAI executor
    pd := &ProviderData{
        ClaudeExecutor: &mockExecutor{name: "claude"},
        OpenAIExecutor: nil,
    }

    _, _, err := GetExecutorForModel("openai-gpt-4o", pd)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "OpenAI not configured")
}
```

### Claude Integration Tests

**File:** `test/provider_llm_claude_test.go`

```go
func TestProviderLLMClaude_ProjectCreateSingleFileSuccess(t *testing.T) {
    testDir := createTestDirectory(t, "TestProviderLLMClaude_ProjectCreateSingleFile")

    // Use embedded config
    configPath := filepath.Join(testDir, "project.tofu")
    err := os.WriteFile(configPath, []byte(ConfigLLMClaudeHelloWorld), 0644)
    require.NoError(t, err)

    // Initialize and apply
    runTerraformInit(t, testDir)
    runTerraformApply(t, testDir)

    // Verify file created
    outputPath := filepath.Join(testDir, "output", "hello.txt")
    assert.FileExists(t, outputPath)

    content, err := os.ReadFile(outputPath)
    require.NoError(t, err)
    assert.Contains(t, string(content), "Hello from Claude")
}

func TestProviderLLMClaude_QuerySimpleSuccess(t *testing.T) {
    testDir := createTestDirectory(t, "TestProviderLLMClaude_QuerySimple")

    configPath := filepath.Join(testDir, "query.tofu")
    err := os.WriteFile(configPath, []byte(ConfigLLMQueryClaude), 0644)
    require.NoError(t, err)

    runTerraformInit(t, testDir)
    output := runTerraformApply(t, testDir)

    // Verify query result in output
    assert.Contains(t, output, "result")
}
```

### OpenAI Integration Tests

**File:** `test/provider_llm_openai_test.go`

```go
func TestProviderLLMOpenAI_ProjectCreateSingleFileSuccess(t *testing.T) {
    // Require OPENAI_API_KEY in environment
    if os.Getenv("OPENAI_API_KEY") == "" {
        t.Skip("OPENAI_API_KEY not set, skipping OpenAI test")
    }

    testDir := createTestDirectory(t, "TestProviderLLMOpenAI_ProjectCreateSingleFile")

    configPath := filepath.Join(testDir, "project.tofu")
    err := os.WriteFile(configPath, []byte(ConfigLLMOpenAIHelloWorld), 0644)
    require.NoError(t, err)

    runTerraformInit(t, testDir)
    runTerraformApply(t, testDir)

    // Verify file created
    outputPath := filepath.Join(testDir, "output", "hello.txt")
    assert.FileExists(t, outputPath)

    content, err := os.ReadFile(outputPath)
    require.NoError(t, err)
    assert.Contains(t, string(content), "Hello from OpenAI")
}

func TestProviderLLMOpenAI_QuerySimpleSuccess(t *testing.T) {
    if os.Getenv("OPENAI_API_KEY") == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    testDir := createTestDirectory(t, "TestProviderLLMOpenAI_QuerySimple")

    configPath := filepath.Join(testDir, "query.tofu")
    err := os.WriteFile(configPath, []byte(ConfigLLMQueryOpenAI), 0644)
    require.NoError(t, err)

    runTerraformInit(t, testDir)
    output := runTerraformApply(t, testDir)

    assert.Contains(t, output, "result")
}

func TestProviderLLMOpenAI_MissingAPIKeyError(t *testing.T) {
    // Temporarily unset env var
    oldKey := os.Getenv("OPENAI_API_KEY")
    os.Unsetenv("OPENAI_API_KEY")
    defer os.Setenv("OPENAI_API_KEY", oldKey)

    testDir := createTestDirectory(t, "TestProviderLLMOpenAI_MissingAPIKey")

    // Config without openai_api_key
    config := `
provider "tofukit" {
  model = "openai-gpt-4o"
}
`
    configPath := filepath.Join(testDir, "main.tofu")
    err := os.WriteFile(configPath, []byte(config), 0644)
    require.NoError(t, err)

    runTerraformInit(t, testDir)

    // Should fail with clear error
    _, err = runTerraformPlanExpectError(t, testDir)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "OpenAI not configured")
}
```

### Mixed Model Tests

**File:** `test/provider_llm_claude_test.go` or separate file

```go
func TestProviderLLMMixed_QueryMultipleProvidersSuccess(t *testing.T) {
    if os.Getenv("OPENAI_API_KEY") == "" {
        t.Skip("OPENAI_API_KEY not set")
    }

    testDir := createTestDirectory(t, "TestProviderLLMMixed_QueryMultipleProviders")

    configPath := filepath.Join(testDir, "queries.tofu")
    err := os.WriteFile(configPath, []byte(ConfigLLMMixedModels), 0644)
    require.NoError(t, err)

    runTerraformInit(t, testDir)
    output := runTerraformApply(t, testDir)

    // Verify both queries executed successfully
    assert.Contains(t, output, "claude_result")
    assert.Contains(t, output, "openai_result")
}
```

### Test Configuration Files

**File:** `test/helpers_test.go` (add embedded configs)

```go
//go:embed testdata/configs/llm-claude-hello-world.tofu
var ConfigLLMClaudeHelloWorld string

//go:embed testdata/configs/llm-openai-hello-world.tofu
var ConfigLLMOpenAIHelloWorld string

//go:embed testdata/configs/llm-query-claude.tofu
var ConfigLLMQueryClaude string

//go:embed testdata/configs/llm-query-openai.tofu
var ConfigLLMQueryOpenAI string

//go:embed testdata/configs/llm-mixed-models.tofu
var ConfigLLMMixedModels string
```

**File:** `test/testdata/configs/llm-claude-hello-world.tofu`

```hcl
provider "tofukit" {
  model = "claude-sonnet"
  output_path = "${path.module}/output"
  debug = true
  dangerously_skip_permissions = true
}

resource "tofukit_project" "hello_claude" {
  name    = "hello-claude"
  version = "0.1.0"

  files = {
    "hello.txt" = {
      content = "Hello from Claude!\n"
    }
  }
}
```

**File:** `test/testdata/configs/llm-openai-hello-world.tofu`

```hcl
provider "tofukit" {
  model = "openai-gpt-4o"
  output_path = "${path.module}/output"
  debug = true
  # openai_api_key read from OPENAI_API_KEY env var
}

resource "tofukit_project" "hello_openai" {
  name    = "hello-openai"
  version = "0.1.0"

  files = {
    "hello.txt" = {
      content = "Hello from OpenAI!\n"
    }
  }
}
```

**File:** `test/testdata/configs/llm-mixed-models.tofu`

```hcl
provider "tofukit" {
  model = "claude-sonnet"
  debug = true
}

data "tofukit_query" "claude_query" {
  model = "claude-haiku"

  instruction {
    prompt = "Reply with exactly: 'Claude responded'"
  }
}

data "tofukit_query" "openai_query" {
  model = "openai-gpt-4o"

  instruction {
    prompt = "Reply with exactly: 'OpenAI responded'"
  }
}

output "claude_result" {
  value = data.tofukit_query.claude_query.text
}

output "openai_result" {
  value = data.tofukit_query.openai_query.text
}
```

### CI/CD Integration

**GitHub Actions** (example):

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install dependencies
        run: |
          # Install unbuffer for Claude tests
          sudo apt-get install -y expect

      - name: Run unit tests
        run: go test ./test -v -run TestProviderModelRouting

      - name: Run Claude tests
        run: go test ./test -v -run TestProviderLLMClaude
        env:
          CLAUDE_HOME: ${{ secrets.CLAUDE_HOME }}

      - name: Run OpenAI tests
        run: go test ./test -v -run TestProviderLLMOpenAI
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
```

### Test Coverage Goals

- ✅ Model parsing (valid/invalid formats) - **Unit tests**
- ✅ Executor routing (available/missing executors) - **Unit tests**
- ✅ Provider validation (missing configs, invalid keys) - **Integration tests**
- ✅ Claude project creation - **Integration test**
- ✅ OpenAI project creation - **Integration test**
- ✅ Claude query - **Integration test**
- ✅ OpenAI query - **Integration test**
- ✅ Mixed model usage - **Integration test**
- ✅ Error messages (helpful, actionable) - **All tests**

## Implementation Checklist

### Phase 1: Provider Configuration (2-3 hours)
- [ ] Update `TofukitProviderModel` schema
  - [ ] Add `model` field
  - [ ] Add `openai_api_key` field
  - [ ] Remove `llm` field
  - [ ] Remove `api_key` field
- [ ] Update `ProviderData` structure
  - [ ] Add `DefaultModel` field
  - [ ] Add `ClaudeExecutor` field
  - [ ] Add `OpenAIExecutor` field
  - [ ] Add `GeminiExecutor` field (stub)
- [ ] Implement `Configure()` method
  - [ ] Read new config fields
  - [ ] Handle `OPENAI_API_KEY` env var (override)
  - [ ] Create executors based on config
  - [ ] Validate each executor
  - [ ] Validate default model
- [ ] Update helper functions
  - [ ] `validateModelHasExecutor()`

### Phase 2: Model Routing (1-2 hours)
- [ ] Create `internal/provider/model_router.go`
  - [ ] Implement `ParseModelPrefix()`
  - [ ] Implement `GetExecutorForModel()`
  - [ ] Add validation logic
  - [ ] Add error messages

### Phase 3: OpenAI Implementation (6-8 hours)
- [ ] Update `internal/llm/openai/executor.go`
  - [ ] Add `github.com/openai/openai-go` dependency
  - [ ] Implement `NewExecutor()`
  - [ ] Implement `Validate()`
  - [ ] Implement `Query()`
  - [ ] Implement `Execute()` (project generation)
    - [ ] `buildProjectPrompt()`
    - [ ] `executePrompt()`
    - [ ] `processResponse()`
  - [ ] Implement `RetryExecution()`
  - [ ] Implement `IsProjectGenerated()`
  - [ ] Implement `CleanupProject()`
  - [ ] Add debug output support

### Phase 4: Resource Updates (2-3 hours)
- [ ] Update query data source (`internal/datasources/query.go`)
  - [ ] Use `GetExecutorForModel()` for routing
  - [ ] Pass API model suffix to executor
  - [ ] Update error handling
- [ ] Update project resource (`internal/resources/project.go`)
  - [ ] Use provider default model
  - [ ] Use `GetExecutorForModel()` for routing
  - [ ] Pass API model suffix to executor
- [ ] Update other resources as needed

### Phase 5: Testing (4-6 hours)
- [ ] Create test configs
  - [ ] `llm-claude-hello-world.tofu`
  - [ ] `llm-openai-hello-world.tofu`
  - [ ] `llm-query-claude.tofu`
  - [ ] `llm-query-openai.tofu`
  - [ ] `llm-mixed-models.tofu`
- [ ] Create unit tests (`provider_model_routing_test.go`)
  - [ ] `TestProviderModelRouting_ParsePrefixValidFormatsSuccess`
  - [ ] `TestProviderModelRouting_ParsePrefixInvalidFormatsError`
  - [ ] `TestProviderModelRouting_GetExecutorClaudeSuccess`
  - [ ] `TestProviderModelRouting_GetExecutorOpenAISuccess`
  - [ ] `TestProviderModelRouting_GetExecutorMissingExecutorError`
- [ ] Create Claude tests (`provider_llm_claude_test.go`)
  - [ ] `TestProviderLLMClaude_ProjectCreateSingleFileSuccess`
  - [ ] `TestProviderLLMClaude_QuerySimpleSuccess`
- [ ] Create OpenAI tests (`provider_llm_openai_test.go`)
  - [ ] `TestProviderLLMOpenAI_ProjectCreateSingleFileSuccess`
  - [ ] `TestProviderLLMOpenAI_QuerySimpleSuccess`
  - [ ] `TestProviderLLMOpenAI_MissingAPIKeyError`
- [ ] Create mixed model tests
  - [ ] `TestProviderLLMMixed_QueryMultipleProvidersSuccess`

### Phase 6: Documentation (1-2 hours)
- [ ] Update CLAUDE.md
  - [ ] Multi-LLM support section
  - [ ] Model naming convention
  - [ ] Configuration examples
- [ ] Create migration guide
  - [ ] Document breaking changes
  - [ ] Provide sed commands
  - [ ] Manual steps
- [ ] Update examples
  - [ ] Migrate example configs to new format

### Phase 7: Migration (1 hour)
- [ ] Create migration script (`migrate-to-multi-llm.sh`)
- [ ] Test migration script on examples
- [ ] Update all example configs
- [ ] Validate all examples

### Total Estimated Time: 17-25 hours

## Future Enhancements

### Short-term (Next Phase)
1. **Gemini Implementation**: Complete the gemini executor stub
2. **Per-Resource Model Override**: Add optional `model` field to project/file resources
3. **Model Aliases**: Support short aliases (`haiku` → `claude-haiku`) via provider config

### Long-term
1. **Custom LLM Support**: Plugin system for third-party LLM integrations
2. **Model Fallback**: Automatic fallback if primary model unavailable
3. **Cost Tracking**: Track and report LLM API costs per resource
4. **Streaming Support**: Real-time output for long-running generations

## References

- OpenAI Go SDK: https://github.com/openai/openai-go
- OpenAI API Models: https://platform.openai.com/docs/models
- Terraform Provider Framework: https://developer.hashicorp.com/terraform/plugin/framework
- TofuKit Current Architecture: See `CLAUDE.md`
