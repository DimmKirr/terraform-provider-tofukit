package scaffolds

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// ScaffoldSource represents the source/precedence level of a scaffold
type ScaffoldSource int

const (
	SourceLanguage ScaffoldSource = iota // Lowest precedence
	SourceFramework
	SourceStack
	SourceProject // Highest precedence
)

// ScaffoldWithSource wraps a scaffold with its source information
type ScaffoldWithSource struct {
	Scaffold schemas.ScaffoldModel
	Source   ScaffoldSource
}

// Merger handles merging scaffolds from multiple sources with precedence rules
type Merger struct {
	scaffolds map[string]ScaffoldWithSource // key is file path
}

// NewMerger creates a new scaffold merger
func NewMerger() *Merger {
	return &Merger{
		scaffolds: make(map[string]ScaffoldWithSource),
	}
}

// AddScaffolds adds scaffolds from a specific source
func (m *Merger) AddScaffolds(ctx context.Context, scaffolds []schemas.ScaffoldModel, source ScaffoldSource) {
	sourceName := m.getSourceName(source)

	for _, scaffold := range scaffolds {
		path := scaffold.Path.ValueString()
		if path == "" {
			continue
		}

		// Check if this path already exists
		if existing, exists := m.scaffolds[path]; exists {
			// Only replace if new source has higher precedence
			if source > existing.Source {
				tflog.Debug(ctx, "Overriding scaffold with higher precedence", map[string]interface{}{
					"path":        path,
					"old_source":  m.getSourceName(existing.Source),
					"new_source":  sourceName,
				})
				m.scaffolds[path] = ScaffoldWithSource{
					Scaffold: scaffold,
					Source:   source,
				}
			} else {
				tflog.Debug(ctx, "Keeping existing scaffold with higher precedence", map[string]interface{}{
					"path":           path,
					"existing_source": m.getSourceName(existing.Source),
					"ignored_source":  sourceName,
				})
			}
		} else {
			// New path, add it
			tflog.Debug(ctx, "Adding new scaffold", map[string]interface{}{
				"path":   path,
				"source": sourceName,
			})
			m.scaffolds[path] = ScaffoldWithSource{
				Scaffold: scaffold,
				Source:   source,
			}
		}
	}
}

// GetMergedScaffolds returns the final merged list of scaffolds
func (m *Merger) GetMergedScaffolds() []schemas.ScaffoldModel {
	result := make([]schemas.ScaffoldModel, 0, len(m.scaffolds))

	for _, scaffoldWithSource := range m.scaffolds {
		result = append(result, scaffoldWithSource.Scaffold)
	}

	return result
}

// GetMergedScaffoldsWithSources returns scaffolds with their source information (for debugging)
func (m *Merger) GetMergedScaffoldsWithSources() map[string]ScaffoldWithSource {
	// Return a copy to prevent external modification
	result := make(map[string]ScaffoldWithSource, len(m.scaffolds))
	for k, v := range m.scaffolds {
		result[k] = v
	}
	return result
}

// getSourceName returns a human-readable name for the source
func (m *Merger) getSourceName(source ScaffoldSource) string {
	switch source {
	case SourceLanguage:
		return "language"
	case SourceFramework:
		return "framework"
	case SourceStack:
		return "stack"
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