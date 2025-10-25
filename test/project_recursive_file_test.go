package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProjectRecursiveFileCreateSuccess tests creating an initial nested file
func TestProjectRecursiveFileCreateSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectRecursiveFileCreateSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	// Step 1: Generate single project.tofu with all configuration
	projectTofuContent := `terraform {
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

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }
}
`
	err := os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(projectTofuContent), 0644)
	require.NoError(t, err, "Failed to write project.tofu")

	// Step 2: Check if terraform/tofu is available
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

	// Step 4: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 5: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 6: Run apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Step 7: Verify the nested file was created
	projectPath := filepath.Join(testDir, "output")
	nestedFilePath := filepath.Join(projectPath, "demo", "hello.txt")

	assert.DirExists(t, filepath.Join(projectPath, "demo"), "demo directory should exist")
	assert.FileExists(t, nestedFilePath, "demo/hello.txt should exist")
	verifyFileContent(t, nestedFilePath, "hello from demo\n")

	t.Log("✅ Initial nested file created successfully!")
}

// TestProjectRecursiveFileAddSuccess tests adding another nested file
func TestProjectRecursiveFileAddSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectRecursiveFileAddSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error
	projectPath := filepath.Join(testDir, "output")

	// Step 1: Generate initial project.tofu with ONE nested file
	initialProjectContent := `terraform {
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

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(initialProjectContent), 0644)
	require.NoError(t, err, "Failed to write initial project.tofu")

	// Step 2: Check if terraform/tofu is available
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

	// Step 3: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run initial apply to create first file
	t.Log("Running initial tofu apply (creating first file)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run initial apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	// Step 5: Verify first file exists
	file1Path := filepath.Join(projectPath, "demo", "hello.txt")
	assert.FileExists(t, file1Path, "demo/hello.txt should exist after initial apply")
	verifyFileContent(t, file1Path, "hello from demo\n")
	t.Log("✓ First file verified")

	// Step 6: Update project.tofu to ADD second file
	t.Log("Updating project.tofu to add second file...")
	updatedProjectContent := `terraform {
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

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }

  file {
    path    = "demo/hello2.txt"
    content = "hello2 from demo\n"
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(updatedProjectContent), 0644)
	require.NoError(t, err, "Failed to update project.tofu")

	// Step 7: Run apply to add the second file
	t.Log("Running tofu apply to add second file...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Step 8: Verify both files exist
	file2Path := filepath.Join(projectPath, "demo", "hello2.txt")

	assert.FileExists(t, file1Path, "demo/hello.txt should still exist")
	assert.FileExists(t, file2Path, "demo/hello2.txt should exist")
	verifyFileContent(t, file1Path, "hello from demo\n")
	verifyFileContent(t, file2Path, "hello2 from demo\n")

	t.Log("✅ Additional nested file added successfully!")
}

// TestProjectRecursiveFileDeeperNestingSuccess tests creating deeply nested directories
func TestProjectRecursiveFileDeeperNestingSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectRecursiveFileDeeperNestingSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error
	projectPath := filepath.Join(testDir, "output")

	// Step 1: Generate initial project.tofu with TWO files in demo/
	initialProjectContent := `terraform {
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

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }

  file {
    path    = "demo/hello2.txt"
    content = "hello2 from demo\n"
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(initialProjectContent), 0644)
	require.NoError(t, err, "Failed to write initial project.tofu")

	// Step 2: Check if terraform/tofu is available
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

	// Step 3: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run initial apply to create two files
	t.Log("Running initial tofu apply (creating two files)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run initial apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	// Step 5: Verify first two files exist
	file1Path := filepath.Join(projectPath, "demo", "hello.txt")
	file2Path := filepath.Join(projectPath, "demo", "hello2.txt")
	assert.FileExists(t, file1Path, "demo/hello.txt should exist after initial apply")
	assert.FileExists(t, file2Path, "demo/hello2.txt should exist after initial apply")
	t.Log("✓ Initial files verified")

	// Step 6: Update project.tofu to ADD deeply nested file
	t.Log("Updating project.tofu to add deeply nested file...")
	updatedProjectContent := `terraform {
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

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }

  file {
    path    = "demo/hello2.txt"
    content = "hello2 from demo\n"
  }

  file {
    path    = "demo/subdir/deep/hello3.txt"
    content = "hello3 from deep\n"
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(updatedProjectContent), 0644)
	require.NoError(t, err, "Failed to update project.tofu")

	// Step 7: Run apply to create the deeply nested file
	t.Log("Running tofu apply to add deeply nested file...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Step 8: Verify all files exist including deeply nested one
	file3Path := filepath.Join(projectPath, "demo", "subdir", "deep", "hello3.txt")

	assert.FileExists(t, file1Path, "demo/hello.txt should still exist")
	assert.FileExists(t, file2Path, "demo/hello2.txt should still exist")
	assert.DirExists(t, filepath.Join(projectPath, "demo", "subdir"), "demo/subdir directory should exist")
	assert.DirExists(t, filepath.Join(projectPath, "demo", "subdir", "deep"), "demo/subdir/deep directory should exist")
	assert.FileExists(t, file3Path, "demo/subdir/deep/hello3.txt should exist")
	verifyFileContent(t, file3Path, "hello3 from deep\n")

	t.Log("✅ Deeper nested file created successfully!")
}

