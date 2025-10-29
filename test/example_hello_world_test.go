package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExampleHelloWorldSuccess validates that the /examples/hello-world example works end-to-end
func TestExampleHelloWorldSuccess(t *testing.T) {
	os.Setenv("TF_LOG", "DEBUG")
	defer os.Unsetenv("TF_LOG")

	testDir := createTestDirectory(t, "TestExampleHelloWorldSuccess")
	t.Log("Testing hello-world example...")

	// Get project root
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	// Step 1: Copy the example configuration
	exampleDir := filepath.Join(projectRoot, "examples", "projects", "hello-world")
	projectContent, err := os.ReadFile(filepath.Join(exampleDir, "project.tofu"))
	if err != nil {
		t.Fatalf("Failed to read project.tofu from example: %v", err)
	}

	// Write the project.tofu to test directory
	projectPath := filepath.Join(testDir, "project.tofu")
	if err := os.WriteFile(projectPath, projectContent, 0644); err != nil {
		t.Fatalf("Failed to write project.tofu: %v", err)
	}

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

	// Step 4: Run plan
	t.Log("Running tofu plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run plan: %s", planOutput)
	t.Log("✓ Plan completed successfully")

	// Step 5: Run apply
	t.Log("Running tofu apply --auto-approve...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color", "-parallelism=1")
	applyCmd.Dir = testDir
	applyOutput, err := applyCmd.CombinedOutput()
	require.NoError(t, err, "Failed to run apply: %s", applyOutput)
	t.Log("✓ Apply completed successfully")

	// Step 6: Verify files were created
	// Note: Files are created directly in testDir, not in output/ subdirectory
	outputPath := testDir

	// Verify deterministic inline file
	inlineDeterministicPath := filepath.Join(outputPath, "inline-deterministic.txt")
	assert.FileExists(t, inlineDeterministicPath, "inline-deterministic.txt should exist")
	verifyFileContent(t, inlineDeterministicPath, "Inline file with determenistic content")

	// Verify LLM-generated inline file (German sky color)
	inlineNonDeterministicPath := filepath.Join(outputPath, "inline-non-deterministic.txt")
	assert.FileExists(t, inlineNonDeterministicPath, "inline-non-deterministic.txt should exist")
	content, _ := os.ReadFile(inlineNonDeterministicPath)
	assert.Contains(t, string(content), "blau", "inline-non-deterministic.txt should contain 'blau' (German for blue)")

	// Verify subdirectory file
	subdirPath := filepath.Join(outputPath, "subdirectory", "hello-subdirectory-deterministic.txt")
	assert.FileExists(t, subdirPath, "subdirectory/hello-subdirectory-deterministic.txt should exist")
	verifyFileContent(t, subdirPath, "Subdirectory File Contents")

	// Verify file with resource link in content
	resourceLinkPath := filepath.Join(outputPath, "hello-with-resource-link.txt")
	assert.FileExists(t, resourceLinkPath, "hello-with-resource-link.txt should exist")
	verifyFileContent(t, resourceLinkPath, "More info in tofukit://file/README.md")

	// Verify file with resource link in prompt (cross-resource reference)
	resourceLinkInPromptPath := filepath.Join(outputPath, "hello-with-resource-link-in-prompt.txt")
	assert.FileExists(t, resourceLinkInPromptPath, "hello-with-resource-link-in-prompt.txt should exist")
	content, _ = os.ReadFile(resourceLinkInPromptPath)
	assert.Contains(t, string(content), "Animal", "hello-with-resource-link-in-prompt.txt should contain 'Animal' (book name)")

	// Verify README from file resource
	readmePath := filepath.Join(outputPath, "README.md")
	assert.FileExists(t, readmePath, "README.md should exist")
	content, _ = os.ReadFile(readmePath)
	assert.LessOrEqual(t, len(content), 100, "README.md should be less than 100 characters")

	// Verify STORY.md from file resource
	storyPath := filepath.Join(outputPath, "STORY.md")
	assert.FileExists(t, storyPath, "STORY.md should exist")
	content, _ = os.ReadFile(storyPath)
	assert.Contains(t, string(content), "JONES", "STORY.md should contain 'JONES' from Animal Farm")

	t.Log("✅ Example hello-world validated successfully!")
}
