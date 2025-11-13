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

func TestE2EProjectExampleFlaskAPIWeatherAppSuccess(t *testing.T) {
	t.Log("Testing flask-api-nyc-weather example...")

	// Create test directory
	testDir := createTestDirectory(t, "TestE2EProjectExampleFlaskAPIWeatherAppSuccess")

	// Get project root
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	// Copy the example project files
	exampleSrcPath := filepath.Join(projectRoot, "examples", "projects", "flask-api-nyc-weather")

	// Copy integration.tofu
	integrationContent, err := os.ReadFile(filepath.Join(exampleSrcPath, "integration.tofu"))
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(testDir, "integration.tofu"), integrationContent, 0644)
	require.NoError(t, err)

	// Copy features.tofu
	featuresContent, err := os.ReadFile(filepath.Join(exampleSrcPath, "features.tofu"))
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(testDir, "features.tofu"), featuresContent, 0644)
	require.NoError(t, err)

	// Copy and modify project.tofu to use test output directory
	projectContent, err := os.ReadFile(filepath.Join(exampleSrcPath, "project.tofu"))
	require.NoError(t, err)

	// Add output_path to the provider block
	modifiedContent := string(projectContent)
	// Insert output_path after debug = true in provider block
	modifiedContent = strings.Replace(modifiedContent,
		`debug                 = true`,
		`debug                 = true
  output_path           = "output"`, 1)

	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(modifiedContent), 0644)
	require.NoError(t, err)

	// Determine which Terraform CLI to use
	var tfCmd string
	if _, err := exec.LookPath("tofu"); err == nil {
		tfCmd = "tofu"
		t.Log("Using OpenTofu")
	} else if _, err := exec.LookPath("terraform"); err == nil {
		tfCmd = "terraform"
		t.Log("Using Terraform")
	} else {
		t.Skip("Neither terraform nor tofu available - skipping test")
	}

	// Run tofu init
	t.Log("Running tofu init...")
	initCmd := exec.Command(tfCmd, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Run tofu plan and capture output
	t.Log("Running tofu plan...")
	planCmd := exec.Command(tfCmd, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	if err != nil {
		t.Logf("Plan output: %s", planOutput)
	}
	require.NoError(t, err, "Failed to run plan")
	t.Log("✓ Plan completed successfully")
	t.Logf("Plan output:\n%s", planOutput)

	// Run tofu apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(tfCmd, "apply", "--auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()

	// Save apply output to log file for analysis
	logFile := filepath.Join(testDir, "apply-output.log")
	err2 := os.WriteFile(logFile, applyOutput, 0644)
	require.NoError(t, err2)
	t.Logf("Apply output saved to: %s", logFile)

	if err != nil {
		t.Logf("Apply output: %s", applyOutput)
	}
	require.NoError(t, err, "Failed to run apply")
	t.Log("✓ Apply completed successfully")

	// Determine output directory
	outputDir := filepath.Join(testDir, "output")

	// Verify generated files exist
	expectedFiles := []string{
		"app.py",
		"requirements.txt",
		"README.md",
		".env.example",
		".gitignore",
	}

	for _, file := range expectedFiles {
		filePath := filepath.Join(outputDir, file)
		assert.FileExists(t, filePath, "File %s should exist", file)
	}

	// Verify app.py contains Flask and Open-Meteo API URL
	appPyPath := filepath.Join(outputDir, "app.py")
	appPyContent, err := os.ReadFile(appPyPath)
	require.NoError(t, err)
	appPyStr := string(appPyContent)

	assert.Contains(t, appPyStr, "from flask import Flask", "app.py should import Flask")
	assert.Contains(t, appPyStr, "api.open-meteo.com", "app.py should contain Open-Meteo API URL")
	assert.Contains(t, appPyStr, "/weather", "app.py should have /weather endpoint")

	// Verify requirements.txt contains expected dependencies
	requirementsPath := filepath.Join(outputDir, "requirements.txt")
	requirementsContent, err := os.ReadFile(requirementsPath)
	require.NoError(t, err)
	requirementsStr := string(requirementsContent)

	assert.Contains(t, requirementsStr, "Flask", "requirements.txt should contain Flask")
	assert.Contains(t, requirementsStr, "requests", "requirements.txt should contain requests")

	// Verify README.md contains integration information
	readmePath := filepath.Join(outputDir, "README.md")
	readmeContent, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	readmeStr := string(readmeContent)

	assert.Contains(t, readmeStr, "Open-Meteo", "README should mention Open-Meteo integration")
	assert.Contains(t, readmeStr, "NYC", "README should mention NYC weather")

	t.Log("✅ Example flask-api-nyc-weather validated successfully!")
}
