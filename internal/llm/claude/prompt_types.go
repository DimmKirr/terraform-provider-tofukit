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

// InstructionGroup represents a named group of instructions with constraints
type InstructionGroup struct {
	Name        string   `json:"name"`
	Prompt      string   `json:"prompt"`
	Constraints []string `json:"constraints,omitempty"`
}

// PromptRequest contains the actual project implementation request
type PromptRequest struct {
	Type             string                   `json:"type"`
	ProjectInfo      ProjectInfo              `json:"project_info"`
	Specification    map[string]interface{}   `json:"specification"`
	ResourceRegistry map[string]interface{}   `json:"resource_registry,omitempty"`
	ProjectContext   map[string]interface{}   `json:"project_context,omitempty"`
	Instructions     []InstructionGroup       `json:"instructions"`
	FileOperations   []map[string]interface{} `json:"file_operations,omitempty"`
}

// ProjectInfo contains basic project metadata
type ProjectInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
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

{{range .Request.Instructions}}### {{.Name}}
**{{.Prompt}}**
{{if .Constraints}}
Constraints:
{{range .Constraints}}- {{.}}
{{end}}{{end}}
{{end}}
{{if .Request.FileOperations}}
## File Operations

**IMPORTANT**: You must perform the following operations in the order specified:

