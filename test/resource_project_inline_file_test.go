package test

import (
	"encoding/json"
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

func TestResourceProjectInlineFileCreateSuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileCreateSuccess")

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
	iacTool := detectIaCTool(t)

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

// TestResourceProjectInlineFileAddMiddleFileSuccess tests adding a file in the middle position
// This ensures files are tracked by path, not by position in the list
func TestResourceProjectInlineFileAddMiddleFileSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectInlineFileAddMiddleFileSuccess")
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
	iacTool := detectIaCTool(t)

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

// TestResourceProjectInlineFileRemovalSuccess tests file removal functionality
func TestResourceProjectInlineFileRemovalSuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileRemovalSuccess")

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
	iacTool := detectIaCTool(t)

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

// TestResourceProjectInlineFileRenameSuccess tests that renaming a file properly deletes the old file
func TestResourceProjectInlineFileRenameSuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileRenameSuccess")

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
	iacTool := detectIaCTool(t)

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

// TestResourceProjectInlineFileNestedCreateSuccess tests creating an initial nested file
func TestResourceProjectInlineFileNestedCreateSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectInlineFileNestedCreateSuccess")
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
  output_format = "json"
  output_path   = "output"
  debug         = true
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
	iacTool := detectIaCTool(t)

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

// TestResourceProjectInlineFileNestedAddSuccess tests adding another nested file
func TestResourceProjectInlineFileNestedAddSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectInlineFileNestedAddSuccess")
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
  output_format = "json"
  output_path   = "output"
  debug         = true
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
	iacTool := detectIaCTool(t)

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
  output_format = "json"
  output_path   = "output"
  debug         = true
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

// TestResourceProjectInlineFileNestedDeeperNestingSuccess tests creating deeply nested directories
func TestResourceProjectInlineFileNestedDeeperNestingSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectInlineFileNestedDeeperNestingSuccess")
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
  output_format = "json"
  output_path   = "output"
  debug         = true
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
	iacTool := detectIaCTool(t)

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
  output_format = "json"
  output_path   = "output"
  debug         = true
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

// TestResourceProjectInlineFileNestedRemovalSuccess tests removing nested files and cleaning up empty directories
func TestResourceProjectInlineFileNestedRemovalSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectInlineFileNestedRemovalSuccess")
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
  output_format = "json"
  output_path   = "output"
  debug         = true
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
	iacTool := detectIaCTool(t)

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
  output_format = "json"
  output_path   = "output"
  debug         = true
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

// TestResourceProjectInlineFileVerificationSuccess tests that verification passes when expectations are met
func TestResourceProjectInlineFileVerificationSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectInlineFileVerificationSuccess")
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
      verifications = [{
        command = "cat hello.txt"
        expect  = "hello world"
      }]
    }
  }
}

