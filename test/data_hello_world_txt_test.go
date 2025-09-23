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

func TestDataQuerySimple(t *testing.T) {
	// Set TF_LOG=DEBUG for debug logging
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestDataQuerySimple")

	// CLAUDE_HOME is not needed when using --dangerously-skip-permissions
	// The flag bypasses all authentication and permission checks
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

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

	// Step 2: No need to setup test files for simple query
	t.Log("Skipping test file setup - using simple query")

	// Step 3: Generate main.tofu with provider configuration and data query
	mainTofuContent := `# Terraform configuration for data query test
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
  instructions = [
    "What color is grass?",
    "Answer with exactly one word, in lowercase",
    "Do not include any punctuation or additional text",
    "Just return the single word: green"
  ]

  format = "text"
}

# Output the query results
output "query_output_json" {
  value = data.tofukit_query.grass_color.output_json
  description = "JSON output from the query"
}

output "query_output_data" {
  value = data.tofukit_query.grass_color.output_data
  description = "Raw output from the query"
}

output "query_id" {
  value = data.tofukit_query.grass_color.id
  description = "Query ID"
}
`
	err = os.WriteFile(filepath.Join(testDir, "main.tofu"), []byte(mainTofuContent), 0644)
	require.NoError(t, err, "Failed to write main.tofu")

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
		if os.Getenv("SKIP_DESTROY") == "true" {
			t.Log("Skipping cleanup due to SKIP_DESTROY=true")
			t.Logf("Test directory preserved at: %s", testDir)
		} else {
			t.Log("Debug mode enabled - preserving output directory for inspection")
			t.Logf("Debug files should be in: %s/output/.debug/", testDir)
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
	if err != nil {
		t.Logf("Apply output: %s", applyOutput)
	}
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

	// Verify query_id exists
	queryID, ok := outputs["query_id"].(map[string]interface{})
	require.True(t, ok, "query_id output not found")
	assert.NotEmpty(t, queryID["value"], "Query ID should not be empty")
	t.Logf("Query ID: %v", queryID["value"])

	// Verify query_output_data exists and contains expected content
	queryData, ok := outputs["query_output_data"].(map[string]interface{})
	require.True(t, ok, "query_output_data not found")
	rawOutput := queryData["value"].(string)
	t.Logf("Raw query output: %s", rawOutput)

	// Verify the output contains "green"
	queryDataValue := queryData["value"].(string)
	// Clean up the output - remove any whitespace, newlines, etc.
	cleanedOutput := strings.TrimSpace(strings.ToLower(queryDataValue))
	t.Logf("Cleaned output: '%s'", cleanedOutput)

	// The output should contain the word "green"
	assert.Contains(t, cleanedOutput, "green", "Output should contain the word 'green'")

	t.Log("✓ Data query test completed successfully")

}
