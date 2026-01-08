package test

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// TestResourceFile_OpenAIIKBImage verifies OpenAI image generation with different model resolution scenarios
// Creates International Klein Blue (IKB) monochrome painting using openai/gpt-image-1-mini model
// Tests 3 scenarios:
// Step 1: Both file.model and project.model set to same value (openai/gpt-image-1-mini)
// Step 2: Only file.model set (project.model commented out) - tests unanimous file model detection
// Step 3: file.model=openai/gpt-image-1-mini, project.model=anthropic/claude-haiku - tests project override
func TestResourceFile_OpenAIIKBImage(t *testing.T) {
	// Skip if no OpenAI API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set - skipping OpenAI integration test")
	}

	testDir := createTestDirectory(t, "TestResourceFile_OpenAIIKB")

	// Detect IaC tool (terraform or tofu)
	iacTool := detectIaCTool(t)

	// Helper function to verify IKB image generation
	verifyIKBImage := func(t *testing.T, step string) {
		imagePath := filepath.Join(testDir, "output", "ikb.png")
		require.FileExists(t, imagePath, "[%s] Generated image should exist", step)

		// Open and decode image
		file, err := os.Open(imagePath)
		require.NoError(t, err)
		defer file.Close()

		img, format, err := image.Decode(file)
		require.NoError(t, err)
		assert.Equal(t, "png", format, "[%s] Image should be PNG format", step)

		// Verify image dimensions
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()
		t.Logf("[%s] Image dimensions: %dx%d", step, width, height)
		assert.Equal(t, 1024, width, "[%s] Image width should be 1024", step)
		assert.Equal(t, 1024, height, "[%s] Image height should be 1024", step)

		// Count blue pixels (IKB is ultramarine blue: high blue channel, low red/green)
		totalPixels := width * height
		bluePixels := 0

		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				// Convert to 8-bit
				r8 := uint8(r >> 8)
				g8 := uint8(g >> 8)
				b8 := uint8(b >> 8)

				// Check if pixel is blue-ish (blue channel dominant)
				// IKB is ultramarine: blue > 100, and blue > red, blue > green
				if b8 > 100 && b8 > r8 && b8 > g8 {
					bluePixels++
				}
			}
		}

		bluePercent := float64(bluePixels) / float64(totalPixels) * 100
		t.Logf("[%s] Blue pixels: %d/%d (%.2f%%)", step, bluePixels, totalPixels, bluePercent)

		// IKB paintings should have significant blue content
		// AI interprets IKB in two modes: literal (~90-95%) or artistic (~26-47%)
		// Threshold set to 20% to accommodate both interpretations while ensuring blue is present
		assert.Greater(t, bluePercent, 20.0, "[%s] IKB painting should have significant blue content (>20%%)", step)

		t.Logf("✓ [%s] OpenAI IKB generation successful: verified 1024x1024 PNG with %.2f%% blue pixels", step, bluePercent)
	}

	// Helper function to clean up between steps
	cleanupStep := func(t *testing.T, step string) {
		// Destroy resources
		runTerraformDestroy(t, iacTool, testDir)

		// Remove output directory
		outputDir := filepath.Join(testDir, "output")
		os.RemoveAll(outputDir)
		t.Logf("[%s] Cleanup complete", step)
	}

	// ============================================================
	// STEP 1: Both file.model and project.model set to same value
	// Expected: Uses openai/gpt-image-1-mini (project.model takes precedence but same value)
	// ============================================================
	t.Log("=== STEP 1: Both file.model and project.model = openai/gpt-image-1-mini ===")

	config1 := `
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
  debug       = true
}

resource "tofukit_file" "ikb" {
  name = "ikb.png"

  instructions = [{
    prompt = "Create an International Klein Blue (IKB), pure ultramarine blue monochrome painting. Solid IKB blue #002FA7, minimalist monochrome artwork, deep saturated ultramarine blue."
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

resource "tofukit_project" "test" {
  name    = "openai-ikb-test"
  version = "1.0.0"
  model   = "openai/gpt-image-1-mini"

  files = {
    "ikb.png" = tofukit_file.ikb
  }
}
`

	configPath := filepath.Join(testDir, "main.tofu")
	err := os.WriteFile(configPath, []byte(config1), 0644)
	require.NoError(t, err)

	// Run terraform init (only needed once)
	runTerraformInit(t, iacTool, testDir)

	// Run terraform apply
	runTerraformApply(t, iacTool, testDir)

	// Verify image
	verifyIKBImage(t, "Step1")

	// Cleanup before next step
	cleanupStep(t, "Step1")

	// ============================================================
	// STEP 2: Only file.model set, project.model NOT set
	// Expected: Uses openai/gpt-image-1-mini via unanimous file model detection
	// ============================================================
	t.Log("=== STEP 2: Only file.model = openai/gpt-image-1-mini (project.model NOT set) ===")

	config2 := `
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
  debug       = true
}

resource "tofukit_file" "ikb" {
  name = "ikb.png"

  instructions = [{
    prompt = "Create an International Klein Blue (IKB), pure ultramarine blue monochrome painting. Solid IKB blue #002FA7, minimalist monochrome artwork, deep saturated ultramarine blue."
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

resource "tofukit_project" "test" {
  name    = "openai-ikb-test"
  version = "1.0.0"
  // model NOT set - should use file's model via unanimous detection

  files = {
    "ikb.png" = tofukit_file.ikb
  }
}
`

	err = os.WriteFile(configPath, []byte(config2), 0644)
	require.NoError(t, err)

	// Run terraform apply (no init needed, already initialized)
	runTerraformApply(t, iacTool, testDir)

	// Verify image
	verifyIKBImage(t, "Step2")

	// Cleanup before next step
	cleanupStep(t, "Step2")

	// ============================================================
	// STEP 3: file.model=openai/gpt-image-1-mini, project.model=anthropic/claude-haiku
	// Expected: file.model takes precedence (per new hierarchy), so OpenAI is used
	// This should produce a valid image because the file's model is used
	// ============================================================
	t.Log("=== STEP 3: file.model=openai/gpt-image-1-mini, project.model=anthropic/claude-haiku ===")
	t.Log("NOTE: This step tests model override behavior - file.model should take precedence")

	config3 := `
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
  debug       = true
}

resource "tofukit_file" "ikb" {
  name = "ikb.png"

  instructions = [{
    prompt = "Create an International Klein Blue (IKB), pure ultramarine blue monochrome painting. Solid IKB blue #002FA7, minimalist monochrome artwork, deep saturated ultramarine blue."
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

resource "tofukit_project" "test" {
  name    = "openai-ikb-test"
  version = "1.0.0"
  model   = "anthropic/claude-haiku"

  files = {
    "ikb.png" = tofukit_file.ikb
  }
}
`

	err = os.WriteFile(configPath, []byte(config3), 0644)
	require.NoError(t, err)

	// Run terraform apply
	// Note: This may fail because Claude cannot generate images, or produce unexpected output
	// The test documents the current behavior where project.model overrides file.model
	runTerraformApply(t, iacTool, testDir)

	// Check if an image was created - should be valid since file.model takes precedence
	imagePath := filepath.Join(testDir, "output", "ikb.png")
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Error("[Step3] No image created - file.model should have taken precedence over project.model")
	} else {
		t.Log("[Step3] File exists - verifying it's a valid image")
		file, err := os.Open(imagePath)
		require.NoError(t, err, "[Step3] Could not open file")
		defer file.Close()
		_, _, err = image.Decode(file)
		require.NoError(t, err, "[Step3] File is not a valid image - file.model should have been used")
		t.Log("[Step3] Valid image produced - confirms file.model takes precedence over project.model")
	}

	// Cleanup before next step
	cleanupStep(t, "Step3")

	// ============================================================
	// STEP 4: Multiple files with different models (multi-model execution)
	// - ikb.png uses openai/gpt-image-1-mini (image generation)
	// - README.md uses anthropic/claude-haiku (text generation)
	// Expected: Each file uses its own model; both are created successfully
	// ============================================================
	t.Log("=== STEP 4: Multiple files with different models (multi-model execution) ===")
	t.Log("NOTE: Testing that different files can use different LLM providers in the same project")

	config4 := `
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
  debug       = true
}

# Image file using OpenAI's image generation model
resource "tofukit_file" "ikb" {
  name = "ikb.png"

  instructions = [{
    prompt = "Create an International Klein Blue (IKB), pure ultramarine blue monochrome painting. Solid IKB blue #002FA7, minimalist monochrome artwork, deep saturated ultramarine blue."
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

# Text file using Claude Haiku
resource "tofukit_file" "readme" {
  name = "README.md"

  instructions = [{
    prompt = "Create a README file explaining what International Klein Blue is and its artistic significance"
    constraints = [
      "Keep it concise (under 200 words)",
      "Include the hex color code #002FA7",
      "Mention Yves Klein as the creator"
    ]
  }]

  # Explicitly use Claude Haiku for text generation
  model = "anthropic/claude-haiku"
}

resource "tofukit_project" "test" {
  name    = "multi-model-test"
  version = "1.0.0"
  # No project-level model - each file uses its own model or provider default

  files = {
    "ikb.png"   = tofukit_file.ikb
    "README.md" = tofukit_file.readme
  }
}
`

	err = os.WriteFile(configPath, []byte(config4), 0644)
	require.NoError(t, err)

	// Run terraform apply
	runTerraformApply(t, iacTool, testDir)

	// Verify ikb.png was created and is a valid image
	imagePath = filepath.Join(testDir, "output", "ikb.png")
	require.FileExists(t, imagePath, "[Step4] ikb.png should exist")

	imgFile, err := os.Open(imagePath)
	require.NoError(t, err, "[Step4] Could not open ikb.png")
	defer imgFile.Close()
	img, _, err := image.Decode(imgFile)
	require.NoError(t, err, "[Step4] ikb.png is not a valid image")
	require.Equal(t, 1024, img.Bounds().Dx(), "[Step4] Image width should be 1024")
	require.Equal(t, 1024, img.Bounds().Dy(), "[Step4] Image height should be 1024")
	t.Log("[Step4] ikb.png: Valid 1024x1024 image generated by OpenAI")

	// Verify README.md was created and has expected content
	readmePath := filepath.Join(testDir, "output", "README.md")
	require.FileExists(t, readmePath, "[Step4] README.md should exist")

	readmeContent, err := os.ReadFile(readmePath)
	require.NoError(t, err, "[Step4] Could not read README.md")
	readmeStr := string(readmeContent)
	require.NotEmpty(t, readmeStr, "[Step4] README.md should not be empty")

	// Check for expected content (Claude should have generated this)
	t.Logf("[Step4] README.md content length: %d bytes", len(readmeStr))
	if strings.Contains(strings.ToLower(readmeStr), "klein") || strings.Contains(readmeStr, "#002FA7") {
		t.Log("[Step4] README.md: Contains expected content about Klein Blue")
	} else {
		t.Log("[Step4] README.md: Content may not contain expected keywords, but file was generated")
	}

	// Cleanup before next step
	cleanupStep(t, "Step4")

	// ============================================================
	// STEP 5: File model overrides project model (different providers)
	// - file.model = openai/gpt-image-1-mini (image generation)
	// - project.model = anthropic/claude-haiku (text generation)
	// Expected: file.model takes precedence, OpenAI is used for the image
	// ============================================================
	t.Log("=== STEP 5: File model (OpenAI) overrides project model (Claude) ===")
	t.Log("NOTE: Testing that file.model takes precedence even when providers differ")

	config5 := `
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
  debug       = true
}

resource "tofukit_file" "ui_weather_sun" {
  name = "ui_weather_sun.png"

  instructions = [{
    prompt = "Create an International Klein Blue (IKB), pure ultramarine blue monochrome painting. Solid IKB blue #002FA7, minimalist monochrome artwork, deep saturated ultramarine blue."
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

resource "tofukit_project" "labs" {
  name        = "labs"
  description = "Generated from kitcut analysis"
  version     = "1.0.0"

  model = "anthropic/claude-haiku"

  files = {
    "ui_weather_sun.png" = tofukit_file.ui_weather_sun
  }
}
`

	err = os.WriteFile(configPath, []byte(config5), 0644)
	require.NoError(t, err)

	// Run terraform apply
	runTerraformApply(t, iacTool, testDir)

	// Verify ui_weather_sun.png was created and is a valid image
	imagePath = filepath.Join(testDir, "output", "ui_weather_sun.png")
	require.FileExists(t, imagePath, "[Step5] ui_weather_sun.png should exist")

	imgFile, err = os.Open(imagePath)
	require.NoError(t, err, "[Step5] Could not open ui_weather_sun.png")
	defer imgFile.Close()
	img, _, err = image.Decode(imgFile)
	require.NoError(t, err, "[Step5] ui_weather_sun.png is not a valid image - file.model (OpenAI) should have been used over project.model (Claude)")
	require.Equal(t, 1024, img.Bounds().Dx(), "[Step5] Image width should be 1024")
	require.Equal(t, 1024, img.Bounds().Dy(), "[Step5] Image height should be 1024")
	t.Log("[Step5] ui_weather_sun.png: Valid 1024x1024 image - confirms file.model (openai/gpt-image-1-mini) overrides project.model (anthropic/claude-haiku)")

	// Cleanup before next step
	cleanupStep(t, "Step5")

	// ============================================================
	// STEP 6: Multiple image files in single project (KIRR-126 bug reproduction)
	// - 3 image files, all using openai/gpt-image-1-mini
	// - Expected: ALL images are generated (not just 1)
	// - Expected: Debug prompts are small (~1KB each, NOT 292KB)
	// - Expected: No "software architect" system prompt in debug files
	// ============================================================
	t.Log("=== STEP 6: Multiple image files in single project (KIRR-126) ===")
	t.Log("NOTE: Testing that ALL images are generated when project has multiple image files")

	config6 := `
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
  debug       = true
}

resource "tofukit_file" "red_square" {
  name = "red_square.png"

  instructions = [{
    prompt = "A solid red square on white background. Simple geometric shape, flat color, no gradients."
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

resource "tofukit_file" "blue_circle" {
  name = "blue_circle.png"

  instructions = [{
    prompt = "A solid blue circle on white background. Simple geometric shape, flat color, no gradients."
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

resource "tofukit_file" "green_triangle" {
  name = "green_triangle.png"

  instructions = [{
    prompt = "A solid green triangle on white background. Simple geometric shape, flat color, no gradients."
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

resource "tofukit_project" "multi_image" {
  name        = "multi-image-test"
  description = "Test multiple images in single project"
  version     = "1.0.0"

  files = {
    "red_square.png"    = tofukit_file.red_square
    "blue_circle.png"   = tofukit_file.blue_circle
    "green_triangle.png" = tofukit_file.green_triangle
  }
}
`

	err = os.WriteFile(configPath, []byte(config6), 0644)
	require.NoError(t, err)

	// Run terraform apply
	runTerraformApply(t, iacTool, testDir)

	// Verify ALL 3 images were created
	imageFiles := []string{"red_square.png", "blue_circle.png", "green_triangle.png"}
	for _, imgName := range imageFiles {
		imgPath := filepath.Join(testDir, "output", imgName)
		require.FileExists(t, imgPath, "[Step6] %s should exist", imgName)

		imgFile, err := os.Open(imgPath)
		require.NoError(t, err, "[Step6] Could not open %s", imgName)

		img, _, err := image.Decode(imgFile)
		imgFile.Close()
		require.NoError(t, err, "[Step6] %s is not a valid image", imgName)
		require.Equal(t, 1024, img.Bounds().Dx(), "[Step6] %s width should be 1024", imgName)
		require.Equal(t, 1024, img.Bounds().Dy(), "[Step6] %s height should be 1024", imgName)

		t.Logf("[Step6] %s: Valid 1024x1024 image generated", imgName)
	}

	// Verify debug prompts are small (not 292KB) and don't contain wrong system prompt
	debugDir := filepath.Join(testDir, "output", ".debug")
	debugFiles, err := os.ReadDir(debugDir)
	if err == nil {
		for _, df := range debugFiles {
			if strings.HasSuffix(df.Name(), ".json") {
				debugPath := filepath.Join(debugDir, df.Name())
				debugContent, err := os.ReadFile(debugPath)
				if err != nil {
					continue
				}

				// Check size - should be small (~1KB for image prompts, not 292KB)
				sizeKB := len(debugContent) / 1024
				if sizeKB > 50 {
					t.Errorf("[Step6] Debug file %s is too large: %dKB (should be <50KB for image prompts)", df.Name(), sizeKB)
				} else {
					t.Logf("[Step6] Debug file %s size: %dKB (OK)", df.Name(), sizeKB)
				}

				// Check for wrong system prompt
				contentStr := string(debugContent)
				if strings.Contains(contentStr, "senior software architect") {
					t.Errorf("[Step6] Debug file %s contains wrong system prompt 'senior software architect' - image generation should not use Claude project prompt", df.Name())
				}
			}
		}
	}

	t.Log("[Step6] All 3 images generated successfully")

	t.Log("✓ All 6 model resolution scenarios tested")
	t.Log("  Step 1: Both file.model and project.model set - uses file.model")
	t.Log("  Step 2: Only file.model set - uses file.model")
	t.Log("  Step 3: file.model and project.model differ - uses file.model (precedence)")
	t.Log("  Step 4: Multiple files with different models - each uses its own model")
	t.Log("  Step 5: file.model (OpenAI) overrides project.model (Claude) - cross-provider precedence works")
	t.Log("  Step 6: Multiple image files in project - all images generated with correct prompts (KIRR-126)")
}

