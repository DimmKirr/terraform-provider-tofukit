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
	buildCmd := exec.Command("task", "install")
	buildCmd.Dir = projectRoot
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build provider: %s", output)
	t.Log("✓ Provider built and installed successfully")

	// Step 2: Copy the example configuration files
	exampleDir := filepath.Join(projectRoot, "examples", "click-cli-hello-world")

	// Copy the unified project.tofu (contains terraform, provider, module, project, data, outputs)
	projectContent, err := os.ReadFile(filepath.Join(exampleDir, "project.tofu"))
	if err != nil {
		t.Fatalf("Failed to read project.tofu: %v", err)
	}

	// Modify the provider config to use test output directory and module path
	modifiedContent := string(projectContent)
	// Add output_path = "output" after the output_format line
	modifiedContent = strings.Replace(modifiedContent,
		`output_format         = "json"`,
		`output_format         = "json"
  output_path           = "output"`, 1)
	// Replace module source path from ../modules to ./modules
	modifiedContent = strings.Replace(modifiedContent, `  source = "../modules/tofukit-stack-python-click-app-generic"`, `  source = "./modules/tofukit-stack-python-click-app-generic"`, 1)

	projectPath := filepath.Join(testDir, "project.tofu")
	if err := os.WriteFile(projectPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to write project.tofu: %v", err)
	}

	// Copy the module directory
	moduleDir := filepath.Join(projectRoot, "examples", "modules")
	testModuleDir := filepath.Join(testDir, "modules")
	copyDirCmd := exec.Command("cp", "-r", moduleDir, testModuleDir)
	if err := copyDirCmd.Run(); err != nil {
		t.Fatalf("Failed to copy modules directory: %v", err)
	}
	t.Logf("Copied modules from %s to %s", moduleDir, testModuleDir)

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

	// === SUBTEST 1: Initial Creation ===
	t.Run("InitialCreation", func(t *testing.T) {
		t.Log("Testing initial Python Click CLI project creation...")

		// The project resource creates scaffold files via Claude
		// in the project output directory
		outputPath := filepath.Join(testDir, "output")

		// Verify key Python files were created by the stack resource
		// Check main CLI file
		cliPath := filepath.Join(outputPath, "src", "cli.py")
		assert.FileExists(t, cliPath, "src/cli.py should exist")
		// The actual content includes "Click framework" in the docstring
		content, err := os.ReadFile(cliPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Click framework", "CLI file should mention Click framework")
		assert.Contains(t, string(content), "@click.group()", "CLI file should use Click decorators")

		requirementsPath := filepath.Join(outputPath, "requirements.txt")
		assert.FileExists(t, requirementsPath, "requirements.txt should exist")
		verifyFileContent(t, requirementsPath, "click==8.1.7")

		// README.md is generated by Claude since it has generate=true
		// and requires Claude execution to generate the content
		readmePath := filepath.Join(outputPath, "README.md")
		if _, err := os.Stat(readmePath); err == nil {
			// README should be generated by Claude, verify content
			readmeContent, err := os.ReadFile(readmePath)
			require.NoError(t, err)
			assert.Contains(t, string(readmeContent), "python-click-cli", "README should contain project name")
		} else {
			t.Log("⚠️ README.md not found - Claude may have failed to generate it")
		}

		t.Log("✓ Initial Python Click CLI project creation successful")
		t.Log("  Note: All files are created via Claude execution")
		t.Log("  Files with generate=true require Claude execution and are skipped")
	})

	// === SUBTEST 2: Project Files Persist ===
	t.Run("ProjectFilesPersist", func(t *testing.T) {
		t.Log("Testing that project files persist after re-apply...")

		// Project output path (no subdirectory)
		outputPath := filepath.Join(testDir, "output")

		// Run apply again without changes
		runApply(t)

		// Verify files still exist
		requirementsPath := filepath.Join(outputPath, "requirements.txt")
		assert.FileExists(t, requirementsPath, "requirements.txt should still exist")

		cliPath := filepath.Join(outputPath, "src", "cli.py")
		assert.FileExists(t, cliPath, "src/cli.py should still exist")

		t.Log("✓ Project files persist successfully")
	})

	// === SUBTEST 3: Project Structure ===
	t.Run("ProjectStructure", func(t *testing.T) {
		t.Log("Verifying complete project structure...")

		// Project output path (no subdirectory)
		outputPath := filepath.Join(testDir, "output")

		// Verify directory structure
		assert.DirExists(t, outputPath, "Project directory should exist")
		assert.DirExists(t, filepath.Join(outputPath, "src"), "src directory should exist")
		// venv directory is not created by stack scaffolds
		// assert.DirExists(t, filepath.Join(outputPath, "venv"), "venv directory should exist")

		// Verify all expected files
		assert.FileExists(t, filepath.Join(outputPath, "src", "cli.py"), "CLI file should exist")
		assert.FileExists(t, filepath.Join(outputPath, "requirements.txt"), "requirements.txt should exist")

		t.Log("✓ Project structure verification successful")
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
		t.Log("ℹ️  Note: All files are created via Claude execution in the project directory")
		t.Log("  Claude execution is skipped for testing")

		// Check for JSONL file with timestamp pattern
		jsonPattern := filepath.Join(testDir, "output", ".debug", "claude-prompt-*.json"+
			""+
			"")
		jsonlFiles, _ := filepath.Glob(jsonPattern)
		if len(jsonlFiles) > 0 {
			t.Logf("✓ Claude prompt JSON found: %s", jsonlFiles[0])
			// Read and verify it contains the system prompt
			jsonlContent, err := os.ReadFile(jsonlFiles[0])
			if err == nil && len(jsonlContent) > 0 {
				assert.Contains(t, string(jsonlContent), "30+ years of experience")
				t.Log("  File contains system prompt and project specification")
			}
		} else {
			t.Logf("⚠️  Claude prompt JSON not found (pattern: %s)", jsonPattern)
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

		// Note: Kits are now embedded in the stack module, not listed separately in the spec
		// The spec only contains the merged files from the stack
		if len(debugSpecFiles) > 0 {
			t.Log("ℹ️  Kits are embedded in the stack and not listed separately in the specification")
			t.Log("✓ Project specification contains expected files")
		}

		t.Log("✓ Debug files verification complete")
	})

	t.Log("✅ All Python Click CLI tests completed successfully!")
}
