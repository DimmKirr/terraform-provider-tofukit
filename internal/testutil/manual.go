package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
)

// ManualTestConfig contains configuration for manual testing
type ManualTestConfig struct {
	ProjectName   string
	Instructions  string
	OutputDir     string
	ClaudeHomeDir string
	Timeout       time.Duration
}

// DefaultManualTestConfig returns a default configuration for manual testing
func DefaultManualTestConfig() *ManualTestConfig {
	return &ManualTestConfig{
		ProjectName:   "manual-test",
		Instructions:  "Create a file called hello.txt with the content 'Hello, World!'",
		OutputDir:     "./test-output",
		ClaudeHomeDir: "~/.claude",
		Timeout:       5 * time.Minute,
	}
}

// RunManualTest executes a manual test with the given configuration
func RunManualTest(config *ManualTestConfig) error {
	fmt.Printf("🚀 Starting manual test: %s\n", config.ProjectName)
	fmt.Printf("📁 Output directory: %s\n", config.OutputDir)

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create project specification
	projectSpec := map[string]interface{}{
		"project": map[string]interface{}{
			"name":        config.ProjectName,
			"description": "Manual test project",
			"version":     "1.0.0",
		},
		"requirements": []map[string]interface{}{
			{
				"name": "Manual test requirement",
				"instructions": []string{
					config.Instructions,
				},
				"verification": map[string]string{
					"command": "ls -la",
					"expect":  "success",
				},
			},
		},
	}

	// Create executor
	executor := claude.NewExecutor(config.ClaudeHomeDir, true) // Skip permissions for tests

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	fmt.Printf("⏳ Executing Claude Code (timeout: %v)...\n", config.Timeout)
	start := time.Now()

	// Execute
	status, err := executor.Execute(ctx, projectSpec, config.OutputDir)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Execution failed after %v: %v\n", duration, err)
		return err
	}

	fmt.Printf("✅ Execution completed in %v\n", duration)
	fmt.Printf("📊 Status: %s\n", status.State)
	fmt.Printf("📂 Project path: %s\n", status.ProjectPath)

	if status.Error != "" {
		fmt.Printf("⚠️  Error: %s\n", status.Error)
	}

	// Show output files
	if err := showOutputFiles(config.OutputDir); err != nil {
		fmt.Printf("⚠️  Failed to show output files: %v\n", err)
	}

	// Show generated project files if not dry run
	if status.ProjectPath != "" {
		fmt.Printf("\n📁 Generated project files:\n")
		if err := showDirectoryContents(status.ProjectPath); err != nil {
			fmt.Printf("⚠️  Failed to show project files: %v\n", err)
		}
	}

	return nil
}

// RunQuickTest runs a quick test with default settings
func RunQuickTest() error {
	config := DefaultManualTestConfig()
	config.ProjectName = "quick-test"
	config.Instructions = "Create a simple hello.txt file with 'Hello from manual test!'"

	return RunManualTest(config)
}

// RunHelloWorldTest runs the classic hello world test
func RunHelloWorldTest() error {
	config := DefaultManualTestConfig()
	config.ProjectName = "hello-world-test"
	config.Instructions = "Create a hello.txt file with 'Hello, World!' and a README.md with project description"
	config.OutputDir = "./test-output/hello-world"

	return RunManualTest(config)
}

