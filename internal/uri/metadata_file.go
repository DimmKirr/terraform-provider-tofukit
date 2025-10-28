package uri

import (
	"fmt"
)

// FileMetadata represents file resource metadata for Claude
type FileMetadata struct {
	Type              string `json:"type"`
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	ContentPreview    string `json:"content_preview,omitempty"`
	ContentLength     int    `json:"content_length"`
	HasInstructions   bool   `json:"has_instructions"`
	HasVerifications  bool   `json:"has_verifications"`
	VerificationCount int    `json:"verification_count"`
}

// getFileMetadata extracts metadata from a file resource
func (b *RegistryBuilder) getFileMetadata(name string) (*FileMetadata, error) {
	// Lookup file in registry by ID
	fileID := fmt.Sprintf("file.%s", name)
	_, exists := b.globalRegistry.GetFile(fileID)
	if !exists {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	// Return basic metadata to avoid import cycle
	metadata := &FileMetadata{
		Type:              "file",
		Name:              name,
		Description:       "",
		ContentLength:     0,
		HasInstructions:   false,
		HasVerifications:  false,
		VerificationCount: 0,
	}

	return metadata, nil
}
