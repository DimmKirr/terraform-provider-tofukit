package uri

import (
	"fmt"

	"github.com/tofukit/opentofu-provider-tofukit/internal/registry"
)

// RegistryBuilder creates resource_registry from URIs
type RegistryBuilder struct {
	globalRegistry *registry.Registry
}

// NewRegistryBuilder creates a new registry builder
func NewRegistryBuilder(reg *registry.Registry) *RegistryBuilder {
	return &RegistryBuilder{globalRegistry: reg}
}

// BuildRegistry creates resource_registry map from URIs
// Returns a map where keys are URIs and values are resource metadata
func (b *RegistryBuilder) BuildRegistry(uris []string) (map[string]interface{}, error) {
	resourceRegistry := make(map[string]interface{})
	scanner := NewScanner()

	for _, uri := range uris {
		components, err := scanner.ParseURI(uri)
		if err != nil {
			return nil, fmt.Errorf("invalid URI %s: %w", uri, err)
		}

		metadata, err := b.getResourceMetadata(components)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve URI %s: %w", uri, err)
		}

		resourceRegistry[uri] = metadata
	}

	return resourceRegistry, nil
}

// getResourceMetadata routes to the appropriate metadata extractor based on type
func (b *RegistryBuilder) getResourceMetadata(c URIComponents) (interface{}, error) {
	switch c.Type {
	case "feature":
		return b.getFeatureMetadata(c.Name)
	case "file":
		return b.getFileMetadata(c.Name)
	case "stack":
		return b.getStackMetadata(c.Name)
	case "kit/language":
		return b.getKitMetadata("language", c.Name)
	case "kit/framework":
		return b.getKitMetadata("framework", c.Name)
	case "kit/tool":
		return b.getKitMetadata("tool", c.Name)
	case "kit/methodology":
		return b.getKitMetadata("methodology", c.Name)
	case "kit/style":
		return b.getKitMetadata("style", c.Name)
	case "kit/infrastructure":
		return b.getKitMetadata("infrastructure", c.Name)
	case "kit/integration":
		return b.getKitMetadata("integration", c.Name)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", c.Type)
	}
}
