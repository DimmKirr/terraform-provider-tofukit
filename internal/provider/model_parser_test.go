package provider

import (
	"testing"
)

func TestParseModel(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantErr      bool
		wantProvider string
		wantModel    string
	}{
		{
			name:         "valid anthropic model",
			input:        "anthropic/claude-3-5-sonnet",
			wantErr:      false,
			wantProvider: "anthropic",
			wantModel:    "claude-3-5-sonnet",
		},
		{
			name:         "valid openai model",
			input:        "openai/gpt-4o",
			wantErr:      false,
			wantProvider: "openai",
			wantModel:    "gpt-4o",
		},
		{
			name:         "valid google model",
			input:        "google/gemini-1.5-pro",
			wantErr:      false,
			wantProvider: "google",
			wantModel:    "gemini-1.5-pro",
		},
		{
			name:         "valid meta model",
			input:        "meta-llama/llama-3.1-70b",
			wantErr:      false,
			wantProvider: "meta-llama",
			wantModel:    "llama-3.1-70b",
		},
		{
			name:    "missing slash",
			input:   "anthropic-claude",
			wantErr: true,
		},
		{
			name:    "too many slashes",
			input:   "anthropic/claude/sonnet",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "only slash",
			input:   "/",
			wantErr: true,
		},
		{
			name:    "empty provider",
			input:   "/claude",
			wantErr: true,
		},
		{
			name:    "empty model",
			input:   "anthropic/",
			wantErr: true,
		},
		{
			name:         "model with whitespace (trimmed)",
			input:        " anthropic / claude-3-5-sonnet ",
			wantErr:      false,
			wantProvider: "anthropic",
			wantModel:    "claude-3-5-sonnet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseModel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Provider != tt.wantProvider {
					t.Errorf("ParseModel() Provider = %v, want %v", got.Provider, tt.wantProvider)
				}
				if got.Model != tt.wantModel {
					t.Errorf("ParseModel() Model = %v, want %v", got.Model, tt.wantModel)
				}
				if got.Original != tt.input {
					t.Errorf("ParseModel() Original = %v, want %v", got.Original, tt.input)
				}
			}
		})
	}
}

func TestSupportedProviders(t *testing.T) {
	providers := SupportedProviders()
	if len(providers) == 0 {
		t.Error("SupportedProviders() returned empty list")
	}

	// Verify anthropic is in the list
	found := false
	for _, p := range providers {
		if p == "anthropic" {
			found = true
			break
		}
	}
	if !found {
		t.Error("SupportedProviders() should include 'anthropic'")
	}
}

func TestIsProviderSupported(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{
			name:     "anthropic is supported",
			provider: "anthropic",
			want:     true,
		},
		{
			name:     "openai is not yet supported",
			provider: "openai",
			want:     false,
		},
		{
			name:     "google is not yet supported",
			provider: "google",
			want:     false,
		},
		{
			name:     "meta-llama is not yet supported",
			provider: "meta-llama",
			want:     false,
		},
		{
			name:     "unknown provider",
			provider: "unknown",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsProviderSupported(tt.provider); got != tt.want {
				t.Errorf("IsProviderSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}