// TestProjectRecursiveFileRemovalSuccess tests removing nested files and cleaning up empty directories
func TestProjectRecursiveFileRemovalSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestProjectRecursiveFileRemovalSuccess")
	t.Log("Skipping CLAUDE_HOME setup (not needed with --dangerously-skip-permissions)")

	var err error
	projectPath := filepath.Join(testDir, "output")

	// Step 1: Generate initial project.tofu with THREE files including deeply nested
	initialProjectContent := `terraform {
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

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }

  file {
    path    = "demo/hello2.txt"
    content = "hello2 from demo\n"
  }

  file {
    path    = "demo/subdir/deep/hello3.txt"
    content = "hello3 from deep\n"
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(initialProjectContent), 0644)
	require.NoError(t, err, "Failed to write initial project.tofu")

	// Step 2: Check if terraform/tofu is available
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

	// Step 3: Run init
	t.Log("Running tofu init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Failed to run init")
	t.Log("✓ Init completed successfully")

	// Step 4: Run initial apply to create all three files
	t.Log("Running initial tofu apply (creating all three files)...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run initial apply: %s", applyOutput)
	t.Log("✓ Initial apply completed successfully")

	// Step 5: Verify all three files exist
	projectPath = filepath.Join(testDir, "output")
	hello1Path := filepath.Join(projectPath, "demo", "hello.txt")
	hello2Path := filepath.Join(projectPath, "demo", "hello2.txt")
	hello3Path := filepath.Join(projectPath, "demo", "subdir", "deep", "hello3.txt")
	demoDir := filepath.Join(projectPath, "demo")
	subdirPath := filepath.Join(projectPath, "demo", "subdir")
	deepPath := filepath.Join(projectPath, "demo", "subdir", "deep")

	assert.FileExists(t, hello1Path, "demo/hello.txt should exist after initial apply")
	assert.FileExists(t, hello2Path, "demo/hello2.txt should exist after initial apply")
	assert.FileExists(t, hello3Path, "demo/subdir/deep/hello3.txt should exist after initial apply")
	assert.DirExists(t, demoDir, "demo directory should exist")
	assert.DirExists(t, subdirPath, "subdir directory should exist")
	assert.DirExists(t, deepPath, "deep directory should exist")
	t.Log("✓ All three files created successfully")

	// Step 6: Update project.tofu to REMOVE hello2.txt and hello3.txt (keep only hello.txt)
	updatedProjectContent := `terraform {
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

resource "tofukit_project" "recursive_test" {
  name        = "recursive-file"
  description = "Test recursive/nested file handling"
  version     = "1.0.0"

  # Only keep the first file - removing hello2.txt and hello3.txt
  file {
    path    = "demo/hello.txt"
    content = "hello from demo\n"
  }
}
`
	err = os.WriteFile(filepath.Join(testDir, "project.tofu"), []byte(updatedProjectContent), 0644)
	require.NoError(t, err, "Failed to write updated project.tofu")
	t.Log("✓ Updated project.tofu to remove two files")

	// Step 7: Run apply to remove the files
	t.Log("Running tofu apply to remove files...")
	applyCmd = exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err = applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply for removal: %s", applyOutput)
	t.Log("✓ Apply completed - files removed")

	// Step 8: Verify only the first file remains
	assert.FileExists(t, hello1Path, "demo/hello.txt should still exist")
	assert.NoFileExists(t, hello2Path, "demo/hello2.txt should be removed")
	assert.NoFileExists(t, hello3Path, "demo/subdir/deep/hello3.txt should be removed")

	// Step 9: Check if empty directories were cleaned up
	assert.NoDirExists(t, deepPath, "Empty deep directory should be removed")
	assert.NoDirExists(t, subdirPath, "Empty subdir directory should be removed")
	assert.DirExists(t, demoDir, "demo directory should still exist (has hello.txt)")

	t.Log("✅ Nested files removed and empty directories cleaned up!")
}
