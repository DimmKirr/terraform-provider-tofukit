package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProjectDriftDetection_StaticFile verifies drift detection for modified static files
func TestProjectDriftDetection_StaticFile(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectDriftDetection_StaticFile")

	var err error

	// Step 1: Create project configuration with static .gitignore
	projectTofuContent := `# Terraform configuration for drift detection test
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

resource "tofukit_project" "drift_test" {
  name        = "drift-test"
  output_path = "output"

  files = {
    ".gitignore" = {
      content = <<-EOF
        *.log
        *.tmp
      EOF
    }
  }
}
`

	projectTofuPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(projectTofuPath, []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 2: Initialize Terraform
	initCmd := exec.Command("tofu", "init")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "tofu init failed: %s", string(initOutput))

	// Step 3: Apply to create initial project
	applyCmd := exec.Command("tofu", "apply", "-auto-approve")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply failed: %s", string(applyOutput))

	t.Log("Initial apply completed successfully")

	// Step 4: Verify initial state - no drift
	showCmd := exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err := showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed: %s", string(showOutput))

	stateJSON := string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected": false`, "Initial state should have drift_detected=false")

	// Step 5: Manually edit .gitignore outside Terraform
	gitignorePath := filepath.Join(testDir, "output", ".gitignore")
	err = os.WriteFile(gitignorePath, []byte("*.log\n*.tmp\n*.cache\n"), 0644)
	require.NoError(t, err, "Failed to manually edit .gitignore")

	t.Log("Manually edited .gitignore to add *.cache")

	// Step 6: Run plan to detect drift
	planCmd := exec.Command("tofu", "plan", "-detailed-exitcode")
	planCmd.Dir = testDir
	planOutput, _ := planCmd.CombinedOutput() // Expecting exit code 2 (changes detected)

	planOutputStr := string(planOutput)
	t.Logf("Plan output:\n%s", planOutputStr)

	// Plan should show changes due to drift
	assert.Contains(t, planOutputStr, "drift_detected", "Plan should show drift_detected change")
	assert.Contains(t, planOutputStr, ".gitignore", "Plan should show .gitignore as drifted")

	// Step 7: Refresh state to detect drift
	refreshCmd := exec.Command("tofu", "apply", "-refresh-only", "-auto-approve")
	refreshCmd.Dir = testDir
	refreshOutput, err := refreshCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply -refresh-only failed: %s", string(refreshOutput))

	t.Log("Refreshed state to detect drift")

	// Step 8: Verify drift detected in state
	showCmd = exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err = showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed after refresh: %s", string(showOutput))

	stateJSON = string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected": true`, "State should have drift_detected=true after refresh")
	assert.Contains(t, stateJSON, ".gitignore", "State should list .gitignore as drifted")

	// Step 9: Apply to restore file
	applyCmd = exec.Command("tofu", "apply", "-auto-approve")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply (restore) failed: %s", string(applyOutput))

	t.Log("Applied to restore drifted file")

	// Step 10: Verify file content restored
	restoredContent, err := os.ReadFile(gitignorePath)
	require.NoError(t, err, "Failed to read restored .gitignore")

	expectedContent := "*.log\n*.tmp\n"
	assert.Equal(t, expectedContent, string(restoredContent), "File content should be restored to original")

	// Step 11: Verify drift cleared in state
	showCmd = exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err = showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed after restore: %s", string(showOutput))

	stateJSON = string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected": false`, "State should have drift_detected=false after restore")

	t.Log("✓ Drift detection and restoration completed successfully")
}

// TestProjectDriftDetection_DeletedFile verifies drift detection for deleted files
func TestProjectDriftDetection_DeletedFile(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectDriftDetection_DeletedFile")

	var err error

	// Step 1: Create project configuration with README.md
	projectTofuContent := `# Terraform configuration for deleted file drift test
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

resource "tofukit_project" "drift_test" {
  name        = "drift-delete-test"
  output_path = "output"

  files = {
    "README.md" = {
      content = "# Test Project\n"
    }
  }
}
`

	projectTofuPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(projectTofuPath, []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 2: Initialize and apply
	initCmd := exec.Command("tofu", "init")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "tofu init failed: %s", string(initOutput))

	applyCmd := exec.Command("tofu", "apply", "-auto-approve")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply failed: %s", string(applyOutput))

	t.Log("Initial apply completed successfully")

	// Step 3: Delete README.md manually
	readmePath := filepath.Join(testDir, "output", "README.md")
	err = os.Remove(readmePath)
	require.NoError(t, err, "Failed to delete README.md")

	t.Log("Manually deleted README.md")

	// Step 4: Refresh state to detect deletion
	refreshCmd := exec.Command("tofu", "apply", "-refresh-only", "-auto-approve")
	refreshCmd.Dir = testDir
	refreshOutput, err := refreshCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply -refresh-only failed: %s", string(refreshOutput))

	// Step 5: Verify drift detected
	showCmd := exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err := showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed: %s", string(showOutput))

	stateJSON := string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected": true`, "State should detect drift from deleted file")
	assert.Contains(t, stateJSON, "README.md", "State should list README.md as drifted")

	// Step 6: Apply to recreate file
	applyCmd = exec.Command("tofu", "apply", "-auto-approve")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply (recreate) failed: %s", string(applyOutput))

	t.Log("Applied to recreate deleted file")

	// Step 7: Verify file recreated
	_, err = os.Stat(readmePath)
	require.NoError(t, err, "README.md should be recreated")

	content, err := os.ReadFile(readmePath)
	require.NoError(t, err, "Failed to read recreated README.md")
	assert.Equal(t, "# Test Project\n", string(content), "Recreated file should have correct content")

	// Step 8: Verify drift cleared
	showCmd = exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err = showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed after recreate: %s", string(showOutput))

	stateJSON = string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected": false`, "State should have drift_detected=false after recreate")

	t.Log("✓ Deleted file drift detection and restoration completed successfully")
}

