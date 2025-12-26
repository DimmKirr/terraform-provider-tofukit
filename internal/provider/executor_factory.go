package provider

import (
	"fmt"

	"github.com/tofukit/opentofu-provider-tofukit/internal/llm"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/models"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/openai"
)

// ExecutorFactory creates LLM executors based on provider/model selection
type ExecutorFactory struct {
	claudeHomeDir              string
	openaiAPIKey               string
	googleAPIKey               string
	metaAPIKey                 string
	dangerouslySkipPermissions bool
	claudeMaxTurns             int
	debug                      bool
	outputPath                 string
}

// NewExecutorFactory creates a new executor factory with configuration
func NewExecutorFactory(
	claudeHomeDir string,
	openaiAPIKey string,
	googleAPIKey string,
	metaAPIKey string,
	dangerouslySkipPermissions bool,
	claudeMaxTurns int,
	debug bool,
	outputPath string,
) *ExecutorFactory {
	return &ExecutorFactory{
		claudeHomeDir:              claudeHomeDir,
		openaiAPIKey:               openaiAPIKey,
		googleAPIKey:               googleAPIKey,
		metaAPIKey:                 metaAPIKey,
		dangerouslySkipPermissions: dangerouslySkipPermissions,
		claudeMaxTurns:             claudeMaxTurns,
		debug:                      debug,
		outputPath:                 outputPath,
	}
}

// SetDebug sets debug mode for all executors
func (f *ExecutorFactory) SetDebug(debug bool) {
	f.debug = debug
}

// SetOutputPath sets output path for all executors
func (f *ExecutorFactory) SetOutputPath(outputPath string) {
	f.outputPath = outputPath
}

// GetExecutor creates an executor for the specified provider and model
// The model parameter is the TofuKit model slug (e.g., "claude-sonnet-4.5", "gpt-5-image")
// The factory will use the models registry to get the provider model ID for SDK calls
func (f *ExecutorFactory) GetExecutor(provider string, model string) (llm.LLMExecutor, error) {
	switch provider {
	case "anthropic":
		return f.createClaudeExecutor(model)
	case "openai":
		return f.createOpenAIExecutor(model)
	case "google":
		return nil, fmt.Errorf("model 'google/%s' selected, but Google Gemini executor is not implemented yet. Supported providers: %v", model, SupportedProviders())
	case "meta-llama", "meta":
		return nil, fmt.Errorf("model 'meta-llama/%s' selected, but Meta LLaMA executor is not implemented yet. Supported providers: %v", model, SupportedProviders())
	default:
		return nil, fmt.Errorf("unsupported provider: %s. Supported providers: %v", provider, SupportedProviders())
	}
}

// createClaudeExecutor creates a Claude executor with the specified model
// This demonstrates the factory pattern using the models registry:
// 1. Takes TofuKit model slug (e.g., "claude-sonnet-4.5")
// 2. Uses models.GetProviderModel() to get SDK model ID (e.g., "claude-sonnet-4-5@20250929")
// 3. Creates executor with the SDK model ID
func (f *ExecutorFactory) createClaudeExecutor(modelSlug string) (llm.LLMExecutor, error) {
	// Get provider model ID from registry (e.g., "claude-sonnet-4.5" -> "claude-sonnet-4-5@20250929")
	providerModelID := models.GetProviderModel("anthropic/" + modelSlug)

	executor := claude.NewExecutor(f.claudeHomeDir, f.dangerouslySkipPermissions, f.claudeMaxTurns)

	// Set the provider model ID (not the TofuKit slug)
	executor.SetModel(providerModelID)

	// Apply common settings
	executor.SetDebug(f.debug)
	if f.outputPath != "" {
		executor.SetOutputPath(f.outputPath)
	}

	// Wrap in adapter to implement llm.LLMExecutor interface
	adapter := newClaudeAdapter(executor)

	return adapter, nil
}

// createOpenAIExecutor creates an OpenAI executor with the specified model
func (f *ExecutorFactory) createOpenAIExecutor(modelSlug string) (llm.LLMExecutor, error) {
	// Get full TofuKit model slug (e.g., "openai/gpt-5-image")
	fullModelSlug := "openai/" + modelSlug

	// Create OpenAI executor with API key and model info
	executor := openai.NewExecutor(f.openaiAPIKey, fullModelSlug)

	// The executor will use models.IsImageModel() internally to route to:
	// - Chat completion API for text models (gpt-5.2)
	// - Image generation API for image models (gpt-5-image, gpt-5-image-mini)

	// Apply common settings
	executor.SetDebug(f.debug)
	if f.outputPath != "" {
		executor.SetOutputPath(f.outputPath)
	}

	// Wrap in adapter to implement llm.LLMExecutor interface
	adapter := newOpenAIAdapter(executor)

	return adapter, nil
}

// GetExecutorForModel is a convenience method that parses model and creates executor
func (f *ExecutorFactory) GetExecutorForModel(modelStr string) (llm.LLMExecutor, error) {
	modelInfo, err := ParseModel(modelStr)
	if err != nil {
		return nil, err
	}

	return f.GetExecutor(modelInfo.Provider, modelInfo.Model)
}
