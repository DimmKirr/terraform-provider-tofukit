package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// ProjectPrompt represents the structured prompt sent to Claude
type ProjectPrompt struct {
	SystemPrompt string        `json:"system_prompt"`
	Request      PromptRequest `json:"request"`
}

// PromptRequest contains the actual project implementation request
type PromptRequest struct {
	Type            string                 `json:"type"`
	ProjectInfo     ProjectInfo            `json:"project_info"`
	Specification   map[string]interface{} `json:"specification"`
	Instructions    []string               `json:"instructions"`
	ScaffoldDetails *ScaffoldInstructions  `json:"scaffold_details,omitempty"`
	Guidelines      []string               `json:"guidelines"`
	Deliverables    []string               `json:"deliverables"`
}

// ProjectInfo contains basic project metadata
type ProjectInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// ScaffoldInstructions provides details about scaffold files
type ScaffoldInstructions struct {
	Description string   `json:"description"`
	Rules       []string `json:"rules"`
}

// ToJSON converts the prompt to JSON string
func (p *ProjectPrompt) ToJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// markdownTemplate is the template for rendering prompts as markdown
const markdownTemplate = `# Project Implementation Instructions for {{.Request.ProjectInfo.Name}}

{{if .SystemPrompt}}## System Context

{{.SystemPrompt}}

{{end}}## Project Information
- **Name**: {{.Request.ProjectInfo.Name}}
- **Description**: {{.Request.ProjectInfo.Description}}
- **Version**: {{.Request.ProjectInfo.Version}}

## Complete Project Specification
The following JSON contains the full project specification:

` + "```json\n{{.SpecificationJSON}}\n```" + `

## Instructions

{{range $i, $instruction := .Request.Instructions}}{{add $i 1}}. {{$instruction}}
{{end}}
{{if .Request.ScaffoldDetails}}## Scaffold Files

{{.Request.ScaffoldDetails.Description}}

{{range .Request.ScaffoldDetails.Rules}}- {{.}}
{{end}}
{{end}}## Key Guidelines

{{range .Request.Guidelines}}- {{.}}
{{end}}
## Expected Deliverables

{{range .Request.Deliverables}}- {{.}}
{{end}}
`

// ToMarkdown renders the prompt as markdown for debug output
func (p *ProjectPrompt) ToMarkdown() string {
	// Create template with helper functions
	tmpl := template.Must(template.New("prompt").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}).Parse(markdownTemplate))

	// Prepare data for template
	type templateData struct {
		*ProjectPrompt
		SpecificationJSON string
	}

	specJSON, _ := json.MarshalIndent(p.Request.Specification, "", "  ")
	data := templateData{
		ProjectPrompt:     p,
		SpecificationJSON: string(specJSON),
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		// Fallback to simple format if template fails
		return fmt.Sprintf("# Project: %s\n\nError rendering template: %v", p.Request.ProjectInfo.Name, err)
	}

	return buf.String()
}

// BuildProjectPrompt creates a structured prompt from project specification
func BuildProjectPrompt(projectSpec map[string]interface{}, customSystemPrompt string) *ProjectPrompt {
	// Extract project info
	projectInfo := ProjectInfo{
		Name:        "Project",
		Description: "A Terraform-generated project",
		Version:     "1.0.0",
	}

	if project, ok := projectSpec["project"].(map[string]interface{}); ok {
		if name, ok := project["name"].(string); ok {
			projectInfo.Name = name
		}
		if desc, ok := project["description"].(string); ok {
			projectInfo.Description = desc
		}
		if version, ok := project["version"].(string); ok {
			projectInfo.Version = version
		}
	}

	// Determine system prompt
	systemPrompt := customSystemPrompt
	if systemPrompt == "" {
		systemPrompt = DefaultSystemPrompt()
	}

	// Build the structured prompt
	prompt := &ProjectPrompt{
		SystemPrompt: systemPrompt,
		Request: PromptRequest{
			Type:          "project_implementation",
			ProjectInfo:   projectInfo,
			Specification: projectSpec,
			Instructions: []string{
				"**Managing scaffold files**: If the specification includes \"scaffolds\", create these files exactly as specified with their exact paths and content. When comparing with existing files, remove any files not in the specification. If removing the last file from a directory, also remove the now-empty directory.",
				"**Analyzing the specification**: Understand all the requirements, kits, and dependencies specified in the JSON",
				"**Creating the project structure**: Set up appropriate directories and files based on the project type and requirements",
				"**Implementing all requirements**: Follow each requirement listed in the \"requirements\" section with proper priority ordering",
				"**Installing and configuring all kits**: Set up all the tools, frameworks, languages, and methodologies specified in the \"kits\" section",
				"**Following verification steps**: Ensure each requirement can be verified as specified",
				"**Creating comprehensive documentation**: Include README, setup instructions, and usage examples",
			},
			ScaffoldDetails: &ScaffoldInstructions{
				Description: "IMPORTANT: If the specification contains a \"scaffolds\" array, you MUST manage these files exactly as specified:",
				Rules: []string{
					"Create each file at the exact path specified in the scaffolds array",
					"If a path contains directories (e.g., 'dir/file.txt'), create the parent directories first",
					"Use the EXACT content provided without ANY modification - preserve all characters including trailing newlines (\\n)",
					"IMPORTANT: If content ends with \\n, the file MUST have a newline at the end. Use echo or printf appropriately",
					"Remove any existing scaffold files that are NOT in the current specification",
					"When removing the last file from a directory, also remove the empty directory",
					"These are template/example files that should be created/updated/removed exactly as-is",
					"When creating files, use: echo -n 'content' > file (for no trailing newline) or echo 'content' > file (for trailing newline)",
				},
			},
			Guidelines: []string{
				"Follow the exact specifications provided in the JSON",
				"Create scaffold files exactly as specified without modification",
				"Create parent directories as needed for nested file paths (e.g., mkdir -p for 'dir/subdir/file.txt')",
				"Clean up empty directories when removing the last file from them (e.g., rmdir or rm -d)",
				"Implement all requirements in priority order (higher numbers first)",
				"Ensure all verification commands work as expected",
				"Create production-ready, well-documented code",
				"Follow best practices for the specified programming language and frameworks",
				"Include proper error handling and logging",
				"Set up development and build toolchains as specified in the kits",
			},
			Deliverables: []string{
				"All scaffold files created exactly as specified",
				"Complete, working project implementation",
				"All files and directories properly structured",
				"Documentation explaining setup and usage",
				"All requirements implemented and verified",
				"Development environment ready for use",
			},
		},
	}

	return prompt
}

// DefaultSystemPrompt returns the default system prompt for Claude
func DefaultSystemPrompt() string {
	return `You are a senior software architect and developer with 30+ years of experience and expertise across multiple programming languages, frameworks, and development methodologies.

Your task is to implement complete, production-ready projects based on detailed specifications. You excel at:

- Setting up project structures and development environments
- Installing and configuring development tools and dependencies
- Implementing features following best practices and coding standards
- Creating comprehensive documentation and examples
- Setting up testing, linting, and build processes
- Following specified methodologies and architectural patterns

When implementing projects:
1. Always follow the exact specifications provided
2. Create well-structured, maintainable code
3. Include comprehensive error handling
4. Set up proper development toolchains
5. Write clear documentation and examples
6. Ensure all verification steps pass
7. Follow language-specific best practices and conventions

Focus on creating production-ready deliverables that developers can immediately use and extend.`
}
