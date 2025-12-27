// Package models provides a hardcoded registry of supported LLM models for TofuKit.
//
// This package maps TofuKit's internal model slugs (e.g., "openai/gpt-5.2") to actual
// provider model IDs (e.g., "gpt-5.2-2025-12-11") that are sent to LLM provider APIs.
//
// # Model Registry
//
// The ModelRegistry contains essential models from OpenAI and Anthropic:
//
// OpenAI Models:
//   - openai/gpt-5.2           → gpt-5.2-chat-latest    (text/code generation, auto-updates)
//   - openai/gpt-5-image       → gpt-image-1.5          (image generation, latest)
//   - openai/gpt-5-image-mini  → gpt-image-1-mini       (image generation, lightweight)
//
// Anthropic Models (Claude Code CLI):
//   - anthropic/claude-sonnet-4.5 → sonnet  (balanced performance, auto-updates)
//   - anthropic/claude-haiku      → haiku   (fast, compact, auto-updates)
//   - anthropic/claude-opus       → opus    (complex tasks, auto-updates)
//
// Google Models (Gemini API):
//   - google/gemini-3-pro              → gemini-3-pro-preview       (best for text, reasoning, 1M context)
//   - google/gemini-2.5-pro            → gemini-2.5-pro             (advanced reasoning, 2M context)
//   - google/gemini-2.5-flash          → gemini-2.5-flash           (balanced performance/speed)
//   - google/gemini-2.5-flash-lite     → gemini-2.5-flash-lite      (fastest/cheapest for text)
//   - google/gemini-3-pro-image        → gemini-3-pro-image-preview (Nano Banana Pro - advanced image gen)
//   - google/gemini-2.5-flash-image    → gemini-2.5-flash-image     (Nano Banana - fast image gen)
//
// # Usage
//
// Get provider model ID from TofuKit slug:
//
//	providerModel := models.GetProviderModel("openai/gpt-5-image")
//	// Returns: "gpt-image-1.5"
//
// Check if a model supports image generation:
//
//	if models.IsImageModel("openai/gpt-5-image") {
//	    // Use image generation API
//	} else {
//	    // Use text generation API
//	}
//
// Get full model metadata:
//
//	model, exists := models.GetModel("openai/gpt-5.2")
//	if exists {
//	    fmt.Println(model.ContextLength) // 272000
//	}
//
// # Relationship to model data sources
//
// OpenAI model IDs are sourced from .context/openai-models.json (direct OpenAI API),
// not .context/models.json (which contains OpenRouter routing slugs). This ensures we
// use actual SDK model IDs that OpenAI's API expects.
//
// The registry is hardcoded for:
//   - Reliability: No runtime file I/O or network calls
//   - Performance: Fast lookups without JSON parsing
//   - Simplicity: Clear mapping visible in code
//   - Latest aliases: Uses "-latest" suffixes where available for auto-updating models
//
// An optional generation script (scripts/generate_model_registry.go) can be used
// to auto-generate this file from openai-models.json if the full registry needs updating.
package models

import "strings"

// Model represents a LLM model configuration
type Model struct {
	Slug             string   `json:"slug"`              // TofuKit model slug (e.g., "openai/gpt-5.2")
	Name             string   `json:"name"`              // Display name
	ProviderModelID  string   `json:"provider_model_id"` // Actual provider API model ID
	ProviderSlug     string   `json:"provider_slug"`     // Provider identifier
	ContextLength    int      `json:"context_length"`    // Max context tokens
	Description      string   `json:"description"`       // Model description
	InputModalities  []string `json:"input_modalities"`  // Supported input types
	OutputModalities []string `json:"output_modalities"` // Supported output types
}

