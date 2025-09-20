package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/tofukit/opentofu-provider-tofukit/internal/provider"
	"github.com/tofukit/opentofu-provider-tofukit/internal/testutil"
)

// TestAccProjectResource_basic tests basic project resource functionality
func TestAccProjectResource_basic(t *testing.T) {
	tmpDir := t.TempDir()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccProjectResourceConfig_basic(tmpDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "name", "test-hello-world"),
					resource.TestCheckResourceAttr("tofukit_project.test", "description", "A simple hello world project"),
					resource.TestCheckResourceAttr("tofukit_project.test", "version", "1.0.0"),
					resource.TestCheckResourceAttr("tofukit_project.test", "execution_status", "dry_run"),
					resource.TestCheckResourceAttrSet("tofukit_project.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "tofukit_project.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccProjectResource_withRequirements tests project with requirements
func TestAccProjectResource_withRequirements(t *testing.T) {
	tmpDir := t.TempDir()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceConfig_withRequirements(tmpDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "name", "test-with-reqs"),
					resource.TestCheckResourceAttr("tofukit_project.test", "requirement.#", "2"),
					resource.TestCheckResourceAttr("tofukit_project.test", "requirement.0.name", "Create hello.txt"),
					resource.TestCheckResourceAttr("tofukit_project.test", "requirement.0.instructions.#", "2"),
					resource.TestCheckResourceAttr("tofukit_project.test", "requirement.1.name", "Create README.md"),
					resource.TestCheckResourceAttr("tofukit_project.test", "execution_status", "dry_run"),
				),
			},
		},
	})
}

// TestAccProjectResource_withExecution tests actual execution (not dry run)
func TestAccProjectResource_withExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping execution test in short mode")
	}

	testutil.SkipIfNoClaudeCredentials(t)
	tmpDir := t.TempDir()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceConfig_withExecution(tmpDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "name", "test-execution"),
					resource.TestMatchResourceAttr("tofukit_project.test", "execution_status",
						regexp.MustCompile("^(completed|failed)$")),
					resource.TestCheckResourceAttrSet("tofukit_project.test", "execution_started"),
					testAccCheckProjectGenerated("tofukit_project.test"),
				),
			},
		},
	})
}

// TestAccProjectResource_update tests project updates
func TestAccProjectResource_update(t *testing.T) {
	tmpDir := t.TempDir()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create initial project
			{
				Config: testAccProjectResourceConfig_basic(tmpDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "description", "A simple hello world project"),
					resource.TestCheckResourceAttr("tofukit_project.test", "version", "1.0.0"),
				),
			},
			// Update project
			{
				Config: testAccProjectResourceConfig_updated(tmpDir),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "description", "Updated hello world project"),
					resource.TestCheckResourceAttr("tofukit_project.test", "version", "1.1.0"),
				),
			},
		},
	})
}

// TestAccProjectResource_validationErrors tests validation errors
func TestAccProjectResource_validationErrors(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProjectResourceConfig_invalidName(),
				ExpectError: regexp.MustCompile("Invalid project name"),
			},
		},
	})
}

