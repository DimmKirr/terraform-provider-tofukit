package files

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// FileSource represents the source/precedence level of a file
type FileSource int

const (
	SourceLanguage FileSource = iota // Lowest precedence
	SourceFramework
	SourceStack
	SourceFeature // Feature files (between stack and project)
	SourceProject // Highest precedence
)

// FileWithSource wraps a file with its source information
type FileWithSource struct {
	File   schemas.FileModelWithPath
	Source FileSource
}

// Merger handles merging files from multiple sources with precedence rules
type Merger struct {
	files     map[string]FileWithSource // key is file path, for fast lookups
	fileOrder []string                  // maintains insertion order of file paths
}

// NewMerger creates a new file merger
func NewMerger() *Merger {
	return &Merger{
		files:     make(map[string]FileWithSource),
		fileOrder: make([]string, 0),
	}
}

// AddFiles adds files from a specific source
func (m *Merger) AddFiles(ctx context.Context, files []schemas.FileModelWithPath, source FileSource) {
	sourceName := m.getSourceName(source)

	for _, file := range files {
		if file.Path == "" {
			continue
		}
		path := file.Path

		// Check if this path already exists
		if existing, exists := m.files[path]; exists {
			// Only replace if new source has higher precedence
			if source > existing.Source {
				tflog.Debug(ctx, "Overriding file with higher precedence", map[string]interface{}{
					"path":       path,
					"old_source": m.getSourceName(existing.Source),
					"new_source": sourceName,
				})
				m.files[path] = FileWithSource{
					File:   file,
					Source: source,
				}
				// Note: Don't add to fileOrder again - already exists
			} else {
				tflog.Debug(ctx, "Keeping existing file with higher precedence", map[string]interface{}{
					"path":            path,
					"existing_source": m.getSourceName(existing.Source),
					"ignored_source":  sourceName,
				})
			}
		} else {
			// New path, add it
			tflog.Debug(ctx, "Adding new file", map[string]interface{}{
				"path":   path,
				"source": sourceName,
			})
			m.files[path] = FileWithSource{
				File:   file,
				Source: source,
			}
			// Track insertion order
			m.fileOrder = append(m.fileOrder, path)
		}
	}
}

// GetMergedFiles returns the final merged list of files in insertion order
func (m *Merger) GetMergedFiles() []schemas.FileModelWithPath {
	result := make([]schemas.FileModelWithPath, 0, len(m.files))

	// Iterate in insertion order to preserve the order from the config
	// This is critical for Terraform plan/apply consistency
	for _, path := range m.fileOrder {
		if fileWithSource, exists := m.files[path]; exists {
			result = append(result, fileWithSource.File)
		}
	}

	return result
}

// GetMergedFilesWithSources returns files with their source information (for debugging)
func (m *Merger) GetMergedFilesWithSources() map[string]FileWithSource {
	// Return a copy to prevent external modification
	result := make(map[string]FileWithSource, len(m.files))
	for k, v := range m.files {
		result[k] = v
	}
	return result
}

// getSourceName returns a human-readable name for the source
func (m *Merger) getSourceName(source FileSource) string {
	switch source {
	case SourceLanguage:
		return "language"
	case SourceFramework:
		return "framework"
	case SourceStack:
		return "stack"
	case SourceFeature:
		return "feature"
	case SourceProject:
		return "project"
	default:
		return fmt.Sprintf("unknown(%d)", source)
	}
}

// MergeContent provides content merging capabilities for special cases
// This could be extended to support markers like MERGE:APPEND, MERGE:PREPEND, etc.
func MergeContent(existing, new string, mergeStrategy string) string {
	switch mergeStrategy {
	case "append":
		if existing == "" {
			return new
		}
		return existing + "\n" + new
	case "prepend":
		if existing == "" {
			return new
		}
		return new + "\n" + existing
	case "replace":
		fallthrough
	default:
		return new
	}
}
