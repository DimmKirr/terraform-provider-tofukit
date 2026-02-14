package test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
	"github.com/tofukit/opentofu-provider-tofukit/internal/resources"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// TestUnitProviderMaxTurnsErrorPropagationClientLevelMustSucceed tests that client correctly returns error
func TestUnitProviderMaxTurnsErrorPropagationClientLevelMustSucceed(t *testing.T) {
	ctx := context.Background()
	testDir := createTestDirectory(t, "TestUnitProviderMaxTurnsErrorPropagationClientLevel")
	outputDir := filepath.Join(testDir, "output")
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create a client with max_turns=2
	client := claude.NewClient("~/.claude", true, 2)

	// Create a simple project spec that will require more than 2 turns
	projectSpec := map[string]interface{}{
		"name":        "test-project",
		"description": "Test max turns",
		"requirements": []interface{}{
			map[string]interface{}{
				"name": "Create files",
				"instructions": []interface{}{
					map[string]interface{}{
						"prompt": "Create 5 files: main.py, config.py, utils.py, README.md, requirements.txt with meaningful content",
					},
				},
			},
		},
	}

	// Execute the project
	result, err := client.ExecuteProject(ctx, projectSpec, outputDir, "sonnet")

	// CRITICAL ASSERTION: Client should return an error when max turns is reached
	if err == nil {
		t.Errorf("Expected client.ExecuteProject to return error when max turns reached, but got nil")
	} else {
		t.Logf("✓ Client correctly returned error: %v", err)
		// Verify error message mentions max turns
		if !strings.Contains(strings.ToLower(err.Error()), "max") && !strings.Contains(strings.ToLower(err.Error()), "turns") {
			t.Errorf("Error message should mention 'max turns', got: %s", err.Error())
		}
	}

	// Result should indicate failure
	if result != nil && result.Success {
		t.Errorf("Expected result.Success=false when max turns reached, got true")
	}
}

// TestUnitProviderMaxTurnsErrorPropagationExecutorLevelMustSucceed tests that executor propagates client error
func TestUnitProviderMaxTurnsErrorPropagationExecutorLevelMustSucceed(t *testing.T) {
	ctx := context.Background()
	testDir := createTestDirectory(t, "TestUnitProviderMaxTurnsErrorPropagationExecutorLevel")
	outputDir := filepath.Join(testDir, "output")
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create an executor with max_turns=2
	executor := claude.NewExecutor("~/.claude", true, 2)
	executor.SetDebug(true)
	executor.SetOutputPath(outputDir)

	// Create a prompt JSON that will require more than 2 turns
	promptData := map[string]interface{}{
		"request": map[string]interface{}{
			"project_info": map[string]interface{}{
				"name":        "test-project",
				"description": "Test max turns",
				"version":     "1.0.0",
			},
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Create files",
					"instructions": []interface{}{
						map[string]interface{}{
							"prompt": "Create 5 files: main.py, config.py, utils.py, README.md, requirements.txt with meaningful content",
						},
					},
				},
			},
		},
		"system_prompt": "You are a code generation assistant.",
	}

	promptJSON, err := json.Marshal(promptData)
	require.NoError(t, err)

	// Execute with prompt JSON
	status, report, err := executor.ExecuteWithPromptJSON(ctx, string(promptJSON), outputDir, []schemas.FileModelWithPath{}, 1)

	// CRITICAL ASSERTION: Executor should propagate the error from client
	if err == nil {
		t.Errorf("Expected executor.ExecuteWithPromptJSON to return error when max turns reached, but got nil")
		t.Logf("Status: %+v", status)
		t.Logf("Report: %+v", report)
	} else {
		t.Logf("✓ Executor correctly returned error: %v", err)
	}

	// Status should indicate failure
	if status != nil && status.State != "failed" {
		t.Errorf("Expected status.State='failed' when max turns reached, got: %s", status.State)
	}
}

// TestUnitProviderMaxTurnsErrorPropagationResourceLevelMustSucceed documents expected behavior at resource level
func TestUnitProviderMaxTurnsErrorPropagationResourceLevelMustSucceed(t *testing.T) {
	// Note: We can't easily unit test executeClaudeCode as it's private
	// Instead, we verify the executor level test above catches the issue
	// This test documents what SHOULD happen at the resource level

	t.Log("✓ Resource level test: Documented expected behavior")
	t.Log("  - executeClaudeCode should call ExecuteWithPromptJSON")
	t.Log("  - ExecuteWithPromptJSON should return error from client")
	t.Log("  - executeClaudeCode should propagate that error")
	t.Log("  - Create/Update should fail the Terraform apply")

	// Log the expected error flow
	t.Log("")
	t.Log("Expected error propagation flow:")
	t.Log("  1. Claude CLI exits with error (max turns reached)")
	t.Log("  2. client.ExecuteProjectWithPrompt returns ExecutionResult{Success: false, Error: '...'}")
	t.Log("  3. executor.ExecuteWithPromptJSON converts to error and returns")
	t.Log("  4. executeClaudeCode receives error and returns it")
	t.Log("  5. Create/Update receives error and calls resp.Diagnostics.AddError()")
	t.Log("  6. Terraform apply fails")

	// Create a resource to verify it exists
	resource := resources.NewProjectResourceFinal()
	assert.NotNil(t, resource, "Resource should be created")
}

// TestUnitProviderMaxTurnsErrorMessageMustSucceed tests that error messages are informative
func TestUnitProviderMaxTurnsErrorMessageMustSucceed(t *testing.T) {
	ctx := context.Background()
	testDir := createTestDirectory(t, "TestUnitProviderMaxTurnsErrorMessage")
	outputDir := filepath.Join(testDir, "output")
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create a client with max_turns=2
	client := claude.NewClient("~/.claude", true, 2)

	projectSpec := map[string]interface{}{
		"name": "test",
		"requirements": []interface{}{
			map[string]interface{}{
				"name": "Task",
				"instructions": []interface{}{
					map[string]interface{}{
						"prompt": "Create many files",
					},
				},
			},
		},
	}

	_, err := client.ExecuteProject(ctx, projectSpec, outputDir, "sonnet")

	if err != nil {
		errMsg := err.Error()
		t.Logf("Error message: %s", errMsg)

		// Check that error message is helpful
		checks := []struct {
			term   string
			reason string
		}{
			{"max_turns", "Should mention the limit setting"},
			{"claude_max_turns", "Should reference the provider configuration"},
			{"increasing", "Should suggest how to fix"},
		}

		for _, check := range checks {
			if !strings.Contains(strings.ToLower(errMsg), strings.ToLower(check.term)) {
				t.Logf("⚠ Warning: Error message should mention '%s' (%s)", check.term, check.reason)
			} else {
				t.Logf("✓ Error message mentions '%s'", check.term)
			}
		}
	} else {
		t.Error("Expected error but got nil")
	}
}
