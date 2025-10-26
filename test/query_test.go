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

// TestQuerySimpleSuccess tests the tofukit_query data source with simple questions
func TestQuerySimpleSuccess(t *testing.T) {
	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestQuerySimpleSuccess")

	// CLAUDE_HOME is not needed when using --dangerously-skip-permissions
	// The flag bypasses all authentication and permission checks
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	// Assume provider is already built and installed via `make install`
	// Run `make install` manually before running tests if needed

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `# Terraform configuration for data query test
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
  # Claude execution is always performed
  debug                 = true       # Enable debug mode for detailed output
  # claude_home_directory not needed with --dangerously-skip-permissions
}

# Data source query - simple color question
data "tofukit_query" "grass_color" {
  instruction {
    prompt = "What color is grass?"
    constraints = [
      "Answer with exactly one word",
      "Use lowercase only",
      "Do NOT include any punctuation or additional text"
    ]
  }
}

# Data source query - simple material question
data "tofukit_query" "rock_hardness" {
  instruction {
    prompt = "Is rock a hard or a soft material?"
    constraints = [
      "Answer with exactly one word",
      "Use lowercase only",
      "Do NOT include any punctuation or additional text"
    ]
  }
}


# Output the query results
output "grass_color_id" {
  value = data.tofukit_query.grass_color.id
  description = "ID from grass color query"
}

output "grass_color_json" {
  value = data.tofukit_query.grass_color.json
  description = "JSON output from grass color query"
}

output "grass_color_text" {
  value = data.tofukit_query.grass_color.text
  description = "Text output from grass color query"
}

output "rock_hardness_json" {
  value = data.tofukit_query.rock_hardness.json
  description = "JSON output from rock hardness query"
}

output "rock_hardness_text" {
  value = data.tofukit_query.rock_hardness.text
  description = "Text output from rock hardness query"
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

	// Clean up resources at the end
	defer func() {
		if os.Getenv("SKIP_DESTROY") == "true" {
			t.Logf("Skipping cleanup due to SKIP_DESTROY=true")
			t.Logf("Test directory preserved at: %s", testDir)
		} else {
			t.Logf("Debug mode enabled - preserving output directory for inspection")
			t.Logf("Debug files should be in: %s/output/.debug/", testDir)
		}
	}()

	// Step 5: Run init to set up the provider
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	err = initCmd.Run()
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 6: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planCmd.Stdout = os.Stdout
	planCmd.Stderr = os.Stderr
	err = planCmd.Run()
	require.NoError(t, err, "Failed to run plan")
	t.Log("✓ Plan completed successfully")

	// Step 7: Run apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyCmd.Stdout = os.Stdout
	applyCmd.Stderr = os.Stderr
	err = applyCmd.Run()
	require.NoError(t, err, "Failed to run apply")
	t.Log("✓ Apply completed successfully")

	// Step 8: Get outputs and verify
	t.Log("Getting outputs...")
	outputCmd := exec.Command(iacTool, "output", "-json")
	outputCmd.Dir = testDir
	outputBytes, err := outputCmd.Output() // Use Output() instead of CombinedOutput() to avoid stderr mixing
	if err != nil {
		t.Logf("Output command failed: %v", err)
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Logf("Stderr: %s", exitErr.Stderr)
		}
		require.NoError(t, err, "Failed to get outputs")
	}

	// Parse outputs
	var outputs map[string]interface{}
	err = json.Unmarshal(outputBytes, &outputs)
	if err != nil {
		t.Logf("Raw output bytes: %q", string(outputBytes))
		t.Logf("Parse error: %v", err)
	}
	require.NoError(t, err, "Failed to parse outputs JSON")

	// Save outputs to outputs.json
	outputsFile := filepath.Join(testDir, "outputs.json")
	outputsJSON, err := json.MarshalIndent(outputs, "", "  ")
	require.NoError(t, err, "Failed to marshal outputs")
	err = os.WriteFile(outputsFile, outputsJSON, 0644)
	require.NoError(t, err, "Failed to write outputs.json")
	t.Logf("Outputs saved to: %s", outputsFile)

	// Save state to state.json
	stateFile := filepath.Join(testDir, "state.json")
	stateContent, err := os.ReadFile(filepath.Join(testDir, "terraform.tfstate"))
	require.NoError(t, err, "Failed to read terraform.tfstate")
	err = os.WriteFile(stateFile, stateContent, 0644)
	require.NoError(t, err, "Failed to write state.json")
	t.Logf("State saved to: %s", stateFile)

	// Verify grass_color outputs
	t.Log("Verifying grass_color query outputs...")

	grassColorID, ok := outputs["grass_color_id"].(map[string]interface{})
	require.True(t, ok, "grass_color_id output not found")
	assert.NotEmpty(t, grassColorID["value"], "Grass color query ID should not be empty")
	t.Logf("Grass color query ID: %v", grassColorID["value"])

	grassColorText, ok := outputs["grass_color_text"].(map[string]interface{})
	require.True(t, ok, "grass_color_text not found")
	grassColorTextValue := grassColorText["value"].(string)
	t.Logf("Grass color text output: '%s'", grassColorTextValue)

	// Verify the output contains "green" (should already be clean)
	cleanedGrassColor := strings.TrimSpace(strings.ToLower(grassColorTextValue))
	assert.Contains(t, cleanedGrassColor, "green", "Grass color output should contain 'green'")
	t.Log("✓ Grass color query verified: contains 'green'")

	// Verify rock_hardness outputs
	t.Log("Verifying rock_hardness query outputs...")

	rockHardnessJSON, ok := outputs["rock_hardness_json"].(map[string]interface{})
	require.True(t, ok, "rock_hardness_json output not found")
	assert.NotEmpty(t, rockHardnessJSON["value"], "Rock hardness JSON should not be empty")
	t.Logf("Rock hardness JSON output: %v", rockHardnessJSON["value"])

	rockHardnessText, ok := outputs["rock_hardness_text"].(map[string]interface{})
	require.True(t, ok, "rock_hardness_text not found")
	rockHardnessTextValue := rockHardnessText["value"].(string)
	t.Logf("Rock hardness text output: '%s'", rockHardnessTextValue)

	// Verify the output contains "hard" (rocks are hard materials)
	cleanedRockHardness := strings.TrimSpace(strings.ToLower(rockHardnessTextValue))
	assert.Contains(t, cleanedRockHardness, "hard", "Rock hardness output should contain 'hard'")
	t.Log("✓ Rock hardness query verified: contains 'hard'")

	t.Log("✓ All data query tests completed successfully")

}
