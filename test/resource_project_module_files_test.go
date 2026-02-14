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

// TestTFK17_ModuleFilesResourceRegistry validates that resource_registry is populated
// when files come from module outputs (e.g., module.ui.files.readme)
//
// TFK-17: FileModel struct missing `model` field causes resource_registry to be empty
// This test reproduces the issue where files from modules don't get their URIs
// extracted, causing empty resource_registry in Claude prompts.
func TestTFK17_ModuleFilesResourceRegistry(t *testing.T) {
	testDir := createTestDirectory(t, "TestTFK17_ModuleFilesResourceRegistry")
	t.Log("Testing TFK-17: Module files resource_registry...")

	// Create module directory
	moduleDir := filepath.Join(testDir, "files_module")
	require.NoError(t, os.MkdirAll(moduleDir, 0755))

	// Module: defines tofukit_file resources
	moduleMain := `
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

resource "tofukit_file" "readme" {
  name = "readme"
  path = "README.md"

  instructions = [{
    prompt = "Create a simple README that references tofukit://file/story"
    constraints = ["Keep it under 100 characters"]
  }]
}

resource "tofukit_file" "story" {
  name = "story"
  path = "STORY.md"

  instructions = [{
    prompt = "First sentence of Orwell's Animal Farm"
    constraints = ["Use all caps"]
  }]
}
`

	// Module outputs
	moduleOutputs := `
output "files" {
  description = "Map of file resources"
  value = {
    readme = tofukit_file.readme
    story  = tofukit_file.story
  }
}
`

	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "main.tofu"), []byte(moduleMain), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "outputs.tofu"), []byte(moduleOutputs), 0644))

	// Root config: references module files
	rootConfig := `
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
}

module "files" {
  source = "./files_module"
}

resource "tofukit_project" "test" {
  name        = "module-files-test"
  description = "Test that module files populate resource_registry"
  version     = "1.0.0"

  # Reference files from module output (TFK-17 reproduction)
  files = {
    "README.md" = module.files.files.readme
    "STORY.md"  = module.files.files.story
  }
}
`

	require.NoError(t, os.WriteFile(filepath.Join(testDir, "main.tofu"), []byte(rootConfig), 0644))

	// Run terraform
	iacTool := detectIaCTool(t)

	// Init
	t.Log("Running init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")

	// Apply
	t.Log("Running apply...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	if err != nil {
		t.Logf("Apply output: %s", applyOutput)
	}
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)

	// Check debug files for resource_registry
	debugDir := filepath.Join(testDir, "output", ".debug")
	debugFiles, err := os.ReadDir(debugDir)
	require.NoError(t, err, "Debug directory should exist")

	var foundRegistry bool
	var registryContent string

	for _, df := range debugFiles {
		if strings.HasPrefix(df.Name(), "claude-prompt-") && strings.HasSuffix(df.Name(), ".json") {
			debugPath := filepath.Join(debugDir, df.Name())
			content, err := os.ReadFile(debugPath)
			require.NoError(t, err)

			contentStr := string(content)

			// Check if resource_registry exists and has content
			if strings.Contains(contentStr, `"resource_registry"`) {
				foundRegistry = true

				// Check if it has actual entries (not just empty {})
				if strings.Contains(contentStr, `"tofukit://file/`) {
					registryContent = "populated"
					t.Log("✓ resource_registry found with tofukit:// URIs")
				} else {
					registryContent = "empty"
					t.Log("✗ resource_registry exists but has no tofukit:// entries")
				}
			}

			// Log a snippet for debugging
			if len(contentStr) > 1000 {
				t.Logf("Debug file %s: %d bytes", df.Name(), len(contentStr))
			}
		}
	}

	// Assertions
	assert.True(t, foundRegistry, "resource_registry should be present in Claude prompt")
	assert.Equal(t, "populated", registryContent,
		"TFK-17: resource_registry should contain tofukit://file/ URIs when files come from modules")

	// Verify output files exist
	assert.FileExists(t, filepath.Join(testDir, "output", "README.md"))
	assert.FileExists(t, filepath.Join(testDir, "output", "STORY.md"))

	t.Log("✓ TFK-17 test completed")
}

// TestTFK21_KitcutModuleStructure tests the exact kitcut directory structure
// where files are declared in submodules and provider is in root.
// This reproduces TFK-21 module path issue.
//
// Structure (exact match of .context/kitcut/.tofukit/):
//
//	.tofukit/
//	├── providers.tofu      # provider "tofukit" { output_path = "../" }
//	├── main.tofu           # tofukit_project referencing module.ui.files.*
//	├── core.tofu           # module "ui" { source = "./ui" }
//	└── ui/                 # Submodule for UI files
//	    ├── terraform.tofu  # required_providers
//	    ├── pages.tofu      # TEXT files (HTML) with model = "anthropic/claude-haiku"
//	    ├── images.tofu     # IMAGE files with model = "openai/gpt-image-1"
//	    └── outputs.tofu    # Exports files map
func TestTFK21_KitcutModuleStructure(t *testing.T) {
	testDir := createTestDirectory(t, "TestTFK21_KitcutModuleStructure")
	t.Log("Testing TFK-21: Kitcut module structure path resolution...")

	// Create .tofukit directory structure (exact kitcut mirror)
	tofukitDir := filepath.Join(testDir, ".tofukit")
	uiDir := filepath.Join(tofukitDir, "ui")
	require.NoError(t, os.MkdirAll(uiDir, 0755))

	// .tofukit/providers.tofu - Provider config
	providersConfig := `
provider "tofukit" {
  output_path   = "../"
  debug         = true
  image_mode    = "wireframe"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tofukitDir, "providers.tofu"), []byte(providersConfig), 0644))

	// .tofukit/ui/terraform.tofu - Required providers for submodule
	uiTerraform := `
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "~> 0.1.0"
    }
  }
}
`
	require.NoError(t, os.WriteFile(filepath.Join(uiDir, "terraform.tofu"), []byte(uiTerraform), 0644))

	// .tofukit/ui/pages.tofu - TEXT file (HTML) with Claude model
	uiPages := `
