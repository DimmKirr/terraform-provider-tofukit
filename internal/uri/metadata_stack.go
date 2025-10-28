package uri

import (
	"fmt"
)

// StackMetadata represents stack resource metadata for Claude
type StackMetadata struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Files       []string `json:"files"`
	FileCount   int      `json:"file_count"`
	Kits        []string `json:"kits,omitempty"`
	Features    []string `json:"features,omitempty"`
}

// getStackMetadata extracts metadata from a stack resource
func (b *RegistryBuilder) getStackMetadata(name string) (*StackMetadata, error) {
	// Lookup stack in registry by ID
	stackID := fmt.Sprintf("stack.%s", name)
	_, exists := b.globalRegistry.GetStack(stackID)
	if !exists {
		return nil, fmt.Errorf("stack not found: %s", name)
	}

	// Return basic metadata to avoid import cycle
	metadata := &StackMetadata{
		Type:        "stack",
		Name:        name,
		Description: "",
		Files:       []string{},
		Kits:        []string{},
		Features:    []string{},
		FileCount:   0,
	}

	return metadata, nil
}
