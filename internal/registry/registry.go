package registry

// Registry stores all defined resources for dependency resolution
type Registry struct {
	Components map[string]interface{}
	Stacks     map[string]interface{}
	Projects   map[string]interface{}
}

// New creates a new Registry instance
func New() *Registry {
	return &Registry{
		Components: make(map[string]interface{}),
		Stacks:     make(map[string]interface{}),
		Projects:   make(map[string]interface{}),
	}
}
