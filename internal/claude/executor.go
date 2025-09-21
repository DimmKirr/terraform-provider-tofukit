package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Executor manages Claude Code execution lifecycle for Terraform resources
type Executor struct {
	client *Client
}

// NewExecutor creates a new Claude Code executor
func NewExecutor(claudeHomeDir string, dryRun bool) *Executor {
	return &Executor{
		client: NewClient(claudeHomeDir, dryRun),
	}
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

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(projectPath), 0755); err != nil {
		tflog.Error(ctx, "Failed to create output directory", map[string]interface{}{
			"path":  filepath.Dir(projectPath),
			"error": err.Error(),
		})
		status.State = "failed"
		status.CompletedAt = time.Now().Format(time.RFC3339)
		status.Error = fmt.Sprintf("Failed to create output directory: %v", err)
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
	status.State = "completed"
	status.CompletedAt = time.Now().Format(time.RFC3339)
	status.Output = result.Output
	status.ProjectPath = result.ProjectPath

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
	metadataPath := filepath.Join(outputDir, "claude-execution-metadata.json")

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

	tflog.Debug(ctx, "Saved execution metadata", map[string]interface{}{
		"metadata_path": metadataPath,
	})

	return nil
}

// LoadExecutionStatus loads execution status from metadata file
func (e *Executor) LoadExecutionStatus(ctx context.Context, outputDir string) (*ExecutionStatus, error) {
	metadataPath := filepath.Join(outputDir, "claude-execution-metadata.json")

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
