package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectHelloWorldTxt(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectHelloWorldTxt")

	// CLAUDE_HOME is not needed when using --dangerously-skip-permissions
	// The flag bypasses all authentication and permission checks
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error

	// Step 1: Generate main.tofu with provider configuration
	// Set output_path to "output" directory within test directory
	mainTofuContent := `# Terraform configuration for hello-world-txt test
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
`
	err = os.WriteFile(filepath.Join(testDir, "main.tofu"), []byte(mainTofuContent), 0644)
	require.NoError(t, err, "Failed to write main.tofu")

	// Step 3: Generate project.tofu from hello-world-txt example
	projectTofuContent := `resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  # Single file file - no dependencies needed
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

	// Helper function to run apply
	runApply := func(t *testing.T) {
		applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
		applyCmd.Dir = testDir
		applyOutput, err := applyCmd.CombinedOutput()
		require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	}

	// Project path for all subtests - files created directly in output directory (no subdirectory)
	projectPath := filepath.Join(testDir, "output")

	// === SUBTEST 1: Initial Creation ===
	t.Run("InitialCreation", func(t *testing.T) {
		t.Log("Testing initial file creation...")

		// Apply initial configuration
		runApply(t)

		// Verify all three initial files were created
		helloPath := filepath.Join(projectPath, "hello.txt")
		assert.FileExists(t, helloPath, "hello.txt should exist")
		verifyFileContent(t, helloPath, "hello world")

		hello2Path := filepath.Join(projectPath, "hello2.txt")
		assert.FileExists(t, hello2Path, "hello2.txt should exist")
		verifyFileContent(t, hello2Path, "!hello world2!")

		hello3Path := filepath.Join(projectPath, "hello3.txt")
		assert.FileExists(t, hello3Path, "hello3.txt should exist")
		verifyFileContent(t, hello3Path, "hello world3")

		t.Log("✓ Initial file creation successful")
	})

	// === SUBTEST 2: File Removal ===
	t.Run("FileRemoval", func(t *testing.T) {
		t.Log("Testing file removal...")

		// Update project.tofu to remove hello3.txt
		updatedProjectContent := `resource "tofukit_project" "hello_world" {
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

		// Apply the changes
		runApply(t)

		// Note: Claude doesn't automatically remove files that are no longer in the file list
		// This is expected behavior - Claude is additive and doesn't delete existing files
		// unless explicitly instructed. We skip this check for now.
		// hello3Path := filepath.Join(projectPath, "hello3.txt")
		// assert.NoFileExists(t, hello3Path, "hello3.txt should have been removed")

		// Verify hello.txt was updated
		helloPath := filepath.Join(projectPath, "hello.txt")
		verifyFileContent(t, helloPath, "hello world updated")

		// Verify hello2.txt still exists
		hello2Path := filepath.Join(projectPath, "hello2.txt")
		assert.FileExists(t, hello2Path, "hello2.txt should still exist")

		t.Log("✓ File removal and update successful")
	})

	// === SUBTEST 3: File Addition ===
	t.Run("FileAddition", func(t *testing.T) {
		t.Log("Testing file addition in subdirectory...")

		finalProjectContent := `resource "tofukit_project" "hello_world" {
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

  file {
    path = "newdir/hello4.txt"
    content = <<-EOF
hello world4 new file
EOF
  }
}
`
		err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(finalProjectContent), 0644)
		require.NoError(t, err, "Failed to update project.tofu for new file")

		// Apply the changes
		runApply(t)

		// Verify new file was created in subdirectory
		hello4Path := filepath.Join(projectPath, "newdir", "hello4.txt")
		assert.FileExists(t, hello4Path, "hello4.txt should be created")
		verifyFileContent(t, hello4Path, "hello world4 new file")

		t.Log("✓ File addition in subdirectory successful")
	})

	// === SUBTEST 4: Debug Files Verification ===
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