{{range .Request.FileOperations}}{{if eq .action "remove"}}### REMOVE: {{.path}}
- Delete this file from the project
- If this is the last file in a directory, also remove the empty directory
{{else if eq .action "rename"}}### RENAME: {{.old_path}} → {{.path}}
- Rename/move the file from {{.old_path}} to {{.path}}
- Preserve the content during the move
- Create parent directories if needed for the new path
{{else if eq .action "add"}}### ADD: {{.path}}
{{if .instructions}}- Generate content following these instructions:
{{range .instructions}}  - Prompt: {{.prompt}}
{{if .constraints}}  - Constraints:
{{range .constraints}}    - {{.}}
{{end}}{{end}}{{end}}{{else}}- Create file with exact content (see specification JSON for content)
{{end}}{{else if eq .action "modify"}}### MODIFY: {{.path}}
{{if .instructions}}- Update content following these instructions:
{{range .instructions}}  - Prompt: {{.prompt}}
{{if .constraints}}  - Constraints:
{{range .constraints}}    - {{.}}
{{end}}{{end}}{{end}}{{else}}- Update file with new content (see specification JSON for content)
{{end}}{{else if eq .action "unchanged"}}### UNCHANGED: {{.path}}
- This file should remain as-is (no action needed)
{{end}}
{{end}}
{{end}}`

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

	// Check if files are specified in the specification
	hasFiles := false
	if filesObj, ok := projectSpec["files"].(map[string]interface{}); ok && len(filesObj) > 0 {
		hasFiles = true
	}

	// Build consolidated instruction groups
	instructions := []InstructionGroup{
		{
			Name:   "Working Directory",
			Prompt: "Execute all file operations in the current working directory",
			Constraints: []string{
				"DO NOT create project-name subdirectories",
				"DO NOT use 'cd' commands",
				"Run 'pwd' first to verify location",
				"NEVER create files in /tmp/ or any other directory",
			},
		},
	}

	// File handling instructions - different behavior based on whether files are specified
	if hasFiles {
		instructions = append(instructions, InstructionGroup{
			Name:   "File Management",
			Prompt: "Create ONLY the files listed in the specification's \"files\" object - this is an EXHAUSTIVE list",
			Constraints: []string{
				"Do NOT create any files beyond those explicitly listed",
				"Files listed are MANDATORY - create them even if you think they're unnecessary",
				"If 'content' field exists, use EXACT content without ANY modification",
				"If 'generate' is true with 'instructions', generate content following ALL instructions",
				"Remove existing files NOT in the current specification",
				"Remove empty directories when last file is removed",
			},
		})
	} else {
		instructions = append(instructions, InstructionGroup{
			Name:   "File Creation",
			Prompt: "Create files as needed to fulfill the requirements",
			Constraints: []string{
				"Use your expertise to determine appropriate file structure",
				"Only create files necessary for the requirements",
				"Follow language/framework best practices for naming and organization",
			},
		})
	}

	// Common instructions
	instructions = append(instructions,
		InstructionGroup{
			Name:   "Quality Standards",
			Prompt: "Produce production-ready, well-documented code",
			Constraints: []string{
				"No TODO comments or placeholder code",
				"Include proper error handling",
				"Follow language-specific best practices",
				"Ensure all verification commands pass",
			},
		},
		InstructionGroup{
			Name:   "Implementation",
			Prompt: "Implement all requirements and configure all kits",
			Constraints: []string{
				"Process requirements in priority order (higher numbers first)",
				"Set up all tools, frameworks, and languages specified in kits",
				"Follow each requirement's verification steps",
			},
		},
	)

	// If this is a fix request, prepend fix instructions
	if fixRequest, ok := projectSpec["_fix_request"].(map[string]interface{}); ok {
		if fixInstructions, ok := fixRequest["instructions"].(string); ok {
			instructions = append([]InstructionGroup{{
				Name:        "Fix Request",
				Prompt:      fixInstructions,
				Constraints: []string{},
			}}, instructions...)
		}
	}

	// Extract file operations if present
	var fileOperations []map[string]interface{}
	if ops, ok := projectSpec["_file_operations"].([]map[string]interface{}); ok {
		fileOperations = ops
	}

	// Extract resource registry if present (from _resource_registry in spec)
	// NOTE: Do NOT delete from projectSpec - this function may be called multiple times
	// and we need to preserve the registry in the source map
	var resourceRegistry map[string]interface{}
	if registry, ok := projectSpec["_resource_registry"].(map[string]interface{}); ok {
		resourceRegistry = registry
		// Enhance system prompt with tofukit URI instructions
		systemPrompt += "\n\n## Resource URI Linking (tofukit://)\n\n" +
			"URIs like `tofukit://TYPE/NAME` reference other resources in this project.\n\n" +
			"**How to resolve URIs:**\n" +
			"1. Look up the URI in `resource_registry` for complete metadata\n" +
			"2. For file resources: `resource_registry[\"tofukit://file/NAME\"]` contains:\n" +
			"   - `name`: The resource identifier (used in linking)\n" +
			"   - `path`: The actual filesystem path where the file exists\n" +
			"   - `description`: What the file does\n" +
			"3. For feature resources: Contains name, description, and associated files\n\n" +
			"**Example:** If you see `tofukit://file/logo` in a prompt:\n" +
			"- Look up `resource_registry[\"tofukit://file/logo\"]`\n" +
			"- Use the `path` field (e.g., `assets/logo.png`) when referencing the actual file"
	}

	// Extract project context if present (from _project_context in spec)
	// NOTE: Do NOT delete from projectSpec - this function may be called multiple times
	// and we need to preserve the context in the source map
	var projectContext map[string]interface{}
	if ctx, ok := projectSpec["_project_context"].(map[string]interface{}); ok {
		projectContext = ctx
		// Enhance system prompt with lean project context instructions
		systemPrompt += "\n\n## Project Context\n\n" +
			"The `project_context` field contains metadata about this project:\n" +
			"- `project_info`: Project name, description, version\n" +
			"- `features`: Feature names and their associated file lists\n\n" +
			"To resolve `tofukit://` URIs, use `resource_registry` (see above)."
	}

	// Create a clean specification without internal fields
	// Internal fields (_resource_registry, _project_context, _file_operations) should
	// appear at the request level, not duplicated in specification
	cleanSpec := make(map[string]interface{})
	for k, v := range projectSpec {
		// Skip internal fields - they are exposed at request level
		if k == "_resource_registry" || k == "_project_context" || k == "_file_operations" {
			continue
		}
		cleanSpec[k] = v
	}

	// Build the structured prompt
	prompt := &ProjectPrompt{
		SystemPrompt: systemPrompt,
		Request: PromptRequest{
			Type:             "project_implementation",
			ProjectInfo:      projectInfo,
			Specification:    cleanSpec,
			ResourceRegistry: resourceRegistry,
			ProjectContext:   projectContext,
			Instructions:     instructions,
			FileOperations:   fileOperations,
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
- Setting up testing, linting, and build processes
- Following specified methodologies and architectural patterns

CRITICAL EXECUTION MODE:

This is a NON-INTERACTIVE automated execution environment. You MUST:
- Execute immediately without asking ANY questions
- NEVER ask for clarification, confirmation, or approval
- NEVER enter "design thinking" or planning mode that requires user input
- Proceed directly with implementation based on the specifications provided
- If specifications are unclear, make reasonable assumptions and proceed

CRITICAL FILE OPERATION RULES:

1. File Deletion: NEVER remove any file from the output directory unless you receive an explicit instruction with action="remove" and the full file path

2. Directory Deletion: NEVER remove any directory unless it is completely empty (no files, no subdirectories)

3. File Discovery: If you discover files in the output directory that are NOT in your files array, leave them alone. Do not remove, modify, or clean them up.

4. Scope Limitation: Only operate on files explicitly listed in the files array you receive. Ignore all other files in the directory.

When implementing projects:
1. Always follow the exact specifications provided
2. Create well-structured, maintainable code
3. Include comprehensive error handling
4. Set up proper development toolchains
5. Ensure all verification steps pass
6. Follow language-specific best practices and conventions

Focus on creating production-ready deliverables that developers can immediately use and extend.`
}
