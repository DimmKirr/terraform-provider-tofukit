package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/registry"
)

// BaseComponentModel contains fields common to all component types
type BaseComponentModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Version     types.String `tfsdk:"version"`
}

// BaseComponent provides common functionality for all component resources
type BaseComponent struct {
	Kind         string
	ProviderData interface{}
}

// Configure adds the provider configured data to the resource
func (r *BaseComponent) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {

	// Prevent panic if the provider is not configured
	if req.ProviderData == nil {
		return
	}

	r.ProviderData = req.ProviderData
}

// SaveToRegistry saves the component to the provider's registry
func (r *BaseComponent) SaveToRegistry(ctx context.Context, id string, data interface{}) {
	tflog.Debug(ctx, fmt.Sprintf("Saving %s component to registry: %s", r.Kind, id))

	if provData, ok := r.ProviderData.(interface {
		GetRegistry() *registry.Registry
	}); ok {
		registry := provData.GetRegistry()
		switch r.Kind {
		case "stack":
			registry.SetStack(id, data)
		default:
			registry.SetComponent(id, data)
		}
	}
}

// GetFromRegistry retrieves a component from the registry
func (r *BaseComponent) GetFromRegistry(ctx context.Context, id string) (interface{}, bool) {
	tflog.Debug(ctx, fmt.Sprintf("Getting %s component from registry: %s", r.Kind, id))

	if provData, ok := r.ProviderData.(interface {
		GetRegistry() *registry.Registry
	}); ok {
		registry := provData.GetRegistry()
		switch r.Kind {
		case "stack":
			return registry.GetStack(id)
		default:
			return registry.GetComponent(id)
		}
	}
	return nil, false
}

// RemoveFromRegistry removes a component from the registry
func (r *BaseComponent) RemoveFromRegistry(ctx context.Context, id string) {
	tflog.Debug(ctx, fmt.Sprintf("Removing %s component from registry: %s", r.Kind, id))

	if provData, ok := r.ProviderData.(interface {
		GetRegistry() *registry.Registry
	}); ok {
		registry := provData.GetRegistry()
		switch r.Kind {
		case "stack":
			registry.RemoveStack(id)
		default:
			registry.RemoveComponent(id)
		}
	}
}

// ValidateConfig performs common validation for all components
// This is now commented out to let each resource handle its own validation
// func (r *BaseComponent) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
// 	// Each resource should implement its own ValidateConfig if needed
// }
