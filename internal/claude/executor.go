package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Executor manages Claude Code execution lifecycle for Terraform resources
type Executor struct {
	client     *Client
	debug      bool
	outputPath string
}

// NewExecutor creates a new Claude Code executor
func NewExecutor(claudeHomeDir string) *Executor {
	return &Executor{
		client:     NewClient(claudeHomeDir),
		debug:      false,
		outputPath: "",
	}
}

// SetDebug enables or disables debug mode
func (e *Executor) SetDebug(debug bool) {
	e.debug = debug
	// Note: client.SetDebug is not available in current implementation
}

// SetOutputPath sets the output path for debug files
func (e *Executor) SetOutputPath(outputPath string) {
	e.outputPath = outputPath
	// Note: client.SetOutputPath is not available in current implementation
}

// SetSystemPrompt sets the custom system prompt
func (e *Executor) SetSystemPrompt(systemPrompt string) {
	// Store system prompt for later use
	// Note: client.SetSystemPrompt is not available in current implementation
}

// ExecutionStatus represents the status of a Claude Code execution
type ExecutionStatus struct {
	State       string            `json:"state"`        // pending, running, completed, failed
	StartedAt   string            `json:"started_at"`   // RFC3339 timestamp
	CompletedAt string            `json:"completed_at"` // RFC3339 timestamp
	ProjectPath string            `json:"project_path"`
	Error       string            `json:"error,omitempty"`
	Output      string            `json:"output,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Execute runs Claude Code with the provided project specification
func (e *Executor) Execute(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*ExecutionStatus, error) {
	tflog.Info(ctx, "Starting Claude Code execution", map[string]interface{}{
		"output_dir": outputDir,
	})

	// Validate Claude Code is available
	if err := e.client.ValidateClaudeCodeAvailability(ctx); err != nil {
		tflog.Error(ctx, "Claude Code validation failed", map[string]interface{}{
			"error": err.Error(),
		})
		return &ExecutionStatus{
			State:     "failed",
			StartedAt: time.Now().Format(time.RFC3339),
			Error:     fmt.Sprintf("Claude Code validation failed: %v", err),
		}, err
	}

	// Create execution status
	status := &ExecutionStatus{
		State:     "running",
		StartedAt: time.Now().Format(time.RFC3339),
		Metadata:  make(map[string]string),
	}

	// Extract project information for metadata
	if project, ok := projectSpec["project"].(map[string]interface{}); ok {
		if name, ok := project["name"].(string); ok {
			status.Metadata["project_name"] = name
		}
		if version, ok := project["version"].(string); ok {
			status.Metadata["project_version"] = version
		}
	}

	// Determine project path - use project name from spec or default
	projectName := "project"
	if project, ok := projectSpec["project"].(map[string]interface{}); ok {
		if name, ok := project["name"].(string); ok {
			projectName = name
		}
	}
	projectPath := filepath.Join(outputDir, projectName)
	status.ProjectPath = projectPath

	// Ensure project directory exists
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		tflog.Error(ctx, "Failed to create project directory", map[string]interface{}{
			"path":  projectPath,
			"error": err.Error(),
		})
		status.State = "failed"
		status.CompletedAt = time.Now().Format(time.RFC3339)
		status.Error = fmt.Sprintf("Failed to create project directory: %v", err)
		return status, err
	}

	// Execute Claude Code with timeout context
	execCtx, cancel := context.WithTimeout(ctx, 10*time.Minute) // 10 minute timeout
	defer cancel()

	result, err := e.client.ExecuteProject(execCtx, projectSpec, projectPath)
	if err != nil {
		tflog.Error(ctx, "Claude Code execution failed", map[string]interface{}{
			"error": err.Error(),
		})
		status.State = "failed"
		status.CompletedAt = time.Now().Format(time.RFC3339)
		status.Error = err.Error()

		// Save failure metadata for debugging
		if saveErr := e.saveExecutionMetadata(ctx, status, outputDir); saveErr != nil {
			tflog.Warn(ctx, "Failed to save failure metadata", map[string]interface{}{
				"error": saveErr.Error(),
			})
		}

		return status, err
	}

	// Update status with results
	status.CompletedAt = time.Now().Format(time.RFC3339)
	status.Output = result.Output
	status.ProjectPath = result.ProjectPath

	// Check if output contains an error message
	// Common error patterns from Claude CLI
	if strings.Contains(result.Output, "Error:") ||
		strings.Contains(result.Output, "error:") ||
		strings.Contains(result.Output, "Reached max turns") ||
		strings.Contains(result.Output, "Failed to") {
		status.State = "failed"
		// Extract error message if possible
		if idx := strings.Index(result.Output, "Error:"); idx != -1 {
			// Get the error line
			errorMsg := result.Output[idx:]
			if endIdx := strings.Index(errorMsg, "\n"); endIdx != -1 {
				errorMsg = errorMsg[:endIdx]
			}
			status.Error = errorMsg
		} else {
			status.Error = "Claude execution failed - check output for details"
		}

		// Save execution metadata and return error
		if err := e.saveExecutionMetadata(ctx, status, outputDir); err != nil {
			tflog.Warn(ctx, "Failed to save execution metadata", map[string]interface{}{
				"error": err.Error(),
			})
		}

		return status, fmt.Errorf("Claude execution failed: %s", status.Error)
	}

	// If no errors detected, mark as completed
	status.State = "completed"

	// Save execution metadata to a file for debugging and tracking
	if err := e.saveExecutionMetadata(ctx, status, outputDir); err != nil {
		tflog.Warn(ctx, "Failed to save execution metadata", map[string]interface{}{
			"error": err.Error(),
		})
		// Don't fail the entire operation for metadata save issues
	}

	tflog.Info(ctx, "Claude Code execution completed successfully", map[string]interface{}{
		"project_path": status.ProjectPath,
		"duration":     e.calculateDuration(status.StartedAt, status.CompletedAt),
	})

	return status, nil
}

// Validate checks if Claude Code execution would be possible with the given configuration
func (e *Executor) Validate(ctx context.Context) error {
	return e.client.ValidateClaudeCodeAvailability(ctx)
}

// saveExecutionMetadata saves execution information to a metadata file
func (e *Executor) saveExecutionMetadata(ctx context.Context, status *ExecutionStatus, outputDir string) error {
	// Create .debug directory
	debugDir := filepath.Join(outputDir, ".debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("failed to create .debug directory: %w", err)
	}

	timestamp := time.Now().Unix()
	metadataPath := filepath.Join(debugDir, fmt.Sprintf("claude-execution-metadata-%d.json", timestamp))

	// Create metadata structure
	metadata := map[string]interface{}{
		"execution_status": status,
		"created_at":       time.Now().Format(time.RFC3339),
		"terraform_provider": map[string]string{
			"name":   "tofukit",
			"action": "claude_code_execution",
		},
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write to file
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	// Generate markdown version for easier reading
	markdownPath := filepath.Join(debugDir, fmt.Sprintf("claude-execution-metadata-%d.md", timestamp))
	if err := e.generateExecutionMarkdown(status, markdownPath); err != nil {
		// Log but don't fail if markdown generation fails
		tflog.Warn(ctx, "Failed to generate markdown metadata", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		tflog.Debug(ctx, "Generated markdown metadata", map[string]interface{}{
			"markdown_path": markdownPath,
		})
	}

	tflog.Debug(ctx, "Saved execution metadata", map[string]interface{}{
		"metadata_path": metadataPath,
	})

	return nil
}

// generateExecutionMarkdown creates a human-readable markdown version of the execution metadata
func (e *Executor) generateExecutionMarkdown(status *ExecutionStatus, markdownPath string) error {
	// Process the output to remove ANSI escape codes and make it more readable
	cleanOutput := status.Output
	// Remove ANSI escape sequences (like \u001b[?25h)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\[[?][0-9]+[hl]`)
	cleanOutput = ansiRegex.ReplaceAllString(cleanOutput, "")

	// Build markdown content
	var content strings.Builder

	content.WriteString("# Claude Execution Metadata\n\n")

	// Execution details
	content.WriteString("## Execution Details\n\n")
	content.WriteString(fmt.Sprintf("- **State**: %s\n", status.State))
	content.WriteString(fmt.Sprintf("- **Started**: %s\n", status.StartedAt))
	content.WriteString(fmt.Sprintf("- **Completed**: %s\n", status.CompletedAt))

	// Calculate duration
	if status.StartedAt != "" && status.CompletedAt != "" {
		start, _ := time.Parse(time.RFC3339, status.StartedAt)
		end, _ := time.Parse(time.RFC3339, status.CompletedAt)
		duration := end.Sub(start)
		content.WriteString(fmt.Sprintf("- **Duration**: %v\n", duration))
	}

	content.WriteString(fmt.Sprintf("- **Project Path**: `%s`\n", status.ProjectPath))

	// Project metadata
	if status.Metadata != nil && len(status.Metadata) > 0 {
		content.WriteString("\n## Project Metadata\n\n")
		if name, ok := status.Metadata["project_name"]; ok && name != "" {
			content.WriteString(fmt.Sprintf("- **Name**: %s\n", name))
		}
		if version, ok := status.Metadata["project_version"]; ok && version != "" {
			content.WriteString(fmt.Sprintf("- **Version**: %s\n", version))
		}
		if desc, ok := status.Metadata["project_description"]; ok && desc != "" {
			content.WriteString(fmt.Sprintf("- **Description**: %s\n", desc))
		}
	}

	// Error section if present
	if status.Error != "" {
		content.WriteString("\n## Error\n\n")
		content.WriteString("```\n")
		content.WriteString(status.Error)
		content.WriteString("\n```\n")
	}

	// Claude's output
	content.WriteString("\n## Claude Output\n\n")
	content.WriteString("---\n\n")

	// The output from Claude often contains markdown, so we'll render it as-is
	content.WriteString(cleanOutput)

	// If output doesn't end with newline, add one
	if !strings.HasSuffix(cleanOutput, "\n") {
		content.WriteString("\n")
	}

	// Write to file
	return os.WriteFile(markdownPath, []byte(content.String()), 0644)
}

