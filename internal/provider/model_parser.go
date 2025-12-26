package provider

import (
	"fmt"
	"strings"
)

// ModelInfo contains parsed model information
type ModelInfo struct {
	Provider string
	Model    string
	Original string
}

// ParseModel parses a model string in provider/model format
// Examples: "anthropic/claude-3-5-sonnet", "openai/dall-e-3"
func ParseModel(modelStr string) (*ModelInfo, error) {
	if modelStr == "" {
		return nil, fmt.Errorf("model string cannot be empty")
	}

	parts := strings.Split(modelStr, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid model format: %s (expected provider/model)", modelStr)
	}

	provider := strings.TrimSpace(parts[0])
	model := strings.TrimSpace(parts[1])

	if provider == "" {
		return nil, fmt.Errorf("provider cannot be empty in model: %s", modelStr)
	}

	if model == "" {
		return nil, fmt.Errorf("model cannot be empty in model: %s", modelStr)
	}

	return &ModelInfo{
		Provider: provider,
		Model:    model,
		Original: modelStr,
	}, nil
}

// SupportedProviders returns a list of supported provider names
func SupportedProviders() []string {
	return []string{"anthropic"}
}

// IsProviderSupported checks if a provider is currently supported
func IsProviderSupported(provider string) bool {
	for _, p := range SupportedProviders() {
		if p == provider {
			return true
		}
	}
	return false
}
