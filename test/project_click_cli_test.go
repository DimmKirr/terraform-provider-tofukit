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

// TestProjectClickCLI tests the Python Click CLI project generation and management
func TestProjectClickCLI(t *testing.T) {
	// Set debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectClickCLI")

	// Get project root
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	// Step 1: Build and install the provider locally
	t.Log("Building and installing provider...")
	buildCmd := exec.Command("make", "install")
	buildCmd.Dir = projectRoot
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build provider: %s", output)
	t.Log("✓ Provider built and installed successfully")

	// Step 2: Copy the example configuration files
	exampleDir := filepath.Join(projectRoot, "examples", "click-cli-hello-world")

	// Read and modify the main.tofu to use test settings
	mainContent := `# Terraform configuration for Click CLI test
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration for testing
provider "tofukit" {
  output_format         = "json"
  output_path           = "output"  # Output will be in {test_directory}/output/
  dry_run               = false     # Enable actual file creation
  claude_home_directory = "~/.claude"
  debug                 = true      # Enable debug mode for detailed output
}
`

	// Write the main configuration
	mainPath := filepath.Join(testDir, "main.tofu")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("Failed to write main.tofu: %v", err)
	}

	// Copy kits.tofu
	kitsContent, err := os.ReadFile(filepath.Join(exampleDir, "kits.tofu"))
	if err != nil {
		t.Fatalf("Failed to read kits.tofu: %v", err)
	}
	kitsPath := filepath.Join(testDir, "kits.tofu")
	if err := os.WriteFile(kitsPath, kitsContent, 0644); err != nil {
		t.Fatalf("Failed to write kits.tofu: %v", err)
	}

	// Copy project.tofu (we'll store it for later modification)
	projectContent, err := os.ReadFile(filepath.Join(exampleDir, "project.tofu"))
	if err != nil {
		t.Fatalf("Failed to read project.tofu: %v", err)
	}
	projectPath := filepath.Join(testDir, "project.tofu")
	if err := os.WriteFile(projectPath, projectContent, 0644); err != nil {
		t.Fatalf("Failed to write project.tofu: %v", err)
	}

	// Copy outputs.tofu
	outputsContent, err := os.ReadFile(filepath.Join(exampleDir, "outputs.tofu"))
	if err != nil {
		t.Fatalf("Failed to read outputs.tofu: %v", err)
	}
	outputsPath := filepath.Join(testDir, "outputs.tofu")
	if err := os.WriteFile(outputsPath, outputsContent, 0644); err != nil {
		t.Fatalf("Failed to write outputs.tofu: %v", err)
	}

	// Step 3: Check if terraform/tofu is available
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
			}
		}
	}()

	// Step 4: Run init to set up the provider
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
	t.Logf("Plan output:\n%s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 6: Run apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Helper function to run apply
	runApply := func(t *testing.T) {
		applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color")
		applyCmd.Dir = testDir
		applyOutput, err := applyCmd.CombinedOutput()
		require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	}

	// === SUBTEST 1: Initial Creation ===
	t.Run("InitialCreation", func(t *testing.T) {
		t.Log("Testing initial Python Click CLI project creation...")

		// Note: With dry_run=true, the tofukit_project resource doesn't create files
		// However, the tofukit_stack resource still creates its scaffolds
		stackPath := filepath.Join(testDir, "output", "stacks", "click-application-stack")

		// Verify key Python files were created by the stack resource
		// Check main CLI file
		cliPath := filepath.Join(stackPath, "src", "cli.py")
		assert.FileExists(t, cliPath, "src/cli.py should exist")
		verifyFileContent(t, cliPath, "Click framework")

		// Check test file
		testPath := filepath.Join(stackPath, "tests", "test_cli.py")
		assert.FileExists(t, testPath, "tests/test_cli.py should exist")
		verifyFileContent(t, testPath, "Unit tests for the CLI")

		// Check configuration files
		pyprojectPath := filepath.Join(stackPath, "pyproject.toml")
		assert.FileExists(t, pyprojectPath, "pyproject.toml should exist")
		verifyFileContent(t, pyprojectPath, "build-system")

		requirementsPath := filepath.Join(stackPath, "requirements.txt")
		assert.FileExists(t, requirementsPath, "requirements.txt should exist")
		verifyFileContent(t, requirementsPath, "click==8.1.7")

		// Check Makefile
		makefilePath := filepath.Join(stackPath, "Makefile")
		assert.FileExists(t, makefilePath, "Makefile should exist")
		verifyFileContent(t, makefilePath, "help")

		// Check README
		readmePath := filepath.Join(stackPath, "README.md")
		assert.FileExists(t, readmePath, "README.md should exist")
		verifyFileContent(t, readmePath, "Python Click CLI")

		// Check Docker and CI files
		dockerPath := filepath.Join(stackPath, "Dockerfile")
		assert.FileExists(t, dockerPath, "Dockerfile should exist")

		ciPath := filepath.Join(stackPath, ".github", "workflows", "ci.yml")
		assert.FileExists(t, ciPath, ".github/workflows/ci.yml should exist")

		gitignorePath := filepath.Join(stackPath, ".gitignore")
		assert.FileExists(t, gitignorePath, ".gitignore should exist")

		t.Log("✓ Initial Python Click CLI project creation successful")
		t.Log("  Note: Files created by tofukit_stack resource in stacks/ directory")
	})

	// === SUBTEST 2: Scaffold Update ===
	t.Run("ScaffoldUpdate", func(t *testing.T) {
		t.Log("Testing scaffold update...")

		// Stack output path (stack resource creates files here)
		stackPath := filepath.Join(testDir, "output", "stacks", "click-application-stack")

		// Update project.tofu to modify requirements.txt
		updatedProjectContent := strings.Replace(
			string(projectContent),
			"click==8.1.7",
			"click==8.1.7\nrich==13.7.0",
			1,
		)

		if err := os.WriteFile(projectPath, []byte(updatedProjectContent), 0644); err != nil {
			t.Fatalf("Failed to update project.tofu: %v", err)
		}

		// Apply the changes
		runApply(t)

		// Verify the requirements.txt was updated (in stack directory)
		requirementsPath := filepath.Join(stackPath, "requirements.txt")
		assert.FileExists(t, requirementsPath, "requirements.txt should still exist")
		content, err := os.ReadFile(requirementsPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "rich==13.7.0", "requirements.txt should contain rich package")

		t.Log("✓ Scaffold update successful")
	})

	// === SUBTEST 3: New Scaffold Addition ===
	t.Run("NewScaffoldAddition", func(t *testing.T) {
		t.Log("Testing new scaffold addition...")

		// Read current project content
		currentContent, err := os.ReadFile(projectPath)
		require.NoError(t, err)

		// Add a new scaffold to project.tofu
		newScaffold := `

  scaffold {
    path = "src/utils.py"
    content = <<-EOF
    """Utility functions for the CLI application."""

    def format_output(data):
        """Format output data."""
        return str(data)
    EOF
  }`
		// Find the end of the tofukit_project resource block
		// Look for the last scaffold's closing brace within the resource
		lastScaffoldEnd := strings.LastIndex(string(currentContent), "  }\n}")
		if lastScaffoldEnd == -1 {
			// Try alternative pattern
			lastScaffoldEnd = strings.LastIndex(string(currentContent), "EOF\n  }")
			if lastScaffoldEnd == -1 {
				t.Fatal("Could not find proper insertion point in project.tofu")
			}
			// Move to after the closing brace of the scaffold
			lastScaffoldEnd = strings.Index(string(currentContent[lastScaffoldEnd:]), "\n  }") + lastScaffoldEnd + 4
		} else {
			lastScaffoldEnd += 3 // Position after the first }
		}

		// Insert the new scaffold after the last scaffold but before the resource closing brace
		updatedContent := string(currentContent[:lastScaffoldEnd]) + newScaffold + "\n" + string(currentContent[lastScaffoldEnd:])

		if err := os.WriteFile(projectPath, []byte(updatedContent), 0644); err != nil {
			t.Fatalf("Failed to update project.tofu with new scaffold: %v", err)
		}

		// Apply the changes
		runApply(t)

		// Stack output path (stack resource creates files here)
		stackPath := filepath.Join(testDir, "output", "stacks", "click-application-stack")

		// Verify the new file was created
		utilsPath := filepath.Join(stackPath, "src", "utils.py")
		assert.FileExists(t, utilsPath, "src/utils.py should be created")
		verifyFileContent(t, utilsPath, "def format_output(data):")

		t.Log("✓ New scaffold addition successful")
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
			assert.Contains(t, string(spec), "python-click-cli")
			t.Log("  This file contains the complete project specification")
		} else {
			t.Logf("⚠️ Debug specification file not found (pattern: %s)", debugSpecPattern)
		}

		// Note about Claude execution mode
		t.Log("ℹ️  Note: With dry_run=false, actual scaffold files are created")
		t.Log("  Claude execution is still skipped (would require Claude CLI)")

		// Check for JSONL file with timestamp pattern
		jsonlPattern := filepath.Join(testDir, "output", ".debug", "claude-prompt-*.jsonl")
		jsonlFiles, _ := filepath.Glob(jsonlPattern)
		if len(jsonlFiles) > 0 {
			t.Logf("✓ Claude prompt JSONL found: %s", jsonlFiles[0])
			// Read and verify it contains the system prompt
			jsonlContent, err := os.ReadFile(jsonlFiles[0])
			if err == nil && len(jsonlContent) > 0 {
				assert.Contains(t, string(jsonlContent), "30+ years of experience")
				t.Log("  File contains system prompt and project specification")
			}
		} else {
			t.Logf("⚠️  Claude prompt JSONL not found (pattern: %s)", jsonlPattern)
		}

		// Check for markdown prompt files
		mdPattern := filepath.Join(testDir, "output", ".debug", "claude-prompt-*.md")
		mdFiles, _ := filepath.Glob(mdPattern)
		if len(mdFiles) > 0 {
			t.Logf("✓ Claude prompt markdown found: %s", mdFiles[0])
			// Verify it contains project info
			mdContent, err := os.ReadFile(mdFiles[0])
			if err == nil {
				assert.Contains(t, string(mdContent), "python-click-cli")
				assert.Contains(t, string(mdContent), "Click framework")
				t.Log("  File contains formatted prompt for Claude")
			}
		} else {
			t.Logf("⚠️ Claude prompt markdown not found (pattern: %s)", mdPattern)
		}

		// Check for kit configuration in project spec
		if len(debugSpecFiles) > 0 {
			spec, _ := os.ReadFile(debugSpecFiles[0])
			assert.Contains(t, string(spec), "language.python", "Should include Python language kit")
			assert.Contains(t, string(spec), "tool.pip", "Should include pip tool kit")
			assert.Contains(t, string(spec), "tool.black", "Should include black formatter kit")
			assert.Contains(t, string(spec), "tool.ruff", "Should include ruff linter kit")
			assert.Contains(t, string(spec), "methodology.python_standards", "Should include Python standards methodology kit")
			t.Log("✓ All expected kits found in project specification")
		}

		t.Log("✓ Debug files verification complete")
	})

	t.Log("✅ All Python Click CLI tests completed successfully!")
}
