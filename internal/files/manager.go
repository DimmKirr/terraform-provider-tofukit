package files

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

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
func (m *Manager) ProcessFileChanges(ctx context.Context, oldFiles, newFiles []schemas.FileModel) error {
	// Create a map of new files for quick lookup
	newFileMap := make(map[string]schemas.FileModel)
	for _, file := range newFiles {
		path := file.Path.ValueString()
		if path != "" {
			newFileMap[path] = file
		}
	}

	// Check for removed files and delete their files
	for _, oldFile := range oldFiles {
		oldPath := oldFile.Path.ValueString()
		if oldPath == "" {
			continue
		}

		// If this file is not in the new list, it was removed
		if _, exists := newFileMap[oldPath]; !exists {
			fullPath := filepath.Join(m.BaseDir, oldPath)
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
			return fmt.Errorf("failed to write file %s: %w", file.Path.ValueString(), err)
		}
	}

	return nil
}

// WriteFile writes a single file
func (m *Manager) WriteFile(ctx context.Context, file schemas.FileModel) error {
	path := file.Path.ValueString()
	if path == "" {
		return nil // Skip empty paths
	}

	fullPath := filepath.Join(m.BaseDir, path)

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
func (m *Manager) RemoveAllFiles(ctx context.Context, files []schemas.FileModel) {
	for _, file := range files {
		path := file.Path.ValueString()
		if path == "" {
			continue
		}

		fullPath := filepath.Join(m.BaseDir, path)
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
func (m *Manager) VerifyFiles(ctx context.Context, files []schemas.FileModel) error {
	var missingFiles []string
	var wrongContent []string

	for _, file := range files {
		path := file.Path.ValueString()
		if path == "" {
			continue
		}

		fullPath := filepath.Join(m.BaseDir, path)

		// Check if file exists
		fileContent, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				missingFiles = append(missingFiles, path)
				tflog.Warn(ctx, "File file missing", map[string]interface{}{
					"path": fullPath,
				})
			} else {
				return fmt.Errorf("error reading file %s: %w", path, err)
			}
			continue
		}

		// Check content matches
		expectedContent := file.Content.ValueString()
		if string(fileContent) != expectedContent {
			wrongContent = append(wrongContent, path)
			tflog.Warn(ctx, "File content mismatch", map[string]interface{}{
				"path": fullPath,
				"expected_length": len(expectedContent),
				"actual_length": len(fileContent),
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
		return fmt.Errorf(errMsg)
	}

	tflog.Info(ctx, "All files verified successfully", map[string]interface{}{
		"file_count": len(files),
	})
	return nil
}

// RunVerifications executes verification commands for all files and checks expected output
func (m *Manager) RunVerifications(ctx context.Context, files []schemas.FileModel) error {
	var verificationErrors []string
	totalVerifications := 0
	passedVerifications := 0

	for _, file := range files {
		path := file.Path.ValueString()
		if path == "" || len(file.Verification) == 0 {
			continue
		}

		tflog.Debug(ctx, "Running verifications for file", map[string]interface{}{
			"path": path,
			"verification_count": len(file.Verification),
		})

		for i, verification := range file.Verification {
			command := verification.Command.ValueString()
			expect := verification.Expect.ValueString()

			if command == "" {
				continue
			}

			totalVerifications++

			// Execute the verification command in the base directory
			cmd := exec.Command("sh", "-c", command)
			cmd.Dir = m.BaseDir

			output, err := cmd.CombinedOutput()
			outputStr := strings.TrimSpace(string(output))

			tflog.Debug(ctx, "Verification command executed", map[string]interface{}{
				"file": path,
				"verification_index": i,
				"command": command,
				"output": outputStr,
				"expect": expect,
				"error": err,
			})

			// Check if command failed
			if err != nil {
				errMsg := fmt.Sprintf("File '%s' verification #%d failed: command '%s' returned error: %v (output: %s)",
					path, i+1, command, err, outputStr)
				verificationErrors = append(verificationErrors, errMsg)
				tflog.Warn(ctx, "Verification command failed", map[string]interface{}{
					"file": path,
					"command": command,
					"error": err.Error(),
					"output": outputStr,
				})
				continue
			}

			// Check if output matches expected
			if !strings.Contains(outputStr, expect) {
				errMsg := fmt.Sprintf("File '%s' verification #%d failed: command '%s' output '%s' does not contain expected '%s'",
					path, i+1, command, outputStr, expect)
				verificationErrors = append(verificationErrors, errMsg)
				tflog.Warn(ctx, "Verification output mismatch", map[string]interface{}{
					"file": path,
					"command": command,
					"output": outputStr,
					"expected": expect,
				})
			} else {
				passedVerifications++
				tflog.Info(ctx, "Verification passed", map[string]interface{}{
					"file": path,
					"command": command,
				})
			}
		}
	}

	// Report results
	if len(verificationErrors) > 0 {
		tflog.Error(ctx, "Verification failures detected", map[string]interface{}{
			"total": totalVerifications,
			"passed": passedVerifications,
			"failed": len(verificationErrors),
		})

		return fmt.Errorf("verification failed (%d/%d passed):\n%s",
			passedVerifications, totalVerifications,
			strings.Join(verificationErrors, "\n"))
	}

	if totalVerifications > 0 {
		tflog.Info(ctx, "All verifications passed", map[string]interface{}{
			"total": totalVerifications,
		})
	}

	return nil
}