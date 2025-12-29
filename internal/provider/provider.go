package provider

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tofukit/opentofu-provider-tofukit/internal/datasources"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm"
	"github.com/tofukit/opentofu-provider-tofukit/internal/registry"
	"github.com/tofukit/opentofu-provider-tofukit/internal/resources"
)

// Ensure TofukitProvider satisfies various provider interfaces.
var _ provider.Provider = &TofukitProvider{}

// TofukitProvider defines the provider implementation.
type TofukitProvider struct {
	// version is the provider version, set during build time.
	version string
}

// TofukitProviderModel describes the provider data model.
type TofukitProviderModel struct {
	OutputFormat               types.String `tfsdk:"output_format"`
	OutputPath                 types.String `tfsdk:"output_path"`
	Model                      types.String `tfsdk:"model"`
	OpenAIAPIKey               types.String `tfsdk:"openai_api_key"`
	GoogleAPIKey               types.String `tfsdk:"google_api_key"`
	MetaAPIKey                 types.String `tfsdk:"meta_api_key"`
	ClaudeHomeDirectory        types.String `tfsdk:"claude_home_directory"`
	Debug                      types.Bool   `tfsdk:"debug"`
	MaxRetries                 types.Int64  `tfsdk:"max_retries"`
	ClaudeMaxTurns             types.Int64  `tfsdk:"claude_max_turns"`
	DangerouslySkipPermissions types.Bool   `tfsdk:"dangerously_skip_permissions"`
	DryRun                     types.Bool   `tfsdk:"dry_run"`
}

func (p *TofukitProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tofukit"
	resp.Version = p.version
}

func (p *TofukitProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"output_format": schema.StringAttribute{
				MarkdownDescription: "Output format for generated contexts (json, yaml, markdown)",
				Optional:            true,
			},
			"output_path": schema.StringAttribute{
				MarkdownDescription: "Path where to write generated context files",
				Optional:            true,
			},
			"model": schema.StringAttribute{
				MarkdownDescription: "Model to use in provider/model format (e.g., anthropic/claude-sonnet-4.5, openai/gpt-5.2). Default: anthropic/claude-sonnet-4.5",
				Optional:            true,
			},
			"openai_api_key": schema.StringAttribute{
				MarkdownDescription: "OpenAI API key for OpenAI models (e.g., openai/gpt-4o, openai/dall-e-3)",
				Optional:            true,
				Sensitive:           true,
			},
			"google_api_key": schema.StringAttribute{
				MarkdownDescription: "Google API key for Gemini models (e.g., google/gemini-1.5-pro)",
				Optional:            true,
				Sensitive:           true,
			},
			"meta_api_key": schema.StringAttribute{
				MarkdownDescription: "Meta API key for LLaMA models (e.g., meta-llama/llama-3.1-70b)",
				Optional:            true,
				Sensitive:           true,
			},
			"claude_home_directory": schema.StringAttribute{
				MarkdownDescription: "Claude home directory for authentication and configuration (default: ~/.claude, only used for anthropic models)",
				Optional:            true,
			},
			"debug": schema.BoolAttribute{
				MarkdownDescription: "Enable debug mode to output LLM debug information",
				Optional:            true,
			},
			"max_retries": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of verification retry attempts (default: 3). If verification fails, the LLM will receive the errors and retry until success or max retries.",
				Optional:            true,
			},
			"claude_max_turns": schema.Int64Attribute{
				MarkdownDescription: "Maximum turns for Claude CLI execution (default: 100). Only applies to anthropic models. Higher values allow more complex projects but take longer. A 'turn' is one user message + Claude's response(s).",
				Optional:            true,
			},
			"dangerously_skip_permissions": schema.BoolAttribute{
				MarkdownDescription: "Skip Claude CLI permission prompts (default: true). When enabled, Claude can create/modify files without prompting. When combined with --add-dir, Claude's access is still restricted to the output directory.",
				Optional:            true,
			},
			"dry_run": schema.BoolAttribute{
				MarkdownDescription: "Generate prompt JSON and write debug files but skip LLM execution (default: false). Useful for testing prompt generation without waiting for LLM responses.",
				Optional:            true,
			},
		},
	}
}

