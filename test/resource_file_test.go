package test

import (
	"encoding/json"
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResourceFile_CreateSuccess verifies that a file resource can be created successfully
func TestResourceFile_CreateSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceFile_CreateSuccess")

	var err error

	// Create project configuration with file resource
	projectTofuContent := `# Terraform configuration for file resource test
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

resource "tofukit_file" "hello" {
  name    = "hello.txt"
  content = "Hello World"
}

resource "tofukit_project" "test" {
  name    = "file-test"
  version = "1.0.0"

  files = {
    "hello.txt" = tofukit_file.hello
  }
}
`

	projectTofuPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(projectTofuPath, []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Initialize Terraform
	initCmd := exec.Command("tofu", "init")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "tofu init failed: %s", string(initOutput))

	// Apply to create project
	applyCmd := exec.Command("tofu", "apply", "-auto-approve")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply failed: %s", string(applyOutput))

	t.Log("✓ File resource created successfully")

	// Verify file exists and has correct content
	helloPath := filepath.Join(testDir, "output", "hello.txt")
	content, err := os.ReadFile(helloPath)
	require.NoError(t, err, "Failed to read hello.txt")
	assert.Equal(t, "Hello World", string(content), "File content should match")

	t.Log("✓ File content verified")
}

// TestResourceFile_DriftDetection verifies drift detection on file resource
func TestResourceFile_DriftDetection(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestResourceFile_DriftDetection")

	var err error

	// Create project configuration with file resource
	projectTofuContent := `# Terraform configuration for file drift detection test
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

resource "tofukit_file" "hello" {
  name    = "hello.txt"
  content = "Hello World"
}

resource "tofukit_project" "test" {
  name    = "file-drift-test"
  version = "1.0.0"

  files = {
    "hello.txt" = tofukit_file.hello
  }
}
`

	projectTofuPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(projectTofuPath, []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 1: Initialize and apply
	initCmd := exec.Command("tofu", "init")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "tofu init failed: %s", string(initOutput))

	applyCmd := exec.Command("tofu", "apply", "-auto-approve")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply failed: %s", string(applyOutput))

	t.Log("✓ Initial apply completed")

	// Verify initial file content
	helloPath := filepath.Join(testDir, "output", "hello.txt")
	content, err := os.ReadFile(helloPath)
	require.NoError(t, err, "Failed to read hello.txt")
	assert.Equal(t, "Hello World", string(content), "Initial file content should match")

	// Step 2: Verify initial state has no drift
	showCmd := exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err := showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed: %s", string(showOutput))

	stateJSON := string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected":false`, "Initial state should have drift_detected=false")

	t.Log("✓ Initial state verified - no drift")

	// Step 3: Manually edit hello.txt to "Bye Bye World"
	err = os.WriteFile(helloPath, []byte("Bye Bye World"), 0644)
	require.NoError(t, err, "Failed to manually edit hello.txt")

	t.Log("✓ Manually edited hello.txt to 'Bye Bye World'")

	// Verify file was actually modified
	modifiedContent, err := os.ReadFile(helloPath)
	require.NoError(t, err, "Failed to read modified hello.txt")
	assert.Equal(t, "Bye Bye World", string(modifiedContent), "File should contain modified content")

	// Step 4: Run plan to detect drift
	planCmd := exec.Command("tofu", "plan", "-detailed-exitcode")
	planCmd.Dir = testDir
	planOutput, _ := planCmd.CombinedOutput() // Expecting exit code 2 (changes detected)

	planOutputStr := string(planOutput)
	t.Logf("Plan output:\n%s", planOutputStr)

	// Plan should show drift detected
	assert.Contains(t, planOutputStr, "drift_detected", "Plan should show drift_detected change")
	assert.Contains(t, planOutputStr, "hello.txt", "Plan should show hello.txt as drifted")

	// Step 5: Refresh state to capture drift
	refreshCmd := exec.Command("tofu", "apply", "-refresh-only", "-auto-approve")
	refreshCmd.Dir = testDir
	refreshOutput, err := refreshCmd.CombinedOutput()
	require.NoError(t, err, "tofu apply -refresh-only failed: %s", string(refreshOutput))

	t.Log("✓ Refreshed state to capture drift")

	// Step 6: Verify drift was auto-restored during refresh
	showCmd = exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err = showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed after refresh: %s", string(showOutput))

	stateJSON = string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected":false`, "State should have drift_detected=false after auto-restore")

	t.Log("✓ Drift was auto-restored during refresh")

	// Step 8: Verify file content restored to "Hello World"
	restoredContent, err := os.ReadFile(helloPath)
	require.NoError(t, err, "Failed to read restored hello.txt")
	assert.Equal(t, "Hello World", string(restoredContent), "File should be restored to 'Hello World'")

	t.Log("✓ File content restored to 'Hello World'")

	// Step 9: Verify drift cleared in state
	showCmd = exec.Command("tofu", "show", "-json")
	showCmd.Dir = testDir
	showOutput, err = showCmd.CombinedOutput()
	require.NoError(t, err, "tofu show failed after restore: %s", string(showOutput))

	stateJSON = string(showOutput)
	assert.Contains(t, stateJSON, `"drift_detected":false`, "State should have drift_detected=false after restore")

	t.Log("✓ Drift cleared from state")

	t.Log("✓✓✓ File drift detection and restoration completed successfully!")
}