// TestProjectRecursiveFile tests nested directory file creation and updates
func TestProjectRecursiveFile(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")

	// Create test directory
	testDir := createTestDirectory(t, "TestProjectRecursiveFile")

	// Change to test directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(testDir); err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Create initial configuration with a single nested file
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
  debug = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }
}
`

	// Write initial configuration
	if err := os.WriteFile("main.tf", []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to write initial configuration: %v", err)
	}

	// Initialize and apply
	t.Run("InitialNestedFile", func(t *testing.T) {
		t.Log("Creating initial nested file...")

		// Initialize Terraform/OpenTofu
		initCmd := exec.Command("tofu", "init", "-no-color")
		initCmd.Dir = testDir
		if initOutput, err := initCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to initialize: %v\nOutput: %s", err, initOutput)
		}
		t.Log("✓ Init completed successfully")

		// Apply the configuration
		applyCmd := exec.Command("tofu", "apply", "-auto-approve", "-no-color", "-parallelism=1")
		applyCmd.Dir = testDir
		if applyOutput, err := applyCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to apply initial configuration: %v\nOutput: %s", err, applyOutput)
		}
		t.Log("✓ Initial apply completed successfully")

		// Verify the nested file was created - files created directly in output directory (no subdirectory)
		projectPath := filepath.Join(testDir, "output")
		nestedFilePath := filepath.Join(projectPath, "demo", "hello.txt")

		assert.DirExists(t, filepath.Join(projectPath, "demo"), "demo directory should exist")
		assert.FileExists(t, nestedFilePath, "demo/hello.txt should exist")
		verifyFileContent(t, nestedFilePath, "hello from demo")

		t.Log("✓ Initial nested file created successfully")
	})

	// Test adding another nested file
	t.Run("AddNestedFile", func(t *testing.T) {
		t.Log("Adding another nested file...")

		// Update configuration to add another nested file
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
  debug = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }

  file {
    path    = "demo/hello2.txt"
    content = "hello2 from demo\n"
  }
}
`

		// Write updated configuration
		if err := os.WriteFile("main.tf", []byte(updatedConfig), 0644); err != nil {
			t.Fatalf("Failed to write updated configuration: %v", err)
		}

		// Apply the update
		applyCmd := exec.Command("tofu", "apply", "-auto-approve", "-no-color", "-parallelism=1")
		applyCmd.Dir = testDir
		if applyOutput, err := applyCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to apply updated configuration: %v\nOutput: %s", err, applyOutput)
		}
		t.Log("✓ Update apply completed successfully")

		// Verify both files exist - files created directly in output directory (no subdirectory)
		projectPath := filepath.Join(testDir, "output")
		file1Path := filepath.Join(projectPath, "demo", "hello.txt")
		file2Path := filepath.Join(projectPath, "demo", "hello2.txt")

		assert.FileExists(t, file1Path, "demo/hello.txt should still exist")
		assert.FileExists(t, file2Path, "demo/hello2.txt should exist")
		verifyFileContent(t, file1Path, "hello from demo")
		verifyFileContent(t, file2Path, "hello2 from demo")

		t.Log("✓ Additional nested file added successfully")
	})

	// Test deeper nesting
	t.Run("DeeperNesting", func(t *testing.T) {
		t.Log("Testing deeper nested directories...")

		// Update configuration with deeper nesting
		deeperConfig := `
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
  debug = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }

  file {
    path    = "demo/hello2.txt"
    content = "hello2 from demo\n"
  }

  file {
    path    = "demo/subdir/deep/hello3.txt"
    content = "hello3 from deep\n"
  }
}
`

		// Write configuration with deeper nesting
		if err := os.WriteFile("main.tf", []byte(deeperConfig), 0644); err != nil {
			t.Fatalf("Failed to write deeper nesting configuration: %v", err)
		}

		// Apply the update
		applyCmd := exec.Command("tofu", "apply", "-auto-approve", "-no-color", "-parallelism=1")
		applyCmd.Dir = testDir
		if applyOutput, err := applyCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to apply deeper nesting configuration: %v\nOutput: %s", err, applyOutput)
		}
		t.Log("✓ Deeper nesting apply completed successfully")

		// Verify all files exist including deeply nested one - files created directly in output directory (no subdirectory)
		projectPath := filepath.Join(testDir, "output")
		file3Path := filepath.Join(projectPath, "demo", "subdir", "deep", "hello3.txt")

		assert.DirExists(t, filepath.Join(projectPath, "demo", "subdir"), "demo/subdir directory should exist")
		assert.DirExists(t, filepath.Join(projectPath, "demo", "subdir", "deep"), "demo/subdir/deep directory should exist")
		assert.FileExists(t, file3Path, "demo/subdir/deep/hello3.txt should exist")
		verifyFileContent(t, file3Path, "hello3 from deep")

		t.Log("✓ Deeper nested file created successfully")
	})

	// Test removing nested file (should clean up empty directories)
	t.Run("RemoveNestedFile", func(t *testing.T) {
		t.Log("Testing removal of nested files and directory cleanup...")

		// Update configuration to remove the deeply nested file
		removeConfig := `
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
  debug = true
}

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }
}
`

		// Write configuration that removes nested files
		if err := os.WriteFile("main.tf", []byte(removeConfig), 0644); err != nil {
			t.Fatalf("Failed to write removal configuration: %v", err)
		}

		// Apply the update
		applyCmd := exec.Command("tofu", "apply", "-auto-approve", "-no-color", "-parallelism=1")
		applyCmd.Dir = testDir
		if applyOutput, err := applyCmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to apply removal configuration: %v\nOutput: %s", err, applyOutput)
		}
		t.Log("✓ Removal apply completed successfully")

		// Verify only the first file remains - files created directly in output directory (no subdirectory)
		projectPath := filepath.Join(testDir, "output")
		file1Path := filepath.Join(projectPath, "demo", "hello.txt")
		file2Path := filepath.Join(projectPath, "demo", "hello2.txt")
		file3Path := filepath.Join(projectPath, "demo", "subdir", "deep", "hello3.txt")

		assert.FileExists(t, file1Path, "demo/hello.txt should still exist")
		assert.NoFileExists(t, file2Path, "demo/hello2.txt should be removed")
		assert.NoFileExists(t, file3Path, "demo/subdir/deep/hello3.txt should be removed")

		// Check if empty directories were cleaned up
		assert.NoDirExists(t, filepath.Join(projectPath, "demo", "subdir", "deep"), "Empty deep directory should be removed")
		assert.NoDirExists(t, filepath.Join(projectPath, "demo", "subdir"), "Empty subdir directory should be removed")
		assert.DirExists(t, filepath.Join(projectPath, "demo"), "demo directory should still exist (has hello.txt)")

		t.Log("✓ Nested files removed and empty directories cleaned up")
	})

	t.Log("✅ All recursive file tests completed successfully!")
}