func (p *TofukitProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data TofukitProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values
	outputFormat := "json"
	outputPath := "./"
	modelStr := "anthropic/claude-sonnet-4.5"
	claudeHomeDir := "~/.claude"
	debug := false
	maxRetries := 3
	dangerouslySkipPermissions := true
	dryRun := false

	if !data.OutputFormat.IsNull() {
		outputFormat = data.OutputFormat.ValueString()
	}

	if !data.OutputPath.IsNull() {
		outputPath = data.OutputPath.ValueString()
	}

	if !data.Model.IsNull() {
		modelStr = data.Model.ValueString()
	}

	// Parse model to get provider and model name
	modelInfo, err := ParseModel(modelStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Model Format",
			fmt.Sprintf("Failed to parse model '%s': %v. Expected format: provider/model (e.g., anthropic/claude-sonnet-4.5)", modelStr, err),
		)
		return
	}

	// Extract API keys
	var openaiAPIKey, googleAPIKey, metaAPIKey string

	if !data.OpenAIAPIKey.IsNull() {
		openaiAPIKey = data.OpenAIAPIKey.ValueString()
	}

	if !data.GoogleAPIKey.IsNull() {
		googleAPIKey = data.GoogleAPIKey.ValueString()
	}

	if !data.MetaAPIKey.IsNull() {
		metaAPIKey = data.MetaAPIKey.ValueString()
	}

	if !data.ClaudeHomeDirectory.IsNull() {
		claudeHomeDir = data.ClaudeHomeDirectory.ValueString()
	}

	if !data.Debug.IsNull() {
		debug = data.Debug.ValueBool()
	}

	if !data.MaxRetries.IsNull() {
		maxRetries = int(data.MaxRetries.ValueInt64())
		if maxRetries < 1 {
			maxRetries = 1
		}
	}

	claudeMaxTurns := 100
	if !data.ClaudeMaxTurns.IsNull() {
		claudeMaxTurns = int(data.ClaudeMaxTurns.ValueInt64())
		if claudeMaxTurns < 1 {
			claudeMaxTurns = 1
		}
	}

	if !data.DangerouslySkipPermissions.IsNull() {
		dangerouslySkipPermissions = data.DangerouslySkipPermissions.ValueBool()
	}

	if !data.DryRun.IsNull() {
		dryRun = data.DryRun.ValueBool()
	}

	// Validate unbuffer for Claude/Anthropic provider
	if modelInfo.Provider == "anthropic" {
		if err := validateUnbufferAvailable(); err != nil {
			resp.Diagnostics.AddError(
				"Missing Required Command",
				fmt.Sprintf("unbuffer command is required for Claude CLI usage: %v\n\nPlease install expect package:\n  - Ubuntu/Debian: apt-get install expect\n  - macOS: brew install expect\n  - RHEL/CentOS: yum install expect", err),
			)
			return
		}
	}

	// Create executor factory with debug and outputPath from provider config
	factory := NewExecutorFactory(
		claudeHomeDir,
		openaiAPIKey,
		googleAPIKey,
		metaAPIKey,
		dangerouslySkipPermissions,
		claudeMaxTurns,
		debug,
		outputPath,
	)

	// Get executor for the default model
	llmExecutor, err := factory.GetExecutor(modelInfo.Provider, modelInfo.Model)
	if err != nil {
		resp.Diagnostics.AddError(
			"Executor Creation Failed",
			fmt.Sprintf("Failed to create executor for model '%s': %v", modelStr, err),
		)
		return
	}

	// Create provider data that will be passed to resources
	providerData := &ProviderData{
		OutputFormat:               outputFormat,
		OutputPath:                 outputPath,
		Model:                      modelStr,
		ClaudeHomeDirectory:        claudeHomeDir,
		Debug:                      debug,
		MaxRetries:                 maxRetries,
		ClaudeMaxTurns:             claudeMaxTurns,
		DangerouslySkipPermissions: dangerouslySkipPermissions,
		DryRun:                     dryRun,
		Registry:                   registry.New(),
		ExecutorFactory:            factory,
		LLMExecutor:                llmExecutor,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *TofukitProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewProjectResourceFinal,
		resources.NewStackResource,
		resources.NewFileResource,
		resources.NewFeatureResource,
		resources.NewLanguageResource,
		resources.NewLibraryResource,
		resources.NewToolResource,
		resources.NewIntegrationResource,
	}
}

