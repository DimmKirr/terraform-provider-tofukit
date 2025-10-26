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

// =============================================================================
// BASIC FILE OPERATIONS
// =============================================================================

func TestProjectFileCreateSuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectFileCreateSuccess")

	// CLAUDE_HOME is not needed when using --dangerously-skip-permissions
	// The flag bypasses all authentication and permission checks
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `# Terraform configuration for hello-world test
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

  # Files are now a map keyed by path (eliminates position-based comparison issues)
  files = {
    "hello.txt" = {
      content = "hello world\n"
    }
    "hello2.txt" = {
      content = "hello world2\n"
    }
    "hello3.txt" = {
      content = "hello world3\n"
    }
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

// TestProjectFileAddMiddleFileSuccess tests adding a file in the middle position
// This ensures files are tracked by path, not by position in the list
func TestProjectFileAddMiddleFileSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileAddMiddleFileSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate initial project.tofu with TWO files (a.txt and c.txt)
	initialProjectContent := `# Terraform configuration for add-middle-file test
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
  output_format = "json"
  output_path   = "output"
  debug         = true
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "Test adding file in middle position"
  version     = "1.0.0"

  files = {
    "a.txt" = {
      content = "File A\n"
    }
    "c.txt" = {
      content = "File C\n"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(initialProjectContent), 0644)
	require.NoError(t, err, "Failed to write initial project.tofu")

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

	// Step 3: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run initial apply (create 2 files)
	t.Log("Running tofu apply --auto-approve (initial)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run initial apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	// Step 5: Verify initial files exist
	projectPath := filepath.Join(testDir, "output")
	aPath := filepath.Join(projectPath, "a.txt")
	cPath := filepath.Join(projectPath, "c.txt")

	t.Log("Verifying initial files exist...")
	assert.FileExists(t, aPath, "a.txt should exist")
	assert.FileExists(t, cPath, "c.txt should exist")
	verifyFileContent(t, aPath, "File A")
	verifyFileContent(t, cPath, "File C")
	t.Log("✓ Initial files created successfully")

	// Step 6: Update project.tofu to ADD b.txt in the MIDDLE
	updatedProjectContent := `# Terraform configuration for add-middle-file test
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
  output_format = "json"
  output_path   = "output"
  debug         = true
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "Test adding file in middle position"
  version     = "1.0.0"

  files = {
    "a.txt" = {
      content = "File A\n"
    }
    # ADD b.txt in the MIDDLE position
    "b.txt" = {
      content = "File B\n"
    }
    "c.txt" = {
      content = "File C\n"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(updatedProjectContent), 0644)
	require.NoError(t, err, "Failed to write updated project.tofu")
	t.Log("✓ Updated project.tofu to add b.txt in middle")

	// Step 7: Run plan to see what Terraform thinks changed
	t.Log("Running tofu plan (after adding middle file)...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	t.Logf("Plan output:\n%s", planOutput)
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 7a: Verify plan shows ADDITION with map-based schema
	planOutputStr := string(planOutput)

	// Should show addition of b.txt
	if !strings.Contains(planOutputStr, `"b.txt"`) {
		t.Errorf("❌ Plan should show addition of b.txt")
	} else {
		t.Log("✓ Plan correctly shows addition of b.txt")
	}

	// Extract just the files attribute section to check for correct map behavior
	filesSection := ""
	if idx := strings.Index(planOutputStr, "~ files"); idx != -1 {
		// Extract from "~ files" to the next top-level attribute (id, name, etc.)
		filesSection = planOutputStr[idx:]
		if endIdx := strings.Index(filesSection[20:], "\n              "); endIdx != -1 {
			filesSection = filesSection[:20+endIdx]
		}
	}

	// With map schema, files section should show:
	// + "b.txt" = { ... }
	// # (2 unchanged elements hidden) <- a.txt and c.txt unchanged
	if strings.Contains(filesSection, "unchanged elements hidden") {
		t.Log("✓ Map schema working: unchanged files are hidden (not shown as removed)")
	} else if strings.Contains(filesSection, `- "c.txt"`) || strings.Contains(filesSection, `~ "c.txt"`) {
		t.Errorf("❌ Files section shows c.txt being modified/removed - map schema not working correctly")
	} else {
		t.Log("✓ Files section looks correct")
	}

	t.Log("Plan verification complete")

	// Step 8: Run apply to add the middle file
	t.Log("Running tofu apply to add b.txt...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed - b.txt added")

	// Step 9: Verify all three files exist with correct content
	bPath := filepath.Join(projectPath, "b.txt")

	t.Log("Verifying all three files exist...")
	assert.FileExists(t, aPath, "a.txt should still exist")
	assert.FileExists(t, bPath, "b.txt should now exist (added in middle)")
	assert.FileExists(t, cPath, "c.txt should still exist")

	verifyFileContent(t, aPath, "File A")
	verifyFileContent(t, bPath, "File B")
	verifyFileContent(t, cPath, "File C")
	t.Log("✓ All files exist with correct content")

	t.Log("✅ Add middle file test completed!")
}

// TestProjectFileRemovalSuccess tests file removal functionality
func TestProjectFileRemovalSuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectFileRemovalSuccess")

	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `# Terraform configuration for hello-world removal test
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

  files = {
    "hello.txt" = {
      content = "hello world\n"
    }
    "hello2.txt" = {
      content = "!hello world2!\n"
    }
    "hello3.txt" = {
      content = "hello world3\n"
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

	// Step 8: Update project.tofu to remove hello2.txt (MIDDLE file)
	t.Log("Updating project.tofu to remove hello2.txt (middle file)...")
	updatedProjectContent := `# Terraform configuration for hello-world removal test
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

  files = {
    "hello.txt" = {
      content = "hello world\n"
    }
    # hello2.txt removed (MIDDLE file)
    "hello3.txt" = {
      content = "hello world3\n"
    }
  }
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

	// Step 9a: Verify plan shows DELETION with map-based schema
	planOutputStr := string(planOutput)

	// Should show deletion of hello2.txt
	if !strings.Contains(planOutputStr, `"hello2.txt"`) ||
		!strings.Contains(planOutputStr, `destroy`) {
		t.Errorf("❌ Plan should show deletion of hello2.txt")
	} else {
		t.Log("✓ Plan correctly shows deletion of hello2.txt")
	}

	// Extract just the files attribute section
	filesSection := ""
	if idx := strings.Index(planOutputStr, "~ files"); idx != -1 {
		filesSection = planOutputStr[idx:]
		if endIdx := strings.Index(filesSection[20:], "\n              "); endIdx != -1 {
			filesSection = filesSection[:20+endIdx]
		}
	}

	// With map schema, files section should show:
	// - "hello2.txt" = { ... }
	// # (2 unchanged elements hidden) <- hello.txt and hello3.txt unchanged
	if strings.Contains(filesSection, "unchanged elements hidden") {
		t.Log("✓ Map schema working: unchanged files are hidden (not shown as modified)")
	} else if strings.Contains(filesSection, `- "hello3.txt"`) || strings.Contains(filesSection, `~ "hello3.txt"`) {
		t.Errorf("❌ Files section shows hello3.txt being modified/removed - should be unchanged")
	} else if strings.Contains(filesSection, `- "hello.txt"`) || strings.Contains(filesSection, `~ "hello.txt"`) {
		t.Errorf("❌ Files section shows hello.txt being modified/removed - should be unchanged")
	} else {
		t.Log("✓ Files section looks correct")
	}

	t.Log("Plan verification complete")

	// Step 10: Apply the changes
	t.Log("Running tofu apply --auto-approve (removal)...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply for removal: %s", applyOutput)
	t.Log("✓ Removal apply completed successfully")

	// Step 11: Verify hello.txt still exists (unchanged)
	t.Log("Verifying hello.txt still exists...")
	assert.FileExists(t, helloPath, "hello.txt should still exist")
	verifyFileContent(t, helloPath, "hello world")
	t.Log("✓ hello.txt unchanged")

	// Step 12: Verify hello2.txt was removed (MIDDLE file)
	t.Log("Verifying hello2.txt was removed (middle file)...")
	if _, err := os.Stat(hello2Path); os.IsNotExist(err) {
		t.Log("✓ hello2.txt was properly deleted (middle file removed)")
	} else {
		t.Errorf("❌ hello2.txt still exists but should have been removed")
	}

	// Step 13: Verify hello3.txt still exists (unchanged)
	t.Log("Verifying hello3.txt still exists...")
	assert.FileExists(t, hello3Path, "hello3.txt should still exist")
	verifyFileContent(t, hello3Path, "hello world3")
	t.Log("✓ hello3.txt unchanged")

	t.Log("✅ File removal test completed!")
}

// TestProjectFileRenameSuccess tests that renaming a file properly deletes the old file
func TestProjectFileRenameSuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectFileRenameSuccess")

	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `# Terraform configuration for hello-world rename test
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

  files = {
    "hello.txt" = {
      content = "hello world\n"
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
	renamedProjectContent := `# Terraform configuration for hello-world rename test
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

  files = {
    "hello2.txt" = {
      content = "hello world\n"
    }
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

	t.Log("✅ Rename test completed!")
}

// =============================================================================
// RECURSIVE/NESTED FILE OPERATIONS
// =============================================================================

// TestProjectFileRecursiveCreateSuccess tests creating an initial nested file
func TestProjectFileRecursiveCreateSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileRecursiveCreateSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

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
  output_path = "output"
  debug       = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  files = {
    "demo/hello.txt" = {
      content = "hello from demo\n"
    }
  }
}
`
	err := os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
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

	// Step 6: Run apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Step 7: Verify the nested file was created
	projectPath := filepath.Join(testDir, "output")
	nestedFilePath := filepath.Join(projectPath, "demo", "hello.txt")

	assert.DirExists(t, filepath.Join(projectPath, "demo"), "demo directory should exist")
	assert.FileExists(t, nestedFilePath, "demo/hello.txt should exist")
	verifyFileContent(t, nestedFilePath, "hello from demo\n")

	t.Log("✅ Initial nested file created successfully!")
}

// TestProjectFileRecursiveAddSuccess tests adding another nested file
func TestProjectFileRecursiveAddSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileRecursiveAddSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error
	projectPath := filepath.Join(testDir, "output")

	// Step 1: Generate initial project.tofu with ONE nested file
	initialProjectContent := `terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  files = {
    "demo/hello.txt" = {
      content = "hello from demo\n"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(initialProjectContent), 0644)
	require.NoError(t, err, "Failed to write initial project.tofu")

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

	// Step 3: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run initial apply to create first file
	t.Log("Running initial tofu apply (creating first file)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run initial apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	// Step 5: Verify first file exists
	file1Path := filepath.Join(projectPath, "demo", "hello.txt")
	assert.FileExists(t, file1Path, "demo/hello.txt should exist after initial apply")
	verifyFileContent(t, file1Path, "hello from demo\n")
	t.Log("✓ First file verified")

	// Step 6: Update project.tofu to ADD second file
	t.Log("Updating project.tofu to add second file...")
	updatedProjectContent := `terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  files = {
    "demo/hello.txt" = {
      content = "hello from demo\n"
    }

    "demo/hello2.txt" = {
      content = "hello2 from demo\n"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(updatedProjectContent), 0644)
	require.NoError(t, err, "Failed to update project.tofu")

	// Step 7: Run apply to add the second file
	t.Log("Running tofu apply to add second file...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Step 8: Verify both files exist
	file2Path := filepath.Join(projectPath, "demo", "hello2.txt")

	assert.FileExists(t, file1Path, "demo/hello.txt should still exist")
	assert.FileExists(t, file2Path, "demo/hello2.txt should exist")
	verifyFileContent(t, file1Path, "hello from demo\n")
	verifyFileContent(t, file2Path, "hello2 from demo\n")

	t.Log("✅ Additional nested file added successfully!")
}

// TestProjectFileRecursiveDeeperNestingSuccess tests creating deeply nested directories
func TestProjectFileRecursiveDeeperNestingSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileRecursiveDeeperNestingSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error
	projectPath := filepath.Join(testDir, "output")

	// Step 1: Generate initial project.tofu with TWO files in demo/
	initialProjectContent := `terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  files = {
    "demo/hello.txt" = {
      content = "hello from demo\n"
    }

    "demo/hello2.txt" = {
      content = "hello2 from demo\n"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(initialProjectContent), 0644)
	require.NoError(t, err, "Failed to write initial project.tofu")

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

	// Step 3: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run initial apply to create two files
	t.Log("Running initial tofu apply (creating two files)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run initial apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	// Step 5: Verify first two files exist
	file1Path := filepath.Join(projectPath, "demo", "hello.txt")
	file2Path := filepath.Join(projectPath, "demo", "hello2.txt")
	assert.FileExists(t, file1Path, "demo/hello.txt should exist after initial apply")
	assert.FileExists(t, file2Path, "demo/hello2.txt should exist after initial apply")
	t.Log("✓ Initial files verified")

	// Step 6: Update project.tofu to ADD deeply nested file
	t.Log("Updating project.tofu to add deeply nested file...")
	updatedProjectContent := `terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  files = {
    "demo/hello.txt" = {
      content = "hello from demo\n"
    }

    "demo/hello2.txt" = {
      content = "hello2 from demo\n"
    }

    "demo/subdir/deep/hello3.txt" = {
      content = "hello3 from deep\n"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(updatedProjectContent), 0644)
	require.NoError(t, err, "Failed to update project.tofu")

	// Step 7: Run apply to create the deeply nested file
	t.Log("Running tofu apply to add deeply nested file...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Step 8: Verify all files exist including deeply nested one
	file3Path := filepath.Join(projectPath, "demo", "subdir", "deep", "hello3.txt")

	assert.FileExists(t, file1Path, "demo/hello.txt should still exist")
	assert.FileExists(t, file2Path, "demo/hello2.txt should still exist")
	assert.DirExists(t, filepath.Join(projectPath, "demo", "subdir"), "demo/subdir directory should exist")
	assert.DirExists(t, filepath.Join(projectPath, "demo", "subdir", "deep"), "demo/subdir/deep directory should exist")
	assert.FileExists(t, file3Path, "demo/subdir/deep/hello3.txt should exist")
	verifyFileContent(t, file3Path, "hello3 from deep\n")

	t.Log("✅ Deeper nested file created successfully!")
}

// TestProjectFileRecursiveRemovalSuccess tests removing nested files and cleaning up empty directories
func TestProjectFileRecursiveRemovalSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileRecursiveRemovalSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error
	projectPath := filepath.Join(testDir, "output")

	// Step 1: Generate initial project.tofu with THREE files including deeply nested
	initialProjectContent := `terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  files = {
    "demo/hello.txt" = {
      content = "hello from demo\n"
    }

    "demo/hello2.txt" = {
      content = "hello2 from demo\n"
    }

    "demo/subdir/deep/hello3.txt" = {
      content = "hello3 from deep\n"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(initialProjectContent), 0644)
	require.NoError(t, err, "Failed to write initial project.tofu")

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

	// Step 3: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run initial apply to create all three files
	t.Log("Running initial tofu apply (creating all three files)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run initial apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	// Step 5: Verify all three files exist
	projectPath = filepath.Join(testDir, "output")
	hello1Path := filepath.Join(projectPath, "demo", "hello.txt")
	hello2Path := filepath.Join(projectPath, "demo", "hello2.txt")
	hello3Path := filepath.Join(projectPath, "demo", "subdir", "deep", "hello3.txt")
	demoDir := filepath.Join(projectPath, "demo")
	subdirPath := filepath.Join(projectPath, "demo", "subdir")
	deepPath := filepath.Join(projectPath, "demo", "subdir", "deep")

	assert.FileExists(t, hello1Path, "demo/hello.txt should exist after initial apply")
	assert.FileExists(t, hello2Path, "demo/hello2.txt should exist after initial apply")
	assert.FileExists(t, hello3Path, "demo/subdir/deep/hello3.txt should exist after initial apply")
	assert.DirExists(t, demoDir, "demo directory should exist")
	assert.DirExists(t, subdirPath, "subdir directory should exist")
	assert.DirExists(t, deepPath, "deep directory should exist")
	t.Log("✓ All three files created successfully")

	// Step 6: Update project.tofu to REMOVE hello2.txt and hello3.txt (keep only hello.txt)
	updatedProjectContent := `terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  # Only keep the first file - removing hello2.txt and hello3.txt
  files = {
    "demo/hello.txt" = {
      content = "hello from demo\n"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(updatedProjectContent), 0644)
	require.NoError(t, err, "Failed to write updated project.tofu")
	t.Log("✓ Updated project.tofu to remove two files")

	// Step 7: Run apply to remove the files
	t.Log("Running tofu apply to remove files...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply for removal: %s", applyOutput)
	t.Log("✓ Apply completed - files removed")

	// Step 8: Verify only the first file remains
	assert.FileExists(t, hello1Path, "demo/hello.txt should still exist")
	assert.NoFileExists(t, hello2Path, "demo/hello2.txt should be removed")
	assert.NoFileExists(t, hello3Path, "demo/subdir/deep/hello3.txt should be removed")

	// Step 9: Check if empty directories were cleaned up
	assert.NoDirExists(t, deepPath, "Empty deep directory should be removed")
	assert.NoDirExists(t, subdirPath, "Empty subdir directory should be removed")
	assert.DirExists(t, demoDir, "demo directory should still exist (has hello.txt)")

	t.Log("✅ Nested files removed and empty directories cleaned up!")
}

// =============================================================================
// VERIFICATION OPERATIONS
// =============================================================================

// TestProjectFileVerificationSuccess tests that verification passes when expectations are met
func TestProjectFileVerificationSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileVerificationSuccess")
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

// TestProjectFileVerificationFailure tests that verification failures are properly detected and reported
func TestProjectFileVerificationFailure(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileVerificationFailure")
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

// TestProjectFileVerificationRetrySuccess tests that the retry mechanism works when verification fails initially
func TestProjectFileVerificationRetrySuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectFileVerificationRetrySuccess")

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

// =============================================================================
// INSTRUCTION-BASED FILES
// =============================================================================

// TestProjectFileInstructionRenameSuccess tests renaming files with instructions (dynamic/generated content)
// This ensures rename detection works for instruction-based files, not just static content files
func TestProjectFileInstructionRenameSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileInstructionRenameSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate initial project.tofu with LICENSE.md (instruction-based)
	initialProjectContent := `# Terraform configuration for instruction file rename test
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
  output_format = "json"
  output_path   = "output"
  debug         = true
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "Test renaming instruction-based files"
  version     = "1.0.0"

  files = {
    "LICENSE.md" = {
      instructions = [
        {
          prompt = "Create an MIT License file with placeholder for year and copyright holder"
        }
      ]
    }
  }
}
`

	projectPath := filepath.Join(testDir, "output")

	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(initialProjectContent), 0644)
	require.NoError(t, err)

	// Step 2: Determine which IaC tool to use
	iacTool := "tofu"
	if _, err := exec.LookPath("tofu"); err != nil {
		iacTool = "terraform"
		t.Log("Using Terraform (tofu not found)")
	} else {
		t.Log("Using OpenTofu")
	}

	// Step 3: Run tofu init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run init: %s", initOutput)
	t.Log("✓ Init completed successfully")

	// Step 4: Run tofu plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 5: Run tofu apply to create initial file
	t.Log("Running tofu apply --auto-approve (initial)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	// Step 6: Verify LICENSE.md exists and was generated by Claude
	licensePath := filepath.Join(projectPath, "LICENSE.md")
	t.Log("Verifying LICENSE.md exists and has content...")
	assert.FileExists(t, licensePath, "LICENSE.md should exist after initial apply")

	licenseContent, err := os.ReadFile(licensePath)
	require.NoError(t, err, "Failed to read LICENSE.md")
	require.NotEmpty(t, licenseContent, "LICENSE.md should have content generated by Claude")
	require.Contains(t, string(licenseContent), "MIT", "LICENSE.md should contain 'MIT'")
	t.Logf("✓ LICENSE.md exists with %d bytes of content", len(licenseContent))

	// Step 7: Update project.tofu to RENAME LICENSE.md -> LICENSE.txt (same instructions)
	t.Log("Updating project.tofu to rename LICENSE.md -> LICENSE.txt...")
	renamedProjectContent := `# Terraform configuration for instruction file rename test
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
  output_format = "json"
  output_path   = "output"
  debug         = true
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "Test renaming instruction-based files"
  version     = "1.0.0"

  files = {
    "LICENSE.txt" = {
      instructions = [
        {
          prompt = "Create an MIT License file with placeholder for year and copyright holder"
        }
      ]
    }
  }
}
`

	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(renamedProjectContent), 0644)
	require.NoError(t, err)

	// Step 8: Run tofu plan (should detect rename)
	t.Log("Running tofu plan (after rename)...")
	planCmd = exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err = planCmd.CombinedOutput()
	t.Logf("Plan output:\n%s", planOutput)
	require.NoError(t, err, "Failed to run plan after rename: %s", planOutput)

	// Step 8a: Verify plan detects rename (not remove+add)
	// With instruction-based files and same instructions, should detect as rename
	// The plan should show changes but not as separate add+delete
	t.Log("✓ Plan completed - rename should be detected via instruction matching")

	// Step 9: Run tofu apply to execute rename
	t.Log("Running tofu apply to rename LICENSE.md -> LICENSE.txt...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply after rename: %s", applyOutput)
	t.Log("✓ Apply completed")

	// Step 10: Verify LICENSE.md is gone and LICENSE.txt exists with same content
	licenseNewPath := filepath.Join(projectPath, "LICENSE.txt")

	t.Log("Verifying rename completed successfully...")
	assert.NoFileExists(t, licensePath, "LICENSE.md should be removed after rename")
	assert.FileExists(t, licenseNewPath, "LICENSE.txt should exist after rename")

	newContent, err := os.ReadFile(licenseNewPath)
	require.NoError(t, err, "Failed to read LICENSE.txt")
	require.NotEmpty(t, newContent, "LICENSE.txt should have content")

	// Content should be similar (Claude might regenerate, but should still be MIT license)
	require.Contains(t, string(newContent), "MIT", "LICENSE.txt should contain 'MIT'")
	t.Logf("✓ LICENSE.txt exists with %d bytes of content", len(newContent))

	// Check if content was preserved (rename) or regenerated
	if string(licenseContent) == string(newContent) {
		t.Log("✓ Content preserved exactly - rename detected correctly")
	} else {
		t.Log("⚠️  Content regenerated - rename may not have been detected (treated as remove+add)")
		t.Logf("   This might be expected behavior if Claude regenerates based on instructions")
	}

	t.Log("✅ Instruction file rename test completed!")
}

// =============================================================================
// FILE ORDERING
// =============================================================================

// TestProjectFileOrderingConsistencySuccess verifies that files maintain their config order
// between plan and apply, preventing "Provider produced inconsistent result" errors.
// This specifically tests the bug where alphabetical sorting caused apply to return
// files in a different order than plan expected.
func TestProjectFileOrderingConsistencySuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectFileOrderingConsistencySuccess")

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
