package uri

import (
	"fmt"
)

// KitMetadata represents kit resource metadata for Claude
type KitMetadata struct {
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// getKitMetadata extracts metadata from a kit resource (language, framework, tool, etc.)
func (b *RegistryBuilder) getKitMetadata(subtype, name string) (*KitMetadata, error) {
	// Lookup kit in registry by ID
	kitID := fmt.Sprintf("%s.%s", subtype, name)

	_, exists := b.globalRegistry.GetComponent(kitID)
	if !exists {
		return nil, fmt.Errorf("%s kit not found: %s", subtype, name)
	}

	// Return basic metadata to avoid import cycle
	return &KitMetadata{
		Type:        "kit",
		Subtype:     subtype,
		Name:        name,
		Version:     "",
		Description: "",
	}, nil
}
