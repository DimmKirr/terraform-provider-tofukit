package testutil

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tofukit/opentofu-provider-tofukit/internal/claude"
	"github.com/tofukit/opentofu-provider-tofukit/internal/resources"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// TestProjectSpec returns a sample project specification for testing
func TestProjectSpec() map[string]interface{} {
	return map[string]interface{}{
		"project": map[string]interface{}{
			"name":        "test-hello-world",
			"description": "A simple hello world project for testing",
			"version":     "1.0.0",
		},
		"requirements": []map[string]interface{}{
			{
				"name": "Create hello.txt file",
				"instructions": []string{
					"Create a file named hello.txt in the project root",
					"Add the content 'Hello, World!' to the file",
				},
				"verification": map[string]string{
					"command": "test -f hello.txt && grep -q 'Hello, World!' hello.txt",
					"expect":  "success",
				},
			},
		},
	}
}

// TestProjectModelFinal returns a sample ProjectModelFinal for testing
func TestProjectModelFinal() resources.ProjectModelFinal {
	return resources.ProjectModelFinal{
		ID:          types.StringValue("project.test-hello"),
		Name:        types.StringValue("test-hello"),
		Description: types.StringValue("Test project"),
		Version:     types.StringValue("1.0.0"),
		Requirements: []schemas.RequirementModel{
			{
				Name: types.StringValue("Create hello.txt"),
				Instructions: []types.String{
					types.StringValue("Create a file named hello.txt"),
					types.StringValue("Add content: Hello, World!"),
				},
				Verification: &schemas.VerificationModel{
					Command: types.StringValue("test -f hello.txt"),
					Expect:  types.StringValue("success"),
				},
			},
		},
		ExecutionStatus:    types.StringValue("pending"),
		ExecutionStarted:   types.StringValue(""),
		ExecutionCompleted: types.StringValue(""),
		ProjectPath:        types.StringValue(""),
		ExecutionError:     types.StringValue(""),
	}
}

// CreateTempProjectDir creates a temporary directory with test project structure
func CreateTempProjectDir(t *testing.T) string {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "test-project")

	err := os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create temp project directory: %v", err)
	}

	return tmpDir
}

// WriteTestFile writes a test file with specified content
func WriteTestFile(t *testing.T, dir, filename, content string) {
	filepath := filepath.Join(dir, filename)
	err := os.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file %s: %v", filepath, err)
	}
}

// ReadTestFile reads content from a test file
func ReadTestFile(t *testing.T, dir, filename string) string {
	filepath := filepath.Join(dir, filename)
	content, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", filepath, err)
	}
	return string(content)
}

// AssertFileExists checks if a file exists at the given path
func AssertFileExists(t *testing.T, filepath string) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to exist, but it doesn't", filepath)
	}
}

// AssertFileNotExists checks if a file does not exist at the given path
func AssertFileNotExists(t *testing.T, filepath string) {
	if _, err := os.Stat(filepath); err == nil {
		t.Errorf("Expected file %s to not exist, but it does", filepath)
	}
}

// AssertExecutionStatus validates an execution status object
func AssertExecutionStatus(t *testing.T, status *claude.ExecutionStatus, expectedState string) {
	if status == nil {
		t.Fatal("Expected execution status to be non-nil")
	}

	if status.State != expectedState {
		t.Errorf("Expected state %s, got %s", expectedState, status.State)
	}

	if status.StartedAt == "" {
		t.Error("Expected StartedAt to be set")
	}

	if _, err := time.Parse(time.RFC3339, status.StartedAt); err != nil {
		t.Errorf("StartedAt is not a valid RFC3339 timestamp: %v", err)
	}

	if expectedState == "completed" && status.CompletedAt == "" {
		t.Error("Expected CompletedAt to be set for completed execution")
	}
}

// AssertMetadataFile validates the metadata file content
func AssertMetadataFile(t *testing.T, outputDir string) *claude.ExecutionStatus {
	metadataPath := filepath.Join(outputDir, "claude-execution-metadata.json")
	AssertFileExists(t, metadataPath)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to read metadata file: %v", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	statusData, ok := metadata["execution_status"].(map[string]interface{})
	if !ok {
		t.Fatal("execution_status not found in metadata")
	}

	statusJSON, err := json.Marshal(statusData)
	if err != nil {
		t.Fatalf("Failed to marshal status data: %v", err)
	}

	var status claude.ExecutionStatus
	if err := json.Unmarshal(statusJSON, &status); err != nil {
		t.Fatalf("Failed to unmarshal execution status: %v", err)
	}

	return &status
}

// WithTimeout runs a test function with a timeout
func WithTimeout(t *testing.T, timeout time.Duration, fn func(ctx context.Context)) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan bool)
	go func() {
		fn(ctx)
		done <- true
	}()

	select {
	case <-done:
		// Test completed normally
	case <-ctx.Done():
		t.Fatalf("Test timed out after %v", timeout)
	}
}

// SkipIfNoClaudeCredentials skips the test if Claude credentials are not available
func SkipIfNoClaudeCredentials(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot determine home directory")
	}

	credPath := filepath.Join(homeDir, ".claude", ".credentials.json")
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		t.Skip("Claude credentials not found, skipping integration test")
	}
}
