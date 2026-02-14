package llm

import "context"

// LLMExecutor defines the interface for LLM provider implementations
type LLMExecutor interface {
	// Execute runs the LLM with the provided project specification
	Execute(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*ExecutionStatus, error)

	// Query sends a simple query to the LLM and returns the response (no project creation)
	// The model parameter specifies which model to use (e.g., "haiku", "sonnet" for Claude)
	Query(ctx context.Context, instructions []string, model string) (string, error)

	// Validate checks if the LLM is properly configured and available
	Validate(ctx context.Context) error

	// SetDebug enables or disables debug mode
	SetDebug(debug bool)

	// SetOutputPath sets the output path for debug files
	SetOutputPath(outputPath string)

	// SetSystemPrompt sets a custom system prompt
	SetSystemPrompt(systemPrompt string)

	// RetryExecution attempts to retry a failed execution
	RetryExecution(ctx context.Context, projectSpec map[string]interface{}, outputDir string, maxRetries int) (*ExecutionStatus, error)

	// IsProjectGenerated checks if a project has been generated at the specified path
	IsProjectGenerated(ctx context.Context, projectPath string) bool

	// CleanupProject removes the generated project directory
	CleanupProject(ctx context.Context, projectPath string) error
}