// LoadExecutionStatus loads execution status from metadata file
func (e *Executor) LoadExecutionStatus(ctx context.Context, outputDir string) (*ExecutionStatus, error) {
	debugDir := filepath.Join(outputDir, ".debug")
	metadataPath := filepath.Join(debugDir, "claude-execution-metadata.json")

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No previous execution
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Extract execution status
	if statusData, ok := metadata["execution_status"].(map[string]interface{}); ok {
		statusJSON, err := json.Marshal(statusData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal status data: %w", err)
		}

		var status ExecutionStatus
		if err := json.Unmarshal(statusJSON, &status); err != nil {
			return nil, fmt.Errorf("failed to unmarshal execution status: %w", err)
		}

		return &status, nil
	}

	return nil, fmt.Errorf("no execution status found in metadata")
}

// calculateDuration calculates the duration between start and end times
func (e *Executor) calculateDuration(startTime, endTime string) string {
	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return "unknown"
	}

	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return "unknown"
	}

	duration := end.Sub(start)
	return duration.String()
}

// IsProjectGenerated checks if a project has been generated at the specified path
func (e *Executor) IsProjectGenerated(ctx context.Context, projectPath string) bool {
	if projectPath == "" {
		return false
	}

	// Check if project directory exists and has content
	if stat, err := os.Stat(projectPath); err == nil && stat.IsDir() {
		// Check if directory has any files
		entries, err := os.ReadDir(projectPath)
		if err == nil && len(entries) > 0 {
			tflog.Debug(ctx, "Project directory exists with content", map[string]interface{}{
				"project_path": projectPath,
				"file_count":   len(entries),
			})
			return true
		}
	}

	return false
}

