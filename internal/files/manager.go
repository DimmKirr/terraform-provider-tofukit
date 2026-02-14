package files

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// VerificationResult represents the result of a single verification
type VerificationResult struct {
	FilePath string
	Command  string
	Expected string
	Actual   string
	Passed   bool
	Error    error
}

// VerificationReport contains results from all verifications
type VerificationReport struct {
	AllPassed   bool
	FailedCount int
	PassedCount int
	Results     []VerificationResult
}

// GetFailureSummary returns a formatted string of all failures for Claude
func (r *VerificationReport) GetFailureSummary() string {
	var failures []string
	for _, result := range r.Results {
		if !result.Passed {
			if result.Error != nil {
				failures = append(failures, fmt.Sprintf(
					"❌ File '%s': Command '%s' failed with error: %v (output: %s)",
					result.FilePath, result.Command, result.Error, result.Actual,
				))
			} else {
				failures = append(failures, fmt.Sprintf(
					"❌ File '%s': Command '%s' expected '%s' but got '%s'",
					result.FilePath, result.Command, result.Expected, result.Actual,
				))
			}
		}
	}
	return strings.Join(failures, "\n")
}

// Manager handles file operations
type Manager struct {
	BaseDir string
}

// NewManager creates a new file manager
func NewManager(baseDir string) *Manager {
	return &Manager{
		BaseDir: baseDir,
	}
}

// ProcessFileChanges handles file additions and removals
func (m *Manager) ProcessFileChanges(ctx context.Context, oldFiles, newFiles []schemas.FileModelWithPath) error {
	// Create a map of new files for quick lookup
	newFileMap := make(map[string]schemas.FileModelWithPath)
	for _, file := range newFiles {
		if file.Path != "" {
			newFileMap[file.Path] = file
		}
	}

	// Check for removed files and delete their files
	for _, oldFile := range oldFiles {
		if oldFile.Path == "" {
			continue
		}

		// If this file is not in the new list, it was removed
		if _, exists := newFileMap[oldFile.Path]; !exists {
			fullPath := filepath.Join(m.BaseDir, oldFile.Path)
			tflog.Info(ctx, fmt.Sprintf("Removing file: %s", fullPath))

			// Remove the file
			if err := os.Remove(fullPath); err != nil {
				// Log the error but don't fail if file doesn't exist
				if !os.IsNotExist(err) {
					tflog.Warn(ctx, fmt.Sprintf("Failed to remove file %s: %v", fullPath, err))
				}
			} else {
				tflog.Info(ctx, fmt.Sprintf("Successfully removed file: %s", fullPath))
			}

			// Clean up empty parent directories
			m.cleanupEmptyDirs(ctx, filepath.Dir(fullPath))
		}
	}

	// Create or update files for new/modified files
	for _, file := range newFiles {
		if err := m.WriteFile(ctx, file); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}
	}

	return nil
}

// WriteFile writes a single file
func (m *Manager) WriteFile(ctx context.Context, file schemas.FileModelWithPath) error {
	if file.Path == "" {
		return nil // Skip empty paths
	}

	fullPath := filepath.Join(m.BaseDir, file.Path)

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
	}

	// Write the file content
	content := file.Content.ValueString()
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Wrote file: %s", fullPath))
	return nil
}

// RemoveAllFiles removes all files
func (m *Manager) RemoveAllFiles(ctx context.Context, files []schemas.FileModelWithPath) {
	for _, file := range files {
		if file.Path == "" {
			continue
		}

		fullPath := filepath.Join(m.BaseDir, file.Path)
		tflog.Debug(ctx, fmt.Sprintf("Removing file: %s", fullPath))

		if err := os.Remove(fullPath); err != nil {
			if !os.IsNotExist(err) {
				tflog.Warn(ctx, fmt.Sprintf("Failed to remove file %s: %v", fullPath, err))
			}
		}

		// Clean up empty parent directories
		m.cleanupEmptyDirs(ctx, filepath.Dir(fullPath))
	}
}

// cleanupEmptyDirs removes empty parent directories up to the base directory
func (m *Manager) cleanupEmptyDirs(ctx context.Context, dir string) {
	// Don't remove the base directory or anything outside it
	if dir == m.BaseDir || !hasPrefix(dir, m.BaseDir) {
		return
	}

	// Check if directory is empty
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	if len(entries) == 0 {
		// Directory is empty, remove it
		if err := os.Remove(dir); err == nil {
			tflog.Debug(ctx, fmt.Sprintf("Removed empty directory: %s", dir))
			// Recursively check parent
			m.cleanupEmptyDirs(ctx, filepath.Dir(dir))
		}
	}
}

// hasPrefix checks if a path has a given prefix (compatibility helper)
func hasPrefix(path, prefix string) bool {
	// Clean both paths to ensure consistent comparison
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)

	// Check if path starts with prefix
	if path == prefix {
		return true
	}

	// Ensure prefix ends with separator for accurate prefix matching
	if !os.IsPathSeparator(prefix[len(prefix)-1]) {
		prefix = prefix + string(os.PathSeparator)
	}

	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

