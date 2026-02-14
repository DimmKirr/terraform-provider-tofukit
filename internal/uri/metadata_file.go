package uri

import (
	"fmt"
	"reflect"
)

// FileMetadata represents file resource metadata for Claude
type FileMetadata struct {
	Type              string `json:"type"`
	Name              string `json:"name"`
	Path              string `json:"path,omitempty"`
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
	fileData, exists := b.globalRegistry.GetFile(fileID)
	if !exists {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	// Extract fields using reflection to avoid import cycles
	metadata := &FileMetadata{
		Type: "file",
		Name: name,
	}

	// Use reflection to extract field values from FileResourceModel
	val := reflect.ValueOf(fileData)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		// Extract Path field
		if pathField := val.FieldByName("Path"); pathField.IsValid() {
			if pathVal := pathField.MethodByName("ValueString"); pathVal.IsValid() {
				results := pathVal.Call(nil)
				if len(results) > 0 {
					metadata.Path = results[0].String()
				}
			}
		}

		// Extract Description field
		if descField := val.FieldByName("Description"); descField.IsValid() {
			if descVal := descField.MethodByName("ValueString"); descVal.IsValid() {
				results := descVal.Call(nil)
				if len(results) > 0 {
					metadata.Description = results[0].String()
				}
			}
		}

		// Extract Content field for length and preview
		if contentField := val.FieldByName("Content"); contentField.IsValid() {
			if contentVal := contentField.MethodByName("ValueString"); contentVal.IsValid() {
				results := contentVal.Call(nil)
				if len(results) > 0 {
					content := results[0].String()
					metadata.ContentLength = len(content)
					if len(content) > 100 {
						metadata.ContentPreview = content[:100] + "..."
					} else if content != "" {
						metadata.ContentPreview = content
					}
				}
			}
		}

		// Check Instructions field
		if instrField := val.FieldByName("Instructions"); instrField.IsValid() {
			if instrField.Kind() == reflect.Slice {
				metadata.HasInstructions = instrField.Len() > 0
			}
		}

		// Check Verifications field
		if verifField := val.FieldByName("Verifications"); verifField.IsValid() {
			if verifField.Kind() == reflect.Slice {
				metadata.HasVerifications = verifField.Len() > 0
				metadata.VerificationCount = verifField.Len()
			}
		}
	}

	return metadata, nil
}
