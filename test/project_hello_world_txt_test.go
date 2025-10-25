package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectHelloWorldTxtCreateSuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectHelloWorldTxtCreateSuccess")

	// CLAUDE_HOME is not needed when using --dangerously-skip-permissions
	// The flag bypasses all authentication and permission checks
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `# Terraform configuration for hello-world-txt test
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration
provider "tofukit" {
  output_format         = "json"
  output_path           = "output"  # Output will be in {test_directory}/output/
  # Claude execution is now required for all tests
  debug                 = true       # Enable debug mode for detailed output
  # claude_home_directory not needed with --dangerously-skip-permissions
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  # Single file file - no dependencies needed
  file {
    path = "hello.txt"
    content = "hello world\n"
  }

  file {
    path = "hello2.txt"
    content = "hello world2\n"
  }

  file {
    path = "hello3.txt"
    content = "hello world3\n"
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 4: Check if terraform/tofu is available
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
		// Skip cleanup if debug mode is enabled to preserve debug files
		if os.Getenv("SKIP_DESTROY") == "true" {
			t.Log("Skipping cleanup due to SKIP_DESTROY=true")
		} else {
			// Check if debug mode is enabled in the test configuration
			debugMode := true // We know debug=true is set in the provider config
			if debugMode {
				t.Log("Debug mode enabled - preserving output directory for inspection")
				t.Logf("Debug files should be in: %s/output/.debug/", testDir)
			} else {
				// For dev overrides, we clean up the output directory manually
				t.Log("Cleaning up test output directory")
				outputPath := filepath.Join(testDir, "output")
				if err := os.RemoveAll(outputPath); err != nil {
					t.Logf("Failed to clean up output directory: %v", err)
				}
			}
		}
	}()

	// Step 5: Run init to set up the provider
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
	t.Logf("Plan output:\n%s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 7: Run apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Project path - files created directly in output directory (no subdirectory)
	projectPath := filepath.Join(testDir, "output")

	// === SUBTEST 1: Initial Creation ===
	t.Log("Testing initial file creation...")

	// Verify all three initial files were created
	helloPath := filepath.Join(projectPath, "hello.txt")
	assert.FileExists(t, helloPath, "hello.txt should exist")
	verifyFileContent(t, helloPath, "hello world\n")

	hello2Path := filepath.Join(projectPath, "hello2.txt")
	assert.FileExists(t, hello2Path, "hello2.txt should exist")
	verifyFileContent(t, hello2Path, "hello world2\n")

	hello3Path := filepath.Join(projectPath, "hello3.txt")
	assert.FileExists(t, hello3Path, "hello3.txt should exist")
	verifyFileContent(t, hello3Path, "hello world3\n")

	t.Log("✓ Initial file creation successful")

	// === SUBTEST 2: Debug Files Verification ===
	t.Run("DebugFilesVerification", func(t *testing.T) {
		t.Log("Verifying debug files...")

		// Check for debug specification file with timestamp pattern
		debugSpecPattern := filepath.Join(testDir, "output", ".debug", "project-*.json")
		debugSpecFiles, _ := filepath.Glob(debugSpecPattern)
		if len(debugSpecFiles) > 0 {
			t.Logf("✓ Debug specification file found: %s", debugSpecFiles[0])
			// Verify content
			spec, err := os.ReadFile(debugSpecFiles[0])
			require.NoError(t, err)
			assert.Contains(t, string(spec), "hello-world")
			assert.Contains(t, string(spec), "files")
			t.Log("  This file contains the project specification that would be sent to Claude")
		} else {
			t.Logf("⚠️ Debug specification file not found (pattern: %s)", debugSpecPattern)
		}

		// Note about Claude execution mode
		t.Log("ℹ️  Note: Claude execution is always performed for tests")
		t.Log("  (including Claude thoughts, command history, and execution logs)")

		// Check for JSONL file with timestamp pattern
		jsonlPattern := filepath.Join(testDir, "output", ".debug", "claude-prompt-*.jsonl")
		jsonlFiles, _ := filepath.Glob(jsonlPattern)
		if len(jsonlFiles) > 0 {
			t.Logf("✓ Claude execution log found: %s", jsonlFiles[0])
			// Read and verify it contains actual Claude interaction
			jsonlContent, err := os.ReadFile(jsonlFiles[0])
			if err == nil && len(jsonlContent) > 0 {
				t.Log("  File contains Claude prompt and response data")
			}
		} else {
			t.Logf("⚠️  Claude execution log not found (pattern: %s)", jsonlPattern)
			t.Log("  This file contains Claude's prompts and responses")
		}

		// Check for markdown prompt files
		mdPattern := filepath.Join(testDir, "output", ".debug", "claude-prompt-*.md")
		mdFiles, _ := filepath.Glob(mdPattern)
		if len(mdFiles) > 0 {
			t.Logf("✓ Claude prompt markdown found: %s", mdFiles[0])
		} else {
			t.Logf("⚠️ Claude prompt markdown not found (pattern: %s)", mdPattern)
		}

		// Check for Claude execution metadata
		execMetadataPattern := filepath.Join(testDir, "output", ".debug", "claude-execution-metadata-*.json")
		execMetadataFiles, _ := filepath.Glob(execMetadataPattern)
		if len(execMetadataFiles) > 0 {
			t.Logf("✓ Claude execution metadata found: %s", execMetadataFiles[0])
		} else {
			t.Log("  Execution metadata is created during actual Claude execution")
		}

		t.Log("✓ Debug files verification complete")
	})

	t.Log("✅ All tests completed successfully!")
}

// TestProjectHelloWorldTxtRemovalSuccess tests file removal functionality
func TestProjectHelloWorldTxtRemovalSuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectHelloWorldTxtRemovalSuccess")

	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `# Terraform configuration for hello-world-txt removal test
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration
provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  debug                 = true
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  file {
    path = "hello.txt"
    content = <<-EOF
hello world
EOF
  }

  file {
    path = "hello2.txt"
    content = <<-EOF
!hello world2!
EOF
  }

  file {
    path = "hello3.txt"
    content = <<-EOF
hello world3
EOF
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

	// Step 4: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 5: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 6: Run initial apply to create all three files
	t.Log("Running tofu apply --auto-approve (initial)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	projectPath := filepath.Join(testDir, "output")

	// Step 7: Verify all three files were created
	t.Log("Verifying initial files exist...")
	helloPath := filepath.Join(projectPath, "hello.txt")
	hello2Path := filepath.Join(projectPath, "hello2.txt")
	hello3Path := filepath.Join(projectPath, "hello3.txt")

	assert.FileExists(t, helloPath, "hello.txt should exist")
	verifyFileContent(t, helloPath, "hello world")

	assert.FileExists(t, hello2Path, "hello2.txt should exist")
	verifyFileContent(t, hello2Path, "!hello world2!")

	assert.FileExists(t, hello3Path, "hello3.txt should exist")
	verifyFileContent(t, hello3Path, "hello world3")

	t.Log("✓ All initial files created successfully")

	// Step 8: Update project.tofu to remove hello3.txt and update hello.txt
	t.Log("Updating project.tofu to remove hello3.txt...")
	updatedProjectContent := `# Terraform configuration for hello-world-txt removal test
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration
provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  debug                 = true
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  file {
    path = "hello.txt"
    content = <<-EOF
hello world updated
EOF
  }

  file {
    path = "hello2.txt"
    content = <<-EOF
!hello world2!
EOF
  }

  # hello3.txt removed
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(updatedProjectContent), 0644)
	require.NoError(t, err, "Failed to update project.tofu")

	// Step 9: Run plan to see the changes
	t.Log("Running tofu plan (after removal)...")
	planCmd = exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err = planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan after removal: %s", planOutput)
	t.Logf("Plan output:\n%s", planOutput)

	// Step 10: Apply the changes
	t.Log("Running tofu apply --auto-approve (removal)...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply for removal: %s", applyOutput)
	t.Log("✓ Removal apply completed successfully")

	// Step 11: Verify hello.txt was updated
	t.Log("Verifying hello.txt was updated...")
	verifyFileContent(t, helloPath, "hello world updated")
	t.Log("✓ hello.txt updated successfully")

	// Step 12: Verify hello2.txt still exists
	t.Log("Verifying hello2.txt still exists...")
	assert.FileExists(t, hello2Path, "hello2.txt should still exist")
	verifyFileContent(t, hello2Path, "!hello world2!")
	t.Log("✓ hello2.txt unchanged")

	// Step 13: Verify hello3.txt was removed
	t.Log("Verifying hello3.txt was removed...")
	if _, err := os.Stat(hello3Path); os.IsNotExist(err) {
		t.Log("✓ hello3.txt was properly deleted")
	} else {
		t.Errorf("❌ hello3.txt still exists but should have been removed")
	}

	t.Log("✅ File removal test completed!")
}

// TestProjectHelloWorldTxtRenameSuccess tests that renaming a file properly deletes the old file
func TestProjectHelloWorldTxtRenameSuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectHelloWorldTxtRenameSuccess")

	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `# Terraform configuration for hello-world-txt rename test
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration
provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  debug                 = true
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  file {
    path = "hello.txt"
    content = <<-EOF
hello world
EOF
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

	// Step 4: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 5: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 6: Run initial apply to create hello.txt
	t.Log("Running tofu apply --auto-approve (initial)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	projectPath := filepath.Join(testDir, "output")

	// Step 7: Verify hello.txt was created
	t.Log("Verifying hello.txt exists...")
	helloPath := filepath.Join(projectPath, "hello.txt")
	assert.FileExists(t, helloPath, "hello.txt should exist after initial apply")
	verifyFileContent(t, helloPath, "hello world")
	t.Log("✓ hello.txt exists with correct content")

	// Step 8: Update project.tofu to RENAME the file from "hello.txt" to "hello2.txt"
	// Same content, just different path
	t.Log("Updating project.tofu to rename hello.txt -> hello2.txt...")
	renamedProjectContent := `# Terraform configuration for hello-world-txt rename test
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration
provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  debug                 = true
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  file {
    path = "hello2.txt"
    content = <<-EOF
hello world
EOF
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(renamedProjectContent), 0644)
	require.NoError(t, err, "Failed to update project.tofu for rename")

	// Step 9: Run plan to see the rename
	t.Log("Running tofu plan (after rename)...")
	planCmd = exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err = planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan after rename: %s", planOutput)
	t.Logf("Plan output (should show path change):\n%s", planOutput)

	// Step 10: Apply the rename
	t.Log("Running tofu apply --auto-approve (rename)...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply for rename: %s", applyOutput)
	t.Log("✓ Rename apply completed successfully")

	// Step 11: Verify hello2.txt exists with correct content
	t.Log("Verifying hello2.txt exists...")
	hello2Path := filepath.Join(projectPath, "hello2.txt")
	assert.FileExists(t, hello2Path, "hello2.txt should exist after rename")
	verifyFileContent(t, hello2Path, "hello world")
	t.Log("✓ hello2.txt exists with correct content")

	// Step 12: Verify hello.txt was deleted (THIS IS THE BUG - will fail initially)
	t.Log("Verifying hello.txt was deleted...")
	if _, err := os.Stat(helloPath); os.IsNotExist(err) {
		t.Log("✓ hello.txt was properly deleted after rename")
	} else {
		t.Errorf("❌ BUG: hello.txt still exists after renaming to hello2.txt")
		t.Log("This confirms the bug - Claude doesn't know to delete the old file")
	}

	t.Log("✅ Rename test completed!")
}
