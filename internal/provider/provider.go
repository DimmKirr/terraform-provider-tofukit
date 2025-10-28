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
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/gemini"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/openai"
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
	LLM                        types.String `tfsdk:"llm"`
	APIKey                     types.String `tfsdk:"api_key"`
	ClaudeHomeDirectory        types.String `tfsdk:"claude_home_directory"`
	Debug                      types.Bool   `tfsdk:"debug"`
	MaxRetries                 types.Int64  `tfsdk:"max_retries"`
	DangerouslySkipPermissions types.Bool   `tfsdk:"dangerously_skip_permissions"`
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
			"llm": schema.StringAttribute{
				MarkdownDescription: "LLM provider to use (claude, openai, gemini). Default: claude",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for LLM provider (required for openai and gemini, not used for claude)",
				Optional:            true,
				Sensitive:           true,
			},
			"claude_home_directory": schema.StringAttribute{
				MarkdownDescription: "Claude home directory for authentication and configuration (default: ~/.claude, only used when llm=claude)",
				Optional:            true,
			},
			"debug": schema.BoolAttribute{
				MarkdownDescription: "Enable debug mode to output LLM debug information",
				Optional:            true,
			},
			"max_retries": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of verification retry attempts (default: 3). If verification fails, Claude will receive the errors and retry until success or max retries.",
				Optional:            true,
			},
			"dangerously_skip_permissions": schema.BoolAttribute{
				MarkdownDescription: "Skip Claude CLI permission prompts (default: true). When enabled, Claude can create/modify files without prompting. When combined with --add-dir, Claude's access is still restricted to the output directory.",
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
	outputPath := "./"  // Default to current directory
	llmType := "claude" // Default LLM provider
	apiKey := ""
	claudeHomeDir := "~/.claude" // Default Claude home directory
	debug := false
	maxRetries := 3                    // Default to 3 verification retry attempts
	dangerouslySkipPermissions := true // Default to true for backward compatibility

	if !data.OutputFormat.IsNull() {
		outputFormat = data.OutputFormat.ValueString()
	}

	if !data.OutputPath.IsNull() {
		outputPath = data.OutputPath.ValueString()
	}

	if !data.LLM.IsNull() {
		llmType = data.LLM.ValueString()
	}

	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
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
			maxRetries = 1 // Minimum 1 retry
		}
	}

	if !data.DangerouslySkipPermissions.IsNull() {
		dangerouslySkipPermissions = data.DangerouslySkipPermissions.ValueBool()
	}

	// Create LLM executor based on type
	var llmExecutor llm.LLMExecutor
	switch llmType {
	case "claude":
		// Validate that unbuffer is available (required for Claude CLI)
		if err := validateUnbufferAvailable(); err != nil {
			resp.Diagnostics.AddError(
				"Missing Required Command",
				fmt.Sprintf("unbuffer command is required for Claude CLI usage: %v\n\nPlease install expect package:\n  - Ubuntu/Debian: apt-get install expect\n  - macOS: brew install expect\n  - RHEL/CentOS: yum install expect", err),
			)
			return
		}
		llmExecutor = newClaudeAdapter(claude.NewExecutor(claudeHomeDir, dangerouslySkipPermissions))
	case "openai":
		if apiKey == "" {
			resp.Diagnostics.AddError(
				"Missing API Key",
				"api_key is required when llm=openai",
			)
			return
		}
		llmExecutor = openai.NewExecutor(apiKey)
	case "gemini":
		if apiKey == "" {
			resp.Diagnostics.AddError(
				"Missing API Key",
				"api_key is required when llm=gemini",
			)
			return
		}
		llmExecutor = gemini.NewExecutor(apiKey)
	default:
		resp.Diagnostics.AddError(
			"Unsupported LLM Type",
			fmt.Sprintf("Unsupported LLM type: %s. Supported types are: claude, openai, gemini", llmType),
		)
		return
	}

	// Configure the executor
	llmExecutor.SetDebug(debug)
	llmExecutor.SetOutputPath(outputPath)

	// Create provider data that will be passed to resources
	providerData := &ProviderData{
		OutputFormat:               outputFormat,
		OutputPath:                 outputPath,
		LLM:                        llmType,
		ClaudeHomeDirectory:        claudeHomeDir,
		Debug:                      debug,
		MaxRetries:                 maxRetries,
		DangerouslySkipPermissions: dangerouslySkipPermissions,
		Registry:                   registry.New(),
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
		resources.NewBlueprintResource,
		resources.NewLanguageResource,
		resources.NewFrameworkResource,
		resources.NewToolResource,
		resources.NewMethodologyResource,
		resources.NewStyleResource,
		resources.NewInfrastructureResource,
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
	LLM                        string
	ClaudeHomeDirectory        string
	Debug                      bool
	MaxRetries                 int
	DangerouslySkipPermissions bool
	Registry                   *registry.Registry
	LLMExecutor                llm.LLMExecutor
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

// GetDangerouslySkipPermissions returns whether to skip Claude CLI permission prompts
func (p *ProviderData) GetDangerouslySkipPermissions() bool {
	return p.DangerouslySkipPermissions
}

// GetLLMExecutor returns the LLM executor
func (p *ProviderData) GetLLMExecutor() llm.LLMExecutor {
	return p.LLMExecutor
}

// validateUnbufferAvailable checks if the unbuffer command is available in PATH
func validateUnbufferAvailable() error {
	_, err := exec.LookPath("unbuffer")
	if err != nil {
		return fmt.Errorf("unbuffer command not found in PATH")
	}
	return nil
}
