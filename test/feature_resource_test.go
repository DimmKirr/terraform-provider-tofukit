package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