// TestProjectDriftDetection_MultipleFiles verifies drift detection for multiple modified files
func TestProjectDriftDetection_MultipleFiles(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectDriftDetection_MultipleFiles")

	var err error

	// Step 1: Create project configuration with multiple files
	projectTofuContent := `# Terraform configuration for multi-file drift test
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

resource "tofukit_project" "drift_test" {
  name        = "drift-multi-test"
  output_path = "output"

  files = {
    ".gitignore" = {
      content = "*.log\n"
    }
    "README.md" = {
      content = "# Project\n"
    }
    "LICENSE" = {
      content = "MIT License\n"
    }
  }
}
`

	projectTofuPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(projectTofuPath, []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 2: Initialize and apply
	initCmd := exec.Command("tofu", "init")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "tofu init failed: %s", string(initOutput))

	applyCmd := exec.Command("tofu", "apply", "-auto-approve")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply failed: %s", string(applyOutput))

	t.Log("Initial apply completed successfully")

	// Step 3: Verify initial state - no drift
	showCmd := exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err := showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed: %s", string(showOutput))

	stateJSON := string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected": false`, "Initial state should have drift_detected=false")

	// Step 4: Edit 2 out of 3 files manually
	outputDir := filepath.Join(testDir, "output")

	// Edit .gitignore
	gitignorePath := filepath.Join(outputDir, ".gitignore")
	err = os.WriteFile(gitignorePath, []byte("*.log\n*.cache\n"), 0644)
	require.NoError(t, err, "Failed to edit .gitignore")

	// Edit LICENSE
	licensePath := filepath.Join(outputDir, "LICENSE")
	err = os.WriteFile(licensePath, []byte("Apache License\n"), 0644)
	require.NoError(t, err, "Failed to edit LICENSE")

	// Leave README.md unchanged

	t.Log("Manually edited .gitignore and LICENSE")

	// Step 5: Refresh state to detect drift
	refreshCmd := exec.Command("tofu", "apply", "-refresh-only", "-auto-approve")
	refreshCmd.Dir = testDir
	refreshOutput, err := refreshCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply -refresh-only failed: %s", string(refreshOutput))

	t.Log("Refreshed state to detect drift")

	// Step 6: Verify drift detected for both files
	showCmd = exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err = showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed after refresh: %s", string(showOutput))

	stateJSON = string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected": true`, "State should have drift_detected=true")
	assert.Contains(t, stateJSON, ".gitignore", "State should list .gitignore as drifted")
	assert.Contains(t, stateJSON, "LICENSE", "State should list LICENSE as drifted")
	assert.NotContains(t, stateJSON, `"README.md"`, "README.md should NOT be in drifted files (unchanged)")

	// Step 7: Apply to restore files
	applyCmd = exec.Command("tofu", "apply", "-auto-approve")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply (restore) failed: %s", string(applyOutput))

	t.Log("Applied to restore drifted files")

	// Step 8: Verify both files restored
	restoredGitignore, err := os.ReadFile(gitignorePath)
	require.NoError(t, err, "Failed to read restored .gitignore")
	assert.Equal(t, "*.log\n", string(restoredGitignore), ".gitignore should be restored")

	restoredLicense, err := os.ReadFile(licensePath)
	require.NoError(t, err, "Failed to read restored LICENSE")
	assert.Equal(t, "MIT License\n", string(restoredLicense), "LICENSE should be restored")

	// Verify README unchanged
	readmeContent, err := os.ReadFile(filepath.Join(outputDir, "README.md"))
	require.NoError(t, err, "Failed to read README.md")
	assert.Equal(t, "# Project\n", string(readmeContent), "README.md should remain unchanged")

	// Step 9: Verify drift cleared
	showCmd = exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err = showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed after restore: %s", string(showOutput))

	stateJSON = string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected": false`, "State should have drift_detected=false after restore")

	t.Log("✓ Multiple file drift detection and restoration completed successfully")
}
