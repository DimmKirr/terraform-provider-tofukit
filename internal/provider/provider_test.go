package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveModel(t *testing.T) {
	tests := []struct {
		name                 string
		resourceModel        string
		providerDefaultModel string
		expectedModel        string
	}{
		{
			name:                 "Resource override takes precedence",
			resourceModel:        "openai/gpt-5.2",
			providerDefaultModel: "anthropic/claude-sonnet-4.5",
			expectedModel:        "openai/gpt-5.2",
		},
		{
			name:                 "Use provider default when resource model empty",
			resourceModel:        "",
			providerDefaultModel: "anthropic/claude-sonnet-4.5",
			expectedModel:        "anthropic/claude-sonnet-4.5",
		},
		{
			name:                 "Both empty returns empty",
			resourceModel:        "",
			providerDefaultModel: "",
			expectedModel:        "",
		},
		{
			name:                 "Resource model with same value as provider",
			resourceModel:        "anthropic/claude-sonnet-4.5",
			providerDefaultModel: "anthropic/claude-sonnet-4.5",
			expectedModel:        "anthropic/claude-sonnet-4.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveModel(tt.resourceModel, tt.providerDefaultModel)
			assert.Equal(t, tt.expectedModel, result)
		})
	}
}