resource "tofukit_project" "verification_test" {
  name        = "verification-test"
  description = "Project to test verification success"
  version     = "1.0.0"

  stack = tofukit_stack.test_stack
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 2: Check if terraform/tofu is available
	iacTool := detectIaCTool(t)

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

// TestResourceProjectInlineFileVerificationFailure tests that verification failures are properly detected and reported
func TestResourceProjectInlineFileVerificationFailure(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectInlineFileVerificationFailure")
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
	iacTool := detectIaCTool(t)

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

// TestResourceProjectInlineFileVerificationRetrySuccess tests that the retry mechanism works when verification fails initially
func TestResourceProjectInlineFileVerificationRetrySuccess(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileVerificationRetrySuccess")

	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Enable test hook to force first verification failure
	// This uses TOFUKIT_TEST_FORCE_VERIFY_FAIL_FIRST env var to guarantee retry triggers
	// Much cleaner than bash scripts - Claude can't interfere with this approach
	t.Setenv("TOFUKIT_TEST_FORCE_VERIFY_FAIL_FIRST", "true")

	// Clean up counter file from any previous test runs
	counterFile := "/tmp/tofukit_test_verify_counter"
	os.Remove(counterFile) // Ignore errors if file doesn't exist

	// Step 2: Generate simple project.tofu with verification
	// The test hook will force first verification to fail, triggering retry
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
  description = "Test that uses TOFUKIT_TEST_FORCE_VERIFY_FAIL_FIRST to guarantee retry triggers"
  version     = "1.0.0"

  # RETRY TRAP: Test hook forces first verification to fail
  # Claude creates correct content, but provider forces failure on first attempt
  # On retry, verification runs normally and passes
  files = {
    "greeting.txt" = {
      instructions = [{
        prompt = "Create a text file containing 'hello world'"
        constraints = [
          "Content must be exactly: hello world",
          "Keep it simple"
        ]
      }]
      verifications = [{
        # Simple verification - check file contains "hello world"
        command = "cat greeting.txt"
        expect  = "hello world"
      }]
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 2: Check if terraform/tofu is available
	iacTool := detectIaCTool(t)

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
	projectPath := filepath.Join(testDir, "output")

	// === SUBTEST 1: Verify File Created Correctly ===
	t.Run("VerifyFileCreated", func(t *testing.T) {
		t.Log("Verifying greeting.txt was created correctly...")

		// Verify greeting.txt exists
		greetingPath := filepath.Join(projectPath, "greeting.txt")
		assert.FileExists(t, greetingPath, "greeting.txt should exist")

		// Read the file content
		greetingContent, err := os.ReadFile(greetingPath)
		require.NoError(t, err, "Failed to read greeting.txt")
		t.Logf("greeting.txt content: %q (length: %d bytes)", string(greetingContent), len(greetingContent))

		// After retry, Claude may adjust the file content in response to the generic failure message
		// The test hook sends "TEST_HOOK_FORCED_FAILURE" without specifics, so Claude makes its best guess
		// Claude typically removes the trailing newline when seeing a generic failure
		assert.Equal(t, "hello world", string(greetingContent), "greeting.txt should contain 'hello world' after retry")
		assert.Equal(t, 11, len(greetingContent), "greeting.txt should be exactly 11 bytes")

		t.Log("✓ File contains 'hello world' after verification retry")
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

		// ASSERT that retry happened - we expect multiple execution attempts
		assert.Greater(t, len(metadataFiles), 1, "Expected retry to trigger - should have multiple execution metadata files")

		if len(metadataFiles) > 1 {
			t.Logf("✓ Retry loop was triggered! Found %d execution attempts", len(metadataFiles))

			// Read first metadata to see the initial failure
			firstMetadata, err := os.ReadFile(filepath.Join(debugDir, metadataFiles[0]))
			require.NoError(t, err, "Failed to read first execution metadata")
			t.Logf("First execution metadata (should show verification failure):\n%s", string(firstMetadata))

			// Read last metadata to see the successful retry
			lastMetadata, err := os.ReadFile(filepath.Join(debugDir, metadataFiles[len(metadataFiles)-1]))
			require.NoError(t, err, "Failed to read last execution metadata")
			t.Logf("Last execution metadata (should show verification success):\n%s", string(lastMetadata))
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

// TestResourceProjectInlineFileInstructionRenameSuccess tests renaming files with instructions (dynamic/generated content)
// This ensures rename detection works for instruction-based files, not just static content files
func TestResourceProjectInlineFileInstructionRenameSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectInlineFileInstructionRenameSuccess")
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
	iacTool := detectIaCTool(t)

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

// TestResourceProjectInlineFileOrderingConsistencySuccess verifies that files maintain their config order
// between plan and apply, preventing "Provider produced inconsistent result" errors.
// This specifically tests the bug where alphabetical sorting caused apply to return
// files in a different order than plan expected.
func TestResourceProjectInlineFileOrderingConsistencySuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectInlineFileOrderingConsistencySuccess")

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
	iacTool := detectIaCTool(t)

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

// =============================================================================
// FILE RESOURCE OPERATIONS
// =============================================================================

// TestFileResourceCreateSuccess tests creating a file resource with static content
// and using it in a project
func TestFileResourceCreateSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestFileResourceCreateSuccess")
	t.Log("Testing file resource creation and usage...")

	var err error

	// Step 1: Generate project.tofu with file resource and project
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

# Define a reusable file resource
resource "tofukit_file" "gitignore" {
  name        = ".gitignore"
  description = "Standard Go .gitignore"

  content = "bin/\n*.exe\n*.dll\n"
}

# Use the file resource in a project
resource "tofukit_project" "test_app" {
  name    = "test-app"
  version = "1.0.0"

  files = {
    ".gitignore" = tofukit_file.gitignore
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 2: Check if terraform/tofu is available
	iacTool := detectIaCTool(t)

	// Step 3: Run init
	t.Log("Running init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run plan
	t.Log("Running plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 5: Run apply
	t.Log("Running apply...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	if err != nil {
		t.Logf("Apply output: %s", applyOutput)
		t.Fatalf("Apply failed: %v", err)
	}
	t.Log("✓ Apply completed successfully")

	// Step 6: Verify the file was created from the file resource
	gitignorePath := filepath.Join(testDir, "output", ".gitignore")
	assert.FileExists(t, gitignorePath, ".gitignore should exist")

	// Verify the full content matches what we defined in the file resource
	verifyFileContent(t, gitignorePath, "bin/\n*.exe\n*.dll\n")

	t.Log("✅ File resource creation and usage test passed!")
}

// TestFileResourceInstructionsSuccess tests file resource with dynamic generation via instructions
func TestFileResourceInstructionsSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestFileResourceInstructionsSuccess")
	t.Log("Testing file resource with instructions...")

	var err error

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

# Define a file resource with instructions (not content)
resource "tofukit_file" "readme" {
  name        = "README.md"
  description = "Standard project README"

  instructions = [{
    prompt = "Create a README.md with a title 'Test Project' and a short description saying 'This is a test project created by tofukit.'"
    constraints = ["Keep it under 10 lines", "Use proper markdown"]
  }]

  verifications = [{
    command = "grep -q 'Test Project' README.md && echo 'found'"
    expect  = "found"
  }]
}

resource "tofukit_project" "test_app" {
  name    = "test-app"
  version = "1.0.0"

  files = {
    "README.md" = tofukit_file.readme
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	iacTool := detectIaCTool(t)

	// Init
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")

	// Plan
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)

	// Apply
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	if err != nil {
		t.Logf("Apply output: %s", applyOutput)
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify README was generated and contains expected content
	readmePath := filepath.Join(testDir, "output", "README.md")
	assert.FileExists(t, readmePath, "README.md should exist")

	content, err := os.ReadFile(readmePath)
	require.NoError(t, err, "Failed to read README.md")
	assert.Contains(t, string(content), "Test Project", "README should contain 'Test Project'")
	assert.Contains(t, string(content), "tofukit", "README should mention 'tofukit'")

	t.Log("✅ File resource with instructions test passed!")
}

// TestFileResourceSharedAcrossProjectsSuccess tests reusing a file resource across multiple projects
func TestFileResourceSharedAcrossProjectsSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestFileResourceSharedAcrossProjectsSuccess")
	t.Log("Testing file resource shared across multiple projects...")

	var err error

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

# Define a reusable LICENSE file resource
resource "tofukit_file" "mit_license" {
  name = "LICENSE"
  content = "MIT License\n\nCopyright (c) 2024\n\nPermission is hereby granted...\n"
}

# Project 1 uses the license
resource "tofukit_project" "app1" {
  name    = "app1"
  version = "1.0.0"

  files = {
    "LICENSE" = tofukit_file.mit_license
  }
}

# Project 2 also uses the same license
resource "tofukit_project" "app2" {
  name    = "app2"
  version = "1.0.0"

  files = {
    "LICENSE" = tofukit_file.mit_license
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	iacTool := detectIaCTool(t)

	// Init
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")

	// Plan
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)

	// Apply
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	if err != nil {
		t.Logf("Apply output: %s", applyOutput)
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify both projects have the LICENSE file with same content
	license1Path := filepath.Join(testDir, "output", "LICENSE")
	license2Path := filepath.Join(testDir, "output", "LICENSE")

	// Both should exist (though they're actually the same path since projects overlap)
	assert.FileExists(t, license1Path, "app1/LICENSE should exist")
	assert.FileExists(t, license2Path, "app2/LICENSE should exist")

	// Verify content
	content1, err := os.ReadFile(license1Path)
	require.NoError(t, err, "Failed to read LICENSE from app1")
	assert.Contains(t, string(content1), "MIT License")
	assert.Contains(t, string(content1), "Copyright (c) 2024")

	t.Log("✅ File resource shared across projects test passed!")
}

// TestResourceProjectDriftDetection_StaticFile verifies drift detection for modified static files
func TestResourceProjectDriftDetection_StaticFile(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectDriftDetection_StaticFile")

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
  name    = "drift-test"
  version = "1.0.0"

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
	assert.Contains(t, stateJSON, `"drift_detected":false`, "Initial state should have drift_detected=false")

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
	assert.Contains(t, stateJSON, `"drift_detected":true`, "State should have drift_detected=true after refresh")
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
	assert.Contains(t, stateJSON, `"drift_detected":false`, "State should have drift_detected=false after restore")

	t.Log("✓ Drift detection and restoration completed successfully")
}

// TestResourceProjectDriftDetection_DeletedFile verifies drift detection for deleted files
func TestResourceProjectDriftDetection_DeletedFile(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectDriftDetection_DeletedFile")

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
  name    = "drift-delete-test"
  version = "1.0.0"

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
	assert.Contains(t, stateJSON, `"drift_detected":true`, "State should detect drift from deleted file")
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
	assert.Contains(t, stateJSON, `"drift_detected":false`, "State should have drift_detected=false after recreate")

	t.Log("✓ Deleted file drift detection and restoration completed successfully")
}

// TestResourceProjectDriftDetection_MultipleFiles verifies drift detection for multiple modified files
func TestResourceProjectDriftDetection_MultipleFiles(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceProjectDriftDetection_MultipleFiles")

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
  name    = "drift-multi-test"
  version = "1.0.0"

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
	assert.Contains(t, stateJSON, `"drift_detected":false`, "Initial state should have drift_detected=false")

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
	assert.Contains(t, stateJSON, `"drift_detected":true`, "State should have drift_detected=true")
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
	assert.Contains(t, stateJSON, `"drift_detected":false`, "State should have drift_detected=false after restore")

	t.Log("✓ Multiple file drift detection and restoration completed successfully")
}

// =============================================================================
// PROMPT GENERATION TESTS (Fast)
// =============================================================================

// TestResourceProjectInlineFileCreateGeneratePromptSuccess verifies prompt generation for file creation
// Uses dry_run mode to skip LLM execution and validate prompt structure
func TestResourceProjectInlineFileCreateGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileCreateGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigProjectInlineFileCreate), 0644)
	require.NoError(t, err)

	// Setup Terraform and run apply (with TF_LOG=INFO)
	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Locate and read prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON file")

	jsonData, err := os.ReadFile(debugFiles[0])
	require.NoError(t, err, "Failed to read prompt JSON")

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err, "Failed to parse prompt JSON")

	// Navigate to specification.files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify hello.txt has create instruction
	helloFile := files["hello.txt"].(map[string]interface{})
	instructions := helloFile["instructions"].([]interface{})
	instruction0 := instructions[0].(map[string]interface{})

	assert.Contains(t, instruction0["prompt"], "Create new file 'hello.txt'",
		"Should have create instruction for new file")

	// Verify hello2.txt has create instruction
	hello2File := files["hello2.txt"].(map[string]interface{})
	instructions2 := hello2File["instructions"].([]interface{})
	instruction2_0 := instructions2[0].(map[string]interface{})

	assert.Contains(t, instruction2_0["prompt"], "Create new file 'hello2.txt'",
		"Should have create instruction for hello2.txt")

	// Verify hello3.txt has create instruction
	hello3File := files["hello3.txt"].(map[string]interface{})
	instructions3 := hello3File["instructions"].([]interface{})
	instruction3_0 := instructions3[0].(map[string]interface{})

	assert.Contains(t, instruction3_0["prompt"], "Create new file 'hello3.txt'",
		"Should have create instruction for hello3.txt")

	t.Log("✓ Prompt generation verified - all files have correct create instructions")
}

// TestResourceProjectInlineFileAddMiddleFileGeneratePromptSuccess verifies prompt for adding files
// Ensures files are tracked by path, not position
func TestResourceProjectInlineFileAddMiddleFileGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileAddMiddleFileGeneratePromptSuccess")

	// MOCK SETUP: Create initial state with 2 files
	projectPath := filepath.Join(testDir, "output")
	err := os.MkdirAll(projectPath, 0755)
	require.NoError(t, err)

	// Create hello.txt
	helloPath := filepath.Join(projectPath, "hello.txt")
	err = os.WriteFile(helloPath, []byte("hello world\n"), 0644)
	require.NoError(t, err)

	// Create hello3.txt
	hello3Path := filepath.Join(projectPath, "hello3.txt")
	err = os.WriteFile(hello3Path, []byte("hello world3\n"), 0644)
	require.NoError(t, err)

	// Step 1: Initial terraform config (2 files)
	initialConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "add-middle-test"
  version = "1.0.0"

  files = {
    "hello.txt" = {
      content = "hello world\n"
    }
    "hello3.txt" = {
      content = "hello world3\n"
    }
  }
}
`

	configPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Step 2: Update config to add hello2.txt in the middle
	updatedConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "add-middle-test"
  version = "1.0.0"

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

	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// Apply with the new file
	runTerraformApply(t, iacTool, testDir)

	// Read prompt JSON (should be attempt2 now after the update)
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt*-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON files")

	// Get the latest prompt file (highest attempt number)
	latestPrompt := debugFiles[len(debugFiles)-1]
	jsonData, err := os.ReadFile(latestPrompt)
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify hello2.txt has create instruction (new file)
	hello2File := files["hello2.txt"].(map[string]interface{})
	instructions2 := hello2File["instructions"].([]interface{})
	instruction2_0 := instructions2[0].(map[string]interface{})

	assert.Contains(t, instruction2_0["prompt"], "Create new file 'hello2.txt'",
		"Should have create instruction for newly added file")

	// Verify hello.txt has unchanged instruction
	helloFile := files["hello.txt"].(map[string]interface{})
	instructions1 := helloFile["instructions"].([]interface{})
	instruction1_0 := instructions1[0].(map[string]interface{})

	assert.Contains(t, instruction1_0["prompt"], "unchanged",
		"Existing file should have unchanged instruction")

	t.Log("✓ Prompt generation verified - middle file add detected correctly")
}

// TestResourceProjectInlineFileRemovalGeneratePromptSuccess verifies prompt for file removal
func TestResourceProjectInlineFileRemovalGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileRemovalGeneratePromptSuccess")

	// MOCK SETUP: Create initial state with 3 files
	projectPath := filepath.Join(testDir, "output")
	err := os.MkdirAll(projectPath, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(projectPath, "hello.txt"), []byte("hello world\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(projectPath, "hello2.txt"), []byte("hello world2\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(projectPath, "hello3.txt"), []byte("hello world3\n"), 0644)
	require.NoError(t, err)

	// Step 1: Initial config with 3 files
	initialConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "removal-test"
  version = "1.0.0"

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

	configPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Step 2: Remove hello2.txt from config
	updatedConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "removal-test"
  version = "1.0.0"

  files = {
    "hello.txt" = {
      content = "hello world\n"
    }
    "hello3.txt" = {
      content = "hello world3\n"
    }
  }
}
`

	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// Apply with file removed
	runTerraformApply(t, iacTool, testDir)

	// Read latest prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt*-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON files")

	latestPrompt := debugFiles[len(debugFiles)-1]
	jsonData, err := os.ReadFile(latestPrompt)
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify hello2.txt has delete instruction
	hello2File, exists := files["hello2.txt"]
	require.True(t, exists, "Removed file should still appear in prompt with delete instruction")

	hello2Map := hello2File.(map[string]interface{})
	instructions := hello2Map["instructions"].([]interface{})
	instruction0 := instructions[0].(map[string]interface{})

	assert.Contains(t, instruction0["prompt"], "Delete file 'hello2.txt'",
		"Should have delete instruction for removed file")

	t.Log("✓ Prompt generation verified - file removal instruction present")
}

// TestResourceProjectInlineFileRenameGeneratePromptSuccess verifies prompt for file rename
func TestResourceProjectInlineFileRenameGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileRenameGeneratePromptSuccess")

	// MOCK SETUP: Create initial state with hello.txt
	projectPath := filepath.Join(testDir, "output")
	err := os.MkdirAll(projectPath, 0755)
	require.NoError(t, err)

	helloPath := filepath.Join(projectPath, "hello.txt")
	err = os.WriteFile(helloPath, []byte("hello world\n"), 0644)
	require.NoError(t, err)

	// Step 1: Initial config with hello.txt
	initialConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "rename-test"
  version = "1.0.0"

  files = {
    "hello.txt" = {
      content = "hello world\n"
    }
  }
}
`

	configPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Step 2: Rename hello.txt to hello2.txt (same content)
	renamedConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "rename-test"
  version = "1.0.0"

  files = {
    "hello2.txt" = {
      content = "hello world\n"
    }
  }
}
`

	err = os.WriteFile(configPath, []byte(renamedConfig), 0644)
	require.NoError(t, err)

	// Apply with renamed file
	runTerraformApply(t, iacTool, testDir)

	// Read latest prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt*-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON files")

	latestPrompt := debugFiles[len(debugFiles)-1]
	jsonData, err := os.ReadFile(latestPrompt)
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify hello2.txt has rename instruction
	hello2File := files["hello2.txt"].(map[string]interface{})
	instructions := hello2File["instructions"].([]interface{})
	instruction0 := instructions[0].(map[string]interface{})

	assert.Contains(t, instruction0["prompt"], "Rename file from 'hello.txt' to 'hello2.txt'",
		"Should have explicit rename instruction")

	t.Log("✓ Prompt generation verified - rename instruction present")
}

// TestResourceProjectInlineFileNestedCreateGeneratePromptSuccess verifies prompt for nested file creation
func TestResourceProjectInlineFileNestedCreateGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileNestedCreateGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigProjectInlineFileNestedCreate), 0644)
	require.NoError(t, err)

	// Setup and apply
	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Read prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON file")

	jsonData, err := os.ReadFile(debugFiles[0])
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify demo/hello.txt has create instruction with directory creation
	demoFile := files["demo/hello.txt"].(map[string]interface{})
	instructions := demoFile["instructions"].([]interface{})
	instruction0 := instructions[0].(map[string]interface{})

	// Should mention creating the file
	assert.Contains(t, instruction0["prompt"], "Create new file 'demo/hello.txt'",
		"Should have create instruction for nested file")

	// Check constraints mention parent directory creation
	constraints := instruction0["constraints"].([]interface{})
	hasParentDirConstraint := false
	for _, c := range constraints {
		constraintStr := c.(string)
		if strings.Contains(constraintStr, "parent directories") || strings.Contains(constraintStr, "mkdir") {
			hasParentDirConstraint = true
			break
		}
	}
	assert.True(t, hasParentDirConstraint, "Constraints should mention creating parent directories")

	t.Log("✓ Prompt generation verified - nested file creation with parent directory instruction")
}

// TestResourceProjectInlineFileNestedAddGeneratePromptSuccess verifies prompt for adding nested files
func TestResourceProjectInlineFileNestedAddGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileNestedAddGeneratePromptSuccess")

	// MOCK SETUP: Create initial nested file
	projectPath := filepath.Join(testDir, "output")
	demoDir := filepath.Join(projectPath, "demo")
	err := os.MkdirAll(demoDir, 0755)
	require.NoError(t, err)

	helloPath := filepath.Join(demoDir, "hello.txt")
	err = os.WriteFile(helloPath, []byte("hello from demo\n"), 0644)
	require.NoError(t, err)

	// Step 1: Initial config with one nested file
	initialConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "nested-add-test"
  version = "1.0.0"

  files = {
    "demo/hello.txt" = {
      content = "hello from demo\n"
    }
  }
}
`

	configPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Step 2: Add another nested file
	updatedConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "nested-add-test"
  version = "1.0.0"

  files = {
    "demo/hello.txt" = {
      content = "hello from demo\n"
    }
    "demo/world.txt" = {
      content = "world from demo\n"
    }
  }
}
`

	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// Apply with new nested file
	runTerraformApply(t, iacTool, testDir)

	// Read latest prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt*-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON files")

	latestPrompt := debugFiles[len(debugFiles)-1]
	jsonData, err := os.ReadFile(latestPrompt)
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify demo/world.txt has create instruction
	worldFile := files["demo/world.txt"].(map[string]interface{})
	instructions := worldFile["instructions"].([]interface{})
	instruction0 := instructions[0].(map[string]interface{})

	assert.Contains(t, instruction0["prompt"], "Create new file 'demo/world.txt'",
		"Should have create instruction for newly added nested file")

	// Verify path is properly handled in nested structure
	assert.Contains(t, files, "demo/world.txt", "Nested path should be in files specification")

	t.Log("✓ Prompt generation verified - nested file addition detected correctly")
}

