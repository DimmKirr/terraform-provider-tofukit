package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProjectVerificationSuccess tests that verification passes when expectations are met
func TestProjectVerificationSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectVerificationSuccess")
	t.Log("Testing verification success...")

	var err error

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  debug                 = true
  claude_home_directory = "~/.claude"
}

resource "tofukit_stack" "test_stack" {
  name        = "test-verification-stack"
  description = "Stack with correct verification"

  files = {
    "hello.txt" = {
      content = "hello world"

      # This verification will pass - content matches expectation
      verifications = [
        {
          command = "cat hello.txt"
          expect  = "hello world"
        }
      ]
    }
  }
}

resource "tofukit_project" "verification_test" {
  name        = "verification-test"
  description = "Project to test verification success"
  version     = "1.0.0"

  depends_on = [tofukit_stack.test_stack]
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

	// Step 5: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 6: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 7: Run apply - This should SUCCEED with passing verification
	t.Log("Running tofu apply --auto-approve (expecting success)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()

	if err != nil {
		t.Logf("Apply output: %s", applyOutput)
		t.Fatalf("Apply should succeed with correct verification, but failed: %v", err)
	}

	t.Log("✓ Apply succeeded with passing verification")

	// Step 8: Verify the file was created with correct content
	helloPath := filepath.Join(testDir, "output", "hello.txt")
	assert.FileExists(t, helloPath, "hello.txt should exist after successful apply")
	verifyFileContent(t, helloPath, "hello world")

	t.Log("✅ Verification success test completed!")
}

// TestProjectVerificationFailure tests that verification failures are properly detected and reported
func TestProjectVerificationFailure(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectVerificationFailure")
	t.Log("Testing verification failure detection...")

	var err error

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  debug                 = true
  claude_home_directory = "~/.claude"
}

resource "tofukit_stack" "test_stack" {
  name        = "test-verification-stack"
  description = "Stack with verification that should fail"

  files = {
    "hello.txt" = {
      content = "hello world"

      # This verification will fail because the file contains "hello world" not "goodbye world"
      verifications = [
        {
          command = "cat hello.txt"
          expect  = "goodbye world"
        }
      ]
    }
  }
}

resource "tofukit_project" "verification_test" {
  name        = "verification-test"
  description = "Project to test verification failure"
  version     = "1.0.0"

  depends_on = [tofukit_stack.test_stack]
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

	// Step 5: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 6: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 7: Run apply - This should FAIL due to verification failure
	t.Log("Running tofu apply --auto-approve (expecting verification failure)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()

	// We expect apply to fail due to verification failure
	if err != nil {
		t.Log("✓ Apply failed as expected due to verification failure")
		t.Logf("Apply output: %s", applyOutput)

		// Verify the error message contains information about verification failure
		outputStr := string(applyOutput)
		assert.Contains(t, outputStr, "verification", "Error should mention verification")

		t.Log("✅ Verification failure detection test completed!")
	} else {
		// If apply succeeded when it should have failed, that's a test failure
		t.Errorf("❌ Apply succeeded but should have failed due to verification mismatch")
		t.Logf("Apply output: %s", applyOutput)
	}
}
