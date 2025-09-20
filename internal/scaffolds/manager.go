package scaffolds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// Manager handles scaffold file operations
type Manager struct {
	BaseDir string
}

// NewManager creates a new scaffold manager
func NewManager(baseDir string) *Manager {
	return &Manager{
		BaseDir: baseDir,
	}
}

// ProcessScaffoldChanges handles scaffold additions and removals
func (m *Manager) ProcessScaffoldChanges(ctx context.Context, oldScaffolds, newScaffolds []schemas.ScaffoldModel) error {
	// Create a map of new scaffolds for quick lookup
	newScaffoldMap := make(map[string]schemas.ScaffoldModel)
	for _, scaffold := range newScaffolds {
		path := scaffold.Path.ValueString()
		if path != "" {
			newScaffoldMap[path] = scaffold
		}
	}

	// Check for removed scaffolds and delete their files
	for _, oldScaffold := range oldScaffolds {
		oldPath := oldScaffold.Path.ValueString()
		if oldPath == "" {
			continue
		}

		// If this scaffold is not in the new list, it was removed
		if _, exists := newScaffoldMap[oldPath]; !exists {
			fullPath := filepath.Join(m.BaseDir, oldPath)
			tflog.Info(ctx, fmt.Sprintf("Removing scaffold file: %s", fullPath))

			// Remove the file
			if err := os.Remove(fullPath); err != nil {
				// Log the error but don't fail if file doesn't exist
				if !os.IsNotExist(err) {
					tflog.Warn(ctx, fmt.Sprintf("Failed to remove scaffold file %s: %v", fullPath, err))
				}
			} else {
				tflog.Info(ctx, fmt.Sprintf("Successfully removed scaffold file: %s", fullPath))
			}

			// Clean up empty parent directories
			m.cleanupEmptyDirs(ctx, filepath.Dir(fullPath))
		}
	}

	// Create or update files for new/modified scaffolds
	for _, scaffold := range newScaffolds {
		if err := m.WriteScaffold(ctx, scaffold); err != nil {
			return fmt.Errorf("failed to write scaffold %s: %w", scaffold.Path.ValueString(), err)
		}
	}

	return nil
}

// WriteScaffold writes a single scaffold file
func (m *Manager) WriteScaffold(ctx context.Context, scaffold schemas.ScaffoldModel) error {
	path := scaffold.Path.ValueString()
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
	content := scaffold.Content.ValueString()
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Wrote scaffold file: %s", fullPath))
	return nil
}

// RemoveAllScaffolds removes all scaffold files
func (m *Manager) RemoveAllScaffolds(ctx context.Context, scaffolds []schemas.ScaffoldModel) {
	for _, scaffold := range scaffolds {
		path := scaffold.Path.ValueString()
		if path == "" {
			continue
		}

		fullPath := filepath.Join(m.BaseDir, path)
		tflog.Debug(ctx, fmt.Sprintf("Removing scaffold file: %s", fullPath))

		if err := os.Remove(fullPath); err != nil {
			if !os.IsNotExist(err) {
				tflog.Warn(ctx, fmt.Sprintf("Failed to remove scaffold file %s: %v", fullPath, err))
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