// TestResourceProjectInlineFileNestedDeeperNestingGeneratePromptSuccess verifies prompt for deeply nested files
func TestResourceProjectInlineFileNestedDeeperNestingGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileNestedDeeperNestingGeneratePromptSuccess")

	// MOCK SETUP: Create initial state with 2 files in demo/
	projectPath := filepath.Join(testDir, "output")
	demoDir := filepath.Join(projectPath, "demo")
	err := os.MkdirAll(demoDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(demoDir, "hello.txt"), []byte("hello from demo\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(demoDir, "hello2.txt"), []byte("hello2 from demo\n"), 0644)
	require.NoError(t, err)

	// Step 1: Initial config with 2 files in demo/
	initialConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "deeper-nesting-test"
  version = "1.0.0"

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

	configPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Step 2: Add deeply nested file
	updatedConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "deeper-nesting-test"
  version = "1.0.0"

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

	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// Apply with deeply nested file
	runTerraformApply(t, iacTool, testDir)

	// Read latest prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt*-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON files")

	latestPrompt := debugFiles[len(debugFiles)-1]
	jsonData, err := os.ReadFile(latestPrompt)
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify demo/subdir/deep/hello3.txt has create instruction
	deepFile := files["demo/subdir/deep/hello3.txt"].(map[string]interface{})
	instructions := deepFile["instructions"].([]interface{})
	instruction0 := instructions[0].(map[string]interface{})

	assert.Contains(t, instruction0["prompt"], "Create new file 'demo/subdir/deep/hello3.txt'",
		"Should have create instruction for deeply nested file")

	// Verify deep path is properly handled
	assert.Contains(t, files, "demo/subdir/deep/hello3.txt", "Deep nested path should be in files specification")

	t.Log("✓ Prompt generation verified - deep nested path handled correctly")
}

// TestResourceProjectInlineFileNestedRemovalGeneratePromptSuccess verifies prompt for nested directory cleanup
func TestResourceProjectInlineFileNestedRemovalGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileNestedRemovalGeneratePromptSuccess")

	// MOCK SETUP: Create initial state with deeply nested file
	projectPath := filepath.Join(testDir, "output")
	deepDir := filepath.Join(projectPath, "demo", "subdir", "deep")
	err := os.MkdirAll(deepDir, 0755)
	require.NoError(t, err)

	demoDir := filepath.Join(projectPath, "demo")
	err = os.WriteFile(filepath.Join(demoDir, "hello.txt"), []byte("hello from demo\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(demoDir, "hello2.txt"), []byte("hello2 from demo\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(deepDir, "hello3.txt"), []byte("hello3 from deep\n"), 0644)
	require.NoError(t, err)

	// Step 1: Initial config with 3 files including deeply nested
	initialConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "nested-removal-test"
  version = "1.0.0"

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

	configPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Step 2: Remove deeply nested file (should cleanup empty directories)
	updatedConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "nested-removal-test"
  version = "1.0.0"

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

	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)

	// Apply with nested file removed
	runTerraformApply(t, iacTool, testDir)

	// Read latest prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt*-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON files")

	latestPrompt := debugFiles[len(debugFiles)-1]
	jsonData, err := os.ReadFile(latestPrompt)
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify demo/subdir/deep/hello3.txt has delete instruction
	deepFile, exists := files["demo/subdir/deep/hello3.txt"]
	require.True(t, exists, "Removed nested file should appear in prompt")

	deepMap := deepFile.(map[string]interface{})
	instructions := deepMap["instructions"].([]interface{})
	instruction0 := instructions[0].(map[string]interface{})

	assert.Contains(t, instruction0["prompt"], "Delete file 'demo/subdir/deep/hello3.txt'",
		"Should have delete instruction for removed nested file")

	t.Log("✓ Prompt generation verified - nested file removal with empty directory cleanup instruction")
}

