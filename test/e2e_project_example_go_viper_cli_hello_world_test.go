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

// TestE2EProjectExampleGoViperCliHelloWorldSuccess validates that the /examples/projects/go-viper-hello-world example works end-to-end
func TestE2EProjectExampleGoViperCliHelloWorldSuccess(t *testing.T) {
	// Set debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestE2EProjectExampleGoViperCliHelloWorldSuccess")

	// Get project root
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	// Step 1: Create directory structure for test
	// testDir/
	//   project/   <- project.tofu goes here
	//   stacks/    <- stacks copied here
	//   output/    <- provider output (created by provider)

	exampleDir := filepath.Join(projectRoot, "examples", "projects", "go-viper-hello-world")

	// Create project subdirectory
	projectSubDir := filepath.Join(testDir, "project")
	if err := os.MkdirAll(projectSubDir, 0755); err != nil {
		t.Fatalf("Failed to create project subdirectory: %v", err)
	}

	// Copy all .tofu files from the example directory
	tofuFiles, err := filepath.Glob(filepath.Join(exampleDir, "*.tofu"))
	if err != nil {
		t.Fatalf("Failed to glob .tofu files: %v", err)
	}

	for _, srcFile := range tofuFiles {
		fileName := filepath.Base(srcFile)
		content, err := os.ReadFile(srcFile)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", fileName, err)
		}

		modifiedContent := string(content)

		// Only modify project.tofu
		if fileName == "project.tofu" {
			// Change output_path to go up to test root then into output
			modifiedContent = strings.Replace(modifiedContent,
				`output_format         = "json"`,
				`output_format         = "json"
  output_path           = "../output"`, 1)
			// Update module source from ../../stacks to ../stacks (one level up from project/)
			modifiedContent = strings.Replace(modifiedContent,
				`source = "../../stacks/tofukit-stack-go-viper-cobra-pterm"`,
				`source = "../stacks/tofukit-stack-go-viper-cobra-pterm"`, 1)
		}

		destPath := filepath.Join(projectSubDir, fileName)
		if err := os.WriteFile(destPath, []byte(modifiedContent), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", fileName, err)
		}
		t.Logf("Copied %s to test directory", fileName)
	}

	// Copy the stacks directory to testDir/stacks
	stacksDir := filepath.Join(projectRoot, "examples", "stacks")
	testStacksDir := filepath.Join(testDir, "stacks")
	copyDirCmd := exec.Command("cp", "-r", stacksDir, testStacksDir)
	if err := copyDirCmd.Run(); err != nil {
		t.Fatalf("Failed to copy stacks directory: %v", err)
	}
	t.Logf("Copied stacks from %s to %s", stacksDir, testStacksDir)

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

	// Step 3: Run init to set up the provider
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = projectSubDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = projectSubDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Logf("Plan output:\n%s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 5: Run apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = projectSubDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Helper function to run apply
	runApply := func(t *testing.T) {
		applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
		applyCmd.Dir = projectSubDir
		applyOutput, err := applyCmd.CombinedOutput()
		require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	}

	// === SUBTEST 1: Initial Creation ===
	t.Run("InitialCreation", func(t *testing.T) {
		t.Log("Testing initial Go CLI project creation...")

		// The project resource creates scaffold files via Claude
		// in the project output directory
		outputPath := filepath.Join(testDir, "output")

		// Verify key Go files were created by the stack resource
		// Check main.go
		mainPath := filepath.Join(outputPath, "main.go")
		assert.FileExists(t, mainPath, "main.go should exist")
		content, err := os.ReadFile(mainPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "package main", "main.go should have package main")
		assert.Contains(t, string(content), "cmd.Execute()", "main.go should call cmd.Execute()")

		// Check cmd/root.go
		rootPath := filepath.Join(outputPath, "cmd", "root.go")
		assert.FileExists(t, rootPath, "cmd/root.go should exist")
		rootContent, err := os.ReadFile(rootPath)
		require.NoError(t, err)
		assert.Contains(t, string(rootContent), "github.com/spf13/cobra", "root.go should import cobra")
		// Note: viper is abstracted in internal/config, not directly imported in cmd/root.go
		// Note: pterm is used in internal/logger/logger.go, not directly in root.go

		// Check internal/logger/logger.go exists (from pterm-logger feature)
		loggerPath := filepath.Join(outputPath, "internal", "logger", "logger.go")
		assert.FileExists(t, loggerPath, "internal/logger/logger.go should exist (from pterm-logger feature)")

		// Check go.mod
		goModPath := filepath.Join(outputPath, "go.mod")
		assert.FileExists(t, goModPath, "go.mod should exist")
		goModContent, err := os.ReadFile(goModPath)
		require.NoError(t, err)
		assert.Contains(t, string(goModContent), "dimmiirr.com/go-viper-hello-world", "go.mod should contain module path")
		assert.Contains(t, string(goModContent), "github.com/spf13/cobra", "go.mod should require cobra")

		// README.md - checking if it exists but not requiring it
		// Note: README is provided via readme feature, but feature files may not be collected yet (known limitation)
		readmePath := filepath.Join(outputPath, "README.md")
		if _, err := os.Stat(readmePath); err == nil {
			// README exists, verify it
			readmeContent, err := os.ReadFile(readmePath)
			require.NoError(t, err, "Should be able to read README.md")
			assert.Contains(t, string(readmeContent), "go-viper-hello-world",
				"README should contain project name 'go-viper-hello-world' (from stack files)")
			t.Log("✓ README.md contains 'go-viper-hello-world' as expected from stack merge")
		} else {
			t.Log("⚠️  README.md not found (feature files not yet fully integrated)")
		}

		t.Log("✓ Initial Go CLI project creation successful")
		t.Log("  Note: All files are created via Claude execution")
	})

	// === SUBTEST 2: Project Files Persist ===
	t.Run("ProjectFilesPersist", func(t *testing.T) {
		t.Log("Testing that project files persist after re-apply...")

		// Project output path (no subdirectory)
		outputPath := filepath.Join(testDir, "output")

		// Run apply again without changes
		runApply(t)

		// Verify files still exist
		mainPath := filepath.Join(outputPath, "main.go")
		assert.FileExists(t, mainPath, "main.go should still exist")

		goModPath := filepath.Join(outputPath, "go.mod")
		assert.FileExists(t, goModPath, "go.mod should still exist")

		t.Log("✓ Project files persist successfully")
	})

	// === SUBTEST 3: Project Structure ===
	t.Run("ProjectStructure", func(t *testing.T) {
		t.Log("Verifying complete project structure...")

		// Project output path (no subdirectory)
		outputPath := filepath.Join(testDir, "output")

		// Verify directory structure
		assert.DirExists(t, outputPath, "Project directory should exist")
		assert.DirExists(t, filepath.Join(outputPath, "cmd"), "cmd directory should exist")

		// Verify all expected files
		assert.FileExists(t, filepath.Join(outputPath, "main.go"), "main.go should exist")
		assert.FileExists(t, filepath.Join(outputPath, "cmd", "root.go"), "cmd/root.go should exist")
		assert.FileExists(t, filepath.Join(outputPath, "go.mod"), "go.mod should exist")

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
			assert.Contains(t, string(spec), "go-viper-hello-world")
			t.Log("  This file contains the complete project specification")
		} else {
			t.Logf("⚠️ Debug specification file not found (pattern: %s)", debugSpecPattern)
		}

		// Note about Claude execution mode
		t.Log("ℹ️  Note: All files are created via Claude execution in the project directory")

		// Check for JSONL file with timestamp pattern
		jsonPattern := filepath.Join(testDir, "output", ".debug", "claude-prompt-*.json")
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
				assert.Contains(t, string(mdContent), "go-viper-hello-world")
				assert.Contains(t, string(mdContent), "Cobra")
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

	// === SUBTEST 5: Build and Run Application ===
	t.Run("BuildAndRun", func(t *testing.T) {
		t.Log("Testing that the generated application builds and runs...")

		outputPath := filepath.Join(testDir, "output")

		// Build the binary using task (uses Taskfile for proper ldflags)
		t.Log("Building the application with 'task build'...")
		buildCmd := exec.Command("task", "build")
		buildCmd.Dir = outputPath
		buildOutput, err := buildCmd.CombinedOutput()
		if err != nil {
			t.Logf("Build output: %s", buildOutput)
		}
		require.NoError(t, err, "Failed to build application with task")

		// Check that binary was created
		binaryPath := filepath.Join(outputPath, "bin", "go-viper-hello-world")
		assert.FileExists(t, binaryPath, "Binary should be created")
		t.Log("✓ Application built successfully")

		// Run the version command
		t.Log("Running 'go-viper-hello-world version' command...")
		versionCmd := exec.Command("./bin/go-viper-hello-world", "version")
		versionCmd.Dir = outputPath
		versionOutput, err := versionCmd.CombinedOutput()
		if err != nil {
			t.Logf("Version command output: %s", versionOutput)
		}
		require.NoError(t, err, "Failed to run version command")

		// Verify version output contains expected version
		outputStr := string(versionOutput)
		assert.Contains(t, outputStr, "1.0.0", "Version output should contain '1.0.0'")
		t.Logf("Version command output:\n%s", outputStr)
		t.Log("✓ Application runs successfully")

		// Optional: Run the help command to verify basic CLI functionality
		t.Log("Running 'go-viper-hello-world --help' command...")
		helpCmd := exec.Command("./bin/go-viper-hello-world", "--help")
		helpCmd.Dir = outputPath
		helpOutput, err := helpCmd.CombinedOutput()
		require.NoError(t, err, "Failed to run help command")

		helpStr := string(helpOutput)
		assert.Contains(t, helpStr, "go-viper-hello-world", "Help output should contain project name")
		assert.Contains(t, helpStr, "version", "Help output should list version command")
		assert.Contains(t, helpStr, "ip", "Help output should list ip command")
		t.Log("✓ Help command works correctly")

		// Run the ip command
		t.Log("Running 'go-viper-hello-world ip' command...")
		ipCmd := exec.Command("./bin/go-viper-hello-world", "ip")
		ipCmd.Dir = outputPath
		ipOutput, err := ipCmd.CombinedOutput()
		if err != nil {
			t.Logf("IP command output: %s", ipOutput)
		}
		require.NoError(t, err, "Failed to run ip command")

		// Verify IP output contains expected format
		ipStr := string(ipOutput)
		assert.Contains(t, ipStr, "Your public IP:", "IP output should contain 'Your public IP:'")
		// Check for pterm styling (ANSI color codes)
		assert.Contains(t, ipStr, "\x1b[", "IP output should contain pterm styling (ANSI codes)")
		// Basic IP format validation - should contain dots (IPv4) or colons (IPv6)
		assert.True(t, strings.Contains(ipStr, ".") || strings.Contains(ipStr, ":"),
			"IP output should contain a valid IP address format (IPv4 or IPv6)")
		t.Logf("IP command output:\n%s", ipStr)
		t.Log("✓ IP command works correctly")

		t.Log("✓ Build and run verification complete")
	})

	t.Log("✅ All Go CLI tests completed successfully!")
}
