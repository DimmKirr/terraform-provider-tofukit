package provider

import (
	"context"

	"github.com/tofukit/opentofu-provider-tofukit/internal/llm"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
)

// claudeAdapter adapts claude.Executor to implement llm.LLMExecutor
type claudeAdapter struct {
	executor *claude.Executor
}

// newClaudeAdapter creates a new adapter for claude.Executor
func newClaudeAdapter(executor *claude.Executor) *claudeAdapter {
	return &claudeAdapter{executor: executor}
}

// Execute implements llm.LLMExecutor
func (a *claudeAdapter) Execute(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*llm.ExecutionStatus, error) {
	status, err := a.executor.Execute(ctx, projectSpec, outputDir)
	if err != nil {
		return nil, err
	}
	// Convert claude.ExecutionStatus to llm.ExecutionStatus
	return &llm.ExecutionStatus{
		State:       status.State,
		StartedAt:   status.StartedAt,
		CompletedAt: status.CompletedAt,
		ProjectPath: status.ProjectPath,
		Error:       status.Error,
		Output:      status.Output,
		Metadata:    status.Metadata,
	}, nil
}

// Query implements llm.LLMExecutor
func (a *claudeAdapter) Query(ctx context.Context, instructions []string, model string) (string, error) {
	return a.executor.Query(ctx, instructions, model)
}

// Validate implements llm.LLMExecutor
func (a *claudeAdapter) Validate(ctx context.Context) error {
	return a.executor.Validate(ctx)
}

// SetDebug implements llm.LLMExecutor
func (a *claudeAdapter) SetDebug(debug bool) {
	a.executor.SetDebug(debug)
}

// SetOutputPath implements llm.LLMExecutor
func (a *claudeAdapter) SetOutputPath(outputPath string) {
	a.executor.SetOutputPath(outputPath)
}

// SetSystemPrompt implements llm.LLMExecutor
func (a *claudeAdapter) SetSystemPrompt(systemPrompt string) {
	a.executor.SetSystemPrompt(systemPrompt)
}

// RetryExecution implements llm.LLMExecutor
func (a *claudeAdapter) RetryExecution(ctx context.Context, projectSpec map[string]interface{}, outputDir string, maxRetries int) (*llm.ExecutionStatus, error) {
	status, err := a.executor.RetryExecution(ctx, projectSpec, outputDir, maxRetries)
	if err != nil {
		return nil, err
	}
	// Convert claude.ExecutionStatus to llm.ExecutionStatus
	return &llm.ExecutionStatus{
		State:       status.State,
		StartedAt:   status.StartedAt,
		CompletedAt: status.CompletedAt,
		ProjectPath: status.ProjectPath,
		Error:       status.Error,
		Output:      status.Output,
		Metadata:    status.Metadata,
	}, nil
}

// IsProjectGenerated implements llm.LLMExecutor
func (a *claudeAdapter) IsProjectGenerated(ctx context.Context, projectPath string) bool {
	return a.executor.IsProjectGenerated(ctx, projectPath)
}

// CleanupProject implements llm.LLMExecutor
func (a *claudeAdapter) CleanupProject(ctx context.Context, projectPath string) error {
	return a.executor.CleanupProject(ctx, projectPath)
}
