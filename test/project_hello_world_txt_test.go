package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectHelloWorldTxt(t *testing.T) {
	// Create unique test directory in test-output
	testDir := createTestDirectory(t, "TestProjectHelloWorldTxt")

	// Get project root
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	// Step 1: Build the provider
	t.Log("Building provider...")
	buildCmd := exec.Command("make", "install")
	buildCmd.Dir = projectRoot
	output, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build provider: %s", output)
	t.Log("✓ Provider built successfully")

	// Step 2: Generate main.tofu with provider configuration
	// Set output_path to "output" directory within test directory
	mainTofuContent := `# Terraform configuration for hello-world-txt test
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration
provider "tofukit" {
  output_format         = "json"
  output_path           = "output"  # Output will be in {test_directory}/output/
  dry_run               = false     # Set to false to execute Claude Code
  claude_home_directory = "~/.claude"
}
`
	err = os.WriteFile(filepath.Join(testDir, "main.tofu"), []byte(mainTofuContent), 0644)
	require.NoError(t, err, "Failed to write main.tofu")

	// Step 3: Generate project.tofu from hello-world-txt example
	projectTofuContent := `resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  # Single file scaffold - no dependencies needed
  scaffold {
    path = "hello.txt"
    content = <<-EOF
hello world
EOF
  }

  scaffold {
    path = "hello2.txt"
    content = <<-EOF
hello world2
EOF
  }

  scaffold {
    path = "hello3.txt"
    content = <<-EOF
hello world2
EOF
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 4: Check if terraform/tofu is available, skip terratest if not
	var iacTool string
	if _, err := exec.LookPath("tofu"); err == nil {
		iacTool = "tofu"
		t.Log("Using OpenTofu")
	} else if _, err := exec.LookPath("terraform"); err == nil {
		iacTool = "terraform"
		t.Log("Using Terraform")
	} else {
		t.Skip("Neither terraform nor tofu available - skipping terratest portion")
	}

	// Configure Terraform options with auto-approve
	terraformOptions := &terraform.Options{
		TerraformDir:    testDir,
		TerraformBinary: iacTool,
		NoColor:         true,
		// Auto-approve for apply and destroy
		PlanFilePath: "",
		Upgrade:      false,
	}

	// Clean up resources at the end
	defer func() {
		if os.Getenv("SKIP_DESTROY") != "true" {
			// Destroy also uses auto-approve by default in terratest
			terraform.Destroy(t, terraformOptions)
		}
	}()

	// Step 5: Run tofu/terraform init using terratest
	t.Log("Running tofu init...")
	terraform.Init(t, terraformOptions)
	t.Log("✓ Init completed successfully")

	// Step 6: Run tofu/terraform apply with auto-approve using terratest
	t.Log("Running tofu apply --auto-approve...")
	// terraform.Apply in terratest automatically uses -auto-approve
	terraform.Apply(t, terraformOptions)
	t.Log("✓ Apply completed successfully")

	// Step 7: Verify scaffolds were created
	t.Log("Verifying scaffold files...")
	// Files should be in {test_directory}/output/hello-world as specified in provider output_path
	projectPath := filepath.Join(testDir, "output", "hello-world")

	// Assert hello.txt exists and has correct content
	helloPath := filepath.Join(projectPath, "hello.txt")
	assert.FileExists(t, helloPath, "hello.txt should exist")
	verifyFileContent(t, helloPath, "hello world")

	// Assert hello2.txt exists and has correct content
	hello2Path := filepath.Join(projectPath, "hello2.txt")
	assert.FileExists(t, hello2Path, "hello2.txt should exist")
	verifyFileContent(t, hello2Path, "hello world2")

	// Assert hello3.txt exists and has correct content
	hello3Path := filepath.Join(projectPath, "hello3.txt")
	assert.FileExists(t, hello3Path, "hello3.txt should exist")
	verifyFileContent(t, hello3Path, "hello world2")

	// Step 8: Test scaffold removal
	t.Log("Testing scaffold removal...")

	// Update project.tofu to remove hello3.txt
	updatedProjectContent := `resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  scaffold {
    path = "hello.txt"
    content = <<-EOF
hello world updated
EOF
  }

  scaffold {
    path = "hello2.txt"
    content = <<-EOF
hello world2
EOF
  }

  # hello3.txt removed
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(updatedProjectContent), 0644)
	require.NoError(t, err, "Failed to update project.tofu")

	// Apply the changes with auto-approve
	t.Log("Running tofu apply --auto-approve for scaffold removal...")
	terraform.Apply(t, terraformOptions)
	t.Log("✓ Apply completed for scaffold removal")

	// Verify hello3.txt was removed
	assert.NoFileExists(t, hello3Path, "hello3.txt should have been removed")
	t.Log("✓ hello3.txt successfully removed")

	// Verify hello.txt was updated
	verifyFileContent(t, helloPath, "hello world updated")
	t.Log("✓ hello.txt successfully updated")

	// Verify hello2.txt still exists
	assert.FileExists(t, hello2Path, "hello2.txt should still exist")
	t.Log("✓ hello2.txt still exists")

	// Step 9: Test adding a new scaffold
	t.Log("Testing adding new scaffold...")

	finalProjectContent := `resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  scaffold {
    path = "hello.txt"
    content = <<-EOF
hello world updated
EOF
  }

  scaffold {
    path = "hello2.txt"
    content = <<-EOF
hello world2
EOF
  }

  scaffold {
    path = "newdir/hello4.txt"
    content = <<-EOF
hello world4 new file
EOF
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(finalProjectContent), 0644)
	require.NoError(t, err, "Failed to update project.tofu for new file")

	// Apply the changes with auto-approve
	t.Log("Running tofu apply --auto-approve for adding new scaffold...")
	terraform.Apply(t, terraformOptions)
	t.Log("✓ Apply completed for adding new scaffold")

	// Verify new file was created
	hello4Path := filepath.Join(projectPath, "newdir", "hello4.txt")
	assert.FileExists(t, hello4Path, "hello4.txt should be created")
	verifyFileContent(t, hello4Path, "hello world4 new file")
	t.Log("✓ hello4.txt successfully created")

	t.Log("✅ Test completed successfully!")
}