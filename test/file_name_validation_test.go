package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileNameValidation_ValidNames tests that valid file names are accepted
func TestFileNameValidation_ValidNames(t *testing.T) {
	validNames := []string{
		"file.txt",
		"README.md",
		".gitignore",
		"src/main.go",
		"my-file_v2.txt",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			testDir := createTestDirectory(t, "TestFileNameValidation_Valid")

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
			iacTool := getIacTool(t)

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
				t.Fatalf("Validate should succeed for valid name %q", name)
			}

			assert.Contains(t, string(validateOutput), "Success", "Validation should succeed for %q", name)
		})
	}
}

// TestFileNameValidation_InvalidNames tests that invalid file names are rejected
func TestFileNameValidation_InvalidNames(t *testing.T) {
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
			testDir := createTestDirectory(t, "TestFileNameValidation_Invalid")

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
			iacTool := getIacTool(t)

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

func getIacTool(t *testing.T) string {
	if _, err := exec.LookPath("tofu"); err == nil {
		return "tofu"
	} else if _, err := exec.LookPath("terraform"); err == nil {
		return "terraform"
	} else {
		t.Skip("Neither terraform nor tofu available")
		return ""
	}
}
