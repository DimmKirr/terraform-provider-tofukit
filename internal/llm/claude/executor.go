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
	"github.com/tofukit/opentofu-provider-tofukit/internal/files"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// Executor manages Claude Code execution lifecycle for Terraform resources
type Executor struct {
	client     *Client
	debug      bool
	outputPath string
	model      string
	maxTurns   int
}

// NewExecutor creates a new Claude Code executor
func NewExecutor(claudeHomeDir string, dangerouslySkipPermissions bool, maxTurns int) *Executor {
	return &Executor{
		client:     NewClient(claudeHomeDir, dangerouslySkipPermissions, maxTurns),
		debug:      false,
		outputPath: "",
		maxTurns:   maxTurns,
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
	e.client.SetSystemPrompt(systemPrompt)
}

// SetModel sets the model to use for execution
func (e *Executor) SetModel(model string) {
	e.model = model
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

	// Files created directly in output directory, not in project-name subdirectory
	status.ProjectPath = outputDir

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		tflog.Error(ctx, "Failed to create output directory", map[string]interface{}{
			"path":  outputDir,
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

	result, err := e.client.ExecuteProject(execCtx, projectSpec, outputDir, e.model)
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

// Query sends a simple query to Claude and returns the response (no project creation)
func (e *Executor) Query(ctx context.Context, instructions []string, model string) (string, error) {
	// Default to "haiku" if no model specified (Claude-specific default)
	if model == "" {
		model = "haiku"
	}

	tflog.Info(ctx, "Executing Claude query", map[string]interface{}{
		"instruction_count": len(instructions),
		"model":             model,
	})

	// Join instructions into a single prompt
	prompt := strings.Join(instructions, "\n")

	// Execute Claude with a simple prompt (no -p flag)
	result, err := e.client.ExecuteQuery(ctx, prompt, model)
	if err != nil {
		tflog.Error(ctx, "Claude query failed", map[string]interface{}{
			"error": err.Error(),
			"model": model,
		})
		return "", err
	}

	tflog.Info(ctx, "Claude query completed successfully", map[string]interface{}{
		"output_length": len(result),
		"model":         model,
	})

	return result, nil
}

// ExecuteWithVerification executes Claude with verification retry loop
// If verifications fail, Claude receives the errors and retries until success or max retries
func (e *Executor) ExecuteWithVerification(
	ctx context.Context,
	projectSpec map[string]interface{},
	outputDir string,
	filesToVerify []schemas.FileModelWithPath,
	maxRetries int,
) (*ExecutionStatus, *files.VerificationReport, error) {
	var lastStatus *ExecutionStatus
	var lastReport *files.VerificationReport

	// Files should be created directly in outputDir, not in a subdirectory
	projectPath := outputDir // Use outputDir directly, no subdirectory

	for attempt := 1; attempt <= maxRetries; attempt++ {
		tflog.Info(ctx, "Executing Claude with verification", map[string]interface{}{
			"attempt":     attempt,
			"max_retries": maxRetries,
		})

		// Execute Claude
		status, err := e.Execute(ctx, projectSpec, outputDir)
		if err != nil {
			tflog.Error(ctx, "Claude execution failed", map[string]interface{}{
				"attempt": attempt,
				"error":   err.Error(),
			})
			return status, nil, err
		}

		lastStatus = status

		// Run verifications if we have files to verify
		if len(filesToVerify) > 0 {
			fileManager := files.NewManager(projectPath)
			report, err := fileManager.RunVerifications(ctx, filesToVerify)
			if err != nil {
				tflog.Error(ctx, "Verification execution failed", map[string]interface{}{
					"attempt": attempt,
					"error":   err.Error(),
				})
				return status, nil, fmt.Errorf("verification execution failed: %w", err)
			}

			lastReport = report

			// Check if all verifications passed
			if report.AllPassed {
				tflog.Info(ctx, "All verifications passed", map[string]interface{}{
					"passed_count": report.PassedCount,
					"attempt":      attempt,
				})
				return status, report, nil
			}

			// Verifications failed - prepare fix request if we have retries left
			if attempt < maxRetries {
				tflog.Warn(ctx, "Verifications failed, requesting fix from Claude", map[string]interface{}{
					"failed_count": report.FailedCount,
					"passed_count": report.PassedCount,
					"attempt":      attempt,
					"remaining":    maxRetries - attempt,
				})

				// Inject fix request into project spec for next attempt
				projectSpec["_fix_request"] = map[string]interface{}{
					"attempt":  attempt,
					"failures": report.GetFailureSummary(),
					"instructions": "🔧 **VERIFICATION FAILURES - IMMEDIATE FIX REQUIRED**\n\n" +
						"Your previous implementation failed verification. This is an automated retry - execute the fix immediately without any discussion or questions.\n\n" +
						"**Original Task:** See the project specification above.\n\n" +
						"**Verification Failures:**\n" +
						report.GetFailureSummary() + "\n\n" +
						"**Required Actions (execute immediately):**\n" +
						"1. Read the verification failure output - it shows exactly what's wrong\n" +
						"2. Identify the root cause of each failure\n" +
						"3. Fix the issues in the affected files\n" +
						"4. Ensure ALL verification commands will pass\n\n" +
						"**IMPORTANT:**\n" +
						"- This is a NON-INTERACTIVE automated environment\n" +
						"- Do NOT ask questions or request clarification\n" +
						"- Do NOT enter design/planning mode\n" +
						"- Execute the fix IMMEDIATELY\n" +
						"- Only modify files causing verification failures\n" +
						"- Do not change files that are working correctly",
				}

				// Wait before retrying (exponential backoff)
				waitTime := time.Duration(attempt) * time.Second
				tflog.Info(ctx, "Waiting before retry", map[string]interface{}{
					"wait_seconds": waitTime.Seconds(),
				})
				time.Sleep(waitTime)
			}
		} else {
			// No verifications to run, success
			tflog.Info(ctx, "No verifications defined, execution successful", map[string]interface{}{
				"attempt": attempt,
			})
			return status, nil, nil
		}
	}

	// All retries exhausted
	tflog.Error(ctx, "Verifications failed after all retry attempts", map[string]interface{}{
		"max_retries":  maxRetries,
		"failed_count": lastReport.FailedCount,
		"passed_count": lastReport.PassedCount,
	})

	return lastStatus, lastReport, fmt.Errorf(
		"verifications failed after %d attempts (%d passed, %d failed):\n%s",
		maxRetries,
		lastReport.PassedCount,
		lastReport.FailedCount,
		lastReport.GetFailureSummary(),
	)
}

// ExecuteWithPromptJSON executes Claude using a pre-built prompt JSON string
// This is used during apply phase when the prompt was already generated and stored during plan
func (e *Executor) ExecuteWithPromptJSON(
	ctx context.Context,
	promptJSON string,
	outputDir string,
	filesToVerify []schemas.FileModelWithPath,
	maxRetries int,
) (*ExecutionStatus, *files.VerificationReport, error) {
	// Parse the prompt JSON to extract project specification
	var projectSpec map[string]interface{}
	if err := json.Unmarshal([]byte(promptJSON), &projectSpec); err != nil {
		return nil, nil, fmt.Errorf("failed to parse prompt JSON: %w", err)
	}

	// Extract project info from the prompt for determining project path
	var projectName string
	if request, ok := projectSpec["request"].(map[string]interface{}); ok {
		if projInfo, ok := request["project_info"].(map[string]interface{}); ok {
			if name, ok := projInfo["name"].(string); ok {
				projectName = name
			}
		}
	}

	if projectName == "" {
		projectName = "project" // fallback
	}

	// Files should be created directly in outputDir, not in a subdirectory
	projectPath := outputDir // Use outputDir directly, no subdirectory
	var lastStatus *ExecutionStatus
	var lastReport *files.VerificationReport

	tflog.Info(ctx, "Executing Claude with pre-built prompt from plan phase", map[string]interface{}{
		"project_name": projectName,
		"prompt_size":  len(promptJSON),
	})

	// Execute Claude with the pre-built prompt
	for attempt := 1; attempt <= maxRetries; attempt++ {
		tflog.Info(ctx, "Executing Claude with verification (from planned prompt)", map[string]interface{}{
			"attempt":     attempt,
			"max_retries": maxRetries,
		})

		// Save the prompt JSON if debug is enabled (so we can see fix requests on retry)
		if e.debug {
			if err := e.savePromptJSON(ctx, promptJSON, outputDir, attempt); err != nil {
				tflog.Warn(ctx, "Failed to save prompt JSON", map[string]interface{}{
					"error":   err.Error(),
					"attempt": attempt,
				})
			}
		}

		// Execute using the pre-built prompt
		result, err := e.client.ExecuteProjectWithPrompt(ctx, promptJSON, projectPath, e.model)
		if err != nil {
			tflog.Error(ctx, "Claude execution failed", map[string]interface{}{
				"attempt": attempt,
				"error":   err.Error(),
			})
			// Convert to ExecutionStatus for consistent return type
			execStatus := &ExecutionStatus{
				State:       "failed",
				StartedAt:   time.Now().Format(time.RFC3339),
				CompletedAt: time.Now().Format(time.RFC3339),
				ProjectPath: projectPath,
				Error:       err.Error(),
			}
			return execStatus, nil, err
		}

		// Convert ExecutionResult to ExecutionStatus
		execStatus := &ExecutionStatus{
			State:       "completed",
			StartedAt:   time.Now().Format(time.RFC3339),
			CompletedAt: time.Now().Format(time.RFC3339),
			ProjectPath: result.ProjectPath,
			Output:      result.Output,
		}
		if !result.Success {
			execStatus.State = "failed"
			execStatus.Error = result.Error
		}

		lastStatus = execStatus

		// Save execution metadata if debug is enabled
		if e.debug {
			if err := e.saveExecutionMetadata(ctx, execStatus, outputDir); err != nil {
				tflog.Warn(ctx, "Failed to save execution metadata", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}

		// Run verifications if we have files to verify
		if len(filesToVerify) > 0 {
			fileManager := files.NewManager(projectPath)
			report, err := fileManager.RunVerifications(ctx, filesToVerify)
			if err != nil {
				tflog.Error(ctx, "Verification execution failed", map[string]interface{}{
					"attempt": attempt,
					"error":   err.Error(),
				})
				return execStatus, nil, fmt.Errorf("verification execution failed: %w", err)
			}

			lastReport = report

			// Check if all verifications passed
			if report.AllPassed {
				tflog.Info(ctx, "All verifications passed", map[string]interface{}{
					"passed_count": report.PassedCount,
					"attempt":      attempt,
				})
				return execStatus, report, nil
			}

			// Verifications failed - prepare fix request if we have retries left
			if attempt < maxRetries {
				tflog.Warn(ctx, "Verifications failed, requesting fix from Claude", map[string]interface{}{
					"failed_count": report.FailedCount,
					"passed_count": report.PassedCount,
					"attempt":      attempt,
					"remaining":    maxRetries - attempt,
				})

				// For retry, we need to regenerate the prompt with fix request
				// Extract the specification from the original prompt
				var spec map[string]interface{}
				if request, ok := projectSpec["request"].(map[string]interface{}); ok {
					if specification, ok := request["specification"].(map[string]interface{}); ok {
						spec = specification
					}
				}

				if spec != nil {
					// Add fix request to the specification
					spec["_fix_request"] = map[string]interface{}{
						"attempt":  attempt,
						"failures": report.GetFailureSummary(),
						"instructions": "🔧 **CRITICAL VERIFICATION FAILURES - IMMEDIATE FIX REQUIRED**\n\n" +
							"❌ Your previous implementation FAILED verification checks. This is an automated retry - execute the fix immediately.\n\n" +
							"**FAILED VERIFICATIONS:**\n" + report.GetFailureSummary() + "\n\n" +
							"**CRITICAL INSTRUCTIONS - EXECUTE IMMEDIATELY:**\n" +
							"1. READ the verification failure output CAREFULLY - it shows EXACTLY what's wrong\n" +
							"2. CHECK the file constraints you were given - you may have IGNORED them\n" +
							"3. For example: If a constraint says 'NO newline', the file must NOT end with \\n\n" +
							"4. If a constraint says 'properly formatted', follow the FORMAT specified in verifications\n" +
							"5. FIX each failed file to satisfy BOTH the instructions AND the verification commands\n" +
							"6. VERIFY your fixes will pass by checking the verification command expectations\n\n" +
							"**IMPORTANT:**\n" +
							"- This is a NON-INTERACTIVE automated environment\n" +
							"- Do NOT ask questions or request clarification\n" +
							"- Do NOT enter design/planning mode\n" +
							"- Execute the fix IMMEDIATELY\n\n" +
							"This is attempt " + fmt.Sprintf("%d", attempt) + " of " + fmt.Sprintf("%d", maxRetries) + ". " +
							"You MUST fix these issues or the operation will FAIL.",
					}

					// Rebuild the prompt with fix request
					prompt := BuildProjectPrompt(spec, e.client.GetSystemPrompt())
					newPromptJSON, err := prompt.ToJSON()
					if err == nil {
						promptJSON = newPromptJSON
					}
				}

				// Wait before retrying (exponential backoff)
				waitTime := time.Duration(attempt) * time.Second
				tflog.Info(ctx, "Waiting before retry", map[string]interface{}{
					"wait_seconds": waitTime.Seconds(),
				})
				time.Sleep(waitTime)
			}
		} else {
			// No verifications, execution succeeded
			return execStatus, nil, nil
		}
	}

	// All retries exhausted
	tflog.Error(ctx, "Verifications failed after all retry attempts", map[string]interface{}{
		"max_retries":  maxRetries,
		"failed_count": lastReport.FailedCount,
		"passed_count": lastReport.PassedCount,
	})

	return lastStatus, lastReport, fmt.Errorf(
		"verification failed after %d attempts:\n%s",
		maxRetries,
		lastReport.GetFailureSummary(),
	)
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

	// Extract and save turn-by-turn breakdown if debug is enabled
	if e.debug {
		if err := e.SaveTurnBreakdown(ctx, outputDir); err != nil {
			tflog.Warn(ctx, "Failed to save turn breakdown", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	return nil
}

// savePromptJSON saves the prompt JSON sent to Claude for debugging
func (e *Executor) savePromptJSON(ctx context.Context, promptJSON string, outputDir string, attempt int) error {
	debugDir := filepath.Join(outputDir, ".debug")

	// Create debug directory if it doesn't exist
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("failed to create debug directory: %w", err)
	}

	timestamp := time.Now().Unix()
	promptPath := filepath.Join(debugDir, fmt.Sprintf("claude-prompt-attempt%d-%d.json", attempt, timestamp))

	// Write raw JSON to file
	if err := os.WriteFile(promptPath, []byte(promptJSON), 0644); err != nil {
		return fmt.Errorf("failed to write prompt JSON: %w", err)
	}

	// Parse and extract fix request if present
	var promptData map[string]interface{}
	if err := json.Unmarshal([]byte(promptJSON), &promptData); err == nil {
		// Create human-readable markdown version
		markdownPath := filepath.Join(debugDir, fmt.Sprintf("claude-prompt-attempt%d-%d.md", attempt, timestamp))
		if err := e.generatePromptMarkdown(promptData, markdownPath, attempt); err != nil {
			tflog.Warn(ctx, "Failed to generate prompt markdown", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	tflog.Debug(ctx, "Saved prompt JSON", map[string]interface{}{
		"prompt_path": promptPath,
		"attempt":     attempt,
	})

	return nil
}

// generatePromptMarkdown creates a human-readable markdown version of the prompt
func (e *Executor) generatePromptMarkdown(promptData map[string]interface{}, markdownPath string, attempt int) error {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# Claude Prompt - Attempt %d\n\n", attempt))
	content.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))

	// Check if this is a retry with fix request
	if request, ok := promptData["request"].(map[string]interface{}); ok {
		if spec, ok := request["specification"].(map[string]interface{}); ok {
			if fixReq, ok := spec["_fix_request"].(map[string]interface{}); ok {
				content.WriteString("## 🔧 FIX REQUEST (Retry)\n\n")
				content.WriteString("**This is a retry attempt after verification failure.**\n\n")

				if attemptNum, ok := fixReq["attempt"]; ok {
					content.WriteString(fmt.Sprintf("**Attempt:** %v\n\n", attemptNum))
				}

				if failures, ok := fixReq["failures"].(string); ok {
					content.WriteString("### Failed Verifications:\n\n")
					content.WriteString("```\n")
					content.WriteString(failures)
					content.WriteString("\n```\n\n")
				}

				if instructions, ok := fixReq["instructions"].(string); ok {
					content.WriteString("### Instructions to Claude:\n\n")
					content.WriteString(instructions)
					content.WriteString("\n\n")
				}
			}
		}
	}

	content.WriteString("## Full Prompt JSON\n\n")
	content.WriteString("See the `.json` file for complete prompt structure.\n")

	return os.WriteFile(markdownPath, []byte(content.String()), 0644)
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

// Turn represents a single conversation turn
type Turn struct {
	TurnNumber int                      `json:"turn_number"`
	User       []map[string]interface{} `json:"user_messages"`
	Assistant  []map[string]interface{} `json:"assistant_messages"`
	Timestamp  string                   `json:"timestamp"`
}

// SaveTurnBreakdown extracts turns from Claude session and saves them individually
func (e *Executor) SaveTurnBreakdown(ctx context.Context, outputDir string) error {
	// Find Claude session directory
	sessionDir, err := e.findClaudeSessionDir(outputDir)
	if err != nil {
		return fmt.Errorf("failed to find Claude session directory: %w", err)
	}

	// Find the conversation JSONL file (largest .jsonl file)
	jsonlPath, err := e.findConversationFile(sessionDir)
	if err != nil {
		return fmt.Errorf("failed to find conversation file: %w", err)
	}

	tflog.Debug(ctx, "Found Claude conversation file", map[string]interface{}{
		"jsonl_path": jsonlPath,
	})

	// Parse turns from JSONL
	turns, err := e.parseTurns(jsonlPath)
	if err != nil {
		return fmt.Errorf("failed to parse turns: %w", err)
	}

	tflog.Info(ctx, "Extracted conversation turns", map[string]interface{}{
		"turn_count": len(turns),
	})

	// Save each turn
	debugDir := filepath.Join(outputDir, ".debug")
	for i, turn := range turns {
		timestamp := time.Now().Unix()
		if err := e.saveTurn(turn, debugDir, timestamp); err != nil {
			tflog.Warn(ctx, "Failed to save turn", map[string]interface{}{
				"turn_number": i + 1,
				"error":       err.Error(),
			})
		}
	}

	return nil
}

// findClaudeSessionDir finds the Claude session directory for this project
func (e *Executor) findClaudeSessionDir(outputDir string) (string, error) {
	claudeHome := e.client.claudeHomeDir
	if claudeHome == "" {
		claudeHome = filepath.Join(os.Getenv("HOME"), ".claude")
	}

	projectsDir := filepath.Join(claudeHome, "projects")

	// Escape the output directory path to match Claude's naming convention
	// Claude replaces / with - and removes other special characters
	escapedPath := strings.ReplaceAll(outputDir, "/", "-")

	// Find directories that match the escaped path
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read projects directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(entry.Name(), escapedPath) {
			return filepath.Join(projectsDir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("no session directory found for output path: %s", outputDir)
}

// findConversationFile finds the main conversation JSONL file (largest file)
func (e *Executor) findConversationFile(sessionDir string) (string, error) {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return "", fmt.Errorf("failed to read session directory: %w", err)
	}

	var largestFile string
	var largestSize int64

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.Size() > largestSize {
				largestSize = info.Size()
				largestFile = filepath.Join(sessionDir, entry.Name())
			}
		}
	}

	if largestFile == "" {
		return "", fmt.Errorf("no conversation JSONL file found")
	}

	return largestFile, nil
}

// parseTurns parses the conversation JSONL into turns
func (e *Executor) parseTurns(jsonlPath string) ([]Turn, error) {
	data, err := os.ReadFile(jsonlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSONL file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var turns []Turn
	var currentUserMsgs []map[string]interface{}
	var currentAssistantMsgs []map[string]interface{}
	turnNumber := 0

	for _, line := range lines {
		if line == "" {
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "user":
			// If we have assistant messages from previous turn, save the turn
			if len(currentAssistantMsgs) > 0 {
				turnNumber++
				turn := Turn{
					TurnNumber: turnNumber,
					User:       currentUserMsgs,
					Assistant:  currentAssistantMsgs,
					Timestamp:  e.extractTimestamp(currentUserMsgs),
				}
				turns = append(turns, turn)

				// Reset for next turn
				currentUserMsgs = nil
				currentAssistantMsgs = nil
			}

			// Add user message to current turn
			currentUserMsgs = append(currentUserMsgs, msg)

		case "assistant":
			// Add assistant message to current turn
			currentAssistantMsgs = append(currentAssistantMsgs, msg)
		}
	}

	// Save last turn if exists
	if len(currentUserMsgs) > 0 || len(currentAssistantMsgs) > 0 {
		turnNumber++
		turn := Turn{
			TurnNumber: turnNumber,
			User:       currentUserMsgs,
			Assistant:  currentAssistantMsgs,
			Timestamp:  e.extractTimestamp(currentUserMsgs),
		}
		turns = append(turns, turn)
	}

	return turns, nil
}

// extractTimestamp extracts timestamp from first message
func (e *Executor) extractTimestamp(msgs []map[string]interface{}) string {
	if len(msgs) > 0 {
		if ts, ok := msgs[0]["timestamp"].(string); ok {
			return ts
		}
	}
	return time.Now().Format(time.RFC3339)
}

// saveTurn saves a single turn to JSON and MD files
func (e *Executor) saveTurn(turn Turn, debugDir string, timestamp int64) error {
	// Save JSON
	jsonPath := filepath.Join(debugDir, fmt.Sprintf("claude-turn-%d-%d.json", turn.TurnNumber, timestamp))
	jsonData, err := json.MarshalIndent(turn, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal turn JSON: %w", err)
	}

	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write turn JSON: %w", err)
	}

	// Save Markdown
	mdPath := filepath.Join(debugDir, fmt.Sprintf("claude-turn-%d-%d.md", turn.TurnNumber, timestamp))
	mdContent := e.generateTurnMarkdown(turn)

	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		return fmt.Errorf("failed to write turn MD: %w", err)
	}

	return nil
}

// generateTurnMarkdown creates a human-readable markdown version of a turn
func (e *Executor) generateTurnMarkdown(turn Turn) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# Turn %d\n\n", turn.TurnNumber))
	content.WriteString(fmt.Sprintf("**Timestamp**: %s\n\n", turn.Timestamp))

	// User messages
	content.WriteString("## User Messages\n\n")
	if len(turn.User) == 0 {
		content.WriteString("*No user messages in this turn*\n\n")
	} else {
		for i, msg := range turn.User {
			content.WriteString(fmt.Sprintf("### User Message %d\n\n", i+1))

			if ts, ok := msg["timestamp"].(string); ok {
				content.WriteString(fmt.Sprintf("**Time**: %s\n\n", ts))
			}

			if contentData, ok := msg["content"]; ok {
				content.WriteString("```\n")
				content.WriteString(fmt.Sprintf("%v", contentData))
				content.WriteString("\n```\n\n")
			}
		}
	}

	// Assistant messages
	content.WriteString("## Assistant Messages\n\n")
	if len(turn.Assistant) == 0 {
		content.WriteString("*No assistant messages in this turn*\n\n")
	} else {
		for i, msg := range turn.Assistant {
			content.WriteString(fmt.Sprintf("### Assistant Message %d\n\n", i+1))

			if ts, ok := msg["timestamp"].(string); ok {
				content.WriteString(fmt.Sprintf("**Time**: %s\n\n", ts))
			}

			if contentData, ok := msg["content"]; ok {
				content.WriteString("```\n")
				content.WriteString(fmt.Sprintf("%v", contentData))
				content.WriteString("\n```\n\n")
			}
		}
	}

	content.WriteString("---\n\n")
	content.WriteString(fmt.Sprintf("*Turn %d extracted from Claude session logs*\n", turn.TurnNumber))

	return content.String()
}
