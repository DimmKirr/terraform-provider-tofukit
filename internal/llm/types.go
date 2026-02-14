package llm

// ExecutionResult contains the result of an LLM execution
type ExecutionResult struct {
	Success     bool   `json:"success"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
	ProjectPath string `json:"project_path,omitempty"`
}

// ExecutionStatus represents the status of an LLM execution
type ExecutionStatus struct {
	State       string            `json:"state"`        // pending, running, completed, failed
	StartedAt   string            `json:"started_at"`   // RFC3339 timestamp
	CompletedAt string            `json:"completed_at"` // RFC3339 timestamp
	ProjectPath string            `json:"project_path"`
	Error       string            `json:"error,omitempty"`
	Output      string            `json:"output,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Config contains configuration for LLM providers
type Config struct {
	// ClaudeHomeDir is the home directory for Claude CLI (claude provider only)
	ClaudeHomeDir string

	// APIKey is the API key for API-based providers (openai, gemini)
	APIKey string

	// Debug enables debug mode
	Debug bool

	// OutputPath is the path for output files
	OutputPath string
}
