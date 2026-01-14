package uri

import (
	"fmt"
	"reflect"
)

// FeatureMetadata represents lean feature resource metadata for Claude
// Intentionally excludes prompts, constraints, instructions for token efficiency
type FeatureMetadata struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Files       []string `json:"files"`
	FileCount   int      `json:"file_count"`
}

// getFeatureMetadata extracts metadata from a feature resource
func (b *RegistryBuilder) getFeatureMetadata(name string) (*FeatureMetadata, error) {
	// Lookup feature in registry by ID
	featureID := fmt.Sprintf("feature.%s", name)
	featureData, exists := b.globalRegistry.GetFeature(featureID)
	if !exists {
		return nil, fmt.Errorf("feature not found: %s", name)
	}

	// Extract fields using reflection to avoid import cycles
	metadata := &FeatureMetadata{
		Type:  "feature",
		Name:  name,
		Files: []string{},
	}

	// Use reflection to extract field values from FeatureResourceModel
	val := reflect.ValueOf(featureData)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		// Extract Description field
		if descField := val.FieldByName("Description"); descField.IsValid() {
			if descVal := descField.MethodByName("ValueString"); descVal.IsValid() {
				results := descVal.Call(nil)
				if len(results) > 0 {
					metadata.Description = results[0].String()
				}
			}
		}

		// Extract Files field (map of file paths)
		if filesField := val.FieldByName("Files"); filesField.IsValid() {
			// Files is types.Map, iterate its keys
			if elemField := filesField.MethodByName("Elements"); elemField.IsValid() {
				// Get map elements
				results := elemField.Call(nil)
				if len(results) > 0 {
					elemMap := results[0]
					if elemMap.Kind() == reflect.Map {
						for _, key := range elemMap.MapKeys() {
							metadata.Files = append(metadata.Files, key.String())
						}
						metadata.FileCount = elemMap.Len()
					}
				}
			}
		}
	}

	return metadata, nil
}
