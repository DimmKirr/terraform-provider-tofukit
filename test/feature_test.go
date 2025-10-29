package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ====================
// Feature Resource Tests
// ====================

// TestFeatureResourceCreateSuccess tests creating a standalone feature resource
func TestFeatureResourceCreateSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestFeatureResourceCreateSuccess")

	// Create Terraform config with a feature resource
	config := `
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
}

resource "tofukit_feature" "hello_cmd" {
  name        = "hello-command"
  description = "A simple hello command"
  prompt      = "Implement a hello command that prints 'Hello from feature!'"

  files = {
    "hello.txt" = {
      content = "Hello from feature!\n"
    }
  }

  verifications = [
    {
      command = "cat hello.txt"
      expect  = "Hello from feature!"
    }
  ]
}
`

	// Write config
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Detect IaC tool
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

	// Run init
	cmd := exec.Command(iacTool, "init")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output:\n%s", output)
		t.Fatalf("Init failed: %v", err)
	}

	// Run apply
	cmd = exec.Command(iacTool, "apply", "-auto-approve")
	cmd.Dir = testDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("Apply output:\n%s", output)
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify the feature resource was created (no files should be generated yet)
	// Features are just registry entries until used by a project
	tfstateContent, err := os.ReadFile(filepath.Join(testDir, "terraform.tfstate"))
	require.NoError(t, err)
	assert.Contains(t, string(tfstateContent), "tofukit_feature")
	assert.Contains(t, string(tfstateContent), "hello_cmd")
	assert.Contains(t, string(tfstateContent), "hello-command")

	// Verify output directory is empty (features don't generate files by themselves)
	outputDir := filepath.Join(testDir, "output")
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("Failed to read output directory: %v", err)
		}
	} else {
		assert.Empty(t, entries, "Feature resource should not generate files by itself")
	}
}

// TestFeatureResourceInlineSuccess tests using an inline feature definition in a project
func TestFeatureResourceInlineSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestFeatureResourceInlineSuccess")

	// Create Terraform config with inline feature
	config := `
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug = true
}

resource "tofukit_project" "test" {
  name    = "inline-feature-test"
  version = "0.1.0"

  features = {
    "hello" = {
      prompt = "Create a hello.txt file with greeting"
      files = {
        "hello.txt" = {
          content = "Hello from inline feature!\n"
        }
      }
    }
  }
}
`

	// Write config
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Detect IaC tool
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

	// Run init
	t.Log("Running init...")
	cmd := exec.Command(iacTool, "init")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output:\n%s", output)
		t.Fatalf("Init failed: %v", err)
	}

	// Run apply
	t.Log("Running apply...")
	cmd = exec.Command(iacTool, "apply", "-auto-approve")
	cmd.Dir = testDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("Apply output:\n%s", output)
		t.Fatalf("Apply failed: %v", err)
	}
	t.Log("✓ Apply completed successfully")

	// Verify hello.txt was created
	helloPath := filepath.Join(testDir, "output", "hello.txt")
	content, err := os.ReadFile(helloPath)
	require.NoError(t, err)
	assert.Equal(t, "Hello from inline feature!\n", string(content))
	t.Log("✅ Inline feature test successful!")
}

// ====================
// Feature Precedence Tests
// ====================

// TestProjectPrecedenceOverFeatureSuccess tests that project files override feature files
func TestProjectPrecedenceOverFeatureSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestProjectPrecedenceOverFeatureSuccess")

	// Create Terraform config with feature and project override
	config := `
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug = true
}

resource "tofukit_project" "test" {
  name    = "precedence-test"
  version = "0.1.0"

  features = {
    "base_config" = {
      prompt = "Provide base configuration"
      files = {
        "config.txt" = {
          content = "Config from feature\n"
        }
      }
    }
  }

  files = {
    "config.txt" = {
      content = "Config from project\n"
    }
  }
}
`

	// Write config
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Detect IaC tool
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

	// Run init
	t.Log("Running init...")
	cmd := exec.Command(iacTool, "init")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output:\n%s", output)
		t.Fatalf("Init failed: %v", err)
	}

	// Run apply
	t.Log("Running apply...")
	cmd = exec.Command(iacTool, "apply", "-auto-approve")
	cmd.Dir = testDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("Apply output:\n%s", output)
		t.Fatalf("Apply failed: %v", err)
	}
	t.Log("✓ Apply completed successfully")

	// Verify project file overrode feature file
	configPath = filepath.Join(testDir, "output", "config.txt")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "Config from project\n", string(content), "Project file should override feature file")
	t.Log("✅ Project precedence over feature verified!")
}

