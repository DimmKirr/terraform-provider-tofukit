package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

// Client wraps the official OpenAI Go SDK client
type Client struct {
	client openai.Client
}

// NewClient creates a new OpenAI client with the provided API key
// If apiKey is empty, the SDK will automatically use the OPENAI_API_KEY environment variable
func NewClient(apiKey string) *Client {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	// If no API key provided, SDK automatically reads from OPENAI_API_KEY env var

	return &Client{
		client: openai.NewClient(opts...),
	}
}

// CreateChatCompletion wraps the official SDK's chat completion API
func (c *Client) CreateChatCompletion(ctx context.Context, model string, systemPrompt string, userMessage string) (string, error) {
	completion, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userMessage),
		},
		Temperature: param.NewOpt(0.7),
		MaxTokens:   param.NewOpt[int64](4096),
	})

	if err != nil {
		return "", err
	}

	return completion.Choices[0].Message.Content, nil
}

// ImageConfig contains configuration for image generation
type ImageConfig struct {
	Size    string
	Quality string
	Style   string
}

// GenerateImage wraps the official SDK's image generation API
// Returns base64-encoded image data
func (c *Client) GenerateImage(ctx context.Context, model string, prompt string, config ImageConfig) (string, error) {
	// Build params - gpt-image-1* models don't support response_format parameter
	// and return base64 by default, while dall-e models need it explicitly
	params := openai.ImageGenerateParams{
		Prompt: prompt,
		N:      param.NewOpt[int64](1),
	}

	// Apply model if specified (important: gpt-image-1.5 has higher prompt limits than gpt-image-1-mini)
	if model != "" {
		params.Model = model
	}

	// Only set ResponseFormat for dall-e models (gpt-image-1* models don't support it)
	if !strings.HasPrefix(model, "gpt-image") {
		params.ResponseFormat = openai.ImageGenerateParamsResponseFormatB64JSON
	}

	// Apply size if specified
	if config.Size != "" {
		params.Size = openai.ImageGenerateParamsSize(config.Size)
	}

	// Apply quality if specified
	// - gpt-image-1*: high, medium, low
	// - dall-e-3: standard, hd
	// - dall-e-2: not supported
	if config.Quality != "" {
		params.Quality = openai.ImageGenerateParamsQuality(config.Quality)
	}

	// Apply style if specified (vivid, natural - DALL-E 3 only)
	if config.Style != "" {
		params.Style = openai.ImageGenerateParamsStyle(config.Style)
	}

	// Generate image
	image, err := c.client.Images.Generate(ctx, params)
	if err != nil {
		return "", err
	}

	// Return base64-encoded image data from first image
	if len(image.Data) > 0 && image.Data[0].B64JSON != "" {
		return image.Data[0].B64JSON, nil
	}

	// If we get here, something went wrong
	return "", fmt.Errorf("OpenAI API returned no base64 image data")
}
