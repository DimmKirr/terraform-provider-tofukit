package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/files"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/models"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// Executor implements the OpenAI LLM executor
type Executor struct {
	apiKey       string
	modelSlug    string // TofuKit model slug (e.g., "openai/gpt-5-image")
	model        string // Provider model ID from models registry
	debug        bool
	outputPath   string
	systemPrompt string
	client       *Client
}

// NewExecutor creates a new OpenAI executor
func NewExecutor(apiKey string, modelSlug string) *Executor {
	// Get provider model ID from registry
	providerModelID := models.GetProviderModel(modelSlug)

	return &Executor{
		apiKey:    apiKey,
		modelSlug: modelSlug,
		model:     providerModelID,
		client:    NewClient(apiKey),
	}
}

// SetDebug enables or disables debug mode
func (e *Executor) SetDebug(debug bool) {
	e.debug = debug
}

// SetOutputPath sets the output path for files
func (e *Executor) SetOutputPath(outputPath string) {
	e.outputPath = outputPath
}

// SetSystemPrompt sets a custom system prompt
func (e *Executor) SetSystemPrompt(systemPrompt string) {
	e.systemPrompt = systemPrompt
}

// Execute implements llm.LLMExecutor
func (e *Executor) Execute(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*llm.ExecutionStatus, error) {
	// Check if this is an image model
	if models.IsImageModel(e.modelSlug) {
		return e.executeImageGeneration(ctx, projectSpec, outputDir)
	}

	// Text/code generation not yet implemented
	return nil, fmt.Errorf("text/code generation with OpenAI not yet implemented (only image generation is supported)")
}

