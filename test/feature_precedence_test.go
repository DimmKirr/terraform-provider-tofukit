package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
func TestFeatureMergeMultipleFeaturesSuccess(t *testing.T) {
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