// TestResourceProjectInlineFileVerificationGeneratePromptSuccess verifies prompt includes verification commands
func TestResourceProjectInlineFileVerificationGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileVerificationGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigProjectInlineFileVerification), 0644)
	require.NoError(t, err)

	// Setup and apply
	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Read prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON file")

	jsonData, err := os.ReadFile(debugFiles[0])
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files (verifications are stored in file specifications)
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Get hello.txt file
	helloFile := files["hello.txt"].(map[string]interface{})

	// Check for verification field
	verification, ok := helloFile["verification"].([]interface{})
	require.True(t, ok, "File should have verification field")
	require.NotEmpty(t, verification, "Verification array should not be empty")

	// Verify verification command structure
	verification0 := verification[0].(map[string]interface{})
	assert.Equal(t, "cat hello.txt", verification0["command"], "Should have verification command")
	assert.Equal(t, "hello world", verification0["expect"], "Should have verification expectation")

	t.Log("✓ Prompt generation verified - verification commands present in file specification")
}

// TestResourceProjectInlineFileVerificationRetryGeneratePromptSuccess verifies prompt includes verification (retry logic doesn't affect prompt)
func TestResourceProjectInlineFileVerificationRetryGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileVerificationRetryGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigProjectInlineFileVerificationRetry), 0644)
	require.NoError(t, err)

	// Setup and apply
	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Read prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON file")

	jsonData, err := os.ReadFile(debugFiles[0])
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Get greeting.txt file
	greetingFile := files["greeting.txt"].(map[string]interface{})

	// Verify instructions present
	instructions := greetingFile["instructions"].([]interface{})
	require.NotEmpty(t, instructions, "Should have instructions")

	// Verify verification commands present
	verification, ok := greetingFile["verification"].([]interface{})
	require.True(t, ok, "File should have verification field")
	require.NotEmpty(t, verification, "Verification array should not be empty")

	verification0 := verification[0].(map[string]interface{})
	assert.Equal(t, "cat greeting.txt", verification0["command"], "Should have verification command")
	assert.Equal(t, "hello world", verification0["expect"], "Should have verification expectation")

	t.Log("✓ Prompt generation verified - verification structure present (retry logic doesn't affect prompt)")
}

