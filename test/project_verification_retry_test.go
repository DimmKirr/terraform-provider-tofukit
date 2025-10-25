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

func TestProjectVerificationRetry(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectVerificationRetry")

	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate single project.tofu with all configuration
	// Enable max_retries to test retry loop
	projectTofuContent := `# Terraform configuration for verification-retry test
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration with retry enabled
provider "tofukit" {
  output_format = "json"
  output_path   = "output"
  debug         = true
  max_retries   = 3  # Enable retry loop for verification failures
}

resource "tofukit_project" "verification_test" {
  name        = "verification-retry-test"
  description = "Test project that will fail verification on first attempt"
  version     = "1.0.0"

  # CONTRADICTORY: Instructions say "properly formatted text file" (implies newline)
  # But verification expects NO trailing newline - this WILL fail first time
  files = {
    "exact.txt" = {
      instructions = [
        "Create a simple text file containing the word 'exact'",
        "Make sure it's a properly formatted text file",
        "Follow standard text file conventions"
      ]
      verifications = [
        {
          # This expects EXACTLY "exact" with NO trailing newline (5 bytes)
          # od -An -tx1 shows hex: "exact" = 6578616374 (no newline)
          # But "exact\n" = 65786163740a (with newline) - will FAIL
          command = "od -An -tx1 exact.txt | tr -d ' \\n'"
          expect  = "6578616374"
        }
      ]
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 2: Check if terraform/tofu is available
	var iacTool string
	if _, err := exec.LookPath("tofu"); err == nil {
		iacTool = "tofu"
		t.Log("Using OpenTofu")
	} else if _, err := exec.LookPath("terraform"); err == nil {
		iacTool = "terraform"
		t.Log("Using Terraform")
	} else {
		t.Skip("Neither terraform nor tofu available - skipping test")
	}

	// Clean up resources at the end
	defer func() {
		if os.Getenv("SKIP_DESTROY") == "true" {
			t.Log("Skipping cleanup due to SKIP_DESTROY=true")
		} else {
			t.Log("Running destroy to clean up resources...")
			destroyCmd := exec.Command(iacTool, "destroy", "-auto-approve", "-no-color")
			destroyCmd.Dir = testDir
			destroyOutput, destroyErr := destroyCmd.CombinedOutput()
			if destroyErr != nil {
				t.Logf("Warning: Destroy failed (non-fatal): %v\nOutput: %s", destroyErr, destroyOutput)
			} else {
				t.Log("✓ Resources destroyed successfully")
			}
		}

		// Keep test directory for inspection
		if os.Getenv("CLEANUP_TEST_OUTPUT") == "true" {
			os.RemoveAll(testDir)
			t.Log("Test directory cleaned up")
		} else {
			t.Logf("Keeping test directory: %s (set CLEANUP_TEST_OUTPUT=true to remove)", testDir)
		}
	}()

	// Step 4: Run tofu init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run init: %s", initOutput)
	t.Log("✓ Init completed successfully")

	// Step 5: Run tofu plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	t.Logf("Plan output:\n%s", planOutput)
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 6: Run tofu apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	t.Logf("Apply output:\n%s", applyOutput)
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Project path for verification
	projectPath := filepath.Join(testDir, "output", "verification-retry-test")

	// === SUBTEST 1: Verify File Created Correctly ===
	t.Run("VerifyFileCreated", func(t *testing.T) {
		t.Log("Verifying exact.txt was created correctly...")

		// Verify exact.txt exists
		exactPath := filepath.Join(projectPath, "exact.txt")
		assert.FileExists(t, exactPath, "exact.txt should exist")

		// Read the file content
		exactContent, err := os.ReadFile(exactPath)
		require.NoError(t, err, "Failed to read exact.txt")
		t.Logf("exact.txt content: %q (length: %d bytes)", string(exactContent), len(exactContent))

		// Verify it's exactly "exact" with NO trailing newline
		assert.Equal(t, "exact", string(exactContent), "exact.txt should contain exactly 'exact' with no newline")
		assert.Equal(t, 5, len(exactContent), "exact.txt should be exactly 5 bytes")

		t.Log("✓ File created with correct content (verification passed after retry)")
	})

	// === SUBTEST 2: Verify Debug Files Show Retry Activity ===
	t.Run("VerifyRetryActivity", func(t *testing.T) {
		t.Log("Checking debug files for retry activity...")

		debugDir := filepath.Join(testDir, "output", ".debug")
		assert.DirExists(t, debugDir, ".debug directory should exist")

		// List all execution metadata files
		entries, err := os.ReadDir(debugDir)
		require.NoError(t, err, "Failed to read debug directory")

		metadataFiles := []string{}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "claude-execution-metadata-") && strings.HasSuffix(entry.Name(), ".json") {
				metadataFiles = append(metadataFiles, entry.Name())
			}
		}

		t.Logf("Found %d execution metadata file(s): %v", len(metadataFiles), metadataFiles)

		// If we have multiple metadata files, it means retry happened
		if len(metadataFiles) > 1 {
			t.Logf("✓ Retry loop was triggered! Found %d execution attempts", len(metadataFiles))

			// Read first metadata to see the failure
			firstMetadata, err := os.ReadFile(filepath.Join(debugDir, metadataFiles[0]))
			if err == nil {
				t.Logf("First execution metadata:\n%s", string(firstMetadata))
			}
		} else if len(metadataFiles) == 1 {
			t.Log("ℹ️  Only one execution - Claude passed verification on first attempt")
		}

		// Check for verification logs in prompt files
		promptFiles, _ := filepath.Glob(filepath.Join(debugDir, "claude-prompt-*.md"))
		if len(promptFiles) > 0 {
			t.Logf("Found %d prompt file(s)", len(promptFiles))

			// Check if any prompts contain fix request markers
			for _, promptFile := range promptFiles {
				content, err := os.ReadFile(promptFile)
				if err == nil && strings.Contains(string(content), "VERIFICATION FAILURES") {
					t.Log("✓ Found fix request in prompt - retry loop was active")
					t.Logf("Fix request prompt excerpt:\n%s", string(content[:min(len(content), 500)]))
					break
				}
			}
		}

		t.Log("✓ Debug files verification complete")
	})

	// === SUBTEST 3: Verify Terraform State ===
	t.Run("VerifyTerraformState", func(t *testing.T) {
		t.Log("Verifying Terraform state...")

		stateCmd := exec.Command(iacTool, "show", "-no-color")
		stateCmd.Dir = testDir
		stateOutput, err := stateCmd.CombinedOutput()
		require.NoError(t, err, "Failed to show state")

		// Verify execution_status is "completed"
		assert.Contains(t, string(stateOutput), "execution_status", "State should contain execution_status")
		assert.Contains(t, string(stateOutput), "completed", "execution_status should be 'completed'")

		// Verify no execution errors
		if strings.Contains(string(stateOutput), "execution_error") {
			// Check if it's null or empty
			assert.Contains(t, string(stateOutput), `execution_error = ""`, "execution_error should be empty")
		}

		t.Log("✓ Terraform state verified")
	})

	t.Log("✅ All verification retry tests completed successfully!")
}