// TestAccProjectResource_disappears tests resource disappears scenario
func TestAccProjectResource_disappears(t *testing.T) {
	tmpDir := t.TempDir()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectResourceConfig_basic(tmpDir),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProjectExists("tofukit_project.test"),
					testAccCheckProjectDisappears("tofukit_project.test"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// Helper functions for acceptance tests

func testAccPreCheck(t *testing.T) {
	// Add any pre-check requirements
	if v := os.Getenv("TOFUKIT_ACC"); v == "" {
		t.Skip("TOFUKIT_ACC must be set for acceptance tests")
	}
}

var testAccProtoV6ProviderFactories = map[string]func() (terraform.ResourceProvider, error){
	"tofukit": func() (terraform.ResourceProvider, error) {
		return provider.New("test")(), nil
	},
}

func testAccProjectResourceConfig_basic(outputDir string) string {
	return fmt.Sprintf(`
provider "tofukit" {
  dry_run     = true
  output_path = %q
}

resource "tofukit_project" "test" {
  name        = "test-hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  requirement {
    name = "Create hello.txt"
    instructions = [
      "Create a file named hello.txt",
      "Add content: Hello, World!"
    ]
    verification {
      command = "test -f hello.txt"
      expect  = "success"
    }
  }
}
`, outputDir)
}

func testAccProjectResourceConfig_withRequirements(outputDir string) string {
	return fmt.Sprintf(`
provider "tofukit" {
  dry_run     = true
  output_path = %q
}

resource "tofukit_project" "test" {
  name        = "test-with-reqs"
  description = "Project with multiple requirements"
  version     = "1.0.0"

  requirement {
    name     = "Create hello.txt"
    priority = 100
    instructions = [
      "Create a file named hello.txt",
      "Add content: Hello, World!"
    ]
    verification {
      command = "test -f hello.txt"
      expect  = "success"
    }
  }

  requirement {
    name     = "Create README.md"
    priority = 90
    instructions = [
      "Create a README.md file",
      "Add project description"
    ]
    verification {
      command = "test -f README.md"
      expect  = "success"
    }
  }
}
`, outputDir)
}

func testAccProjectResourceConfig_withExecution(outputDir string) string {
	return fmt.Sprintf(`
provider "tofukit" {
  dry_run     = false
  output_path = %q
}

resource "tofukit_project" "test" {
  name        = "test-execution"
  description = "Project for testing actual execution"
  version     = "1.0.0"

  requirement {
    name = "Create hello.txt"
    instructions = [
      "Create a file named hello.txt",
      "Add content: Hello from Claude!"
    ]
    verification {
      command = "test -f hello.txt && grep -q 'Hello from Claude!' hello.txt"
      expect  = "success"
    }
  }
}
`, outputDir)
}

func testAccProjectResourceConfig_updated(outputDir string) string {
	return fmt.Sprintf(`
provider "tofukit" {
  dry_run     = true
  output_path = %q
}

resource "tofukit_project" "test" {
  name        = "test-hello-world"
  description = "Updated hello world project"
  version     = "1.1.0"

  requirement {
    name = "Create hello.txt"
    instructions = [
      "Create a file named hello.txt",
      "Add content: Hello, Updated World!"
    ]
    verification {
      command = "test -f hello.txt"
      expect  = "success"
    }
  }
}
`, outputDir)
}

func testAccProjectResourceConfig_invalidName() string {
	return `
provider "tofukit" {
  dry_run = true
}

resource "tofukit_project" "test" {
  name        = ""
  description = "Invalid project"
  version     = "1.0.0"
}
`
}

// Check functions

func testAccCheckProjectExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("Resource ID not set")
		}

		// Add any additional existence checks here
		return nil
	}
}

func testAccCheckProjectGenerated(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource not found: %s", resourceName)
		}

		projectPath := rs.Primary.Attributes["project_path"]
		if projectPath == "" {
			return fmt.Errorf("Project path not set")
		}

		// Check if project directory exists
		if _, err := os.Stat(projectPath); os.IsNotExist(err) {
			return fmt.Errorf("Project directory does not exist: %s", projectPath)
		}

		return nil
	}
}

func testAccCheckProjectDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Resource not found: %s", resourceName)
		}

		projectPath := rs.Primary.Attributes["project_path"]
		if projectPath != "" {
			// Simulate the project being removed externally
			os.RemoveAll(projectPath)
		}

		return nil
	}
}

func testAccCheckProjectDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tofukit_project" {
			continue
		}

		projectPath := rs.Primary.Attributes["project_path"]
		if projectPath != "" {
			if _, err := os.Stat(projectPath); err == nil {
				return fmt.Errorf("Project still exists: %s", projectPath)
			}
		}
	}

	return nil
}