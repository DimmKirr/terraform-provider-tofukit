package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tofukit/opentofu-provider-tofukit/internal/testutil"
	"github.com/tofukit/opentofu-provider-tofukit/internal/testutil/mocks"
)

// TestExecutor_Execute_DryRun tests executor in dry run mode
func TestExecutor_Execute_DryRun(t *testing.T) {
	tmpDir := testutil.CreateTempProjectDir(t)
	executor := NewExecutor("~/.claude", true) // dry run mode

	projectSpec := testutil.TestProjectSpec()
	status, err := executor.Execute(context.Background(), projectSpec, tmpDir)

	if err != nil {
		t.Fatalf("Execute() in dry run should not fail: %v", err)
	}

	testutil.AssertExecutionStatus(t, status, "completed")

	// In dry run, no actual project should be created
	projectPath := filepath.Join(tmpDir, "test-hello-world")
	testutil.AssertFileNotExists(t, projectPath)
}

// TestExecutor_Execute_WithMock tests executor with mocked Claude client
func TestExecutor_Execute_WithMock(t *testing.T) {
	tests := []struct {
		name          string
		mockSetup     func(*mocks.MockClaudeClient)
		projectSpec   map[string]interface{}
		expectedState string
		wantErr       bool
	}{
		{
			name: "successful execution",
			mockSetup: func(m *mocks.MockClaudeClient) {
				m.WithCustomResult(&ExecutionResult{
					Success:     true,
					Output:      "Successfully created hello.txt",
					ProjectPath: "/tmp/test-project",
				})
			},
			projectSpec:   testutil.TestProjectSpec(),
			expectedState: "completed",
			wantErr:       false,
		},
		{
			name: "failed validation",
			mockSetup: func(m *mocks.MockClaudeClient) {
				m.WithFailedValidation(fmt.Errorf("claude CLI not found"))
			},
			projectSpec:   testutil.TestProjectSpec(),
			expectedState: "failed",
			wantErr:       true,
		},
		{
			name: "failed execution",
			mockSetup: func(m *mocks.MockClaudeClient) {
				m.WithFailedExecution()
			},
			projectSpec:   testutil.TestProjectSpec(),
			expectedState: "failed",
			wantErr:       true,
		},
		{
			name: "timeout during execution",
			mockSetup: func(m *mocks.MockClaudeClient) {
				m.WithExecutionDelay(2 * time.Second)
			},
			projectSpec:   testutil.TestProjectSpec(),
			expectedState: "failed",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and configure it
			mockClient := mocks.NewMockClaudeClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mockClient)
			}

			// Create executor with mock client
			executor := &Executor{client: &Client{mockClient: mockClient}}
			tmpDir := testutil.CreateTempProjectDir(t)

			// Execute with timeout for timeout test
			ctx := context.Background()
			if tt.name == "timeout during execution" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 1*time.Second)
				defer cancel()
			}

			status, err := executor.Execute(ctx, tt.projectSpec, tmpDir)

			// Validate error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Validate status
			if status == nil {
				t.Fatal("Execute() returned nil status")
			}

			testutil.AssertExecutionStatus(t, status, tt.expectedState)

			// Validate mock was called appropriately
			if !tt.wantErr || tt.name != "failed validation" {
				if !mockClient.ValidateCalled {
					t.Error("Expected ValidateClaudeCodeAvailability to be called")
				}
			}

			if !tt.wantErr {
				if !mockClient.ExecuteCalled {
					t.Error("Expected ExecuteProject to be called")
				}
			}
		})
	}
}

// TestExecutor_MetadataHandling tests metadata file creation and loading
func TestExecutor_MetadataHandling(t *testing.T) {
	tmpDir := testutil.CreateTempProjectDir(t)
	executor := NewExecutor("~/.claude", true) // Use dry run to avoid actual execution

	projectSpec := testutil.TestProjectSpec()
	status, err := executor.Execute(context.Background(), projectSpec, tmpDir)

	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify metadata file was created
	loadedStatus := testutil.AssertMetadataFile(t, tmpDir)

	// Compare loaded status with original
	if loadedStatus.State != status.State {
		t.Errorf("Loaded state %s doesn't match original %s", loadedStatus.State, status.State)
	}

	if loadedStatus.StartedAt != status.StartedAt {
		t.Errorf("Loaded StartedAt %s doesn't match original %s", loadedStatus.StartedAt, status.StartedAt)
	}
}

