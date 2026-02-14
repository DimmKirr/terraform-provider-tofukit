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

// TestE2EProjectExamplePurlSuccess validates that the /examples/projects/purl example works end-to-end
func TestE2EProjectExamplePurlSuccess(t *testing.T) {
	// Set debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestE2EProjectExamplePurlSuccess")

	// Get project root
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	// Step 1: Create directory structure for test
	// testDir/
	//   project/   <- project.tofu goes here
	//   stacks/    <- stacks copied here
	//   output/    <- provider output (created by provider)

	exampleDir := filepath.Join(projectRoot, "examples", "projects", "purl")

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
  output_path           = "../output"
  claude_max_turns      = 500`, 1)
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

	// === MAIN TEST: Verify purl binary works ===
	t.Run("VerifyPurlBinary", func(t *testing.T) {
		t.Log("Testing purl binary functionality...")

		outputPath := filepath.Join(testDir, "output")

		// Verify main.go was created
		mainPath := filepath.Join(outputPath, "main.go")
		assert.FileExists(t, mainPath, "main.go should exist")

		// Verify cmd/root.go was created
		rootPath := filepath.Join(outputPath, "cmd", "root.go")
		assert.FileExists(t, rootPath, "cmd/root.go should exist")

		// Build the purl binary
		t.Log("Building purl binary...")
		binDir := filepath.Join(outputPath, "bin")
		err := os.MkdirAll(binDir, 0755)
		require.NoError(t, err, "Failed to create bin directory")

		purlBinaryPath := filepath.Join(binDir, "purl")
		buildCmd := exec.Command("go", "build", "-o", purlBinaryPath)
		buildCmd.Dir = outputPath
		buildOutput, err := buildCmd.CombinedOutput()
		if err != nil {
			t.Logf("Build output: %s", buildOutput)
		}
		require.NoError(t, err, "Failed to build purl binary")
		t.Log("✓ purl binary built successfully")

		// Verify binary exists
		assert.FileExists(t, purlBinaryPath, "purl binary should exist at ./bin/purl")
	})

	// === ADDITIONAL TEST: Verify purl ping command ===
	t.Run("VerifyPurlPing", func(t *testing.T) {
		t.Log("Testing purl ping functionality...")

		outputPath := filepath.Join(testDir, "output")
		purlBinaryPath := filepath.Join(outputPath, "bin", "purl")

		// Test: Run `./bin/purl http://ifcfg.me/` with timeout (direct URL, no subcommand)
		t.Log("Running ./bin/purl http://ifcfg.me/...")
		pingCmd := exec.Command("timeout", "3s", purlBinaryPath, "http://ifcfg.me/")
		pingCmd.Dir = outputPath
		pingOutput, err := pingCmd.CombinedOutput()

		pingOutputStr := string(pingOutput)
		t.Logf("purl ping output:\n%s", pingOutputStr)

		// Assert: output contains "200" status code
		if assert.Contains(t, pingOutputStr, "200", "Ping output should contain HTTP 200 status") {
			t.Log("✓ purl ping command works correctly")
		} else {
			// If ping fails, log detailed information
			t.Logf("⚠️  Ping command did not return 200 status")
			if err != nil {
				t.Logf("Command error: %v", err)
			}
			t.Logf("This may indicate the HTTP ping feature needs verification")
			t.FailNow() // Pause here as requested
		}
	})

	t.Log("✓✓✓ Purl E2E test completed successfully!")
}
