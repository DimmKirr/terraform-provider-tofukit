package provider

import (
	"context"

	"github.com/tofukit/opentofu-provider-tofukit/internal/llm"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/openai"
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

// GetClaudeExecutor returns the underlying Claude executor
// This allows access to Claude-specific methods like ExecuteWithPromptJSON
func (a *claudeAdapter) GetClaudeExecutor() *claude.Executor {
	return a.executor
}

// openaiAdapter adapts openai.Executor to implement llm.LLMExecutor
type openaiAdapter struct {
	executor *openai.Executor
}

// newOpenAIAdapter creates a new adapter for openai.Executor
func newOpenAIAdapter(executor *openai.Executor) *openaiAdapter {
	return &openaiAdapter{executor: executor}
}

// Execute implements llm.LLMExecutor
func (a *openaiAdapter) Execute(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*llm.ExecutionStatus, error) {
	return a.executor.Execute(ctx, projectSpec, outputDir)
}

// Query implements llm.LLMExecutor
func (a *openaiAdapter) Query(ctx context.Context, instructions []string, model string) (string, error) {
	return a.executor.Query(ctx, instructions, model)
}

// Validate implements llm.LLMExecutor
func (a *openaiAdapter) Validate(ctx context.Context) error {
	return a.executor.Validate(ctx)
}

// SetDebug implements llm.LLMExecutor
func (a *openaiAdapter) SetDebug(debug bool) {
	a.executor.SetDebug(debug)
}

// SetOutputPath implements llm.LLMExecutor
func (a *openaiAdapter) SetOutputPath(outputPath string) {
	a.executor.SetOutputPath(outputPath)
}

// SetSystemPrompt implements llm.LLMExecutor
func (a *openaiAdapter) SetSystemPrompt(systemPrompt string) {
	a.executor.SetSystemPrompt(systemPrompt)
}

// RetryExecution implements llm.LLMExecutor
func (a *openaiAdapter) RetryExecution(ctx context.Context, projectSpec map[string]interface{}, outputDir string, maxRetries int) (*llm.ExecutionStatus, error) {
	return a.executor.RetryExecution(ctx, projectSpec, outputDir, maxRetries)
}

// IsProjectGenerated implements llm.LLMExecutor
func (a *openaiAdapter) IsProjectGenerated(ctx context.Context, projectPath string) bool {
	return a.executor.IsProjectGenerated(ctx, projectPath)
}

// CleanupProject implements llm.LLMExecutor
func (a *openaiAdapter) CleanupProject(ctx context.Context, projectPath string) error {
	return a.executor.CleanupProject(ctx, projectPath)
}

// GetOpenAIExecutor returns the underlying OpenAI executor
// This allows access to OpenAI-specific methods like ExecuteWithPromptJSON
func (a *openaiAdapter) GetOpenAIExecutor() *openai.Executor {
	return a.executor
}

// svgAdapter adapts claude.SVGExecutor to implement llm.LLMExecutor
type svgAdapter struct {
	executor *claude.SVGExecutor
}

// newSVGAdapter creates a new adapter for claude.SVGExecutor
func newSVGAdapter(executor *claude.SVGExecutor) *svgAdapter {
	return &svgAdapter{executor: executor}
}

// Execute implements llm.LLMExecutor
func (a *svgAdapter) Execute(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*llm.ExecutionStatus, error) {
	return a.executor.Execute(ctx, projectSpec, outputDir)
}

// Query implements llm.LLMExecutor
func (a *svgAdapter) Query(ctx context.Context, instructions []string, model string) (string, error) {
	return a.executor.Query(ctx, instructions, model)
}

// Validate implements llm.LLMExecutor
func (a *svgAdapter) Validate(ctx context.Context) error {
	return a.executor.Validate(ctx)
}

// SetDebug implements llm.LLMExecutor
func (a *svgAdapter) SetDebug(debug bool) {
	a.executor.SetDebug(debug)
}

// SetOutputPath implements llm.LLMExecutor
func (a *svgAdapter) SetOutputPath(outputPath string) {
	a.executor.SetOutputPath(outputPath)
}

// SetSystemPrompt implements llm.LLMExecutor
func (a *svgAdapter) SetSystemPrompt(systemPrompt string) {
	a.executor.SetSystemPrompt(systemPrompt)
}

// RetryExecution implements llm.LLMExecutor
func (a *svgAdapter) RetryExecution(ctx context.Context, projectSpec map[string]interface{}, outputDir string, maxRetries int) (*llm.ExecutionStatus, error) {
	return a.executor.RetryExecution(ctx, projectSpec, outputDir, maxRetries)
}

// IsProjectGenerated implements llm.LLMExecutor
func (a *svgAdapter) IsProjectGenerated(ctx context.Context, projectPath string) bool {
	return a.executor.IsProjectGenerated(ctx, projectPath)
}

// CleanupProject implements llm.LLMExecutor
func (a *svgAdapter) CleanupProject(ctx context.Context, projectPath string) error {
	return a.executor.CleanupProject(ctx, projectPath)
}

// GetSVGExecutor returns the underlying SVG executor
// This allows access to SVG-specific methods like ExecuteWithPromptJSON
func (a *svgAdapter) GetSVGExecutor() *claude.SVGExecutor {
	return a.executor
}