// averageBrightness calculates the average brightness (0-255) for a horizontal slice of the image
func averageBrightness(img image.Image, yStart, yEnd int) float64 {
	bounds := img.Bounds()
	var total float64
	pixels := 0

	for y := yStart; y < yEnd && y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Convert to 0-255 range and calculate brightness
			brightness := (float64(r>>8) + float64(g>>8) + float64(b>>8)) / 3.0
			total += brightness
			pixels++
		}
	}

	if pixels == 0 {
		return 0
	}
	return total / float64(pixels)
}

// TestResourceFile_WireframeImage verifies wireframe mode image generation
// Uses Claude to generate SVG wireframe, then converts to PNG
// This mode does NOT require OpenAI API key - uses Claude for SVG generation
func TestResourceFile_WireframeImage(t *testing.T) {
	testDir := createTestDirectory(t, "TestResourceFile_Wireframe")

	// Detect IaC tool (terraform or tofu)
	iacTool := detectIaCTool(t)

	// Helper function to verify wireframe image generation
	verifyWireframeImage := func(t *testing.T, imagePath string, expectedWidth, expectedHeight int) {
		require.FileExists(t, imagePath, "Generated wireframe image should exist")

		// Open and decode image
		file, err := os.Open(imagePath)
		require.NoError(t, err)
		defer file.Close()

		img, format, err := image.Decode(file)
		require.NoError(t, err)
		assert.Equal(t, "png", format, "Wireframe image should be PNG format")

		// Verify image dimensions
		bounds := img.Bounds()
		width := bounds.Dx()
		height := bounds.Dy()
		t.Logf("Wireframe image dimensions: %dx%d", width, height)

		// Check dimensions are as expected
		assert.Equal(t, expectedWidth, width, "Wireframe image width should match")
		assert.Equal(t, expectedHeight, height, "Wireframe image height should match")

		// Verify image has some content (not entirely blank)
		// Calculate average brightness to ensure it's not all black or all white
		totalPixels := width * height
		var totalBrightness float64

		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				brightness := (float64(r>>8) + float64(g>>8) + float64(b>>8)) / 3.0
				totalBrightness += brightness
			}
		}

		avgBrightness := totalBrightness / float64(totalPixels)
		t.Logf("Wireframe average brightness: %.2f (0=black, 255=white)", avgBrightness)

		// Wireframe should have some variation (not all black or all white)
		// A valid SVG wireframe should have brightness between 10 and 245
		assert.Greater(t, avgBrightness, 5.0, "Wireframe should not be entirely black")
		assert.Less(t, avgBrightness, 250.0, "Wireframe should not be entirely white")

		t.Logf("✓ Wireframe generation successful: verified %dx%d PNG with avg brightness %.2f", width, height, avgBrightness)
	}

	// ============================================================
	// TEST: Wireframe mode generates SVG wireframe and converts to PNG
	// Uses image_mode = "wireframe" in provider config
	// ============================================================
	t.Log("=== Testing wireframe mode (Claude SVG -> PNG) ===")

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
  output_path = "output"
  debug       = true
  image_mode  = "wireframe"
}