// RunComplexProjectTest runs a more complex test scenario
func RunComplexProjectTest() error {
	config := DefaultManualTestConfig()
	config.ProjectName = "complex-test"
	config.OutputDir = "./test-output/complex"
	config.Timeout = 10 * time.Minute

	// More complex project specification
	projectSpec := map[string]interface{}{
		"project": map[string]interface{}{
			"name":        config.ProjectName,
			"description": "Complex test project with multiple requirements",
			"version":     "1.0.0",
		},
		"requirements": []map[string]interface{}{
			{
				"name":     "Setup project structure",
				"priority": 100,
				"instructions": []string{
					"Create a src/ directory",
					"Create a tests/ directory",
					"Create a docs/ directory",
				},
				"verification": map[string]string{
					"command": "test -d src && test -d tests && test -d docs",
					"expect":  "success",
				},
			},
			{
				"name":     "Create main application file",
				"priority": 90,
				"instructions": []string{
					"Create src/main.go with a simple Go hello world program",
					"Include proper package declaration and main function",
				},
				"verification": map[string]string{
					"command": "test -f src/main.go && grep -q 'package main' src/main.go",
					"expect":  "success",
				},
			},
			{
				"name":     "Create test file",
				"priority": 80,
				"instructions": []string{
					"Create tests/main_test.go with a simple test",
					"Test should verify the hello world functionality",
				},
				"verification": map[string]string{
					"command": "test -f tests/main_test.go",
					"expect":  "success",
				},
			},
			{
				"name":     "Create documentation",
				"priority": 70,
				"instructions": []string{
					"Create README.md with project description and usage instructions",
					"Create docs/API.md with API documentation",
				},
				"verification": map[string]string{
					"command": "test -f README.md && test -f docs/API.md",
					"expect":  "success",
				},
			},
		},
	}

	fmt.Printf("🚀 Starting complex project test\n")
	fmt.Printf("📁 Output directory: %s\n", config.OutputDir)

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create executor
	executor := claude.NewExecutor(config.ClaudeHomeDir, true) // Skip permissions for tests

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	fmt.Printf("⏳ Executing complex Claude Code project (timeout: %v)...\n", config.Timeout)
	start := time.Now()

	// Execute
	status, err := executor.Execute(ctx, projectSpec, config.OutputDir)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Complex test failed after %v: %v\n", duration, err)
		return err
	}

	fmt.Printf("✅ Complex test completed in %v\n", duration)
	fmt.Printf("📊 Status: %s\n", status.State)
	fmt.Printf("📂 Project path: %s\n", status.ProjectPath)

	// Show detailed results
	if status.ProjectPath != "" {
		fmt.Printf("\n📁 Generated project structure:\n")
		if err := showDirectoryTree(status.ProjectPath); err != nil {
			fmt.Printf("⚠️  Failed to show project tree: %v\n", err)
		}
	}

	return nil
}

// TestClaudeValidation tests Claude CLI validation
func TestClaudeValidation() error {
	fmt.Printf("🔍 Testing Claude CLI validation...\n")

	executor := claude.NewExecutor("~/.claude", true) // Skip permissions for tests

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := executor.Validate(ctx)
	if err != nil {
		fmt.Printf("❌ Claude CLI validation failed: %v\n", err)
		fmt.Printf("💡 Make sure Claude CLI is installed and you're logged in:\n")
		fmt.Printf("   npm install -g @anthropic-ai/claude-code\n")
		fmt.Printf("   claude login\n")
		return err
	}

	fmt.Printf("✅ Claude CLI validation successful\n")
	return nil
}

// Helper functions

func showOutputFiles(outputDir string) error {
	fmt.Printf("\n📄 Output files in %s:\n", outputDir)

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Printf("  📁 %s/\n", entry.Name())
		} else {
			info, _ := entry.Info()
			fmt.Printf("  📄 %s (%d bytes)\n", entry.Name(), info.Size())

			// Show content of JSON files
			if filepath.Ext(entry.Name()) == ".json" {
				showJSONFileContent(filepath.Join(outputDir, entry.Name()))
			}
		}
	}

	return nil
}

func showDirectoryContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Printf("  📁 %s/\n", entry.Name())
		} else {
			info, _ := entry.Info()
			fmt.Printf("  📄 %s (%d bytes)\n", entry.Name(), info.Size())
		}
	}

	return nil
}

func showDirectoryTree(dir string) error {
	// Try to use 'tree' command if available
	if _, err := exec.LookPath("tree"); err == nil {
		cmd := exec.Command("tree", dir)
		output, err := cmd.Output()
		if err == nil {
			fmt.Printf("%s\n", output)
			return nil
		}
	}

	// Fallback to simple directory listing
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(dir, path)
		if relPath == "." {
			return nil
		}

		depth := len(filepath.SplitList(filepath.ToSlash(relPath))) - 1
		indent := ""
		for i := 0; i < depth; i++ {
			indent += "  "
		}

		if info.IsDir() {
			fmt.Printf("%s📁 %s/\n", indent, info.Name())
		} else {
			fmt.Printf("%s📄 %s (%d bytes)\n", indent, info.Name(), info.Size())
		}

		return nil
	})
}

func showJSONFileContent(filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return
	}

	prettyJSON, err := json.MarshalIndent(jsonData, "    ", "  ")
	if err != nil {
		return
	}

	fmt.Printf("    Content preview:\n")
	lines := string(prettyJSON)
	if len(lines) > 500 {
		lines = lines[:500] + "..."
	}
	fmt.Printf("    %s\n", lines)
}
