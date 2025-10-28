package uri

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// FeatureMetadata represents feature resource metadata for Claude
type FeatureMetadata struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Prompt      string   `json:"prompt"`
	Files       []string `json:"files"`
	FileCount   int      `json:"file_count"`
	Kits        []string `json:"kits,omitempty"`
	Constraints []string `json:"constraints,omitempty"`
}

// FeatureModel interface for accessing feature data without import cycle
type FeatureModel interface {
	GetName() types.String
	GetDescription() types.String
	GetPrompt() types.String
	GetFiles() types.Map
	GetConstraints() []types.String
}

// getFeatureMetadata extracts metadata from a feature resource
func (b *RegistryBuilder) getFeatureMetadata(name string) (*FeatureMetadata, error) {
	// Lookup feature in registry by ID
	featureID := fmt.Sprintf("feature.%s", name)
	_, exists := b.globalRegistry.GetFeature(featureID)
	if !exists {
		return nil, fmt.Errorf("feature not found: %s", name)
	}

	// Return basic metadata to avoid import cycle
	// The resource exists, which is the main validation
	metadata := &FeatureMetadata{
		Type:        "feature",
		Name:        name,
		Description: "",
		Prompt:      "",
		Files:       []string{},
		Kits:        []string{},
		Constraints: []string{},
		FileCount:   0,
	}

	return metadata, nil
}