resource "tofukit_file" "portrait" {
  name = "portrait.png"

  instructions = [{
    prompt = "A portrait of a wizard with a long beard, pointed hat, and a staff. The wizard is standing in front of a mystical forest with glowing mushrooms."
  }]

  image = {
    size = "512x512"
  }

  # Use an OpenAI image model - but wireframe mode will intercept and use Claude SVG instead
  model = "openai/gpt-image-1-mini"
}

resource "tofukit_project" "test" {
  name    = "wireframe-test"
  version = "1.0.0"

  files = {
    "portrait.png" = tofukit_file.portrait
  }
}
`

	configPath := filepath.Join(testDir, "main.tofu")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Run terraform init
	runTerraformInit(t, iacTool, testDir)

	// Run terraform apply
	runTerraformApply(t, iacTool, testDir)

	// Verify wireframe image was created with the size specified in file's image block
	imagePath := filepath.Join(testDir, "output", "portrait.png")
	verifyWireframeImage(t, imagePath, 512, 512)

	// Check if SVG was saved in debug mode (now in .debug/ directory)
	debugDir := filepath.Join(testDir, "output", ".debug")
	if _, err := os.Stat(debugDir); !os.IsNotExist(err) {
		// Look for SVG file in .debug/ directory (saved when debug=true)
		svgPath := filepath.Join(debugDir, "portrait.svg")
		if _, err := os.Stat(svgPath); !os.IsNotExist(err) {
			t.Log("✓ SVG intermediate file found in .debug/ directory")
		}
	}

	// Cleanup
	runTerraformDestroy(t, iacTool, testDir)

	t.Log("✓ Wireframe mode test completed successfully")
}

// TestResourceFile_WireframeSVGGenerationSuccess tests SVG generation via wireframe mode
// Tests the SVG feedback loop (TFK-11 Option B) where invalid SVGs are sent back to Claude for correction
// Uses Claude Haiku for SVG generation via wireframe mode
func TestResourceFile_WireframeSVGGenerationSuccess(t *testing.T) {
	// This test requires Claude to be available for wireframe SVG generation
	// No OpenAI API key needed - uses Claude CLI

	testCases := []struct {
		name           string
		fileName       string
		expectedFormat string
		expectedWidth  int
		expectedHeight int
		prompt         string
	}{
		{
			name:           "SVG_ReactIcon",
			fileName:       "react-icon.svg",
			expectedFormat: "svg",
			expectedWidth:  512,
			expectedHeight: 512,
			prompt: `**CRITICAL: EXECUTE IMMEDIATELY. DO NOT ASK QUESTIONS.**