// executeImageGeneration handles image generation requests
func (e *Executor) executeImageGeneration(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*llm.ExecutionStatus, error) {
	// Extract file specifications from projectSpec
	// The Claude prompt structure is: request.specification.files (map of filename -> file object)
	var filesMap map[string]interface{}

	// Try specification.files first (Claude prompt structure)
	if spec, ok := projectSpec["specification"].(map[string]interface{}); ok {
		if fileData, ok := spec["files"].(map[string]interface{}); ok {
			filesMap = fileData
		}
	}

	// Fallback: try direct "files" key
	if filesMap == nil {
		if fileData, ok := projectSpec["files"].(map[string]interface{}); ok {
			filesMap = fileData
		}
	}

	if len(filesMap) == 0 {
		return nil, fmt.Errorf("no files specified for image generation (checked specification.files and files)")
	}

	// Process each file (should be image files)
	generatedFiles := []map[string]interface{}{}
	for path, fileData := range filesMap {
		fileMap, ok := fileData.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract prompt from instructions
		promptText := extractPromptText(fileMap)
		if promptText == "" {
			continue
		}

		// Extract image config from constraints
		config := parseImageConfig(extractConstraints(fileMap))

		// Generate image
		fmt.Printf("[OpenAI DEBUG] Generating image for path: %s\n", path)
		fmt.Printf("[OpenAI DEBUG] Model: %s, Prompt: %s\n", e.model, promptText)
		fmt.Printf("[OpenAI DEBUG] Config: %+v\n", config)

		imageDataB64, err := e.client.GenerateImage(ctx, e.model, promptText, config)
		if err != nil {
			fmt.Printf("[OpenAI DEBUG] Image generation FAILED: %v\n", err)
			return nil, fmt.Errorf("failed to generate image '%s': %w", path, err)
		}
		fmt.Printf("[OpenAI DEBUG] Image generation SUCCESS, base64 length: %d\n", len(imageDataB64))

		// Save image to file
		fullPath := filepath.Join(outputDir, path)
		fmt.Printf("[OpenAI DEBUG] Saving to: %s\n", fullPath)
		if err := saveImageFromBase64(imageDataB64, fullPath); err != nil {
			fmt.Printf("[OpenAI DEBUG] Save FAILED: %v\n", err)
			return nil, fmt.Errorf("failed to save image '%s': %w", path, err)
		}
		fmt.Printf("[OpenAI DEBUG] Image saved successfully\n")

		// Collect response data for debug logging
		generatedFiles = append(generatedFiles, map[string]interface{}{
			"path":        path,
			"prompt":      promptText,
			"config":      config,
			"base64_size": len(imageDataB64),
			"model":       e.model,
		})
	}

	// Save response data for debugging
	if len(generatedFiles) > 0 {
		responseData := map[string]interface{}{
			"generated_files": generatedFiles,
			"total_files":     len(generatedFiles),
			"timestamp":       time.Now().Format(time.RFC3339),
		}
		if err := e.saveResponseData(ctx, responseData, outputDir, 1); err != nil {
			tflog.Warn(ctx, "Failed to save response data", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	return &llm.ExecutionStatus{
		State:       "completed",
		ProjectPath: outputDir,
	}, nil
}

// saveImageFromBase64 decodes base64 image data and saves it to a file
func saveImageFromBase64(b64Data string, filePath string) error {
	// Decode base64
	imageData, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	// Detect format and ensure correct extension
	filePath = ensureImageExtension(filePath, imageData)

	// Create directory if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ensureImageExtension detects image format via MIME type and ensures correct extension
func ensureImageExtension(filePath string, imageData []byte) string {
	mimeType := http.DetectContentType(imageData)

	var correctExt string
	switch mimeType {
	case "image/png":
		correctExt = ".png"
	case "image/jpeg":
		correctExt = ".jpg"
	case "image/webp":
		correctExt = ".webp"
	default:
		correctExt = ".png" // Default
	}

	// Check current extension
	ext := filepath.Ext(filePath)
	if ext != correctExt {
		// Replace or add correct extension
		base := strings.TrimSuffix(filePath, ext)
		return base + correctExt
	}

	return filePath
}

// parseImageConfig extracts image configuration from constraints
func parseImageConfig(constraints []string) ImageConfig {
	config := ImageConfig{
		Size:    "1024x1024",
		Quality: "standard",
		Style:   "vivid",
	}

	for _, c := range constraints {
		lower := strings.ToLower(c)
		if strings.Contains(lower, "size:") {
			parts := strings.Split(c, ":")
			if len(parts) >= 2 {
				config.Size = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(lower, "quality:") {
			parts := strings.Split(c, ":")
			if len(parts) >= 2 {
				config.Quality = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(lower, "style:") {
			parts := strings.Split(c, ":")
			if len(parts) >= 2 {
				config.Style = strings.TrimSpace(parts[1])
			}
		}
	}

	return config
}

// extractPromptText extracts the prompt text from file specification
func extractPromptText(fileMap map[string]interface{}) string {
	instructions, ok := fileMap["instructions"].([]interface{})
	if !ok || len(instructions) == 0 {
		return ""
	}

	// Get first instruction
	firstInst, ok := instructions[0].(map[string]interface{})
	if !ok {
		return ""
	}

	prompt, _ := firstInst["prompt"].(string)
	return prompt
}

// getKeys returns the keys of a map for debugging
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// extractConstraints extracts constraints from file specification
func extractConstraints(fileMap map[string]interface{}) []string {
	instructions, ok := fileMap["instructions"].([]interface{})
	if !ok || len(instructions) == 0 {
		return []string{}
	}

	// Get first instruction
	firstInst, ok := instructions[0].(map[string]interface{})
	if !ok {
		return []string{}
	}

	constraints, ok := firstInst["constraints"].([]interface{})
	if !ok {
		return []string{}
	}

	// Convert to string slice
	result := make([]string, 0, len(constraints))
	for _, c := range constraints {
		if str, ok := c.(string); ok {
			result = append(result, str)
		}
	}

	return result
}

// Query implements llm.LLMExecutor (not yet implemented for OpenAI)
func (e *Executor) Query(ctx context.Context, instructions []string, model string) (string, error) {
	return "", fmt.Errorf("Query not yet implemented for OpenAI")
}

// Validate implements llm.LLMExecutor
func (e *Executor) Validate(ctx context.Context) error {
	if e.apiKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}
	return nil
}

// RetryExecution implements llm.LLMExecutor (not yet implemented)
func (e *Executor) RetryExecution(ctx context.Context, projectSpec map[string]interface{}, outputDir string, maxRetries int) (*llm.ExecutionStatus, error) {
	return e.Execute(ctx, projectSpec, outputDir)
}

// IsProjectGenerated implements llm.LLMExecutor
func (e *Executor) IsProjectGenerated(ctx context.Context, projectPath string) bool {
	info, err := os.Stat(projectPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// CleanupProject implements llm.LLMExecutor
func (e *Executor) CleanupProject(ctx context.Context, projectPath string) error {
	return os.RemoveAll(projectPath)
}

// ExecuteWithPromptJSON executes OpenAI with a pre-built prompt JSON
// This method is called by the project resource when using OpenAI models
func (e *Executor) ExecuteWithPromptJSON(
	ctx context.Context,
	promptJSON string,
	outputDir string,
	filesToVerify []schemas.FileModelWithPath,
	maxRetries int,
) (*llm.ExecutionStatus, *files.VerificationReport, error) {
	// Save prompt JSON for debugging (attempt 1)
	if err := e.savePromptJSON(ctx, promptJSON, outputDir, 1); err != nil {
		tflog.Warn(ctx, "Failed to save prompt JSON", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Parse the prompt JSON to extract project specification
	var promptData map[string]interface{}
	if err := json.Unmarshal([]byte(promptJSON), &promptData); err != nil {
		return nil, nil, fmt.Errorf("failed to parse prompt JSON: %w", err)
	}

	// The prompt JSON structure is Claude-specific: {request: {project_info, files, ...}, system_prompt: "..."}
	// Extract the request object which contains files
	var projectSpec map[string]interface{}
	if request, ok := promptData["request"].(map[string]interface{}); ok {
		projectSpec = request
	} else {
		// Fallback: use full promptData if no request field
		projectSpec = promptData
	}

	// Execute image generation
	status, err := e.Execute(ctx, projectSpec, outputDir)
	if err != nil {
		return nil, nil, err
	}

	// Save execution metadata for debugging
	if err := e.saveExecutionMetadata(ctx, status, outputDir); err != nil {
		tflog.Warn(ctx, "Failed to save execution metadata", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// For image generation, we don't have verification commands
	// Return a simple "all passed" report
	report := &files.VerificationReport{
		AllPassed:   true,
		FailedCount: 0,
		PassedCount: 0,
		Results:     []files.VerificationResult{},
	}

	return status, report, nil
}

// savePromptJSON saves the prompt JSON sent to OpenAI for debugging
func (e *Executor) savePromptJSON(ctx context.Context, promptJSON string, outputDir string, attempt int) error {
	if !e.debug {
		return nil
	}

	debugDir := filepath.Join(outputDir, ".debug")

	// Create debug directory if it doesn't exist
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("failed to create debug directory: %w", err)
	}

	timestamp := time.Now().Unix()
	promptPath := filepath.Join(debugDir, fmt.Sprintf("openai-prompt-attempt%d-%d.json", attempt, timestamp))

	// Write raw JSON to file
	if err := os.WriteFile(promptPath, []byte(promptJSON), 0644); err != nil {
		return fmt.Errorf("failed to write prompt JSON: %w", err)
	}

	tflog.Debug(ctx, "Saved OpenAI prompt JSON", map[string]interface{}{
		"prompt_path": promptPath,
		"attempt":     attempt,
	})

	return nil
}

// saveResponseData saves the OpenAI API response data for debugging
func (e *Executor) saveResponseData(ctx context.Context, responseData map[string]interface{}, outputDir string, attempt int) error {
	if !e.debug {
		return nil
	}

	debugDir := filepath.Join(outputDir, ".debug")

	// Create debug directory if it doesn't exist
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("failed to create debug directory: %w", err)
	}

	timestamp := time.Now().Unix()
	responsePath := filepath.Join(debugDir, fmt.Sprintf("openai-response-attempt%d-%d.json", attempt, timestamp))

	// Marshal to JSON
	data, err := json.MarshalIndent(responseData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal response data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(responsePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write response file: %w", err)
	}

	tflog.Debug(ctx, "Saved OpenAI response data", map[string]interface{}{
		"response_path": responsePath,
		"attempt":       attempt,
	})

	return nil
}

// saveExecutionMetadata saves execution status metadata for debugging
func (e *Executor) saveExecutionMetadata(ctx context.Context, status *llm.ExecutionStatus, outputDir string) error {
	if !e.debug {
		return nil
	}

	// Create .debug directory
	debugDir := filepath.Join(outputDir, ".debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("failed to create .debug directory: %w", err)
	}

	timestamp := time.Now().Unix()
	metadataPath := filepath.Join(debugDir, fmt.Sprintf("openai-execution-metadata-%d.json", timestamp))

	// Create metadata structure
	metadata := map[string]interface{}{
		"execution_status": status,
		"created_at":       time.Now().Format(time.RFC3339),
		"terraform_provider": map[string]string{
			"name":   "tofukit",
			"action": "openai_execution",
		},
		"model": e.model,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to file
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	tflog.Debug(ctx, "Saved OpenAI execution metadata", map[string]interface{}{
		"metadata_path": metadataPath,
	})

	return nil
}