// TestFeatureMergeMultipleFeaturesSuccess tests merging files from multiple features
// SKIPPED: Inline features with prompt/requirements processing is not yet fully implemented
// Stack-level features work, but project-level inline features don't collect files yet
func TestFeatureMergeMultipleFeaturesSuccess(t *testing.T) {
	t.Skip("Inline feature file collection not yet implemented - only static content in project files works")
	testDir := createTestDirectory(t, "TestFeatureMergeMultipleFeaturesSuccess")

	// Create Terraform config with multiple features
	config := `
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug = true
}

resource "tofukit_project" "test" {
  name    = "multi-feature-test"
  version = "0.1.0"

  features = {
    "logging" = {
      prompt = "Add logging capability"
      files = {
        "log.txt" = {
          content = "Logging enabled\n"
        }
      }
    }
    "config" = {
      prompt = "Add configuration capability"
      files = {
        "config.txt" = {
          content = "Config loaded\n"
        }
      }
    }
    "metrics" = {
      prompt = "Add metrics capability"
      files = {
        "metrics.txt" = {
          content = "Metrics tracking\n"
        }
      }
    }
  }
}
`

	// Write config
	configPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Detect IaC tool
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

	// Run init
	t.Log("Running init...")
	cmd := exec.Command(iacTool, "init")
	cmd.Dir = testDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output:\n%s", output)
		t.Fatalf("Init failed: %v", err)
	}

	// Run apply
	t.Log("Running apply...")
	cmd = exec.Command(iacTool, "apply", "-auto-approve")
	cmd.Dir = testDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("Apply output:\n%s", output)
		t.Fatalf("Apply failed: %v", err)
	}
	t.Log("✓ Apply completed successfully")

	// Verify all feature files were created
	logPath := filepath.Join(testDir, "output", "log.txt")
	logContent, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Equal(t, "Logging enabled\n", string(logContent))

	configPath = filepath.Join(testDir, "output", "config.txt")
	configContent, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "Config loaded\n", string(configContent))

	metricsPath := filepath.Join(testDir, "output", "metrics.txt")
	metricsContent, err := os.ReadFile(metricsPath)
	require.NoError(t, err)
	assert.Equal(t, "Metrics tracking\n", string(metricsContent))
	t.Log("✅ Multiple features merged successfully!")
}

// ====================
// Stack-Feature Integration Tests
// ====================

