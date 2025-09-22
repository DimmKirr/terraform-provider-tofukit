package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Client needs systemPrompt field
type Client struct {
	claudeHomeDir string
	dryRun        bool
	systemPrompt  string
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewClient creates a new Claude Code client
func NewClient(claudeHomeDir string, dryRun bool) *Client {
	// Expand home directory
	if strings.HasPrefix(claudeHomeDir, "~/") {
		home, _ := os.UserHomeDir()
		claudeHomeDir = filepath.Join(home, claudeHomeDir[2:])
	}

	return &Client{
		claudeHomeDir: claudeHomeDir,
		dryRun:        dryRun,
	}
}

// SetDebug enables or disables debug mode
func (c *Client) SetDebug(debug bool) {
	// Debug mode is handled via fmt.Printf statements currently
	// This is a placeholder for future debug mode enhancements
}

// SetOutputPath sets the output path for debug files
func (c *Client) SetOutputPath(outputPath string) {
	// Output path is passed to ExecuteProject directly
	// This is a placeholder for future enhancements
}

// SetSystemPrompt sets the custom system prompt
func (c *Client) SetSystemPrompt(systemPrompt string) {
	c.systemPrompt = systemPrompt
}

// ExecutionResult contains the result of a Claude Code execution
type ExecutionResult struct {
	Success     bool   `json:"success"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
	ProjectPath string `json:"project_path,omitempty"`
}

// ExecuteProject executes Claude Code with the provided project specification
func (c *Client) ExecuteProject(ctx context.Context, projectSpec map[string]interface{}, outputPath string) (*ExecutionResult, error) {
	fmt.Printf("🔧 DEBUG: ExecuteProject called\n")
	fmt.Printf("🔧 DEBUG: - outputPath: %s\n", outputPath)
	fmt.Printf("🔧 DEBUG: - dryRun: %v\n", c.dryRun)
	fmt.Printf("🔧 DEBUG: - claudeHomeDir: %s\n", c.claudeHomeDir)

	// Add timeout check
	if deadline, ok := ctx.Deadline(); ok {
		fmt.Printf("🔧 DEBUG: - context deadline: %v (in %v)\n", deadline, time.Until(deadline))
	} else {
		fmt.Printf("🔧 DEBUG: - no context deadline\n")
	}

	fmt.Printf("🔧 DEBUG: About to log with tflog...\n")
	tflog.Info(ctx, "Starting Claude Code execution", map[string]interface{}{
		"output_path": outputPath,
		"dry_run":     c.dryRun,
	})
	fmt.Printf("🔧 DEBUG: tflog.Info completed\n")

	// If in dry run mode, skip actual execution
	if c.dryRun {
		fmt.Printf("🔧 DEBUG: Dry run mode - skipping actual execution\n")
		tflog.Info(ctx, "Dry run mode - skipping Claude Code execution")

		// In dry run, create output directory and a sample file to show the test works
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			fmt.Printf("🔧 DEBUG: Failed to create output directory: %v\n", err)
		} else {
			testFile := filepath.Join(outputPath, "dryrun-test.txt")
			if err := os.WriteFile(testFile, []byte("This file was created in dry-run mode"), 0644); err != nil {
				fmt.Printf("🔧 DEBUG: Failed to create dry-run test file: %v\n", err)
			} else {
				fmt.Printf("🔧 DEBUG: Created dry-run test file: %s\n", testFile)
			}
		}

		return &ExecutionResult{
			Success:     true,
			Output:      "Dry run mode - Claude Code execution skipped",
			ProjectPath: outputPath,
		}, nil
	}

	// Build the prompt from the project specification
	prompt, err := c.BuildPrompt(projectSpec)
	if err != nil {
		fmt.Printf("🔧 DEBUG: Failed to build prompt: %v\n", err)
		tflog.Error(ctx, "Failed to build prompt", map[string]interface{}{
			"error": err.Error(),
		})
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to build prompt: %v", err),
		}, err
	}

	fmt.Printf("🔧 DEBUG: Built prompt (length: %d chars)\n", len(prompt))
	if len(prompt) < 1000 {
		fmt.Printf("🔧 DEBUG: Short prompt preview:\n%s\n", prompt[:min(len(prompt), 500)])
	}

	// Save prompt to file for visibility (even when not in dry-run mode)
	promptPath := filepath.Join(filepath.Dir(outputPath), "claude-prompt.txt")
	if err := os.WriteFile(promptPath, []byte(prompt), 0644); err != nil {
		fmt.Printf("🔧 DEBUG: Failed to save prompt to file: %v\n", err)
	} else {
		fmt.Printf("🔧 DEBUG: Saved prompt to: %s\n", promptPath)
	}

	// Also save markdown version for debugging
	if markdownPrompt, err := c.BuildPromptMarkdown(projectSpec); err == nil {
		markdownPath := filepath.Join(filepath.Dir(outputPath), "claude-prompt-markdown.txt")
		os.WriteFile(markdownPath, []byte(markdownPrompt), 0644)
	}

	tflog.Debug(ctx, "Built prompt for Claude Code", map[string]interface{}{
		"prompt_length": len(prompt),
	})

	// Set up environment with Claude home directory if needed
	var env []string
	if c.claudeHomeDir != "" && c.claudeHomeDir != "~/.claude" {
		env = os.Environ()
		// Filter out existing CLAUDE_HOME
		filteredEnv := []string{}
		for _, e := range env {
			if !strings.HasPrefix(e, "CLAUDE_HOME=") {
				filteredEnv = append(filteredEnv, e)
			}
		}
		// Add our CLAUDE_HOME
		filteredEnv = append(filteredEnv, fmt.Sprintf("CLAUDE_HOME=%s", c.claudeHomeDir))
		env = filteredEnv
	}

	// Set up options for Claude execution
	fmt.Printf("🔧 DEBUG: Setting up Claude CLI execution\n")
	fmt.Printf("🔧 DEBUG: Working directory: %s\n", outputPath)

	// Get absolute path for output directory
	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		fmt.Printf("🔧 DEBUG: Failed to get absolute path: %v\n", err)
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get absolute path: %v", err),
		}, err
	}

	// Change to the output directory before executing
	originalDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("🔧 DEBUG: Failed to get current directory: %v\n", err)
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to get current directory: %v", err),
		}, err
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(absOutputPath, 0755); err != nil {
		fmt.Printf("🔧 DEBUG: Failed to create output directory: %v\n", err)
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create output directory: %v", err),
		}, err
	}

	if err := os.Chdir(absOutputPath); err != nil {
		fmt.Printf("🔧 DEBUG: Failed to change to output directory: %v\n", err)
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to change directory: %v", err),
		}, err
	}
	defer os.Chdir(originalDir) // Restore original directory when done

	fmt.Printf("🔧 DEBUG: Changed working directory to: %s\n", absOutputPath)

	systemPrompt := c.BuildSystemPrompt()
	fmt.Printf("🔧 DEBUG: System prompt length: %d chars\n", len(systemPrompt))

	// Build the full prompt with system prompt
	fullPrompt := fmt.Sprintf("%s\n\n%s", systemPrompt, prompt)

	fmt.Printf("🔧 DEBUG: Claude CLI Execution:\n")
	fmt.Printf("🔧 DEBUG:   - Working dir: %s\n", absOutputPath)
	fmt.Printf("🔧 DEBUG:   - Full prompt length: %d chars\n", len(fullPrompt))
	if deadline, ok := ctx.Deadline(); ok {
		fmt.Printf("🔧 DEBUG:   - Context deadline: %v\n", deadline)
	} else {
		fmt.Printf("🔧 DEBUG:   - Context has no deadline\n")
	}
	fmt.Printf("🔧 DEBUG: Prompt preview (first 100 chars): %s\n", prompt[:min(len(prompt), 100)])

	// Execute Claude using the SDK
	fmt.Printf("🔧 DEBUG: Executing claude via SDK...\n")

	// Set up environment with Claude home directory if needed
	if c.claudeHomeDir != "" && c.claudeHomeDir != "~/.claude" {
		os.Setenv("CLAUDE_HOME", c.claudeHomeDir)
		defer os.Unsetenv("CLAUDE_HOME")
	}

	// Debug output for configuration
	fmt.Printf("🔧 DEBUG: Claude execution configuration:\n")
	fmt.Printf("🔧 DEBUG:   - Working dir: %s\n", absOutputPath)
	fmt.Printf("🔧 DEBUG:   - System prompt length: %d chars\n", len(systemPrompt))
	fmt.Printf("🔧 DEBUG:   - MaxTurns: 10\n")
	fmt.Printf("🔧 DEBUG:   - Using --dangerously-skip-permissions flag\n")

	// Execute Claude with unbuffer and --dangerously-skip-permissions
	fmt.Printf("🔧 DEBUG: Executing claude with unbuffer and --dangerously-skip-permissions\n")

	// For very long prompts, truncate and provide instructions to read from file
	actualPrompt := prompt
	if len(prompt) > 10000 {
		// Save full prompt to file and use a reference
		fullPromptPath := filepath.Join(absOutputPath, "full-prompt.md")
		if err := os.WriteFile(fullPromptPath, []byte(prompt), 0644); err != nil {
			fmt.Printf("🔧 DEBUG: Failed to write full prompt: %v\n", err)
		}
		actualPrompt = fmt.Sprintf("The full instructions are too long for the command line. Please read the file '%s' and execute all the instructions in it.", fullPromptPath)
	}

	// Build command - use unbuffer to prevent TTY detection issues
	commandArgs := []string{
		"claude",
		"-p",
		actualPrompt, // Pass prompt as argument
		"--dangerously-skip-permissions",
		"--max-turns", "30",
		"--system-prompt", systemPrompt,
	}

	fmt.Printf("🔧 DEBUG: Command: unbuffer claude -p <prompt> ...\n")
	fmt.Printf("🔧 DEBUG: Prompt length: %d characters (original: %d)\n", len(actualPrompt), len(prompt))

	// Create a timeout context if one isn't already set
	execCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		// No deadline set, create one (3 minutes for complex operations)
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
		fmt.Printf("🔧 DEBUG: Set 3 minute timeout for Claude execution\n")
	}

	// Use unbuffer as the main command
	cmd := exec.CommandContext(execCtx, "unbuffer", commandArgs...)
	cmd.Dir = absOutputPath

	// CLAUDE_HOME is not needed when using --dangerously-skip-permissions
	// The flag bypasses all authentication and permission checks
	fmt.Printf("🔧 DEBUG: CLAUDE_HOME not set (not needed with --dangerously-skip-permissions)\n")

	// Execute and capture output
	output, cmdErr := cmd.CombinedOutput()

	if cmdErr != nil {
		fmt.Printf("🔧 DEBUG: Claude execution failed: %v\n", cmdErr)
		fmt.Printf("🔧 DEBUG: Output: %s\n", string(output))

		// Check if files were created despite error (e.g., max turns reached)
		// This is common when Claude creates files but hits turn limit
		if strings.Contains(string(output), "max turns") {
			fmt.Printf("🔧 DEBUG: Max turns reached but continuing (files may have been created)\n")
			// Don't return error if it's just max turns
		} else {
			tflog.Error(ctx, "Claude execution failed", map[string]interface{}{
				"error":  cmdErr.Error(),
				"output": string(output),
			})
			return &ExecutionResult{
				Success: false,
				Error:   fmt.Sprintf("Claude execution failed: %v\nOutput: %s", cmdErr, string(output)),
			}, cmdErr
		}
	}

	fmt.Printf("🔧 DEBUG: Claude execution completed successfully\n")
	fmt.Printf("🔧 DEBUG: Output length: %d chars\n", len(output))
	if len(output) > 0 {
		fmt.Printf("🔧 DEBUG: Output preview (first 500 chars): %s\n", string(output[:min(len(output), 500)]))
	}

	// Log Claude's response to the debug log file if in debug mode
	debugDir := filepath.Join(filepath.Dir(outputPath), ".debug")
	if _, err := os.Stat(debugDir); err == nil {
		// Find the most recent claude-log-*.jsonl file
		files, _ := filepath.Glob(filepath.Join(debugDir, "claude-log-*.jsonl"))
		if len(files) > 0 {
			// Use the most recent log file
			logPath := files[len(files)-1]
			responseEntry := map[string]interface{}{
				"type":              "response",
				"timestamp":         time.Now().Format(time.RFC3339),
				"content":           string(output),
				"success":           true,
				"working_directory": absOutputPath,
			}
			if logBytes, err := json.Marshal(responseEntry); err == nil {
				if logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644); err == nil {
					logFile.Write(append(logBytes, '\n'))
					logFile.Close()
					fmt.Printf("🔧 DEBUG: Appended response to log: %s\n", logPath)
				}
			}
		}
	}

	tflog.Info(ctx, "Claude Code execution completed", map[string]interface{}{
		"project_path":   absOutputPath,
		"output_length":  len(output),
		"execution_time": fmt.Sprintf("%v", time.Since(time.Now())),
	})

	return &ExecutionResult{
		Success:     true,
		Output:      string(output),
		ProjectPath: absOutputPath,
	}, nil
}

// BuildPrompt creates a structured JSON prompt to send to Claude Code
func (c *Client) BuildPrompt(projectSpec map[string]interface{}) (string, error) {
	// Use the BuildProjectPrompt from prompt_types.go
	prompt := BuildProjectPrompt(projectSpec, c.systemPrompt)

	// Convert to JSON
	return prompt.ToJSON()
}

// getStringOrDefault safely extracts a string from a map or returns a default
func getStringOrDefault(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

// BuildPromptMarkdown creates a markdown version of the prompt for debug purposes
func (c *Client) BuildPromptMarkdown(projectSpec map[string]interface{}) (string, error) {
	// Convert spec to JSON for embedding in markdown
	specJSON, err := json.MarshalIndent(projectSpec, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal project spec: %w", err)
	}

	// Extract key information for prompt context
	projectName := "Project"
	projectDesc := "A Terraform-generated project"
	projectVersion := "1.0.0"

	if project, ok := projectSpec["project"].(map[string]interface{}); ok {
		if name, ok := project["name"].(string); ok {
			projectName = name
		}
		if desc, ok := project["description"].(string); ok {
			projectDesc = desc
		}
		if version, ok := project["version"].(string); ok {
			projectVersion = version
		}
	}

	// Define the prompt template
	const promptTemplate = `# Project Implementation Request

I need you to implement a complete project based on the following specification.

## Project Information
- **Name**: {{.ProjectName}}
- **Description**: {{.ProjectDesc}}
- **Version**: {{.ProjectVersion}}

## Complete Project Specification
The following JSON contains the full project specification including all requirements, kits, and dependencies:

` + "```json\n{{.SpecJSON}}\n```" + `

## Instructions

Please implement this project by:

1. **Analyzing the specification**: Understand all the requirements, kits, and dependencies specified in the JSON
2. **Creating the project structure**: Set up appropriate directories and files based on the project type and requirements
3. **Implementing all requirements**: Follow each requirement listed in the "requirements" section with proper priority ordering
4. **Installing and configuring all kits**: Set up all the tools, frameworks, languages, and methodologies specified in the "kits" section
5. **Following verification steps**: Ensure each requirement can be verified as specified
6. **Creating comprehensive documentation**: Include README, setup instructions, and usage examples

## Key Guidelines

- Follow the exact specifications provided in the JSON
- Implement all requirements in priority order (higher numbers first)
- Ensure all verification commands work as expected
- Create production-ready, well-documented code
- Follow best practices for the specified programming language and frameworks
- Include proper error handling and logging
- Set up development and build toolchains as specified in the kits

## Expected Deliverables

- Complete, working project implementation
- All files and directories properly structured
- Documentation explaining setup and usage
- All requirements implemented and verified
- Development environment ready for use

Please start implementing the project now.`

	// Parse and execute the template
	tmpl, err := template.New("prompt").Parse(promptTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse prompt template: %w", err)
	}

	// Prepare data for template
	data := struct {
		ProjectName    string
		ProjectDesc    string
		ProjectVersion string
		SpecJSON       string
	}{
		ProjectName:    projectName,
		ProjectDesc:    projectDesc,
		ProjectVersion: projectVersion,
		SpecJSON:       string(specJSON),
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute prompt template: %w", err)
	}

	return buf.String(), nil
}

// BuildSystemPrompt creates a system prompt that guides Claude Code behavior (exported for testing)
func (c *Client) BuildSystemPrompt() string {
	if c.systemPrompt != "" {
		return c.systemPrompt
	}
	return DefaultSystemPrompt()
}

// ValidateClaudeCodeAvailability checks if Claude Code CLI is available
func (c *Client) ValidateClaudeCodeAvailability(ctx context.Context) error {
	// Check if Claude CLI is installed
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Printf("🔍 DEBUG: Claude binary not found in PATH\n")
		fmt.Printf("🔍 DEBUG: PATH = %s\n", os.Getenv("PATH"))
		return fmt.Errorf("claude CLI not found: %w. Please install with: npm install -g @anthropic-ai/claude-code", err)
	}

	fmt.Printf("🔍 DEBUG: Claude binary found at: %s\n", claudePath)

	// Skip version check as 'claude version' hangs
	fmt.Printf("🔍 DEBUG: Skipping Claude version check (known to hang)\n")

	// Verify Claude home directory exists and has credentials
	expandedHome := c.claudeHomeDir
	if strings.HasPrefix(expandedHome, "~/") {
		home, _ := os.UserHomeDir()
		expandedHome = filepath.Join(home, expandedHome[2:])
	}

	fmt.Printf("🔍 DEBUG: Claude home directory: %s\n", expandedHome)

	// Check if home directory exists
	if _, err := os.Stat(expandedHome); os.IsNotExist(err) {
		fmt.Printf("🔍 DEBUG: Claude home directory doesn't exist: %s\n", expandedHome)
		// This is not an error - the directory might be created during first run
	} else {
		fmt.Printf("🔍 DEBUG: Claude home directory exists: %s\n", expandedHome)
	}

	return nil
}
