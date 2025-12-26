package openai

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

// Client wraps the official OpenAI Go SDK client
type Client struct {
	client openai.Client
}

// NewClient creates a new OpenAI client with the provided API key
func NewClient(apiKey string) *Client {
	return &Client{
		client: openai.NewClient(
			option.WithAPIKey(apiKey),
		),
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
	// Build params with ResponseFormat set to b64_json to get base64 data
	params := openai.ImageGenerateParams{
		Prompt:         prompt,
		N:              param.NewOpt[int64](1),
		ResponseFormat: openai.ImageGenerateParamsResponseFormatB64JSON,
	}

	// Apply size if specified
	if config.Size != "" {
		params.Size = openai.ImageGenerateParamsSize(config.Size)
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
