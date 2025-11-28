package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderClaudeMaxTurnsMustFail validates that setting claude_max_turns=2
// causes Claude to fail when the project requires more turns
func TestProviderClaudeMaxTurnsMustFail(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProviderClaudeMaxTurnsMustFail")
	t.Log("Testing claude_max_turns=2 failure...")

	// Create a Terraform configuration that sets claude_max_turns=2
	// The project will require more than 2 turns to complete, so it should fail
	projectConfig := `
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  claude_max_turns      = 2
  debug                 = true
}

resource "tofukit_project" "test_turns" {
  name = "test-max-turns"
  description = "Test project that should fail due to max_turns=2"
  version = "1.0.0"

  requirements = [
    {
      name = "Multi-file project"
      instructions = [
        {
          prompt = "Create a simple Python CLI application with the following files: main.py (entry point with main function), config.py (configuration with APP_NAME constant), utils.py (utilities with helper function), README.md (documentation with installation section), and requirements.txt (dependencies with at least click package). Each file should have meaningful content."
        }
      ]
      verifications = [
        {
          command = "test -f main.py"
        },
        {
          command = "grep -q 'def main' main.py"
        },
        {
          command = "test -f config.py"
        },
        {
          command = "grep -q 'APP_NAME' config.py"
        },
        {
          command = "test -f utils.py"
        },
        {
          command = "grep -q 'def' utils.py"
        },
        {
          command = "test -f README.md"
        },
        {
          command = "grep -q -i 'installation' README.md"
        },
        {
          command = "test -f requirements.txt"
        },
        {
          command = "grep -q 'click' requirements.txt"
        }
      ]
    }
  ]
}
`

	// Write the project.tofu to test directory
	projectPath := filepath.Join(testDir, "project.tofu")
	require.NoError(t, os.WriteFile(projectPath, []byte(projectConfig), 0644))

	// Detect IaC tool
	iacTool := detectIaCTool(t)

	// Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Run plan (should succeed - plan just prepares the prompt)
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Run apply - THIS SHOULD FAIL due to max_turns=2
	t.Log("Running tofu apply --auto-approve (expecting failure)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()

	// Apply MUST fail
	if err == nil {
		t.Fatalf("Expected apply to fail due to claude_max_turns=2, but it succeeded!\nOutput: %s", applyOutput)
	}

	// Verify the error is related to reaching max turns
	outputStr := string(applyOutput)
	t.Logf("Apply output: %s", outputStr)

	// Check for max turns related messages
	// Claude CLI typically outputs "max turns" when the limit is reached
	hasMaxTurnsMessage := strings.Contains(strings.ToLower(outputStr), "max turns") ||
		strings.Contains(strings.ToLower(outputStr), "turn limit") ||
		strings.Contains(strings.ToLower(outputStr), "turns reached")

	assert.True(t, hasMaxTurnsMessage,
		"Expected error message to contain 'max turns', 'turn limit', or 'turns reached'\nActual output: %s",
		outputStr)

	t.Log("✓ Test passed: Apply failed as expected due to claude_max_turns=2")

	// Verify debug files were created to show the max_turns setting
	debugDir := filepath.Join(testDir, "output", ".debug")
	if _, err := os.Stat(debugDir); err == nil {
		t.Logf("Debug directory exists: %s", debugDir)
	}
}
