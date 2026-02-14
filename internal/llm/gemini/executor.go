package gemini

import (
	"context"
	"fmt"
	"time"

	"github.com/tofukit/opentofu-provider-tofukit/internal/llm"
)

// Executor manages Gemini execution lifecycle (stub implementation)
type Executor struct {
	apiKey     string
	debug      bool
	outputPath string
}

// NewExecutor creates a new Gemini executor
func NewExecutor(apiKey string) *Executor {
	return &Executor{
		apiKey:     apiKey,
		debug:      false,
		outputPath: "",
	}
}

// Execute runs Gemini with the provided project specification (stub)
func (e *Executor) Execute(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*llm.ExecutionStatus, error) {
	return &llm.ExecutionStatus{
		State:       "failed",
		StartedAt:   time.Now().Format(time.RFC3339),
		CompletedAt: time.Now().Format(time.RFC3339),
		Error:       "Gemini provider not yet implemented. Coming soon!",
	}, fmt.Errorf("Gemini provider not yet implemented. Coming soon!")
}

// Query sends a simple query to Gemini and returns the response (stub)
func (e *Executor) Query(ctx context.Context, instructions []string, model string) (string, error) {
	return "", fmt.Errorf("Gemini provider not yet implemented. Coming soon!")
}

// Validate checks if Gemini is properly configured (stub)
func (e *Executor) Validate(ctx context.Context) error {
	if e.apiKey == "" {
		return fmt.Errorf("API key is required for Gemini provider")
	}
	// TODO: Validate API key format and connectivity
	return fmt.Errorf("Gemini provider not yet implemented. Please use 'claude' for now")
}

// SetDebug enables or disables debug mode
func (e *Executor) SetDebug(debug bool) {
	e.debug = debug
}

// SetOutputPath sets the output path for debug files
func (e *Executor) SetOutputPath(outputPath string) {
	e.outputPath = outputPath
}

// SetSystemPrompt sets a custom system prompt
func (e *Executor) SetSystemPrompt(systemPrompt string) {
	// TODO: Store system prompt for Gemini execution
}

// RetryExecution attempts to retry a failed execution (stub)
func (e *Executor) RetryExecution(ctx context.Context, projectSpec map[string]interface{}, outputDir string, maxRetries int) (*llm.ExecutionStatus, error) {
	return e.Execute(ctx, projectSpec, outputDir)
}

// IsProjectGenerated checks if a project has been generated (stub)
func (e *Executor) IsProjectGenerated(ctx context.Context, projectPath string) bool {
	return false
}

// CleanupProject removes the generated project directory (stub)
func (e *Executor) CleanupProject(ctx context.Context, projectPath string) error {
	return nil
}