**Style:** Technology brand logo/icon with React official color scheme

**Composition:**
- Square icon format: centered blue atom/electron structure
- 60% cyan blue (#61DAFB) occupying center space
- Central nucleus circle surrounded by three elliptical electron orbits
- Electron orbits at 120-degree angles creating balanced composition
- Clean lines, minimal design, modern JavaScript library aesthetic
- High contrast: bright cyan on dark/transparent background
- Scalable vector-style design
- Professional technology branding look

**Generation Prompt:**

Create a technology brand icon for React. Design a square icon featuring the distinctive React atom logo. Use a central circle nucleus in cyan blue (#61DAFB or similar React blue) at the center of the composition. Around this nucleus, create three elliptical electron orbit paths positioned at 120-degree angles from each other, creating a balanced, symmetrical design. The orbits should be simple curved lines in the same cyan blue color.`,
		},
		{
			name:           "PNG_BrainIllustration",
			fileName:       "brain-lifting.png",
			expectedFormat: "png",
			expectedWidth:  512,
			expectedHeight: 512,
			prompt: `**CRITICAL: EXECUTE IMMEDIATELY. DO NOT ASK QUESTIONS.**

**Style:** Flat design animated character illustration with cartoon aesthetic, friendly and approachable style

**Composition:**
- Square composition: 20% white top padding, 60% character, 20% white bottom padding
- Centered character at approximately 60% of frame height
- Pink brain shape as character head, 25% of character height
- Rounded, organic brain contour with subtle ridge texture
- Dark eyes (simple dots or ovals) positioned at 40% of head height
- Pink arms extended outward and upward
- Dark gray metal dumbbells in each hand
- Warm pink/magenta color (#FF69B4 or similar) for brain and body
- Dark gray (#4A4A4A) for dumbbells and accent features
- Clean white background

**Generation Prompt:**

Create a flat design character illustration of a friendly cartoon brain lifting dumbbells. The composition should be square with the character centered. The character should be a large rounded pink brain shape. Use warm pink/magenta color (#FF69B4 or similar) for the brain and body. Use dark gray (#4A4A4A) for the dumbbells. The background should be clean white.`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := createTestDirectory(t, "TestResourceFile_WireframeSVG_"+tc.name)

			// Detect IaC tool
			iacTool := detectIaCTool(t)

			// Build configuration
			config := fmt.Sprintf(`
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
  debug       = true
  image_mode  = "wireframe"
}

resource "tofukit_file" "test_image" {
  name = "%s"

  instructions = [{
    prompt = <<-PROMPT
%s
    PROMPT
  }]

  image = {
    size = "%dx%d"
  }

  # Model is intercepted by wireframe mode - Claude is used for SVG generation
  model = "openai/gpt-image-1"
}

resource "tofukit_project" "test" {
  name    = "wireframe-svg-test"
  version = "1.0.0"

  files = {
    "%s" = tofukit_file.test_image
  }
}
`, tc.fileName, tc.prompt, tc.expectedWidth, tc.expectedHeight, tc.fileName)

			configPath := filepath.Join(testDir, "main.tofu")
			err := os.WriteFile(configPath, []byte(config), 0644)
			require.NoError(t, err)

			// Run terraform init and apply
			runTerraformInit(t, iacTool, testDir)
			runTerraformApply(t, iacTool, testDir)

			// Verify output file exists
			outputPath := filepath.Join(testDir, "output", tc.fileName)
			require.FileExists(t, outputPath, "Generated file should exist: %s", tc.fileName)

			// Verify file format and content
			if tc.expectedFormat == "svg" {
				// Read SVG content and verify it's valid XML
				content, err := os.ReadFile(outputPath)
				require.NoError(t, err, "Should be able to read SVG file")

				svgStr := string(content)
				assert.Contains(t, svgStr, "<svg", "SVG file should contain <svg tag")
				assert.Contains(t, svgStr, "</svg>", "SVG file should contain closing </svg> tag")
				assert.Contains(t, svgStr, "xmlns", "SVG should have xmlns attribute")

				// Check that SVG has proper dimensions
				assert.True(t,
					strings.Contains(svgStr, fmt.Sprintf(`width="%d"`, tc.expectedWidth)) ||
						strings.Contains(svgStr, fmt.Sprintf(`width='%d'`, tc.expectedWidth)) ||
						strings.Contains(svgStr, `viewBox`),
					"SVG should have width attribute or viewBox")

				t.Logf("✓ SVG file generated: %d bytes, contains valid SVG structure", len(content))

			} else if tc.expectedFormat == "png" {
				// Open and decode PNG image
				file, err := os.Open(outputPath)
				require.NoError(t, err, "Should be able to open PNG file")
				defer file.Close()

				img, format, err := image.Decode(file)
				require.NoError(t, err, "PNG file should be decodable - SVG feedback loop should have produced valid SVG for conversion")
				assert.Equal(t, "png", format, "File should be PNG format")

				// Verify dimensions
				bounds := img.Bounds()
				assert.Equal(t, tc.expectedWidth, bounds.Dx(), "Image width should match")
				assert.Equal(t, tc.expectedHeight, bounds.Dy(), "Image height should match")

				// Verify image has content (not blank)
				var totalBrightness float64
				totalPixels := bounds.Dx() * bounds.Dy()
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						r, g, b, _ := img.At(x, y).RGBA()
						brightness := (float64(r>>8) + float64(g>>8) + float64(b>>8)) / 3.0
						totalBrightness += brightness
					}
				}
				avgBrightness := totalBrightness / float64(totalPixels)

				assert.Greater(t, avgBrightness, 5.0, "Image should not be entirely black")
				assert.Less(t, avgBrightness, 250.0, "Image should not be entirely white")

				t.Logf("✓ PNG file generated: %dx%d, avg brightness: %.2f", bounds.Dx(), bounds.Dy(), avgBrightness)
			}

			// Check debug directory for SVG intermediate files
			debugDir := filepath.Join(testDir, "output", ".debug")
			if _, err := os.Stat(debugDir); !os.IsNotExist(err) {
				// Look for SVG debug files
				debugFiles, _ := os.ReadDir(debugDir)
				for _, df := range debugFiles {
					if strings.HasSuffix(df.Name(), ".svg") {
						t.Logf("✓ Found SVG debug file: %s", df.Name())
					}
					if strings.Contains(df.Name(), ".failed.svg") {
						// If there's a failed SVG, the feedback loop was triggered
						t.Logf("ℹ Found failed SVG attempt (feedback loop triggered): %s", df.Name())
					}
					if strings.Contains(df.Name(), ".attempt") {
						t.Logf("ℹ Found retry attempt file: %s", df.Name())
					}
				}
			}

			// NOTE: Do NOT run terraform destroy - keep output files for inspection
			// The test directory cleanup is handled by t.Cleanup() in createTestDirectory
			// which only runs if CLEANUP_TEST_OUTPUT=true

			t.Logf("✓ Wireframe %s generation test completed successfully", tc.expectedFormat)
			t.Logf("  Output preserved at: %s/output/", testDir)
		})
	}
}
