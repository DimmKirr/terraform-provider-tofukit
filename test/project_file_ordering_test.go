package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProjectFileOrderingConsistency verifies that files maintain their config order
// between plan and apply, preventing "Provider produced inconsistent result" errors.
// This specifically tests the bug where alphabetical sorting caused apply to return
// files in a different order than plan expected.
func TestProjectFileOrderingConsistency(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileOrderingConsistency")

	var err error

	// Create config with files in NON-ALPHABETICAL order
	// This would trigger the bug: hi2.txt comes before LICENSE.md in config,
	// but alphabetically LICENSE.md < hi2.txt
	projectTofuContent := `terraform {
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

resource "tofukit_project" "hello_world" {
  name        = "file-ordering-test"
  description = "Test file ordering consistency"
  version     = "1.0.0"

  # Files in non-alphabetical order (hi2.txt before LICENSE.md)
  files = {
    "hi2.txt" = {
      content = "Hi from file 2\n"
    }

    "LICENSE.md" = {
      content = "MIT License\n"
    }

    "README.md" = {
      content = "Project README\n"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Check for tofu/terraform
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

	// Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run init: %s", initOutput)
	t.Log("✓ Init completed successfully")

	// Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Run apply - THIS IS WHERE THE BUG MANIFESTS
	// Before the fix, this would fail with:
	// "Provider produced inconsistent result after apply"
	// ".file[0].path: was cty.StringVal("hi2.txt"), but now cty.StringVal("LICENSE.md")"
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()

	// Check for the specific inconsistency error
	applyOutputStr := string(applyOutput)
	if strings.Contains(applyOutputStr, "Provider produced inconsistent result") {
		t.Errorf("❌ APPLY BUG DETECTED: Provider produced inconsistent result after apply")
		t.Logf("This indicates files were returned in different order than planned")
		t.Logf("Apply output:\n%s", applyOutputStr)
	}

	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully without consistency errors")

	// Verify files exist in correct order
	projectPath := filepath.Join(testDir, "output")
	hi2Path := filepath.Join(projectPath, "hi2.txt")
	licensePath := filepath.Join(projectPath, "LICENSE.md")
	readmePath := filepath.Join(projectPath, "README.md")

	require.FileExists(t, hi2Path, "hi2.txt should exist")
	require.FileExists(t, licensePath, "LICENSE.md should exist")
	require.FileExists(t, readmePath, "README.md should exist")

	verifyFileContent(t, hi2Path, "Hi from file 2")
	verifyFileContent(t, licensePath, "MIT License")
	verifyFileContent(t, readmePath, "Project README")

	t.Log("✅ File ordering consistency test passed!")
}