func (p *TofukitProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewQueryDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TofukitProvider{
			version: version,
		}
	}
}

// ProviderData contains data that is passed to all resources
type ProviderData struct {
	OutputFormat               string
	OutputPath                 string
	Model                      string // Model in provider/model format
	ClaudeHomeDirectory        string
	Debug                      bool
	MaxRetries                 int
	ClaudeMaxTurns             int
	DangerouslySkipPermissions bool
	DryRun                     bool
	Registry                   *registry.Registry
	ExecutorFactory            *ExecutorFactory
	LLMExecutor                llm.LLMExecutor // Default executor for provider model
}

// GetClaudeHomeDirectory returns the Claude home directory
func (p *ProviderData) GetClaudeHomeDirectory() string {
	return p.ClaudeHomeDirectory
}

// IsDebug returns whether debug mode is enabled
func (p *ProviderData) IsDebug() bool {
	return p.Debug
}

// GetOutputPath returns the output path
func (p *ProviderData) GetOutputPath() string {
	return p.OutputPath
}

// GetOutputFormat returns the output format
func (p *ProviderData) GetOutputFormat() string {
	return p.OutputFormat
}

// GetRegistry returns the registry
func (p *ProviderData) GetRegistry() *registry.Registry {
	return p.Registry
}

// GetDebug returns the debug setting (for backward compatibility)
func (p *ProviderData) GetDebug() bool {
	return p.Debug
}

// GetMaxRetries returns the maximum number of verification retry attempts
func (p *ProviderData) GetMaxRetries() int {
	return p.MaxRetries
}

// GetClaudeMaxTurns returns the maximum turns for Claude CLI execution
func (p *ProviderData) GetClaudeMaxTurns() int {
	return p.ClaudeMaxTurns
}

// GetDangerouslySkipPermissions returns whether to skip Claude CLI permission prompts
func (p *ProviderData) GetDangerouslySkipPermissions() bool {
	return p.DangerouslySkipPermissions
}

// GetLLMExecutor returns the LLM executor
func (p *ProviderData) GetLLMExecutor() llm.LLMExecutor {
	return p.LLMExecutor
}

// GetDryRun returns whether dry run mode is enabled
func (p *ProviderData) GetDryRun() bool {
	return p.DryRun
}

// GetModel returns the default model in provider/model format
func (p *ProviderData) GetModel() string {
	return p.Model
}

// GetExecutorFactory returns the executor factory
func (p *ProviderData) GetExecutorFactory() *ExecutorFactory {
	return p.ExecutorFactory
}

// GetExecutorForModel creates an executor for the specified model
func (p *ProviderData) GetExecutorForModel(model string) (llm.LLMExecutor, error) {
	return p.ExecutorFactory.GetExecutorForModel(model)
}

// ResolveModel determines which model to use for a resource
// Priority: resource model > provider default model
func ResolveModel(resourceModel string, providerDefaultModel string) string {
	if resourceModel != "" {
		return resourceModel
	}
	return providerDefaultModel
}

// validateUnbufferAvailable checks if the unbuffer command is available in PATH
func validateUnbufferAvailable() error {
	_, err := exec.LookPath("unbuffer")
	if err != nil {
		return fmt.Errorf("unbuffer command not found in PATH")
	}
	return nil
}
