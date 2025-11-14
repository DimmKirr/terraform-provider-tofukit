package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestKitVerificationEnforcement validates that kit verifications are actually run and enforced
// This is an end-to-end test that runs 'tofu init' and 'tofu apply' to verify the provider
// enforces kit verification failures and returns appropriate error messages.
func TestKitVerificationEnforcement(t *testing.T) {
	testDir := createTestDirectory(t, "TestKitVerificationEnforcement")

	// Create a project that requires a non-existent tool
	projectContent := `
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format = "json"
  output_path   = "output"
  debug         = true
}

resource "tofukit_tool" "fake_tool" {
  name        = "nonexistent-tool"
  description = "A tool that doesn't exist"
  version     = "1.0.0"

  requirements = [
    {
      name = "Fake Tool Installation"

      instructions = [{
        prompt = "This tool doesn't exist and can't be installed"
      }]

      verifications = [
        {
          command = "false"
          expect  = ""
        }
      ]
    }
  ]
}

resource "tofukit_project" "test" {
  name        = "kit-verification-test"
  description = "Test that kit verifications are enforced"
  version     = "1.0.0"

  kits = [tofukit_tool.fake_tool]

  files = {
    "test.txt" = {
      content = "hello\n"
    }
  }
}
`

	projectPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	// Run tofu init
	iacTool := detectIaCTool(t)
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "Init should succeed: %s", string(initOutput))

	// Run tofu apply - should FAIL because kit verification fails
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()

	// Expect FAILURE with verification error message
	require.Error(t, err, "Apply should fail when kit verification fails")
	require.Contains(t, string(applyOutput), "verification failed",
		"Output should mention verification failure")
	require.Contains(t, string(applyOutput), "Command 'false'",
		"Output should mention the failed command")
}