resource "tofukit_file" "ui_page_home" {
  name = "ui_page_home"
  path = "index.html"

  instructions = [{
    prompt = "Generate minimal HTML5: doctype, html, head with title 'TFK-21 Test', body with h1 'Module Path Test'."
    constraints = ["Valid HTML5", "Under 15 lines"]
  }]

  model = "anthropic/claude-haiku"
  description = "Landing page from submodule"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(uiDir, "pages.tofu"), []byte(uiPages), 0644))

	// .tofukit/ui/images.tofu - IMAGE file with OpenAI model
	uiImages := `
resource "tofukit_file" "ui_image_logo" {
  name = "ui_image_logo"
  path = "_next/static/media/logo.png"

  image = {
    quality = "low"
    size    = "64x64"
  }

  instructions = [{
    prompt = "Create a simple blue square icon."
    constraints = ["Minimal design"]
  }]

  model = "openai/gpt-image-1"
  description = "Logo from submodule"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(uiDir, "images.tofu"), []byte(uiImages), 0644))

	// .tofukit/ui/outputs.tofu - Export files map
	uiOutputs := `
output "files" {
  value = {
    ui_page_home  = tofukit_file.ui_page_home
    ui_image_logo = tofukit_file.ui_image_logo
  }
}
`
	require.NoError(t, os.WriteFile(filepath.Join(uiDir, "outputs.tofu"), []byte(uiOutputs), 0644))

	// .tofukit/core.tofu - Module import
	coreConfig := `
module "ui" {
  source = "./ui"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tofukitDir, "core.tofu"), []byte(coreConfig), 0644))

	// .tofukit/main.tofu - Project resource referencing module files
	mainConfig := `
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "~> 0.1.0"
    }
  }
}

resource "tofukit_project" "website" {
  name        = "tfk21-kitcut-structure"
  description = "Test kitcut module structure"
  version     = "1.0.0"

  model = "anthropic/claude-haiku"

  # Files come from submodule - this is the TFK-21 scenario
  files = {
    "index.html"                  = module.ui.files.ui_page_home
    "_next/static/media/logo.png" = module.ui.files.ui_image_logo
  }
}
`
	require.NoError(t, os.WriteFile(filepath.Join(tofukitDir, "main.tofu"), []byte(mainConfig), 0644))

	// Run from .tofukit directory (like kitcut does)
	iacTool := detectIaCTool(t)

	// Init
	t.Log("Running init from .tofukit/...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = tofukitDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")

	// Apply
	t.Log("Running apply from .tofukit/...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color")
	applyCmd.Dir = tofukitDir
	applyOutput, err := applyCmd.CombinedOutput()
	t.Logf("Apply output: %s", applyOutput)
	require.NoError(t, err, "Failed to run apply")

	// Check output files - should be in testDir/ (relative to .tofukit/../)
	outputDir := testDir

	// 1. Check HTML file (TEXT - TFK-21 main issue)
	htmlPath := filepath.Join(outputDir, "index.html")
	htmlExists := false
	if _, err := os.Stat(htmlPath); err == nil {
		content, _ := os.ReadFile(htmlPath)
		if len(content) > 0 {
			htmlExists = true
			t.Logf("✓ TEXT file exists: index.html (%d bytes)", len(content))
		} else {
			t.Log("✗ TEXT file EXISTS but EMPTY: index.html")
		}
	} else {
		t.Log("✗ TEXT file MISSING: index.html")
	}

	// 2. Check IMAGE file (should work per TFK-21)
	imagePath := filepath.Join(outputDir, "_next", "static", "media", "logo.png")
	imageExists := false
	if _, err := os.Stat(imagePath); err == nil {
		imageExists = true
		t.Logf("✓ IMAGE file exists: _next/static/media/logo.png")
	} else {
		t.Log("✗ IMAGE file MISSING: _next/static/media/logo.png")
	}

	// TFK-21 Bug: Image works but text doesn't
	if imageExists && !htmlExists {
		t.Fatalf("TFK-21 BUG REPRODUCED: Image from submodule works, but text file skipped!")
	}

	// Also check if files were created in wrong location (.tofukit/ instead of ../)
	wrongHtmlPath := filepath.Join(tofukitDir, "index.html")
	if _, err := os.Stat(wrongHtmlPath); err == nil {
		t.Error("BUG: index.html created in .tofukit/ instead of ../")
	}

	require.True(t, htmlExists, "TFK-21: Text file should be created from submodule")
	require.True(t, imageExists, "Image file should be created from submodule")

	t.Log("✓ TFK-21 kitcut module structure test passed")
}