// TestFeatureFilesIndependent tests that feature resources provide files to stacks
// SKIPPED: Feature files from stack features are not being collected properly yet
func TestFeatureFilesIndependent(t *testing.T) {
	t.Skip("Feature file collection from stack features not yet implemented")
	testDir := createTestDirectory(t, "TestFeatureFilesIndependent")

	// Create a minimal test with inline feature and stack
	projectContent := `
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  debug                 = true
  claude_home_directory = "~/.claude"
}

# Define a simple feature with a test file
resource "tofukit_feature" "test_feature" {
  name = "test-feature"

  requirements = [
    {
      name = "Test File"
      instructions = [
        {
          prompt = "Add a simple test file"
        }
      ]
    }
  ]

  files = {
    "test.txt" = {
      content = "Hello from feature\n"
    }
  }
}

# Define a stack that uses the feature
resource "tofukit_stack" "test_stack" {
  name        = "test-stack"
  description = "Test stack with feature"

  features = [
    tofukit_feature.test_feature
  ]

  files = {}
}

# Project uses the stack
resource "tofukit_project" "test" {
  name        = "test-feature-files"
  description = "Test feature files"
  version     = "1.0.0"
  model       = "haiku"

  stack = tofukit_stack.test_stack
}
`

	projectPath := filepath.Join(testDir, "project.tofu")
	err := os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	t.Log("Testing independent feature files...")

	// Determine which IaC tool to use
	iacTool := "tofu"
	if _, err := exec.LookPath("tofu"); err != nil {
		iacTool = "terraform"
	}
	t.Logf("Using %s", iacTool)

	// Initialize
	t.Log("Running init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "Init failed: %s", string(initOutput))

	// Apply
	t.Log("Running apply...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Apply failed: %s", string(applyOutput))

	// Check debug output
	outputDir := filepath.Join(testDir, "output")
	debugDir := filepath.Join(outputDir, ".debug")
	require.DirExists(t, debugDir, "Debug directory should exist")

	// Find and read project JSON
	debugFiles, err := filepath.Glob(filepath.Join(debugDir, "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have project JSON file")

	projectJSONPath := debugFiles[0]
	t.Logf("Reading project JSON: %s", projectJSONPath)

	jsonData, err := os.ReadFile(projectJSONPath)
	require.NoError(t, err, "Failed to read project JSON")

	var projectJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &projectJSON)
	require.NoError(t, err, "Failed to parse project JSON")

	// Verify the feature file is present
	specification, ok := projectJSON["specification"].(map[string]interface{})
	require.True(t, ok, "specification should be a map")
	files, ok := specification["files"].(map[string]interface{})
	require.True(t, ok, "files should be a map")

	t.Logf("Files in project JSON: %d", len(files))
	for path := range files {
		t.Logf("  - %s", path)
	}

	if len(files) == 0 {
		t.Log("⚠️  WARNING: No files found in project JSON!")
		t.Log("This indicates that stack features are not being collected properly")

		// Print the full JSON for debugging
		prettyJSON, _ := json.MarshalIndent(projectJSON, "", "  ")
		t.Logf("Full project JSON:\n%s", string(prettyJSON))

		t.Fatal("Expected files from feature, but got none")
	}

	assert.Contains(t, files, "test.txt", "Should have test.txt from feature")

	// Verify the content is correct
	testFile := files["test.txt"].(map[string]interface{})
	content, ok := testFile["content"].(string)
	require.True(t, ok, "content should be a string")
	assert.Equal(t, "Hello from feature\n", content, "Content should match")

	t.Log("✅ Independent feature files test completed successfully!")
}

// TestStackFeaturesFromModule tests that features from a stack module are collected
// SKIPPED: Stack features from modules don't have their files collected properly yet
func TestStackFeaturesFromModule(t *testing.T) {
	t.Skip("Feature file collection from module stacks not yet implemented")
	testDir := createTestDirectory(t, "TestStackFeaturesFromModule")

	// Get project root and copy stacks
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	stacksDir := filepath.Join(projectRoot, "examples", "stacks")
	testStacksDir := filepath.Join(testDir, "stacks")

	copyCmd := exec.Command("cp", "-r", stacksDir, testStacksDir)
	err = copyCmd.Run()
	require.NoError(t, err, "Failed to copy stacks")

	// Create test project that uses the Python Click stack
	projectContent := `
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  output_path           = "output"
  debug                 = true
  claude_home_directory = "~/.claude"
}

locals {
  project_name    = "test-stack-features"
  project_version = "1.0.0"
}

module "click_app" {
  source = "./stacks/tofukit-stack-python-click-app"

  project_name    = local.project_name
  project_version = local.project_version
}

resource "tofukit_project" "test" {
  name        = local.project_name
  description = "Test project for stack features"
  version     = local.project_version
  model       = "haiku"

  stack = module.click_app.stack
}
`

	projectPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(projectPath, []byte(projectContent), 0644)
	require.NoError(t, err)

	t.Log("Testing stack features from module...")

	// Determine which IaC tool to use
	iacTool := "tofu"
	if _, err := exec.LookPath("tofu"); err != nil {
		iacTool = "terraform"
	}

	// Init and apply
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	require.NoError(t, err, "Init failed: %s", string(initOutput))

	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	if err != nil {
		t.Logf("Apply output:\n%s", string(applyOutput))
	}
	require.NoError(t, err, "Apply failed")

	// Read project JSON
	debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles)

	jsonData, err := os.ReadFile(debugFiles[0])
	require.NoError(t, err)

	var projectJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &projectJSON)
	require.NoError(t, err)

	specification, ok := projectJSON["specification"].(map[string]interface{})
	require.True(t, ok, "specification should be a map")
	files, ok := specification["files"].(map[string]interface{})
	require.True(t, ok, "files should be a map")

	t.Logf("Files in project JSON: %d", len(files))
	for path := range files {
		t.Logf("  - %s", path)
	}

	if len(files) == 0 {
		t.Log("⚠️  WARNING: No files found - stack features not collected!")
		prettyJSON, _ := json.MarshalIndent(projectJSON, "", "  ")
		t.Logf("Full project JSON:\n%s", string(prettyJSON))
		t.Fatal("Expected files from stack features")
	}

	// Verify expected files from features
	expectedFiles := []string{
		"src/__init__.py",    // from version_command feature
		"src/__version__.py", // from version_command feature
		"README.md",          // from readme feature
	}

	for _, expectedFile := range expectedFiles {
		assert.Contains(t, files, expectedFile, "Should have %s from stack features", expectedFile)
	}

	t.Log("✅ Stack features from module test completed successfully!")
}
