package mocks

import (
	"context"
	"fmt"
	"time"

	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
)

// MockClaudeClient implements a mock version of the Claude client for testing
type MockClaudeClient struct {
	// Test behavior controls
	ShouldFailValidation bool
	ShouldFailExecution  bool
	ExecutionDelay       time.Duration
	MockExecutionResult  *claude.ExecutionResult
	MockValidationError  error

	// Call tracking
	ValidateCalled   bool
	ExecuteCalled    bool
	ExecuteCallCount int
	LastProjectSpec  map[string]interface{}
	LastOutputPath   string
}

// NewMockClaudeClient creates a new mock Claude client
func NewMockClaudeClient() *MockClaudeClient {
	return &MockClaudeClient{
		MockExecutionResult: &claude.ExecutionResult{
			Success:     true,
			Output:      "Mock execution completed successfully",
			ProjectPath: "/tmp/test-project",
		},
	}
}

// WithFailedValidation configures the mock to fail validation
func (m *MockClaudeClient) WithFailedValidation(err error) *MockClaudeClient {
	m.ShouldFailValidation = true
	m.MockValidationError = err
	return m
}

// WithFailedExecution configures the mock to fail execution
func (m *MockClaudeClient) WithFailedExecution() *MockClaudeClient {
	m.ShouldFailExecution = true
	return m
}

// WithExecutionDelay adds a delay to execution
func (m *MockClaudeClient) WithExecutionDelay(delay time.Duration) *MockClaudeClient {
	m.ExecutionDelay = delay
	return m
}

// WithCustomResult sets a custom execution result
func (m *MockClaudeClient) WithCustomResult(result *claude.ExecutionResult) *MockClaudeClient {
	m.MockExecutionResult = result
	return m
}

// ValidateClaudeCodeAvailability mocks the validation method
func (m *MockClaudeClient) ValidateClaudeCodeAvailability(ctx context.Context) error {
	m.ValidateCalled = true

	if m.ShouldFailValidation {
		if m.MockValidationError != nil {
			return m.MockValidationError
		}
		return fmt.Errorf("mock validation failed")
	}

	return nil
}

// ExecuteProject mocks the execution method
func (m *MockClaudeClient) ExecuteProject(ctx context.Context, projectSpec map[string]interface{}, outputPath string) (*claude.ExecutionResult, error) {
	m.ExecuteCalled = true
	m.ExecuteCallCount++
	m.LastProjectSpec = projectSpec
	m.LastOutputPath = outputPath

	// Add delay if configured
	if m.ExecutionDelay > 0 {
		select {
		case <-time.After(m.ExecutionDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if m.ShouldFailExecution {
		return &claude.ExecutionResult{
			Success: false,
			Error:   "mock execution failed",
		}, fmt.Errorf("mock execution failed")
	}

	return m.MockExecutionResult, nil
}

// GetProjectPath mocks the project path method
func (m *MockClaudeClient) GetProjectPath(projectSpec map[string]interface{}, basePath string) string {
	if project, ok := projectSpec["project"].(map[string]interface{}); ok {
		if name, ok := project["name"].(string); ok {
			return fmt.Sprintf("%s/%s", basePath, name)
		}
	}
	return fmt.Sprintf("%s/mock-project", basePath)
}

// Reset clears all call tracking
func (m *MockClaudeClient) Reset() {
	m.ValidateCalled = false
	m.ExecuteCalled = false
	m.ExecuteCallCount = 0
	m.LastProjectSpec = nil
	m.LastOutputPath = ""
}
