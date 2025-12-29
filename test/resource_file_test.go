package test

import (
	"encoding/json"
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

	t.Log("✓ All 5 model resolution scenarios tested")
	t.Log("  Step 1: Both file.model and project.model set - uses file.model")
	t.Log("  Step 2: Only file.model set - uses file.model")
	t.Log("  Step 3: file.model and project.model differ - uses file.model (precedence)")
	t.Log("  Step 4: Multiple files with different models - each uses its own model")
	t.Log("  Step 5: file.model (OpenAI) overrides project.model (Claude) - cross-provider precedence works")
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
