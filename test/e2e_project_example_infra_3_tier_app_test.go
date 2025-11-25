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

// TestE2EProjectExampleInfra3TierAppSuccess validates that the /examples/projects/infra-3-tier-app example works end-to-end
func TestE2EProjectExampleInfra3TierAppSuccess(t *testing.T) {
	// Set debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestE2EProjectExampleInfra3TierAppSuccess")

	// Get project root
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	// Step 1: Copy the example configuration files
	exampleDir := filepath.Join(projectRoot, "examples", "projects", "infra-3-tier-app")

	// Copy the tools directory (required for module references)
	toolsSourceDir := filepath.Join(projectRoot, "examples", "tools")
	toolsDestDir := filepath.Join(filepath.Dir(testDir), "tools")

	// Check if tools directory exists and needs to be copied
	if _, err := os.Stat(toolsSourceDir); err == nil {
		// Remove existing tools directory in test-output if it exists
		os.RemoveAll(toolsDestDir)

		// Copy tools directory
		if err := copyDir(toolsSourceDir, toolsDestDir); err != nil {
			t.Fatalf("Failed to copy tools directory: %v", err)
		}
		t.Logf("✓ Copied tools directory to: %s", toolsDestDir)
	}

	// Copy the features directory (required for diagram module reference)
	featuresSourceDir := filepath.Join(projectRoot, "examples", "features")
	featuresDestDir := filepath.Join(filepath.Dir(testDir), "features")

	// Check if features directory exists and needs to be copied
	if _, err := os.Stat(featuresSourceDir); err == nil {
		// Remove existing features directory in test-output if it exists
		os.RemoveAll(featuresDestDir)

		// Copy features directory
		if err := copyDir(featuresSourceDir, featuresDestDir); err != nil {
			t.Fatalf("Failed to copy features directory: %v", err)
		}
		t.Logf("✓ Copied features directory to: %s", featuresDestDir)
	}

	// Copy the project.tofu
	projectContent, err := os.ReadFile(filepath.Join(exampleDir, "project.tofu"))
	if err != nil {
		t.Fatalf("Failed to read project.tofu: %v", err)
	}

	// Modify the provider config to use test output directory
	modifiedContent := string(projectContent)
	// Add output_path = "output" after the output_format line
	modifiedContent = strings.Replace(modifiedContent,
		`output_format         = "json"`,
		`output_format         = "json"
  output_path           = "output"`, 1)

	// Update module source path to point to the copied tools directory
	// Original: source = "../../tools/tofukit-tool-opentofu"
	// New: source = "../tools/tofukit-tool-opentofu" (relative to test-output/<test-name>/)
	modifiedContent = strings.Replace(modifiedContent,
		`source = "../../tools/tofukit-tool-opentofu"`,
		`source = "../tools/tofukit-tool-opentofu"`, 1)

	// Update module source path to point to the copied features directory
	// Original: source = "../../features/tofukit-feature-diagram-drawio"
	// New: source = "../features/tofukit-feature-diagram-drawio" (relative to test-output/<test-name>/)
	modifiedContent = strings.Replace(modifiedContent,
		`source = "../../features/tofukit-feature-diagram-drawio"`,
		`source = "../features/tofukit-feature-diagram-drawio"`, 1)

	projectPath := filepath.Join(testDir, "project.tofu")
	if err := os.WriteFile(projectPath, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to write project.tofu: %v", err)
	}

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
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Logf("Plan output:\n%s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 5: Run apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// === SUBTEST 1: Diagram File Creation ===
	t.Run("DiagramFileCreation", func(t *testing.T) {
		t.Log("Testing draw.io diagram generation...")

		// The project should create architecture.drawio in the output directory
		outputPath := filepath.Join(testDir, "output")
		diagramPath := filepath.Join(outputPath, "architecture.drawio")

		// Verify diagram file was created
		assert.FileExists(t, diagramPath, "architecture.drawio should exist")

		// Read file content
		content, err := os.ReadFile(diagramPath)
		require.NoError(t, err, "Should be able to read architecture.drawio")

		// Verify it's valid XML (contains draw.io markers)
		contentStr := string(content)
		assert.Contains(t, contentStr, "<mxGraphModel", "Diagram should contain mxGraphModel element (draw.io XML)")
		assert.Contains(t, contentStr, "<mxCell", "Diagram should contain mxCell elements")

		// Verify it contains feature names from the project (discovered via introspection)
		// The diagram should discover and visualize: load_balancer, web_server, database features
		assert.Contains(t, contentStr, "load_balancer", "Diagram should mention load_balancer feature")
		assert.Contains(t, contentStr, "web_server", "Diagram should mention web_server feature")
		assert.Contains(t, contentStr, "database", "Diagram should mention database feature")

		t.Log("✓ Draw.io diagram file created successfully with expected components")
	})

	// === SUBTEST 2: PNG Export Verification ===
	t.Run("PNGExportVerification", func(t *testing.T) {
		t.Log("Testing PNG export from draw.io diagram...")

		outputPath := filepath.Join(testDir, "output")
		pngPath := filepath.Join(outputPath, "architecture.drawio.png")

		// PNG should have been created by the verification step during apply
		// If drawio CLI is not available, the apply should have failed (not skipped)
		// because we have a tofukit_tool definition with installation instructions
		assert.FileExists(t, pngPath, "architecture.drawio.png should exist (created by verification). "+
			"If missing, ensure drawio CLI is installed: "+
			"brew install --cask drawio (macOS) or see https://github.com/jgraph/drawio-desktop/releases")

		// Check file size is reasonable (not empty)
		info, err := os.Stat(pngPath)
		require.NoError(t, err, "Failed to stat PNG file")
		assert.Greater(t, info.Size(), int64(1000), "PNG file should be larger than 1KB")

		t.Log("✓ PNG export successful")
	})

	// === SUBTEST 3: Project Structure ===
	t.Run("ProjectStructure", func(t *testing.T) {
		t.Log("Verifying project structure...")

		outputPath := filepath.Join(testDir, "output")

		// Verify directory structure
		assert.DirExists(t, outputPath, "Output directory should exist")

		// Verify diagram file
		diagramPath := filepath.Join(outputPath, "architecture.drawio")
		assert.FileExists(t, diagramPath, "Diagram file should exist")

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
			assert.Contains(t, string(spec), "infra-3-tier-app", "Debug spec should contain project name")
			t.Log("  This file contains the complete project specification")
		} else {
			t.Logf("⚠️ Debug specification file not found (pattern: %s)", debugSpecPattern)
		}

		// Check for Claude prompt JSON file
		jsonPattern := filepath.Join(testDir, "output", ".debug", "claude-prompt-*.json")
		jsonlFiles, _ := filepath.Glob(jsonPattern)
		if len(jsonlFiles) > 0 {
			t.Logf("✓ Claude prompt JSON found: %s", jsonlFiles[0])
			// Read and verify it contains diagram-related content
			jsonlContent, err := os.ReadFile(jsonlFiles[0])
			if err == nil && len(jsonlContent) > 0 {
				assert.Contains(t, string(jsonlContent), "architecture.drawio", "Prompt should reference diagram file")
				assert.Contains(t, string(jsonlContent), "project_context", "Prompt should contain project context for introspection")
				t.Log("  File contains system prompt and diagram specifications")
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
				assert.Contains(t, string(mdContent), "architecture", "Markdown should mention architecture")
				t.Log("  File contains formatted prompt for Claude")
			}
		} else {
			t.Logf("⚠️ Claude prompt markdown not found (pattern: %s)", mdPattern)
		}

		t.Log("✓ Debug files verification complete")
	})

	// === SUBTEST 5: Diagram Content Validation ===
	t.Run("DiagramContentValidation", func(t *testing.T) {
		t.Log("Validating diagram XML content structure...")

		outputPath := filepath.Join(testDir, "output")
		diagramPath := filepath.Join(outputPath, "architecture.drawio")

		content, err := os.ReadFile(diagramPath)
		require.NoError(t, err)

		contentStr := string(content)

		// Verify essential draw.io XML structure
		assert.Contains(t, contentStr, "<?xml", "Should be valid XML with declaration")
		assert.Contains(t, contentStr, "<mxfile", "Should have mxfile root element")
		assert.Contains(t, contentStr, "<diagram", "Should have diagram element")
		assert.Contains(t, contentStr, "<mxGraphModel", "Should have mxGraphModel element")
		assert.Contains(t, contentStr, "<root>", "Should have root element")

		// Verify all three features are represented (discovered via introspection)
		features := []string{"load_balancer", "web_server", "database"}
		for _, feature := range features {
			assert.Contains(t, contentStr, feature, "Diagram should contain %s", feature)
		}

		// Verify diagram has multiple cells (shapes)
		cellCount := strings.Count(contentStr, "<mxCell")
		assert.GreaterOrEqual(t, cellCount, 3, "Should have at least 3 mxCell elements for components")

		t.Log("✓ Diagram XML content validation successful")
		t.Logf("  Found %d mxCell elements in diagram", cellCount)
	})

	t.Log("✅ All infra-3-tier-app diagram generation tests completed successfully!")
}
