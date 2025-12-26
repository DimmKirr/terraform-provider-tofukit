package models

import (
	"testing"
)

func TestGetProviderModel(t *testing.T) {
	tests := []struct {
		name      string
		modelSlug string
		want      string
	}{
		{
			name:      "OpenAI GPT-5.2",
			modelSlug: "openai/gpt-5.2",
			want:      "gpt-5.2-2025-12-11",
		},
		{
			name:      "OpenAI GPT-5 Image",
			modelSlug: "openai/gpt-5-image",
			want:      "gpt-5-2025-08-07",
		},
		{
			name:      "OpenAI GPT-5 Image Mini",
			modelSlug: "openai/gpt-5-image-mini",
			want:      "gpt-5-mini-2025-08-07",
		},
		{
			name:      "Claude Sonnet 4.5",
			modelSlug: "anthropic/claude-sonnet-4.5",
			want:      "claude-sonnet-4-5@20250929",
		},
		{
			name:      "Claude Haiku",
			modelSlug: "anthropic/claude-haiku",
			want:      "claude-haiku-20250101",
		},
		{
			name:      "Claude Opus",
			modelSlug: "anthropic/claude-opus",
			want:      "claude-opus-20250115",
		},
		{
			name:      "Unknown model with provider prefix",
			modelSlug: "openai/unknown-model",
			want:      "unknown-model",
		},
		{
			name:      "Unknown model without provider prefix",
			modelSlug: "unknown-model",
			want:      "unknown-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetProviderModel(tt.modelSlug)
			if got != tt.want {
				t.Errorf("GetProviderModel(%q) = %q, want %q", tt.modelSlug, got, tt.want)
			}
		})
	}
}

func TestIsImageModel(t *testing.T) {
	tests := []struct {
		name      string
		modelSlug string
		want      bool
	}{
		{
			name:      "GPT-5.2 is not image model",
			modelSlug: "openai/gpt-5.2",
			want:      false,
		},
		{
			name:      "GPT-5 Image is image model",
			modelSlug: "openai/gpt-5-image",
			want:      true,
		},
		{
			name:      "GPT-5 Image Mini is image model",
			modelSlug: "openai/gpt-5-image-mini",
			want:      true,
		},
		{
			name:      "Claude Sonnet is not image model",
			modelSlug: "anthropic/claude-sonnet-4.5",
			want:      false,
		},
		{
			name:      "Claude Haiku is not image model",
			modelSlug: "anthropic/claude-haiku",
			want:      false,
		},
		{
			name:      "Claude Opus is not image model",
			modelSlug: "anthropic/claude-opus",
			want:      false,
		},
		{
			name:      "Unknown model with 'image' in name",
			modelSlug: "provider/some-image-model",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsImageModel(tt.modelSlug)
			if got != tt.want {
				t.Errorf("IsImageModel(%q) = %v, want %v", tt.modelSlug, got, tt.want)
			}
		})
	}
}

func TestGetModel(t *testing.T) {
	tests := []struct {
		name      string
		modelSlug string
		wantFound bool
		wantName  string
	}{
		{
			name:      "OpenAI GPT-5.2 exists",
			modelSlug: "openai/gpt-5.2",
			wantFound: true,
			wantName:  "OpenAI GPT-5.2",
		},
		{
			name:      "OpenAI GPT-5 Image exists",
			modelSlug: "openai/gpt-5-image",
			wantFound: true,
			wantName:  "OpenAI GPT-5 Image",
		},
		{
			name:      "Unknown model does not exist",
			modelSlug: "provider/unknown",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, found := GetModel(tt.modelSlug)
			if found != tt.wantFound {
				t.Errorf("GetModel(%q) found = %v, want %v", tt.modelSlug, found, tt.wantFound)
			}
			if found && model.Name != tt.wantName {
				t.Errorf("GetModel(%q).Name = %q, want %q", tt.modelSlug, model.Name, tt.wantName)
			}
		})
	}
}

func TestModelRegistryCompleteness(t *testing.T) {
	expectedModels := []string{
		"openai/gpt-5.2",
		"openai/gpt-5-image",
		"openai/gpt-5-image-mini",
		"anthropic/claude-sonnet-4.5",
		"anthropic/claude-haiku",
		"anthropic/claude-opus",
	}

	for _, slug := range expectedModels {
		t.Run(slug, func(t *testing.T) {
			model, exists := ModelRegistry[slug]
			if !exists {
				t.Errorf("ModelRegistry missing expected model: %q", slug)
			}
			if model.Slug != slug {
				t.Errorf("ModelRegistry[%q].Slug = %q, want %q", slug, model.Slug, slug)
			}
			if model.ProviderModelID == "" {
				t.Errorf("ModelRegistry[%q].ProviderModelID is empty", slug)
			}
			if model.Name == "" {
				t.Errorf("ModelRegistry[%q].Name is empty", slug)
			}
		})
	}

	if len(ModelRegistry) != len(expectedModels) {
		t.Errorf("ModelRegistry has %d models, want %d", len(ModelRegistry), len(expectedModels))
	}
}