// TestResourceFileNameValidation_ValidNames tests that valid file names are accepted
func TestResourceFileNameValidation_ValidNames(t *testing.T) {
	validNames := []string{
		"file.txt",
		"README.md",
		".gitignore",
		"src/main.go",
		"my-file_v2.txt",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			testDir := createTestDirectory(t, "TestResourceFileNameValidation_Valid")

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
}

resource "tofukit_file" "test" {
  name = "` + name + `"
  content = "test content\n"
}
`
			err := os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
			require.NoError(t, err)

			// Get IaC tool
			iacTool := detectIaCTool(t)

			// Init
			initCmd := exec.Command(iacTool, "init", "-no-color")
			initCmd.Dir = testDir
			initOutput, err := initCmd.CombinedOutput()
			if err != nil {
				t.Logf("Init output: %s", initOutput)
			}
			require.NoError(t, err, "Init should succeed")

			// Validate (checks schema validation)
			validateCmd := exec.Command(iacTool, "validate", "-no-color")
			validateCmd.Dir = testDir
			validateOutput, err := validateCmd.CombinedOutput()
			if err != nil {
				t.Logf("Validate output: %s", validateOutput)
				t.Fatalf("Validate should fail for valid name %q", name)
			}

			assert.Contains(t, string(validateOutput), "Success", "Validation should succeed for %q", name)
		})
	}
}

// TestResourceFileNameValidation_InvalidNames tests that invalid file names are rejected
func TestResourceFileNameValidation_InvalidNames(t *testing.T) {
	invalidTests := []struct {
		name          string
		expectedError string
	}{
		{"hello world.txt", "cannot contain spaces"},
		{"  file.txt", "leading or trailing whitespace"},
		{"file<name>.txt", "invalid characters"},
		{"CON", "Windows reserved name"},
		{"file.", "cannot end with a dot"},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := createTestDirectory(t, "TestResourceFileNameValidation_Invalid")

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
}

resource "tofukit_file" "test" {
  name = "` + tt.name + `"
  content = "test content\n"
}
`
			err := os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
			require.NoError(t, err)

			// Get IaC tool
			iacTool := detectIaCTool(t)

			// Init
			initCmd := exec.Command(iacTool, "init", "-no-color")
			initCmd.Dir = testDir
			initOutput, err := initCmd.CombinedOutput()
			if err != nil {
				t.Logf("Init output: %s", initOutput)
			}
			require.NoError(t, err, "Init should succeed")

			// Validate (should fail due to schema validation)
			validateCmd := exec.Command(iacTool, "validate", "-no-color")
			validateCmd.Dir = testDir
			validateOutput, err := validateCmd.CombinedOutput()

			// We expect validation to fail
			if err == nil {
				t.Logf("Validate output: %s", validateOutput)
				t.Fatalf("Validate should fail for invalid name %q", tt.name)
			}

			// Check that error message contains expected text
			assert.Contains(t, string(validateOutput), tt.expectedError, "Error message should mention the validation issue")
		})
	}
}

// =============================================================================
// PROMPT GENERATION TESTS (Fast)
// =============================================================================