// TestExecutor_RetryExecution tests the retry mechanism
func TestExecutor_RetryExecution(t *testing.T) {
	mockClient := mocks.NewMockClaudeClient()

	// Configure mock to fail first two attempts, succeed on third
	failCount := 0
	originalExecute := mockClient.ExecuteProject
	mockClient.ExecuteProject = func(ctx context.Context, projectSpec map[string]interface{}, outputPath string) (*ExecutionResult, error) {
		failCount++
		if failCount <= 2 {
			return &ExecutionResult{
				Success: false,
				Error:   fmt.Sprintf("execution failed (attempt %d)", failCount),
			}, fmt.Errorf("execution failed (attempt %d)", failCount)
		}
		return originalExecute(ctx, projectSpec, outputPath)
	}

	executor := &Executor{client: &Client{mockClient: mockClient}}
	tmpDir := testutil.CreateTempProjectDir(t)
	projectSpec := testutil.TestProjectSpec()

	status, err := executor.RetryExecution(context.Background(), projectSpec, tmpDir, 3)

	if err != nil {
		t.Fatalf("RetryExecution() should have succeeded on 3rd attempt: %v", err)
	}

	testutil.AssertExecutionStatus(t, status, "completed")

	if mockClient.ExecuteCallCount != 3 {
		t.Errorf("Expected 3 execution attempts, got %d", mockClient.ExecuteCallCount)
	}
}

// TestExecutor_ProjectGenerated tests project generation checking
func TestExecutor_ProjectGenerated(t *testing.T) {
	tmpDir := testutil.CreateTempProjectDir(t)
	executor := NewExecutor("~/.claude", false)

	// Test with non-existent project
	nonExistentPath := filepath.Join(tmpDir, "non-existent")
	if executor.IsProjectGenerated(context.Background(), nonExistentPath) {
		t.Error("Expected IsProjectGenerated to return false for non-existent path")
	}

	// Create a project directory with some files
	projectPath := filepath.Join(tmpDir, "test-project")
	err := os.MkdirAll(projectPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	testutil.WriteTestFile(t, projectPath, "hello.txt", "Hello, World!")

	// Test with existing project
	if !executor.IsProjectGenerated(context.Background(), projectPath) {
		t.Error("Expected IsProjectGenerated to return true for existing project with files")
	}
}

// TestExecutor_CleanupProject tests project cleanup
func TestExecutor_CleanupProject(t *testing.T) {
	tmpDir := testutil.CreateTempProjectDir(t)
	executor := NewExecutor("~/.claude", false)

	// Create a project directory with files
	projectPath := filepath.Join(tmpDir, "test-project")
	err := os.MkdirAll(projectPath, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	testutil.WriteTestFile(t, projectPath, "hello.txt", "Hello, World!")
	testutil.WriteTestFile(t, projectPath, "README.md", "# Test Project")

	// Verify files exist
	testutil.AssertFileExists(t, filepath.Join(projectPath, "hello.txt"))
	testutil.AssertFileExists(t, filepath.Join(projectPath, "README.md"))

	// Cleanup project
	err = executor.CleanupProject(context.Background(), projectPath)
	if err != nil {
		t.Fatalf("CleanupProject() failed: %v", err)
	}

	// Verify directory is removed
	testutil.AssertFileNotExists(t, projectPath)
}

// Integration test - only runs with actual Claude CLI
func TestExecutor_IntegrationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testutil.SkipIfNoClaudeCredentials(t)

	tmpDir := testutil.CreateTempProjectDir(t)
	executor := NewExecutor("~/.claude", false)

	projectSpec := map[string]interface{}{
		"project": map[string]interface{}{
			"name":        "integration-test",
			"description": "Integration test with real Claude CLI",
			"version":     "1.0.0",
		},
		"requirements": []map[string]interface{}{
			{
				"name": "Create hello.txt",
				"instructions": []string{
					"Create a file named hello.txt",
					"Add content: Hello from Claude!",
				},
				"verification": map[string]string{
					"command": "test -f hello.txt && grep -q 'Hello from Claude!' hello.txt",
					"expect":  "success",
				},
			},
		},
	}

	// Run with timeout to prevent hanging
	testutil.WithTimeout(t, 5*time.Minute, func(ctx context.Context) {
		status, err := executor.Execute(ctx, projectSpec, tmpDir)
		if err != nil {
			t.Fatalf("Integration test failed: %v", err)
		}

		testutil.AssertExecutionStatus(t, status, "completed")

		// Verify the project was actually created
		projectPath := filepath.Join(tmpDir, "integration-test")
		testutil.AssertFileExists(t, projectPath)

		// Check if hello.txt was created with correct content
		helloPath := filepath.Join(projectPath, "hello.txt")
		if _, err := os.Stat(helloPath); err == nil {
			content := testutil.ReadTestFile(t, projectPath, "hello.txt")
			if content != "Hello from Claude!" {
				t.Logf("Unexpected file content: %s", content)
			}
		}

		// Cleanup
		if cleanupErr := executor.CleanupProject(ctx, projectPath); cleanupErr != nil {
			t.Logf("Cleanup failed: %v", cleanupErr)
		}
	})
}

// Benchmark tests
func BenchmarkExecutor_Execute_DryRun(b *testing.B) {
	tmpDir := b.TempDir()
	executor := NewExecutor("~/.claude", true)
	projectSpec := testutil.TestProjectSpec()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.Execute(context.Background(), projectSpec, tmpDir)
		if err != nil {
			b.Fatalf("Execute() failed: %v", err)
		}
	}
}
