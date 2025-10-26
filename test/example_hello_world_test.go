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
	exampleDir := filepath.Join(projectRoot, "examples", "hello-world")
	projectContent, err := os.ReadFile(filepath.Join(exampleDir, "project.tofu"))
	if err != nil {
		t.Fatalf("Failed to read project.tofu from example: %v", err)
	}

	// Modify the provider config to use test output directory
	modifiedContent := string(projectContent)
	modifiedContent = strings.Replace(modifiedContent,
		`output_path           = ".tofukit"`,
		`output_path           = "output"`, 1)

	projectPath := filepath.Join(testDir, "project.tofu")
	if err := os.WriteFile(projectPath, []byte(modifiedContent), 0644); err != nil {
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
	outputPath := filepath.Join(testDir, "output")

	helloPath := filepath.Join(outputPath, "hello.txt")
	assert.FileExists(t, helloPath, "hello.txt should exist")
	verifyFileContent(t, helloPath, "hello world")

	hello2Path := filepath.Join(outputPath, "hello2.txt")
	assert.FileExists(t, hello2Path, "hello2.txt should exist")
	verifyFileContent(t, hello2Path, "hello world2")

	hello4Path := filepath.Join(outputPath, "demo", "hello4.txt")
	assert.FileExists(t, hello4Path, "demo/hello4.txt should exist")
	verifyFileContent(t, hello4Path, "hello world4")

	t.Log("✅ Example hello-world validated successfully!")
}