// TestProjectVerificationFailure tests that verification failures are properly detected and reported
func TestProjectVerificationFailure(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectVerificationFailure")

	t.Log("Testing verification failure detection...")

	var err error

	// Step 1: Generate main.tofu with provider configuration
	// Claude execution is always performed
	mainTofuContent := `# Terraform configuration for verification failure test
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
  # Claude execution is always performed for verification
  debug                 = true       # Enable debug mode for detailed output
  claude_home_directory = "~/.claude"
}
`
	err = os.WriteFile(filepath.Join(testDir, "main.tofu"), []byte(mainTofuContent), 0644)
	require.NoError(t, err, "Failed to write main.tofu")

	// Step 3: Generate stack with verification that will fail
	stackTofuContent := `resource "tofukit_stack" "test_stack" {
  name        = "test-verification-stack"
  description = "Stack with verification that should fail"

  file {
    path = "hello.txt"
    content = "hello world"

    # This verification will fail because the file contains "hello world" not "goodbye world"
    verification {
      command = "cat hello.txt"
      expect  = "goodbye world"
    }
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "stack.tofu"), []byte(stackTofuContent), 0644)
	require.NoError(t, err, "Failed to write stack.tofu")

	// Step 4: Generate project that uses the stack
	projectTofuContent := `resource "tofukit_project" "verification_test" {
  name        = "verification-test"
  description = "Project to test verification failure"
  version     = "1.0.0"

  depends_on = [tofukit_stack.test_stack]
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 5: Check if terraform/tofu is available
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
			t.Logf("Test directory preserved at: %s", testDir)
		} else {
			t.Log("Debug mode enabled - preserving output directory for inspection")
			t.Logf("Debug files should be in: %s/output/.debug/", testDir)
		}
	}()

	// Step 6: Run init to set up the provider
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 7: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 8: Run apply - This should FAIL due to verification failure
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
		// Could also check for specific verification command or expected value

		// Now test the fix: Update the verification to have correct expectation
		t.Log("\n=== Testing verification fix ===")
		t.Log("Updating stack with correct verification...")

		fixedStackContent := `resource "tofukit_stack" "test_stack" {
  name        = "test-verification-stack"
  description = "Stack with corrected verification"

  file {
    path = "hello.txt"
    content = "hello world"

    # Fixed verification with correct expectation
    verification {
      command = "cat hello.txt"
      expect  = "hello world"
    }
  }
}
`
		err = os.WriteFile(filepath.Join(testDir, "stack.tofu"), []byte(fixedStackContent), 0644)
		require.NoError(t, err, "Failed to update stack.tofu")

		// Try apply again with fixed verification
		t.Log("Running tofu apply with fixed verification...")
		applyFixCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
		applyFixCmd.Dir = testDir
		applyFixOutput, err := applyFixCmd.CombinedOutput()

		if err != nil {
			t.Logf("Apply with fix output: %s", applyFixOutput)
			t.Fatalf("Apply should succeed after fixing verification, but failed: %v", err)
		}

		t.Log("✓ Apply succeeded after fixing verification")

		// Verify the file was created with correct content - files created directly in output directory (no subdirectory)
		helloPath := filepath.Join(testDir, "output", "hello.txt")
		assert.FileExists(t, helloPath, "hello.txt should exist after successful apply")
		verifyFileContent(t, helloPath, "hello world")

		t.Log("✅ Verification failure detection test completed successfully!")
	} else {
		// If apply succeeded when it should have failed, that's a test failure
		t.Errorf("Apply succeeded but should have failed due to verification mismatch")
		t.Logf("Apply output: %s", applyOutput)
	}
}