// CleanupProject removes the generated project directory
func (e *Executor) CleanupProject(ctx context.Context, projectPath string) error {
	if projectPath == "" {
		return nil
	}

	if err := os.RemoveAll(projectPath); err != nil {
		tflog.Error(ctx, "Failed to cleanup project directory", map[string]interface{}{
			"project_path": projectPath,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to cleanup project directory: %w", err)
	}

	tflog.Info(ctx, "Project directory cleaned up", map[string]interface{}{
		"project_path": projectPath,
	})
	return nil
}

// RetryExecution attempts to retry a failed execution
func (e *Executor) RetryExecution(ctx context.Context, projectSpec map[string]interface{}, outputDir string, maxRetries int) (*ExecutionStatus, error) {
	var lastErr error
	var lastStatus *ExecutionStatus

	for attempt := 1; attempt <= maxRetries; attempt++ {
		tflog.Info(ctx, "Attempting Claude Code execution", map[string]interface{}{
			"attempt":     attempt,
			"max_retries": maxRetries,
		})

		status, err := e.Execute(ctx, projectSpec, outputDir)
		if err == nil && status.State == "completed" {
			tflog.Info(ctx, "Claude Code execution succeeded", map[string]interface{}{
				"attempt": attempt,
			})
			return status, nil
		}

		lastErr = err
		lastStatus = status

		if attempt < maxRetries {
			// Wait before retrying with exponential backoff
			waitTime := time.Duration(attempt*attempt) * time.Second
			tflog.Info(ctx, "Retrying Claude Code execution", map[string]interface{}{
				"attempt":   attempt,
				"wait_time": waitTime.String(),
				"error":     err.Error(),
			})

			select {
			case <-ctx.Done():
				return lastStatus, ctx.Err()
			case <-time.After(waitTime):
				// Continue to next retry
			}
		}
	}

	tflog.Error(ctx, "All Claude Code execution attempts failed", map[string]interface{}{
		"attempts": maxRetries,
		"error":    lastErr.Error(),
	})

	return lastStatus, fmt.Errorf("execution failed after %d attempts: %w", maxRetries, lastErr)
}
