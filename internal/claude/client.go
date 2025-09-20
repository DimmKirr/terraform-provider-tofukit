package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Client wraps the Claude Code SDK and provides functionality specific to our Terraform provider
type Client struct {
	claudeHomeDir string
	dryRun        bool
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
	prompt, err := c.buildPrompt(projectSpec)
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
	fmt.Printf("🔧 DEBUG: Setting up Claude SDK options\n")
	fmt.Printf("🔧 DEBUG: Working directory: %s\n", outputPath)

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
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		fmt.Printf("🔧 DEBUG: Failed to create output directory: %v\n", err)
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to create output directory: %v", err),
		}, err
	}

	if err := os.Chdir(outputPath); err != nil {
		fmt.Printf("🔧 DEBUG: Failed to change to output directory: %v\n", err)
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to change directory: %v", err),
		}, err
	}
	defer os.Chdir(originalDir) // Restore original directory when done

	fmt.Printf("🔧 DEBUG: Changed working directory to: %s\n", outputPath)

	systemPrompt := c.buildSystemPrompt()
	fmt.Printf("🔧 DEBUG: System prompt length: %d chars\n", len(systemPrompt))

	// Don't use WithCwd as it causes SDK to return nil messages (SDK bug)
	opts := []claudecode.Option{
		claudecode.WithSystemPrompt(systemPrompt),
		claudecode.WithMaxTurns(5),
		claudecode.WithPermissionMode(claudecode.PermissionModeBypassPermissions),
	}

	fmt.Printf("🔧 DEBUG: Claude SDK Options:\n")
	fmt.Printf("🔧 DEBUG:   - SystemPrompt: %d chars\n", len(systemPrompt))
	fmt.Printf("🔧 DEBUG:   - MaxTurns: 5\n")
	fmt.Printf("🔧 DEBUG:   - PermissionMode: bypassPermissions\n")
	fmt.Printf("🔧 DEBUG:   - Working dir (via os.Chdir): %s\n", outputPath)
	if deadline, ok := ctx.Deadline(); ok {
		fmt.Printf("🔧 DEBUG:   - Context deadline: %v\n", deadline)
	} else {
		fmt.Printf("🔧 DEBUG:   - Context has no deadline\n")
	}
	fmt.Printf("🔧 DEBUG: Prompt preview (first 100 chars): %s\n", prompt[:min(len(prompt), 100)])
	fmt.Printf("🔧 DEBUG: Calling claudecode.Query()...\n")

	// Execute the query - Query returns MessageIterator and error
	messages, err := claudecode.Query(ctx, prompt, opts...)
	if err != nil {
		fmt.Printf("🔧 DEBUG: claudecode.Query() failed immediately: %v\n", err)
		tflog.Error(ctx, "Claude Code execution failed", map[string]interface{}{
			"error": err.Error(),
		})
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("Claude Code execution failed: %v", err),
		}, err
	}

	fmt.Printf("🔧 DEBUG: claudecode.Query() returned MessageIterator\n")

	// Collect all messages
	var output strings.Builder
	defer messages.Close()

	fmt.Printf("🔧 DEBUG: Starting message iteration...\n")
	messageCount := 0

	for {
		fmt.Printf("🔧 DEBUG: Calling messages.Next() (iteration %d)...\n", messageCount+1)
		msg, err := messages.Next(ctx)
		if err != nil {
			fmt.Printf("🔧 DEBUG: messages.Next() returned error: %v\n", err)
			// Check if this is end of iteration or actual error
			if err != nil && strings.Contains(err.Error(), "no more messages") {
				fmt.Printf("🔧 DEBUG: End of message stream detected (normal termination)\n")
				break
			}
			fmt.Printf("🔧 DEBUG: Unexpected error during iteration: %v\n", err)
			tflog.Error(ctx, "Error during message iteration", map[string]interface{}{
				"error": err.Error(),
			})
			return &ExecutionResult{
				Success: false,
				Error:   fmt.Sprintf("Message iteration failed: %v", err),
			}, err
		}

		// Check if message is nil - this also indicates end of stream
		if msg == nil {
			fmt.Printf("🔧 DEBUG: Received nil message - end of stream\n")
			break
		}

		messageCount++
		fmt.Printf("🔧 DEBUG: Received message %d\n", messageCount)
		fmt.Printf("🔧 DEBUG: Message type: %T\n", msg)

		// Handle different message types - for now just log what we get
		// The SDK may return different types than expected
		switch msg.(type) {
		default:
			fmt.Printf("🔧 DEBUG: Unknown message type: %T\n", msg)
			// Try to marshal as JSON for debugging
			if msgBytes, err := json.Marshal(msg); err == nil {
				msgStr := string(msgBytes)
				fmt.Printf("🔧 DEBUG: Message JSON: %s\n", msgStr[:min(len(msgStr), 500)])
				output.WriteString(msgStr)
				output.WriteString("\n")
			}
		}
	}

	fmt.Printf("🔧 DEBUG: Message iteration complete. Total messages: %d\n", messageCount)

	// Process the response
	execResult := &ExecutionResult{
		Success:     true,
		Output:      output.String(),
		ProjectPath: outputPath,
	}

	tflog.Info(ctx, "Claude Code execution completed successfully", map[string]interface{}{
		"output_length": len(execResult.Output),
	})

	return execResult, nil
}