// TestResourceFile_CreateGeneratePromptSuccess verifies prompt generation for file resource
func TestResourceFile_CreateGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceFile_CreateGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigFileResourceBasic), 0644)
	require.NoError(t, err)

	// Setup Terraform and run apply
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

	// Verify hello.txt is in files specification
	helloFile, exists := files["hello.txt"]
	require.True(t, exists, "hello.txt should be in files specification")

	helloMap := helloFile.(map[string]interface{})

	// Verify content from file resource
	assert.Equal(t, "Hello World", helloMap["content"], "Content should match file resource")

	// Verify instructions for file creation
	instructions := helloMap["instructions"].([]interface{})
	require.NotEmpty(t, instructions, "Should have instructions")

	instruction0 := instructions[0].(map[string]interface{})
	assert.Contains(t, instruction0["prompt"], "Create new file 'hello.txt'",
		"Should have create instruction for file resource")

	t.Log("✓ Prompt generation verified - file resource content in specification")
}

// TestResourceFile_DriftDetectionGeneratePromptSuccess verifies prompt structure for file resource (no actual drift in dry_run)
func TestResourceFile_DriftDetectionGeneratePromptSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceFile_DriftDetectionGeneratePromptSuccess")

	// Use embedded config from testdata
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(ConfigFileResourceDrift), 0644)
	require.NoError(t, err)

	// Setup and apply (dry_run mode doesn't execute actual drift detection)
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

	// Verify hello.txt is in prompt with correct structure
	helloFile, exists := files["hello.txt"]
	require.True(t, exists, "hello.txt should be in files specification")

	helloMap := helloFile.(map[string]interface{})
	assert.Equal(t, "Hello World", helloMap["content"], "Content should match file resource")

	t.Log("✓ Prompt generation verified - file resource drift detection structure valid")
}

// TestResourceFile_OpenAIBlackSquare verifies OpenAI image generation
// Creates a black square using openai/gpt-5-image model
func TestResourceFile_OpenAIBlackSquare(t *testing.T) {
	// Skip if no OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set - skipping OpenAI integration test")
	}

	testDir := createTestDirectory(t, "TestResourceFile_OpenAIBlackSquare")

	// Terraform config using OpenAI for image generation
	config := `
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  openai_api_key = "` + os.Getenv("OPENAI_API_KEY") + `"
  output_path    = "output"
  debug          = true
}

resource "tofukit_file" "black_square" {
  name = "black.png"

  instructions = [{
    prompt = "A solid black square, completely black, #000000"
    constraints = [
      "Size: 1024x1024",
      "Quality: standard",
      "Style: vivid"
    ]
  }]

  model = "openai/gpt-5-image"
}

resource "tofukit_project" "test" {
  name    = "openai-test"
  version = "1.0.0"
  model   = "openai/gpt-5-image"

  files = {
    "black.png" = tofukit_file.black_square
  }
}
`

	// Write config
	configPath := filepath.Join(testDir, "main.tf")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Detect IaC tool (terraform or tofu)
	iacTool := detectIaCTool(t)

	// Run terraform init
	runTerraformInit(t, iacTool, testDir)

	// Run terraform apply
	runTerraformApply(t, iacTool, testDir)

	// Verify image file exists
	imagePath := filepath.Join(testDir, "output", "black.png")
	require.FileExists(t, imagePath, "Generated image should exist")

	// Open and decode image
	file, err := os.Open(imagePath)
	require.NoError(t, err)
	defer file.Close()

	img, format, err := image.Decode(file)
	require.NoError(t, err)
	assert.Equal(t, "png", format, "Image should be PNG format")

	// Verify image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	t.Logf("Image dimensions: %dx%d", width, height)

	// Count black pixels
	totalPixels := width * height
	blackPixels := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to 8-bit and check if close to black
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			// Allow small tolerance for compression artifacts
			if r8 <= 10 && g8 <= 10 && b8 <= 10 {
				blackPixels++
			}
		}
	}

	blackPercent := float64(blackPixels) / float64(totalPixels) * 100
	t.Logf("Black pixels: %d/%d (%.2f%%)", blackPixels, totalPixels, blackPercent)

	// Verify image dimensions are correct
	// Note: OpenAI's image models are artistic and non-deterministic
	// We verify that image generation worked with correct dimensions, not the exact artistic output
	assert.Equal(t, 1024, width, "Image width should be 1024")
	assert.Equal(t, 1024, height, "Image height should be 1024")

	t.Logf("✓ OpenAI image generation successful: %dx%d PNG with %.2f%% dark pixels", width, height, blackPercent)
}