// ModelRegistry contains all supported models
var ModelRegistry = map[string]Model{
	// OpenAI - Text/Code Generation
	"openai/gpt-5.2": {
		Slug:             "openai/gpt-5.2",
		Name:             "OpenAI GPT-5.2",
		ProviderModelID:  "gpt-5.2-chat-latest",
		ProviderSlug:     "openai",
		ContextLength:    400000,
		Description:      "GPT-5.2 is the most capable model series yet for professional knowledge work",
		InputModalities:  []string{"text", "image", "file"},
		OutputModalities: []string{"text"},
	},

	// OpenAI - Image Generation
	"openai/gpt-5-image": {
		Slug:             "openai/gpt-5-image",
		Name:             "OpenAI GPT-5 Image",
		ProviderModelID:  "gpt-image-1.5",
		ProviderSlug:     "openai",
		ContextLength:    400000,
		Description:      "GPT-5 Image combines GPT-5 reasoning with state-of-the-art image generation",
		InputModalities:  []string{"text", "image", "file"},
		OutputModalities: []string{"image", "text"},
	},

	"openai/gpt-5-image-mini": {
		Slug:             "openai/gpt-5-image-mini",
		Name:             "OpenAI GPT-5 Image Mini",
		ProviderModelID:  "gpt-image-1-mini",
		ProviderSlug:     "openai",
		ContextLength:    400000,
		Description:      "Lightweight version of GPT-5 Image for faster, cost-effective image generation",
		InputModalities:  []string{"text", "image", "file"},
		OutputModalities: []string{"image", "text"},
	},

	// OpenAI - Direct Image Generation Models (alternative naming)
	"openai/gpt-image-1": {
		Slug:             "openai/gpt-image-1",
		Name:             "OpenAI GPT Image 1.5",
		ProviderModelID:  "gpt-image-1.5",
		ProviderSlug:     "openai",
		ContextLength:    400000,
		Description:      "GPT Image 1.5 - advanced image generation model",
		InputModalities:  []string{"text"},
		OutputModalities: []string{"image"},
	},

	"openai/gpt-image-1-mini": {
		Slug:             "openai/gpt-image-1-mini",
		Name:             "OpenAI GPT Image 1 Mini",
		ProviderModelID:  "gpt-image-1-mini",
		ProviderSlug:     "openai",
		ContextLength:    400000,
		Description:      "Lightweight GPT Image model for faster, cost-effective image generation",
		InputModalities:  []string{"text"},
		OutputModalities: []string{"image"},
	},

	// Anthropic - Claude Models (via Claude Code CLI)
	"anthropic/claude-sonnet-4.5": {
		Slug:             "anthropic/claude-sonnet-4.5",
		Name:             "Anthropic Claude Sonnet",
		ProviderModelID:  "sonnet",
		ProviderSlug:     "anthropic",
		ContextLength:    1000000,
		Description:      "Claude Sonnet offers balanced performance for most tasks (auto-updates to latest Sonnet)",
		InputModalities:  []string{"text", "image"},
		OutputModalities: []string{"text"},
	},

	"anthropic/claude-haiku": {
		Slug:             "anthropic/claude-haiku",
		Name:             "Anthropic Claude Haiku",
		ProviderModelID:  "haiku",
		ProviderSlug:     "anthropic",
		ContextLength:    200000,
		Description:      "Claude Haiku is the fastest, most compact model for near-instant responsiveness (auto-updates to latest Haiku)",
		InputModalities:  []string{"text"},
		OutputModalities: []string{"text"},
	},

	"anthropic/claude-opus": {
		Slug:             "anthropic/claude-opus",
		Name:             "Anthropic Claude Opus",
		ProviderModelID:  "opus",
		ProviderSlug:     "anthropic",
		ContextLength:    200000,
		Description:      "Claude Opus delivers top-level performance on highly complex tasks (auto-updates to latest Opus)",
		InputModalities:  []string{"text", "image"},
		OutputModalities: []string{"text"},
	},

	// Google - Gemini Models
	"google/gemini-3-pro": {
		Slug:             "google/gemini-3-pro",
		Name:             "Google Gemini 3 Pro",
		ProviderModelID:  "gemini-3-pro-preview",
		ProviderSlug:     "google",
		ContextLength:    1048576,
		Description:      "Gemini 3 Pro is Google's flagship frontier model for high-precision multimodal reasoning with 1M context window",
		InputModalities:  []string{"text", "image", "video", "audio", "file"},
		OutputModalities: []string{"text"},
	},

	"google/gemini-2.5-flash": {
		Slug:             "google/gemini-2.5-flash",
		Name:             "Google Gemini 2.5 Flash",
		ProviderModelID:  "gemini-2.5-flash",
		ProviderSlug:     "google",
		ContextLength:    1048576,
		Description:      "Gemini 2.5 Flash balances performance and speed for high-throughput tasks",
		InputModalities:  []string{"text", "image", "audio", "file"},
		OutputModalities: []string{"text"},
	},

	"google/gemini-2.5-flash-lite": {
		Slug:             "google/gemini-2.5-flash-lite",
		Name:             "Google Gemini 2.5 Flash-Lite",
		ProviderModelID:  "gemini-2.5-flash-lite",
		ProviderSlug:     "google",
		ContextLength:    1048576,
		Description:      "Gemini 2.5 Flash-Lite is optimized for ultra-low latency and cost efficiency",
		InputModalities:  []string{"text", "image", "audio", "file"},
		OutputModalities: []string{"text"},
	},

	"google/gemini-2.5-pro": {
		Slug:             "google/gemini-2.5-pro",
		Name:             "Google Gemini 2.5 Pro",
		ProviderModelID:  "gemini-2.5-pro",
		ProviderSlug:     "google",
		ContextLength:    2097152,
		Description:      "Gemini 2.5 Pro offers advanced reasoning and complex problem-solving with 2M context window",
		InputModalities:  []string{"text", "image", "video", "audio", "file"},
		OutputModalities: []string{"text"},
	},

	"google/gemini-3-pro-image": {
		Slug:             "google/gemini-3-pro-image",
		Name:             "Google Gemini 3 Pro Image (Nano Banana Pro)",
		ProviderModelID:  "gemini-3-pro-image-preview",
		ProviderSlug:     "google",
		ContextLength:    65536,
		Description:      "Nano Banana Pro is Google's most advanced image generation model with multimodal reasoning and high-fidelity visual synthesis",
		InputModalities:  []string{"text", "image"},
		OutputModalities: []string{"image", "text"},
	},

	"google/gemini-2.5-flash-image": {
		Slug:             "google/gemini-2.5-flash-image",
		Name:             "Google Gemini 2.5 Flash Image (Nano Banana)",
		ProviderModelID:  "gemini-2.5-flash-image",
		ProviderSlug:     "google",
		ContextLength:    32768,
		Description:      "Nano Banana provides fast, efficient image generation with good quality",
		InputModalities:  []string{"text", "image"},
		OutputModalities: []string{"image", "text"},
	},
}

// GetProviderModel returns the provider model ID for a given TofuKit model slug
func GetProviderModel(modelSlug string) string {
	if model, exists := ModelRegistry[modelSlug]; exists {
		return model.ProviderModelID
	}

	// Fallback: extract model part after "provider/"
	parts := strings.Split(modelSlug, "/")
	if len(parts) == 2 {
		return parts[1]
	}

	return modelSlug
}

// IsImageModel checks if a model supports image generation
func IsImageModel(modelSlug string) bool {
	model, exists := ModelRegistry[modelSlug]
	if !exists {
		// Fallback to string matching
		providerModel := GetProviderModel(modelSlug)
		return strings.Contains(providerModel, "image") ||
			strings.Contains(providerModel, "-image")
	}

	// Check if model has image output capability
	for _, modality := range model.OutputModalities {
		if modality == "image" {
			return true
		}
	}

	return false
}

// GetModel returns the full model configuration
func GetModel(modelSlug string) (Model, bool) {
	model, exists := ModelRegistry[modelSlug]
	return model, exists
}