// buildPrompt constructs a prompt for Claude Code based on the project specification
func (c *Client) buildPrompt(projectSpec map[string]interface{}) (string, error) {
	// Convert the project specification to JSON for better readability
	specJSON, err := json.MarshalIndent(projectSpec, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal project specification: %w", err)
	}

	// Extract project information
	var projectName, projectDesc, projectVersion string
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

	prompt := fmt.Sprintf("# Project Implementation Request\n\n"+
		"I need you to implement a complete project based on the following specification.\n\n"+
		"## Project Information\n"+
		"- **Name**: %s\n"+
		"- **Description**: %s\n"+
		"- **Version**: %s\n\n"+
		"## Complete Project Specification\n"+
		"The following JSON contains the full project specification including all requirements, kits, and dependencies:\n\n"+
		"```json\n%s\n```\n\n"+
		"## Instructions\n\n"+
		"Please implement this project by:\n\n"+
		"1. **Analyzing the specification**: Understand all the requirements, kits, and dependencies specified in the JSON\n"+
		"2. **Creating the project structure**: Set up appropriate directories and files based on the project type and requirements\n"+
		"3. **Implementing all requirements**: Follow each requirement listed in the \"requirements\" section with proper priority ordering\n"+
		"4. **Installing and configuring all kits**: Set up all the tools, frameworks, languages, and methodologies specified in the \"kits\" section\n"+
		"5. **Following verification steps**: Ensure each requirement can be verified as specified\n"+
		"6. **Creating comprehensive documentation**: Include README, setup instructions, and usage examples\n\n"+
		"## Key Guidelines\n\n"+
		"- Follow the exact specifications provided in the JSON\n"+
		"- Implement all requirements in priority order (higher numbers first)\n"+
		"- Ensure all verification commands work as expected\n"+
		"- Create production-ready, well-documented code\n"+
		"- Follow best practices for the specified programming language and frameworks\n"+
		"- Include proper error handling and logging\n"+
		"- Set up development and build toolchains as specified in the kits\n\n"+
		"## Expected Deliverables\n\n"+
		"- Complete, working project implementation\n"+
		"- All files and directories properly structured\n"+
		"- Documentation explaining setup and usage\n"+
		"- All requirements implemented and verified\n"+
		"- Development environment ready for use\n\n"+
		"Please start implementing the project now.",
		projectName, projectDesc, projectVersion, string(specJSON))

	return prompt, nil
}

// buildSystemPrompt creates a system prompt that guides Claude Code behavior
func (c *Client) buildSystemPrompt() string {
	return `You are a senior software architect and developer with expertise across multiple programming languages, frameworks, and development methodologies.

Your task is to implement complete, production-ready projects based on detailed specifications. You excel at:

- Setting up project structures and development environments
- Installing and configuring development tools and dependencies
- Implementing features following best practices and coding standards
- Creating comprehensive documentation and examples
- Setting up testing, linting, and build processes
- Following specified methodologies and architectural patterns

When implementing projects:
1. Always follow the exact specifications provided
2. Create well-structured, maintainable code
3. Include comprehensive error handling
4. Set up proper development toolchains
5. Write clear documentation and examples
6. Ensure all verification steps pass
7. Follow language-specific best practices and conventions

Focus on creating production-ready deliverables that developers can immediately use and extend.`
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

	// Check version
	cmd := exec.CommandContext(ctx, claudePath, "--version")
	if c.claudeHomeDir != "" && c.claudeHomeDir != "~/.claude" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("CLAUDE_HOME=%s", c.claudeHomeDir))
	}

	versionOutput, versionErr := cmd.CombinedOutput()
	if versionErr == nil {
		fmt.Printf("🔍 DEBUG: Claude version: %s\n", string(versionOutput))
	} else {
		fmt.Printf("🔍 DEBUG: Failed to get Claude version: %v\n", versionErr)
	}

	// Verify Claude home directory exists and has credentials
	expandedHome := c.claudeHomeDir
	if strings.HasPrefix(expandedHome, "~/") {
		home, _ := os.UserHomeDir()
		expandedHome = filepath.Join(home, expandedHome[2:])
	}

	fmt.Printf("🔍 DEBUG: Claude home directory: %s\n", expandedHome)

	credPath := filepath.Join(expandedHome, ".credentials.json")
	if credInfo, err := os.Stat(credPath); err == nil {
		fmt.Printf("🔍 DEBUG: Credentials found at %s (size: %d bytes)\n", credPath, credInfo.Size())
	} else if os.IsNotExist(err) {
		fmt.Printf("🔍 DEBUG: Credentials NOT found at %s\n", credPath)
		return fmt.Errorf("claude credentials not found at %s. Please run 'claude login' to authenticate", credPath)
	}

	// Check if Claude SDK package is available
	if _, err := exec.LookPath("node"); err == nil {
		nodeCmd := exec.CommandContext(ctx, "node", "-e", "try { require('@anthropic-ai/claude-code'); console.log('SDK found'); } catch(e) { console.log('SDK not found'); }")
		if output, err := nodeCmd.CombinedOutput(); err == nil {
			fmt.Printf("🔍 DEBUG: Node.js Claude SDK check: %s", string(output))
		}
	}

	tflog.Info(ctx, "Claude Code CLI validation successful")
	fmt.Printf("✅ DEBUG: Claude validation complete\n\n")
	return nil
}

// GetProjectPath returns the expected project path based on the specification
func (c *Client) GetProjectPath(projectSpec map[string]interface{}, basePath string) string {
	// Extract project name from specification
	if project, ok := projectSpec["project"].(map[string]interface{}); ok {
		if name, ok := project["name"].(string); ok {
			return filepath.Join(basePath, name)
		}
	}

	// Fallback to a generic project directory
	return filepath.Join(basePath, "generated-project")
}
