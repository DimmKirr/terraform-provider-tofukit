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
	Type             string                   `json:"type"`
	ProjectInfo      ProjectInfo              `json:"project_info"`
	Specification    map[string]interface{}   `json:"specification"`
	ResourceRegistry map[string]interface{}   `json:"resource_registry,omitempty"`
	ProjectContext   map[string]interface{}   `json:"project_context,omitempty"`
	Instructions     []string                 `json:"instructions"`
	FileDetails      *FileInstructions        `json:"file_details,omitempty"`
	FileOperations   []map[string]interface{} `json:"file_operations,omitempty"`
	Guidelines       []string                 `json:"guidelines"`
	Deliverables     []string                 `json:"deliverables"`
}

// ProjectInfo contains basic project metadata
type ProjectInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// FileInstructions provides details about file files
type FileInstructions struct {
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
{{end}}{{if .Request.FileDetails}}## File Files

{{.Request.FileDetails.Description}}

{{range .Request.FileDetails.Rules}}- {{.}}
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
// buildFileDetails creates FileInstructions based on whether files are specified
func buildFileDetails(hasFiles bool) *FileInstructions {
	if hasFiles {
		// Files ARE specified - strict mode
		return &FileInstructions{
			Description: "IMPORTANT: The specification contains a \"files\" object. This is an EXHAUSTIVE list - you MUST manage ONLY these files:",
			Rules: []string{
				"**CRITICAL**: ALL file operations MUST be performed in the current working directory (run 'pwd' first to verify location). NEVER create files in /tmp/ or any other directory",
				"**EXHAUSTIVE LIST**: The \"files\" section contains the COMPLETE list of files. Do NOT create any files beyond those listed, even if requirements suggest additional files are needed. All functionality must be implemented within the specified files.",
				"**NON-NEGOTIABLE**: Files listed in the \"files\" section are MANDATORY. Even if you believe a file shouldn't exist or isn't needed, you MUST create it exactly as specified. If you think main.tf shouldn't exist but it's specified, create main.tf anyway and work around your concerns. The files list is the user's explicit directive and cannot be questioned or skipped.",
				"Create each file at the exact path specified as a key in the files object (relative to current working directory)",
				"If a path contains directories (e.g., 'dir/file.txt'), create the parent directories first",
				"If 'content' field exists: Use the EXACT content provided without ANY modification - preserve all characters including trailing newlines (\\n)",
				"If 'generate' is true and 'instructions' field exists: Generate appropriate content following ALL the instructions provided",
				"IMPORTANT: For generated content, ensure it satisfies ALL instructions AND the verification requirements",
				"IMPORTANT: Generated files should be production-ready and follow best practices for the file type",
				"IMPORTANT: If content ends with \\n, the file MUST have a newline at the end. Use echo or printf appropriately",
				"Remove any existing file files that are NOT in the current specification",
				"When removing the last file from a directory, also remove the empty directory",
				"These are template/example files that should be created/updated/removed as specified",
				"When creating files, use: echo -n 'content' > file (for no trailing newline) or echo 'content' > file (for trailing newline)",
			},
		}
	}

	// NO files specified - flexible mode
	return &FileInstructions{
		Description: "IMPORTANT: No explicit files are specified. Create files as needed to satisfy the requirements:",
		Rules: []string{
			"**CRITICAL**: ALL file operations MUST be performed in the current working directory (run 'pwd' first to verify location). NEVER create files in /tmp/ or any other directory",
			"Analyze the requirements and determine what files are needed to implement them fully",
			"Create appropriate file structure and naming conventions based on best practices for the languages/frameworks involved",
			"If a path contains directories (e.g., 'src/cli.py'), create the parent directories first",
			"Generate production-ready, well-documented code that follows best practices",
			"Ensure all generated content satisfies the requirements AND any verification requirements",
			"IMPORTANT: Only create files that are necessary to fulfill the requirements - do not create unnecessary files",
			"When creating files, use: echo -n 'content' > file (for no trailing newline) or echo 'content' > file (for trailing newline)",
		},
	}
}

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

	// Build instructions - conditional based on whether files are specified
	instructions := []string{
		"**CRITICAL - Working Directory**: ALL files must be created directly in the current working directory. DO NOT create any project-name subdirectories. DO NOT use 'cd' commands. The current directory IS the project directory.",
	}

	// File handling instructions - different behavior based on whether files are specified
	if hasFiles {
		// Files ARE specified - create ONLY those files (exhaustive list)
		instructions = append(instructions,
			"**Managing specified files - EXHAUSTIVE LIST**: The specification includes a \"files\" object. This is the COMPLETE and EXHAUSTIVE list of files you must create - do not create any additional files beyond this list. Create these files exactly as specified **in the current working directory** with their exact paths and content. IMPORTANT: All file operations must be performed in the current working directory (use pwd to verify). Never create files in /tmp/ or other directories. When comparing with existing files, remove any files not in the specification. If removing the last file from a directory, also remove the now-empty directory.",
			"**Requirements when files are specified**: If the specification also contains \"requirements\", treat them as context and constraints for HOW to implement the specified files. Requirements provide implementation guidance but should NOT result in creating additional files beyond those explicitly listed in the \"files\" section. All requirement logic must be incorporated into the specified files.",
		)
	} else {
		// NO files specified - create files as needed based on requirements
		instructions = append(instructions,
			"**Creating files based on requirements**: No explicit files are specified. Create whatever files and directories are needed to fulfill the requirements listed in the \"requirements\" section. Use your expertise to determine the appropriate file structure, naming conventions, and content organization. All paths should be relative to the current working directory - DO NOT create a project-name subdirectory.",
		)
	}

	// Common instructions for both cases
	instructions = append(instructions,
		"**Analyzing the specification**: Understand all the requirements, kits, and dependencies specified in the JSON",
		"**Implementing all requirements**: Follow each requirement listed in the \"requirements\" section",
		"**Installing and configuring all kits**: Set up all the tools, frameworks, languages, and methodologies specified in the \"kits\" section",
		"**Following verification steps**: Ensure each requirement can be verified as specified",
	)

	// If this is a fix request, prepend fix instructions
	if fixRequest, ok := projectSpec["_fix_request"].(map[string]interface{}); ok {
		if fixInstructions, ok := fixRequest["instructions"].(string); ok {
			// Prepend fix instructions to the beginning
			instructions = append([]string{fixInstructions}, instructions...)
		}
	}

	// Extract file operations if present
	var fileOperations []map[string]interface{}
	if ops, ok := projectSpec["_file_operations"].([]map[string]interface{}); ok {
		fileOperations = ops
	}

	// Extract resource registry if present
	var resourceRegistry map[string]interface{}
	if registry, ok := projectSpec["_resource_registry"].(map[string]interface{}); ok {
		resourceRegistry = registry
		// Enhance system prompt with resource URI instructions
		systemPrompt += "\n\n## Resource URI References\n\n" +
			"URIs like tofukit://TYPE/NAME reference other resources in this project. " +
			"Look them up in the resource_registry section for complete details about their files, capabilities, and interfaces. " +
			"When you see these URIs in prompts or instructions, treat them as explicit dependencies that you should integrate with."
	}

	// Extract project context if present
	var projectContext map[string]interface{}
	if ctx, ok := projectSpec["_project_context"].(map[string]interface{}); ok {
		projectContext = ctx

		// Enhance system prompt with project context instructions
		systemPrompt += "\n\n## Project Context Introspection\n\n" +
			"The project_context field in the specification contains complete metadata about this project:\n" +
			"- features: All features defined in this project with their prompts, files, and capabilities\n" +
			"- integrations: All external API/service integrations referenced by this project\n" +
			"- kits: All language/framework/tool kits configured for this project\n" +
			"- requirements: All high-level requirements for this project\n\n" +
			"Features can introspect this context to automatically discover project components without requiring explicit configuration. " +
			"For example, diagram generation features can visualize the entire architecture by reading the project_context field."
	}

	// Build the structured prompt
	prompt := &ProjectPrompt{
		SystemPrompt: systemPrompt,
		Request: PromptRequest{
			Type:             "project_implementation",
			ProjectInfo:      projectInfo,
			Specification:    projectSpec,
			ResourceRegistry: resourceRegistry,
			ProjectContext:   projectContext,
			Instructions:     instructions,
			FileOperations:   fileOperations,
			FileDetails: buildFileDetails(hasFiles),
			Guidelines: []string{
				"Follow the exact specifications provided in the JSON",
				"Create file files exactly as specified without modification",
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
				"All file files created exactly as specified",
				"Complete, working project implementation",
				"All files and directories properly structured",
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
