package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProjectKitDependencyRequirements validates that all kit requirements
// (including those from feature-referenced kits like gotask) are included in the Claude prompt
func TestProjectKitDependencyRequirements(t *testing.T) {
	// Create test directory
	testDir := createTestDirectory(t, "TestProjectKitDependencyRequirements")

	// Get project root
	projectRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	// Copy stacks directory to test directory
	stacksDir := filepath.Join(projectRoot, "examples", "stacks")
	testStacksDir := filepath.Join(testDir, "stacks")
	copyDirCmd := exec.Command("cp", "-r", stacksDir, testStacksDir)
	err = copyDirCmd.Run()
	require.NoError(t, err, "Failed to copy stacks directory")
	t.Logf("Copied stacks from %s to %s", stacksDir, testStacksDir)

	// Write config with corrected module path
	config := ConfigKitDependencyTest
	// Replace module source to point to copied stacks
	config = strings.Replace(config,
		`source = "../../../examples/stacks/tofukit-stack-go-viper-cobra-pterm"`,
		`source = "./stacks/tofukit-stack-go-viper-cobra-pterm"`, 1)

	configPath := filepath.Join(testDir, "project.tofu")
	err = os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Detect IaC tool
	iacTool := detectIaCTool(t)

	// Run init
	t.Log("Running init...")
	initCmd := exec.Command(iacTool, "init", "-no-color")
	initCmd.Dir = testDir
	initOutput, err := initCmd.CombinedOutput()
	if err != nil {
		t.Logf("Init output: %s", initOutput)
	}
	require.NoError(t, err, "Init should succeed")

	// Run plan (this generates the prompt JSON without executing Claude)
	t.Log("Running plan...")
	planCmd := exec.Command(iacTool, "plan", "-no-color")
	planCmd.Dir = testDir
	planOutput, err := planCmd.CombinedOutput()
	if err != nil {
		t.Logf("Plan output: %s", planOutput)
	}
	require.NoError(t, err, "Plan should succeed")

	// Run apply to generate the prompt JSON (will fail in Claude execution, but we only need the prompt)
	t.Log("Running apply to generate prompt JSON...")
	applyCmd := exec.Command(iacTool, "apply", "-auto-approve", "-no-color")
	applyCmd.Dir = testDir
	applyOutput, _ := applyCmd.CombinedOutput()
	// We expect this to fail (Claude execution), but the prompt JSON should be generated
	t.Logf("Apply output (expected to fail): %s", string(applyOutput))

	// Read the generated prompt JSON
	// Note: Provider creates .debug in testDir root, not in testDir/output
	debugPattern := filepath.Join(testDir, ".debug", "claude-prompt-attempt1-*.json")
	debugFiles, err := filepath.Glob(debugPattern)
	require.NoError(t, err)
	require.NotEmpty(t, debugFiles, "Should have generated prompt JSON file")

	promptJSONPath := debugFiles[0]
	t.Logf("Reading prompt JSON: %s", promptJSONPath)

	jsonData, err := os.ReadFile(promptJSONPath)
	require.NoError(t, err, "Should be able to read prompt JSON")

	var promptJSON map[string]interface{}
	err = json.Unmarshal(jsonData, &promptJSON)
	require.NoError(t, err, "Should be able to parse prompt JSON")

	// Extract requirements
	request, ok := promptJSON["request"].(map[string]interface{})
	require.True(t, ok, "prompt should have 'request' object")

	specification, ok := request["specification"].(map[string]interface{})
	require.True(t, ok, "request should have 'specification' object")

	requirements, ok := specification["requirements"].([]interface{})
	require.True(t, ok, "specification should have 'requirements' array")

	// Collect all requirement names
	requirementNames := make([]string, len(requirements))
	requirementMap := make(map[string]map[string]interface{})

	for i, req := range requirements {
		reqMap, ok := req.(map[string]interface{})
		require.True(t, ok, "requirement should be an object")

		name, ok := reqMap["name"].(string)
		require.True(t, ok, "requirement should have string 'name' field")

		requirementNames[i] = name
		requirementMap[name] = reqMap
	}

	t.Logf("Found %d requirements in prompt:", len(requirementNames))
	for i, name := range requirementNames {
		t.Logf("  %d. %s", i+1, name)
	}

	// === TEST 1: Verify expected requirement count ===
	// Stack should provide:
	// - Language (go): 2 requirements
	// - Tool (go_mod): 1 requirement
	// - Tool (gotask): 2 requirements ← THIS IS THE BUG WE'RE TESTING FOR
	// - Framework (cobra): 2 requirements
	// - Framework (viper): 2 requirements
	// - Framework (pterm): 2 requirements
	// - Feature (viper_config): 1 requirement
	// - Feature (version_command): 1 requirement
	// - Feature (pterm_logger): 1 requirement
	// - Feature (taskfile): 1 requirement
	// - Feature (gitignore): 1 requirement
	// - Feature (readme): 1 requirement
	// Total: 17 stack requirements
	// Project adds: 0 requirements (just files)
	// Expected total: 17

	expectedMinCount := 17 // Minimum expected (without gotask bug fix: 15)
	assert.GreaterOrEqual(t, len(requirementNames), expectedMinCount,
		"Should have at least %d requirements from stack", expectedMinCount)

	// === TEST 2: Verify all kit requirements are present ===
	t.Run("LanguageKitRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "Go Runtime Verification",
			"Should include go language verification requirement")
		assert.Contains(t, requirementNames, "Idiomatic Go Code Standards",
			"Should include go language standards requirement")
	})

	t.Run("GoModToolRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "Go Modules Initialization",
			"Should include go_mod tool requirement")
	})

	t.Run("GotaskToolRequirements", func(t *testing.T) {
		// THIS IS THE KEY TEST - these requirements are currently MISSING
		assert.Contains(t, requirementNames, "Task Runner Installation",
			"Should include gotask tool installation requirement (referenced by taskfile feature)")
		assert.Contains(t, requirementNames, "Taskfile v3 Support",
			"Should include gotask tool v3 support requirement (referenced by taskfile feature)")
	})

	t.Run("CobraFrameworkRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "Cobra Installation",
			"Should include cobra framework installation")
		assert.Contains(t, requirementNames, "Idiomatic Cobra Usage",
			"Should include cobra framework usage requirement")
	})

	t.Run("ViperFrameworkRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "Viper Installation",
			"Should include viper framework installation")
		assert.Contains(t, requirementNames, "Idiomatic Viper Usage",
			"Should include viper framework usage requirement")
	})

	t.Run("PtermFrameworkRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "pterm Installation",
			"Should include pterm framework installation")
		assert.Contains(t, requirementNames, "Idiomatic pterm Usage",
			"Should include pterm framework usage requirement")
	})

	t.Run("FeatureRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "Configuration Management",
			"Should include viper_config feature requirement")
		assert.Contains(t, requirementNames, "Version Command Implementation",
			"Should include version_command feature requirement")
		assert.Contains(t, requirementNames, "Logger Implementation",
			"Should include pterm_logger feature requirement")
		assert.Contains(t, requirementNames, "Taskfile Configuration",
			"Should include taskfile feature requirement")
		assert.Contains(t, requirementNames, "Git Ignore Configuration",
			"Should include gitignore feature requirement")
		assert.Contains(t, requirementNames, "README Documentation",
			"Should include readme feature requirement")
	})

	// === TEST 3: Verify each requirement has instructions ===
	t.Run("AllRequirementsHaveInstructions", func(t *testing.T) {
		for name, reqMap := range requirementMap {
			instructions, ok := reqMap["instructions"].([]interface{})
			assert.True(t, ok, "Requirement '%s' should have instructions array", name)
			assert.NotEmpty(t, instructions, "Requirement '%s' should have at least one instruction", name)

			if len(instructions) > 0 {
				firstInstruction, ok := instructions[0].(map[string]interface{})
				assert.True(t, ok, "First instruction of '%s' should be an object", name)

				prompt, ok := firstInstruction["prompt"].(string)
				assert.True(t, ok, "First instruction of '%s' should have prompt string", name)
				assert.NotEmpty(t, prompt, "Prompt for '%s' should not be empty", name)
			}
		}
	})

	// === TEST 4: Verify files are present ===
	t.Run("FilesAreMerged", func(t *testing.T) {
		files, ok := specification["files"].(map[string]interface{})
		require.True(t, ok, "specification should have 'files' map")

		t.Logf("Found %d files in prompt", len(files))

		// Check for some expected stack files
		expectedFiles := []string{
			"main.go",
			"cmd/root.go",
			"go.mod",
			".gitignore",
			"README.md",
			"Taskfile.yaml",
			"internal/config/config.go",
			"internal/logger/logger.go",
			"internal/version/version.go",
			"test.txt", // Project-level file
		}

		for _, expectedFile := range expectedFiles {
			assert.Contains(t, files, expectedFile,
				"Files should include '%s' from stack or project", expectedFile)
		}

		// Verify project file takes precedence (test.txt is from project, not stack)
		testFile, ok := files["test.txt"].(map[string]interface{})
		if ok {
			content, _ := testFile["content"].(string)
			if content != "" {
				assert.Equal(t, "project-level file\n", content,
					"Project-level files should take precedence over stack files")
			}
		}

		// Also verify actual files exist on disk (provider creates in testDir root)
		actualTestFile := filepath.Join(testDir, "test.txt")
		if _, err := os.Stat(actualTestFile); err == nil {
			actualContent, _ := os.ReadFile(actualTestFile)
			assert.Equal(t, "project-level file\n", string(actualContent),
				"Actual file should match project specification")
		}
	})

	t.Log("✅ Kit dependency requirements validation complete")
}