// VerifyFiles checks that all files exist with correct content
func (m *Manager) VerifyFiles(ctx context.Context, files []schemas.FileModelWithPath) error {
	var missingFiles []string
	var wrongContent []string

	for _, file := range files {
		if file.Path == "" {
			continue
		}

		fullPath := filepath.Join(m.BaseDir, file.Path)

		// Check if file exists
		fileContent, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				missingFiles = append(missingFiles, file.Path)
				tflog.Warn(ctx, "File file missing", map[string]interface{}{
					"path": fullPath,
				})
			} else {
				return fmt.Errorf("error reading file %s: %w", file.Path, err)
			}
			continue
		}

		// Check content matches
		expectedContent := file.Content.ValueString()
		if string(fileContent) != expectedContent {
			wrongContent = append(wrongContent, file.Path)
			tflog.Warn(ctx, "File content mismatch", map[string]interface{}{
				"path":            fullPath,
				"expected_length": len(expectedContent),
				"actual_length":   len(fileContent),
			})
		}
	}

	// Report all issues
	if len(missingFiles) > 0 || len(wrongContent) > 0 {
		errMsg := "File verification failed:"
		if len(missingFiles) > 0 {
			errMsg += fmt.Sprintf("\n  Missing files: %v", missingFiles)
		}
		if len(wrongContent) > 0 {
			errMsg += fmt.Sprintf("\n  Wrong content: %v", wrongContent)
		}
		return fmt.Errorf("%s", errMsg)
	}

	tflog.Info(ctx, "All files verified successfully", map[string]interface{}{
		"file_count": len(files),
	})
	return nil
}

// RunVerifications executes verification commands for all files and returns structured results
func (m *Manager) RunVerifications(ctx context.Context, files []schemas.FileModelWithPath) (*VerificationReport, error) {
	report := &VerificationReport{
		Results: []VerificationResult{},
	}

	// TEST HOOK: Force first verification to fail for testing retry mechanism
	// This allows us to test the retry flow without Claude being able to predict/avoid the failure
	if os.Getenv("TOFUKIT_TEST_FORCE_VERIFY_FAIL_FIRST") == "true" {
		counterFile := "/tmp/tofukit_test_verify_counter"
		count := 0

		// Read current counter
		if data, err := os.ReadFile(counterFile); err == nil {
			count, _ = strconv.Atoi(strings.TrimSpace(string(data)))
		}

		// First call - force failure
		if count == 0 {
			tflog.Warn(ctx, "TEST HOOK: Forcing first verification to fail", map[string]interface{}{
				"hook": "TOFUKIT_TEST_FORCE_VERIFY_FAIL_FIRST",
			})

			// Increment counter for next call
			os.WriteFile(counterFile, []byte("1"), 0644)

			// Return a forced failure for the first file
			if len(files) > 0 {
				report.Results = append(report.Results, VerificationResult{
					FilePath: files[0].Path,
					Command:  "TEST_HOOK_FORCED_FAILURE",
					Expected: "PASS",
					Actual:   "FAIL",
					Passed:   false,
				})
				report.FailedCount = 1
				report.AllPassed = false

				return report, nil
			}
		}

		tflog.Info(ctx, "TEST HOOK: Counter > 0, running normal verification", map[string]interface{}{
			"count": count,
		})
	}

	for _, file := range files {
		if file.Path == "" || len(file.Verifications) == 0 {
			continue
		}

		tflog.Debug(ctx, "Running verifications for file", map[string]interface{}{
			"path":               file.Path,
			"verification_count": len(file.Verifications),
		})

		for _, verification := range file.Verifications {
			command := verification.Command.ValueString()
			expect := ""
			if !verification.Expect.IsNull() && !verification.Expect.IsUnknown() {
				expect = verification.Expect.ValueString()
			}

			if command == "" {
				continue
			}

			result := VerificationResult{
				FilePath: file.Path,
				Command:  command,
				Expected: expect,
			}

			// Execute the verification command in the base directory
			cmd := exec.Command("sh", "-c", command)
			cmd.Dir = m.BaseDir

			output, err := cmd.CombinedOutput()
			// IMPORTANT: Do NOT use TrimSpace - it removes trailing newlines that we need to detect!
			// Only trim the final newline that shell commands naturally add
			result.Actual = strings.TrimSuffix(string(output), "\n")
			result.Error = err

			tflog.Debug(ctx, "Verification command executed", map[string]interface{}{
				"file":    file.Path,
				"command": command,
				"output":  result.Actual,
				"expect":  expect,
				"error":   err,
			})

			// Determine if verification passed
			if err != nil {
				result.Passed = false
				report.FailedCount++
				tflog.Warn(ctx, "Verification command failed", map[string]interface{}{
					"file":    file.Path,
					"command": command,
					"error":   err.Error(),
					"output":  result.Actual,
				})
			} else if expect != "" && result.Actual != expect {
				result.Passed = false
				report.FailedCount++
				tflog.Warn(ctx, "Verification output mismatch", map[string]interface{}{
					"file":     file.Path,
					"command":  command,
					"output":   result.Actual,
					"expected": expect,
				})
			} else {
				result.Passed = true
				report.PassedCount++
				tflog.Info(ctx, "Verification passed", map[string]interface{}{
					"file":    file.Path,
					"command": command,
				})
			}

			report.Results = append(report.Results, result)
		}
	}

	report.AllPassed = report.FailedCount == 0

	tflog.Info(ctx, "Verification summary", map[string]interface{}{
		"total":  report.PassedCount + report.FailedCount,
		"passed": report.PassedCount,
		"failed": report.FailedCount,
	})

	return report, nil
}