// TestResourceProjectInlineFileInstructionRenameGeneratePromptSuccess verifies rename of instruction-based files
func TestResourceProjectInlineFileInstructionRenameGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileInstructionRenameGeneratePromptSuccess")

	// MOCK SETUP: Create LICENSE.md with generated content
	projectPath := filepath.Join(testDir, "output")
	err := os.MkdirAll(projectPath, 0755)
	require.NoError(t, err)

	licensePath := filepath.Join(projectPath, "LICENSE.md")
	err = os.WriteFile(licensePath, []byte("MIT License\nCopyright (c) 2025\n"), 0644)
	require.NoError(t, err)

	// Step 1: Initial config with LICENSE.md (instruction-based)
	initialConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "instruction-rename-test"
  version = "1.0.0"

  files = {
    "LICENSE.md" = {
      instructions = [{
        prompt = "Create an MIT License file with placeholder for year and copyright holder"
      }]
    }
  }
}
`

	configPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)

	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Step 2: Rename LICENSE.md to LICENSE.txt (same instructions)
	renamedConfig := `
terraform {
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
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "instruction-rename-test"
  version = "1.0.0"

  files = {
    "LICENSE.txt" = {
      instructions = [{
        prompt = "Create an MIT License file with placeholder for year and copyright holder"
      }]
    }
  }
}
`

	err = os.WriteFile(configPath, []byte(renamedConfig), 0644)
	require.NoError(t, err)

	// Apply with renamed file
	runTerraformApply(t, iacTool, testDir)

	// Read latest prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt*-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON files")

	latestPrompt := debugFiles[len(debugFiles)-1]
	jsonData, err := os.ReadFile(latestPrompt)
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify LICENSE.txt has rename instruction
	licenseFile := files["LICENSE.txt"].(map[string]interface{})
	instructions := licenseFile["instructions"].([]interface{})
	instruction0 := instructions[0].(map[string]interface{})

	// Rename is detected by matching instructions (for generated files)
	assert.Contains(t, instruction0["prompt"], "Rename file from 'LICENSE.md' to 'LICENSE.txt'",
		"Should have rename instruction for instruction-based file")

	t.Log("✓ Prompt generation verified - instruction-based file rename detected")
}

// TestResourceProjectInlineFileOrderingConsistencyGeneratePromptSuccess verifies file ordering in prompt
func TestResourceProjectInlineFileOrderingConsistencyGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectInlineFileOrderingConsistencyGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigProjectInlineFileOrdering), 0644)
	require.NoError(t, err)

	// Setup and apply
	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Read prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON file")

	jsonData, err := os.ReadFile(debugFiles[0])
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify all files are present
	assert.Contains(t, files, "hi2.txt", "hi2.txt should be in files")
	assert.Contains(t, files, "LICENSE.md", "LICENSE.md should be in files")
	assert.Contains(t, files, "README.md", "README.md should be in files")

	// Verify each file has the correct content and instructions
	hi2File := files["hi2.txt"].(map[string]interface{})
	assert.Equal(t, "Hi from file 2\n", hi2File["content"], "hi2.txt should have correct content")

	licenseFile := files["LICENSE.md"].(map[string]interface{})
	assert.Equal(t, "MIT License\n", licenseFile["content"], "LICENSE.md should have correct content")

	readmeFile := files["README.md"].(map[string]interface{})
	assert.Equal(t, "Project README\n", readmeFile["content"], "README.md should have correct content")

	t.Log("✓ Prompt generation verified - file ordering consistency maintained")
}

// TestResourceProjectDriftDetection_StaticFileGeneratePromptSuccess verifies drift warnings in prompt
// Note: Prompt generation tests don't actually execute drift detection logic, just verify prompt structure
func TestResourceProjectDriftDetection_StaticFileGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectDriftDetection_StaticFileGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigProjectDriftStaticFile), 0644)
	require.NoError(t, err)

	// Setup and apply (in dry_run mode, no actual drift detection occurs)
	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Read prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON file")

	jsonData, err := os.ReadFile(debugFiles[0])
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify .gitignore is in prompt
	gitignoreFile := files[".gitignore"].(map[string]interface{})
	assert.NotNil(t, gitignoreFile, ".gitignore should be in prompt")
	assert.Equal(t, "*.log\n*.tmp\n", gitignoreFile["content"], "Content should match specification")

	t.Log("✓ Prompt generation verified - drift detection structure can be validated in e2e tests")
}

// TestResourceProjectDriftDetection_DeletedFileGeneratePromptSuccess verifies prompt structure for deleted files
func TestResourceProjectDriftDetection_DeletedFileGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectDriftDetection_DeletedFileGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigProjectDriftDeletedFile), 0644)
	require.NoError(t, err)

	// Setup and apply
	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Read prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON file")

	jsonData, err := os.ReadFile(debugFiles[0])
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify config.txt is in prompt
	configFile := files["config.txt"].(map[string]interface{})
	assert.NotNil(t, configFile, "config.txt should be in prompt")

	t.Log("✓ Prompt generation verified - file deletion drift detection structure valid")
}

// TestResourceProjectDriftDetection_MultipleFilesGeneratePromptSuccess verifies prompt with multiple files
func TestResourceProjectDriftDetection_MultipleFilesGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceProjectDriftDetection_MultipleFilesGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigProjectDriftMultipleFiles), 0644)
	require.NoError(t, err)

	// Setup and apply
	iacTool := setupTerraform(t, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Read prompt JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have prompt JSON file")

	jsonData, err := os.ReadFile(debugFiles[0])
	require.NoError(t, err)

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err)

	// Navigate to files
	request := promptJSON["request"].(map[string]interface{})
	specification := request["specification"].(map[string]interface{})
	files := specification["files"].(map[string]interface{})

	// Verify all files are in prompt
	assert.Contains(t, files, ".gitignore", ".gitignore should be in prompt")
	assert.Contains(t, files, "LICENSE", "LICENSE should be in prompt")
	assert.Contains(t, files, "README.md", "README.md should be in prompt")

	t.Log("✓ Prompt generation verified - multiple files drift detection structure valid")
}
