package registry

import "sync"

// Registry stores all defined resources for dependency resolution
type Registry struct {
	mu           sync.RWMutex
	Components   map[string]interface{}
	Stacks       map[string]interface{}
	Files        map[string]interface{}
	Features     map[string]interface{}
	Projects     map[string]interface{}
	Integrations map[string]interface{}
}

// New creates a new Registry instance
func New() *Registry {
	return &Registry{
		Components:   make(map[string]interface{}),
		Stacks:       make(map[string]interface{}),
		Files:        make(map[string]interface{}),
		Features:     make(map[string]interface{}),
		Projects:     make(map[string]interface{}),
		Integrations: make(map[string]interface{}),
	}
}

// SetComponent stores a component in the registry
func (r *Registry) SetComponent(id string, data interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Components[id] = data
}

// GetComponent retrieves a component from the registry
func (r *Registry) GetComponent(id string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, exists := r.Components[id]
	return data, exists
}

// RemoveComponent removes a component from the registry
func (r *Registry) RemoveComponent(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Components, id)
}

// SetStack stores a stack in the registry
func (r *Registry) SetStack(id string, data interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Stacks[id] = data
}

// GetStack retrieves a stack from the registry
func (r *Registry) GetStack(id string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, exists := r.Stacks[id]
	return data, exists
}

// RemoveStack removes a stack from the registry
func (r *Registry) RemoveStack(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Stacks, id)
}

// GetAllStacks returns all stacks in the registry
func (r *Registry) GetAllStacks() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range r.Stacks {
		result[k] = v
	}
	return result
}

// SetFile stores a file resource in the registry
func (r *Registry) SetFile(id string, data interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Files[id] = data
}

// GetFile retrieves a file resource from the registry
func (r *Registry) GetFile(id string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, exists := r.Files[id]
	return data, exists
}

// RemoveFile removes a file resource from the registry
func (r *Registry) RemoveFile(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Files, id)
}

// GetAllFiles returns all file resources in the registry
func (r *Registry) GetAllFiles() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range r.Files {
		result[k] = v
	}
	return result
}

// SetFeature stores a feature resource in the registry
func (r *Registry) SetFeature(id string, data interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Features[id] = data
}

// GetFeature retrieves a feature resource from the registry
func (r *Registry) GetFeature(id string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, exists := r.Features[id]
	return data, exists
}

// RemoveFeature removes a feature resource from the registry
func (r *Registry) RemoveFeature(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Features, id)
}

// GetAllFeatures returns all feature resources in the registry
func (r *Registry) GetAllFeatures() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range r.Features {
		result[k] = v
	}
	return result
}

// SetIntegration stores an integration resource in the registry
func (r *Registry) SetIntegration(id string, data interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Integrations[id] = data
	return nil
}

// GetIntegration retrieves an integration resource from the registry
func (r *Registry) GetIntegration(id string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	data, exists := r.Integrations[id]
	return data, exists
}

// RemoveIntegration removes an integration resource from the registry
func (r *Registry) RemoveIntegration(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Integrations, id)
	return nil
}

// GetAllIntegrations returns all integration resources in the registry
func (r *Registry) GetAllIntegrations() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range r.Integrations {
		result[k] = v
	}
	return result
}
