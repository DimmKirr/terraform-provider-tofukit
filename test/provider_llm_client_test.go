package test

import (
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

// TestProviderClaudeMaxTurnsMustFail validates that setting claude_max_turns=2
// causes Claude to fail when the project requires more turns
func TestProviderClaudeMaxTurnsMustFail(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProviderClaudeMaxTurnsMustFail")
	t.Log("Testing claude_max_turns=2 failure...")

	// Create a Terraform configuration that sets claude_max_turns=2
	// The project will require more than 2 turns to complete, so it should fail
	projectConfig := `
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  claude_max_turns      = 2
  debug                 = true
}

resource "tofukit_project" "test_turns" {
  name = "test-max-turns"
  description = "Test project that should fail due to max_turns=2"
  version = "1.0.0"

  requirements = [
    {
      name = "Multi-file project"
      instructions = [
        {
          prompt = "Create a simple Python CLI application with the following files: main.py (entry point with main function), config.py (configuration with APP_NAME constant), utils.py (utilities with helper function), README.md (documentation with installation section), and requirements.txt (dependencies with at least click package). Each file should have meaningful content."
        }
      ]
      verifications = [
        {
          command = "test -f main.py"
        },
        {
          command = "grep -q 'def main' main.py"
        },
        {
          command = "test -f config.py"
        },
        {
          command = "grep -q 'APP_NAME' config.py"
        },
        {
          command = "test -f utils.py"
        },
        {
          command = "grep -q 'def' utils.py"
        },
        {
          command = "test -f README.md"
        },
        {
          command = "grep -q -i 'installation' README.md"
        },
        {
          command = "test -f requirements.txt"
        },
        {
          command = "grep -q 'click' requirements.txt"
        }
      ]
    }
  ]
}
`

	// Write the project.tofu to test directory
	projectPath := filepath.Join(testDir, "project.tofu")
	require.NoError(t, os.WriteFile(projectPath, []byte(projectConfig), 0644))

	// Detect IaC tool
	iacTool := detectIaCTool(t)

	// Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Run plan (should succeed - plan just prepares the prompt)
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Run apply - THIS SHOULD FAIL due to max_turns=2
	t.Log("Running tofu apply --auto-approve (expecting failure)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()

	// Apply MUST fail
	if err == nil {
		t.Fatalf("Expected apply to fail due to claude_max_turns=2, but it succeeded!\nOutput: %s", applyOutput)
	}

	// Verify the error is related to reaching max turns
	outputStr := string(applyOutput)
	t.Logf("Apply output: %s", outputStr)

	// Check for max turns related messages
	// Claude CLI typically outputs "max turns" when the limit is reached
	hasMaxTurnsMessage := strings.Contains(strings.ToLower(outputStr), "max turns") ||
		strings.Contains(strings.ToLower(outputStr), "turn limit") ||
		strings.Contains(strings.ToLower(outputStr), "turns reached")

	assert.True(t, hasMaxTurnsMessage,
		"Expected error message to contain 'max turns', 'turn limit', or 'turns reached'\nActual output: %s",
		outputStr)

	t.Log("✓ Test passed: Apply failed as expected due to claude_max_turns=2")

	// Verify debug files were created to show the max_turns setting
	debugDir := filepath.Join(testDir, "output", ".debug")
	if _, err := os.Stat(debugDir); err == nil {
		t.Logf("Debug directory exists: %s", debugDir)
	}
}

// TestProviderMultiModelExecutorRouting validates that the correct LLM executor
// is called based on the model specified in the file resource.
//
// This test verifies:
// - anthropic/claude-haiku → Claude executor (creates claude-*.json debug files)
// - openai/gpt-image-1-mini → OpenAI executor (creates openai-*.json debug files)
//
// Related to KIRR-126: Image generation should use OpenAI executor directly,
// not route through Claude project implementation flow.
func TestProviderMultiModelExecutorRouting(t *testing.T) {
	testDir := createTestDirectory(t, "TestProviderMultiModelExecutorRouting")
	t.Log("Testing multi-model executor routing...")

	// Create config with files using different providers
	projectConfig := `
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

# Text file using Claude Haiku
resource "tofukit_file" "readme" {
  name = "readme"
  path = "README.md"

  instructions = [{
    prompt = "Create a simple README with just the text 'Hello World' as a heading"
    constraints = ["Keep it under 50 characters total", "Just one line with # Hello World"]
  }]

  model = "anthropic/claude-haiku"
}

# Image file using OpenAI image model
resource "tofukit_file" "icon" {
  name = "icon"
  path = "icon.png"

  instructions = [{
    prompt = "A simple red circle on white background"
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

resource "tofukit_project" "test" {
  name        = "multi-model-routing-test"
  description = "Test that correct executor is used per model"
  version     = "1.0.0"

  files = {
    "README.md" = tofukit_file.readme
    "icon.png"  = tofukit_file.icon
  }
}
`

	// Write config
	configPath := filepath.Join(testDir, "project.tofu")
	require.NoError(t, os.WriteFile(configPath, []byte(projectConfig), 0644))

	// Detect IaC tool and run
	iacTool := detectIaCTool(t)
	runTerraformInit(t, iacTool, testDir)
	runTerraformApply(t, iacTool, testDir)

	// Verify both files were created
	readmePath := filepath.Join(testDir, "output", "README.md")
	iconPath := filepath.Join(testDir, "output", "icon.png")

	require.FileExists(t, readmePath, "README.md should exist (Claude Haiku)")
	require.FileExists(t, iconPath, "icon.png should exist (OpenAI Image)")

	// Verify icon.png is a valid image
	imgFile, err := os.Open(iconPath)
	require.NoError(t, err, "Could not open icon.png")
	defer imgFile.Close()
	img, _, err := image.Decode(imgFile)
	require.NoError(t, err, "icon.png is not a valid image")
	require.Equal(t, 1024, img.Bounds().Dx(), "Image width should be 1024")
	t.Log("✓ icon.png: Valid 1024x1024 image generated by OpenAI")

	// Verify README.md has content
	readmeContent, err := os.ReadFile(readmePath)
	require.NoError(t, err)
	require.NotEmpty(t, readmeContent, "README.md should have content")
	t.Logf("✓ README.md: Created with %d bytes by Claude Haiku", len(readmeContent))

	// Check debug files to verify correct executor routing
	debugDir := filepath.Join(testDir, "output", ".debug")
	debugFiles, err := os.ReadDir(debugDir)
	require.NoError(t, err, "Debug directory should exist")

	var hasClaudeDebug, hasOpenAIDebug bool
	var claudeDebugFiles, openaiDebugFiles []string

	for _, df := range debugFiles {
		name := df.Name()
		if strings.HasPrefix(name, "claude-") && strings.HasSuffix(name, ".json") {
			hasClaudeDebug = true
			claudeDebugFiles = append(claudeDebugFiles, name)
		}
		if strings.HasPrefix(name, "openai-") && strings.HasSuffix(name, ".json") {
			hasOpenAIDebug = true
			openaiDebugFiles = append(openaiDebugFiles, name)
		}
	}

	t.Logf("Claude debug files: %v", claudeDebugFiles)
	t.Logf("OpenAI debug files: %v", openaiDebugFiles)

	// Verify OpenAI executor was used for image (should have openai-*.json)
	assert.True(t, hasOpenAIDebug,
		"OpenAI executor should create openai-*.json debug files for image generation")

	// Verify Claude executor was used for text (should have claude-*.json)
	assert.True(t, hasClaudeDebug,
		"Claude executor should create claude-*.json debug files for text generation")

	// CRITICAL: Verify OpenAI debug files do NOT contain Claude system prompt
	// This is the KIRR-126 bug - OpenAI image generation should NOT use Claude prompts
	for _, df := range debugFiles {
		if strings.HasPrefix(df.Name(), "openai-") && strings.HasSuffix(df.Name(), ".json") {
			debugPath := filepath.Join(debugDir, df.Name())
			content, err := os.ReadFile(debugPath)
			if err != nil {
				continue
			}

			contentStr := string(content)

			// Check for wrong system prompt
			if strings.Contains(contentStr, "senior software architect") {
				t.Errorf("KIRR-126 BUG: OpenAI debug file %s contains Claude's 'senior software architect' system prompt - image generation should not use Claude prompt builder", df.Name())
			}

			// Check size - OpenAI image prompts should be small, not 292KB
			sizeKB := len(content) / 1024
			if sizeKB > 50 {
				t.Errorf("KIRR-126 BUG: OpenAI debug file %s is %dKB (should be <50KB for image prompts)", df.Name(), sizeKB)
			} else {
				t.Logf("✓ OpenAI debug file %s size: %dKB (OK)", df.Name(), sizeKB)
			}
		}
	}

	t.Log("✓ Multi-model executor routing test completed")
}

// TestOpenAI_ModerationBypass_KIRR127 tests that benign prompts are not falsely
// blocked by OpenAI's moderation when using moderation="low" (our default).
//
// KIRR-127: OpenAI image moderation false positive blocks benign rabbit character prompt
// The original bug was that prompts with words like "anthropomorphic", "realistic fur"
// were blocked even for completely family-friendly content.
//
// This test verifies that our moderation="low" default fixes this issue.
func TestOpenAI_ModerationBypass_KIRR127(t *testing.T) {
	testDir := createTestDirectory(t, "TestOpenAI_ModerationBypass_KIRR127")
	t.Log("Testing KIRR-127: OpenAI moderation bypass for benign prompts...")

	// This is a simplified version of the rabbit prompt that triggered KIRR-127
	// The full prompt included words like "anthropomorphic", "realistic fur texture"
	// which falsely triggered moderation with default settings
	rabbitPrompt := `Create a 3D rendered character illustration showing a friendly anthropomorphic rabbit in a warm kitchen setting. The rabbit should have realistic gray fur with soft texture. Feature large upright ears with pink inner ear details. Create a warm, friendly facial expression with large expressive brown eyes and gentle smile. Dress the character in a blue collared shirt. Have the character holding a brown paper bag filled with orange carrots with fresh green carrot tops.`

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
}

resource "tofukit_file" "rabbit" {
  name = "rabbit"
  path = "rabbit.png"

  instructions = [{
    prompt = %q
    constraints = [
      "Do not use realistic rabbit anatomy - must be anthropomorphic with human-like posture",
      "Warm, friendly expression required"
    ]
  }]

  image = {
    size    = "1024x1024"
    quality = "low"
  }

  model = "openai/gpt-image-1-mini"
}

resource "tofukit_project" "rabbit_test" {
  name        = "kirr-127-moderation-test"
  description = "Test that benign rabbit prompt is not blocked by moderation"
  version     = "1.0.0"

  files = {
    "rabbit.png" = tofukit_file.rabbit
  }
}
`, rabbitPrompt)

	configPath := filepath.Join(testDir, "project.tofu")
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0644))

	iacTool := detectIaCTool(t)
	runTerraformInit(t, iacTool, testDir)

	// This is the key test - if moderation="low" works, this should succeed
	// With moderation="auto" (old default), this would fail with moderation_blocked
	t.Log("Running apply with moderation='low' (should succeed)...")
	runTerraformApply(t, iacTool, testDir)

	// Verify image was created
	rabbitPath := filepath.Join(testDir, "output", "rabbit.png")
	require.FileExists(t, rabbitPath, "rabbit.png should exist - moderation bypass worked")

	// Verify it's a valid image
	imgFile, err := os.Open(rabbitPath)
	require.NoError(t, err)
	defer imgFile.Close()

	img, _, err := image.Decode(imgFile)
	require.NoError(t, err, "rabbit.png should be a valid image")
	require.Equal(t, 1024, img.Bounds().Dx(), "Image width should be 1024")
	require.Equal(t, 1024, img.Bounds().Dy(), "Image height should be 1024")

	t.Log("✓ KIRR-127 FIXED: Rabbit prompt successfully generated with moderation='low'")
	t.Log("  - Prompt contained: 'anthropomorphic', 'realistic fur', detailed character descriptions")
	t.Log("  - These words previously triggered false positive moderation blocks")
	t.Log("  - With moderation='low', benign content is allowed through")
}
