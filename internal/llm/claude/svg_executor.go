package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/tofukit/opentofu-provider-tofukit/internal/files"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// SVGExecutor generates images using Claude to create SVG wireframes, then converts to PNG
type SVGExecutor struct {
	claudeExecutor *Executor
	debug          bool
	outputPath     string
}

// NewSVGExecutor creates a new SVG executor that uses Claude for wireframe generation
func NewSVGExecutor(claudeHomeDir string, dangerouslySkipPermissions bool, maxTurns int) *SVGExecutor {
	return &SVGExecutor{
		claudeExecutor: NewExecutor(claudeHomeDir, dangerouslySkipPermissions, maxTurns),
		debug:          false,
	}
}

// SetDebug enables or disables debug mode
func (e *SVGExecutor) SetDebug(debug bool) {
	e.debug = debug
	e.claudeExecutor.SetDebug(debug)
}

// SetOutputPath sets the output path for debug files
func (e *SVGExecutor) SetOutputPath(outputPath string) {
	e.outputPath = outputPath
	e.claudeExecutor.SetOutputPath(outputPath)
}

// SetSystemPrompt sets a custom system prompt (not used for SVG mode)
func (e *SVGExecutor) SetSystemPrompt(systemPrompt string) {
	// SVG executor uses its own system prompt
}

// SetModel sets the model (not used, always uses Claude)
func (e *SVGExecutor) SetModel(model string) {
	e.claudeExecutor.SetModel(model)
}

// wrapPromptForSVG wraps the user's image prompt with instructions for SVG wireframe generation
func wrapPromptForSVG(originalPrompt string, width, height int) string {
	return fmt.Sprintf(`Generate a valid SVG wireframe/schematic representing the following image composition.
This should be a visual layout guide as if you were creating a wireframe for an artist to follow.

Use simple shapes, lines, and labels to represent:
- Main subject placement and proportions
- Background elements and layers
- Key visual elements and their relationships
- Composition guidelines (rule of thirds, focal points)
- Approximate color regions (use solid fills with hex colors)

CRITICAL SVG REQUIREMENTS:
- Output ONLY the SVG code, no explanations or markdown
- Start with <svg and end with </svg>
- ALL attributes MUST have quoted values (e.g., width="512" NOT width=512)
- Use viewBox="0 0 %d %d"
- Use valid XML syntax throughout
- Include descriptive labels/annotations as text elements
- Use a clean, minimalist style

Example of valid SVG start:
<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">

Image to create wireframe for:
%s`, width, height, width, height, width, height, originalPrompt)
}

// stripANSIAndTrim removes ANSI escape sequences and trims whitespace
func stripANSIAndTrim(s string) string {
	// Remove ANSI escape sequences (like \x1b[?25h cursor control)
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)
	s = ansiRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// extractSVGFromResponse extracts SVG content from Claude's response
func extractSVGFromResponse(response string) (string, error) {
	// The response is JSON from Claude CLI's --output-format=json
	// We need to extract the "result" field first to unescape JSON escapes
	content := response

	// Strip ANSI escape codes and whitespace before JSON parsing
	cleanedResponse := stripANSIAndTrim(response)

	// Try to parse as JSON and extract the result field
	if strings.HasPrefix(cleanedResponse, "{") {
		var jsonResponse struct {
			Result string `json:"result"`
		}
		if err := json.Unmarshal([]byte(cleanedResponse), &jsonResponse); err == nil && jsonResponse.Result != "" {
			content = jsonResponse.Result
		}
	}

	// Try to find SVG in code blocks FIRST (Claude often wraps SVG in markdown)
	codeBlockRegex := regexp.MustCompile("(?s)```(?:svg|xml)?\\s*(<svg[^>]*>.*?</svg>)\\s*```")
	matches := codeBlockRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		return sanitizeSVG(matches[1]), nil
	}

	// Try to find SVG content between <svg> tags (direct output)
	svgRegex := regexp.MustCompile(`(?s)<svg[^>]*>.*?</svg>`)
	match := svgRegex.FindString(content)
	if match != "" {
		return sanitizeSVG(match), nil
	}

	// If response starts with <?xml, try to extract from there
	if strings.Contains(content, "<?xml") {
		xmlRegex := regexp.MustCompile(`(?s)(<svg[^>]*>.*?</svg>)`)
		match := xmlRegex.FindString(content)
		if match != "" {
			return sanitizeSVG(match), nil
		}
	}

	return "", fmt.Errorf("no SVG content found in response")
}

// sanitizeSVG fixes common SVG issues that cause XML parsing errors
func sanitizeSVG(svg string) string {
	// Fix backslash-escaped quotes in attribute values (KIRR-129)
	// Claude sometimes outputs JSON-style escaping like: xmlns="\"http://...\""
	// In XML/SVG, backslash is NOT an escape character - quotes use &quot; entity.
	// We fix this with two targeted replacements:
	// 1. ="\" → =" (removes escaped quote after opening quote)
	// 2. \"" → " (removes escaped quote before closing quote)
	svg = strings.ReplaceAll(svg, `="\"`, `="`)
	svg = strings.ReplaceAll(svg, `\""`, `"`)

	// Fix unquoted attribute values (e.g., width=100 -> width="100")
	// This regex finds attribute=value patterns where value is not quoted
	attrRegex := regexp.MustCompile(`(\s)([a-zA-Z-]+)=([^"'\s>][^\s>]*)`)
	svg = attrRegex.ReplaceAllString(svg, `$1$2="$3"`)

	// Ensure xmlns is present
	if !strings.Contains(svg, "xmlns=") {
		svg = strings.Replace(svg, "<svg", `<svg xmlns="http://www.w3.org/2000/svg"`, 1)
	}

	// Fix unescaped ampersands in text content (KIRR-128)
	// We need to escape & that is NOT part of a valid XML entity.
	// Valid entities start with & followed by letter (named) or # (numeric)
	// Go's RE2 doesn't support negative lookahead, so we use ReplaceAllStringFunc
	svg = escapeUnescapedAmpersands(svg)

	// Note: We don't fix empty tag pairs (<tag></tag> -> <tag/>) because Go's
	// regexp doesn't support backreferences. The SVG parser handles both forms.

	return svg
}

// escapeUnescapedAmpersands finds bare & characters and escapes them to &amp;
// It preserves valid XML entities like &amp; &lt; &gt; &quot; &apos; and numeric entities like &#38; &#x26;
func escapeUnescapedAmpersands(s string) string {
	var result strings.Builder
	result.Grow(len(s) + 100) // Pre-allocate with some extra space for escapes

	i := 0
	for i < len(s) {
		if s[i] == '&' {
			// Check if this is a valid XML entity by looking for the complete pattern
			if isValidXMLEntity(s[i:]) {
				// Valid entity, keep as-is
				result.WriteByte('&')
			} else {
				// Bare & not part of a valid entity - escape it
				result.WriteString("&amp;")
			}
			i++
		} else {
			result.WriteByte(s[i])
			i++
		}
	}

	return result.String()
}

// isValidXMLEntity checks if the string starting at s[0] ('&') is a valid XML entity
// Valid patterns: &name; or &#digits; or &#xhexdigits;
func isValidXMLEntity(s string) bool {
	if len(s) < 3 || s[0] != '&' {
		return false
	}

	// Check for numeric entity: &#digits; or &#xhexdigits;
	if s[1] == '#' {
		if len(s) < 4 {
			return false
		}
		i := 2
		if s[2] == 'x' || s[2] == 'X' {
			// Hex entity: &#x[0-9a-fA-F]+;
			i = 3
			for i < len(s) && isHexDigit(s[i]) {
				i++
			}
			if i == 3 { // No hex digits found
				return false
			}
		} else {
			// Decimal entity: &#[0-9]+;
			for i < len(s) && isDigit(s[i]) {
				i++
			}
			if i == 2 { // No digits found
				return false
			}
		}
		return i < len(s) && s[i] == ';'
	}

	// Check for named entity: &[a-zA-Z]+;
	if isLetter(s[1]) {
		i := 2
		for i < len(s) && isLetter(s[i]) {
			i++
		}
		return i < len(s) && s[i] == ';'
	}

	return false
}

// isLetter returns true if the byte is an ASCII letter (a-zA-Z)
func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isDigit returns true if the byte is an ASCII digit (0-9)
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// isHexDigit returns true if the byte is a hex digit (0-9, a-f, A-F)
func isHexDigit(b byte) bool {
	return isDigit(b) || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// convertSVGToPNG converts SVG content to PNG image data
func convertSVGToPNG(svgContent string, width, height int) ([]byte, error) {
	// Parse SVG
	icon, err := oksvg.ReadIconStream(strings.NewReader(svgContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SVG: %w", err)
	}

	// Set target size
	icon.SetTarget(0, 0, float64(width), float64(height))

	// Create RGBA image
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create scanner and rasterize
	scanner := rasterx.NewScannerGV(width, height, rgba, rgba.Bounds())
	raster := rasterx.NewDasher(width, height, scanner)
	icon.Draw(raster, 1.0)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// Execute generates images using Claude SVG wireframes
func (e *SVGExecutor) Execute(ctx context.Context, projectSpec map[string]interface{}, outputDir string) (*llm.ExecutionStatus, error) {
	// Extract files from projectSpec
	var filesMap map[string]interface{}

	if spec, ok := projectSpec["specification"].(map[string]interface{}); ok {
		if fileData, ok := spec["files"].(map[string]interface{}); ok {
			filesMap = fileData
		}
	}

	if filesMap == nil {
		if fileData, ok := projectSpec["files"].(map[string]interface{}); ok {
			filesMap = fileData
		}
	}

	if len(filesMap) == 0 {
		return nil, fmt.Errorf("no files specified for SVG image generation")
	}

	// Process each file
	var (
		generatedFiles []string
		resultsMu      sync.Mutex
		wg             sync.WaitGroup
		errChan        = make(chan error, len(filesMap))
	)

	for path, fileData := range filesMap {
		path := path
		fileData := fileData

		wg.Add(1)
		go func() {
			defer wg.Done()

			fileMap, ok := fileData.(map[string]interface{})
			if !ok {
				return
			}

			// Extract prompt
			promptText := extractPromptTextFromFile(fileMap)
			if promptText == "" {
				return
			}

			// Extract size from image config or use defaults
			width, height := extractImageSize(fileMap)

			// Wrap prompt for SVG generation
			svgPrompt := wrapPromptForSVG(promptText, width, height)

			if e.debug {
				tflog.Debug(ctx, "Generating SVG wireframe", map[string]interface{}{
					"path":   path,
					"width":  width,
					"height": height,
				})
			}

			// Use Claude to generate SVG
			response, err := e.claudeExecutor.Query(ctx, []string{svgPrompt}, "")
			if err != nil {
				errChan <- fmt.Errorf("failed to generate SVG for '%s': %w", path, err)
				return
			}

			// Save raw Claude response for debugging
			if e.debug {
				debugDir := filepath.Join(outputDir, ".debug")
				if err := os.MkdirAll(debugDir, 0755); err == nil {
					rawPath := filepath.Join(debugDir, strings.ReplaceAll(path, "/", "-")+".claude-response.txt")
					_ = os.WriteFile(rawPath, []byte(response), 0644)
				}
			}

			// Extract SVG from response
			svgContent, err := extractSVGFromResponse(response)
			if err != nil {
				errChan <- fmt.Errorf("failed to extract SVG from response for '%s': %w (response length: %d)", path, err, len(response))
				return
			}

			// Save sanitized SVG for debugging if enabled
			if e.debug {
				svgPath := strings.TrimSuffix(filepath.Join(outputDir, path), filepath.Ext(path)) + ".svg"
				if err := os.MkdirAll(filepath.Dir(svgPath), 0755); err == nil {
					_ = os.WriteFile(svgPath, []byte(svgContent), 0644)
				}
			}

			// Convert SVG to PNG
			pngData, err := convertSVGToPNG(svgContent, width, height)
			if err != nil {
				// Save the problematic SVG for debugging
				if e.debug {
					debugDir := filepath.Join(outputDir, ".debug")
					if mkErr := os.MkdirAll(debugDir, 0755); mkErr == nil {
						failedSvgPath := filepath.Join(debugDir, strings.ReplaceAll(path, "/", "-")+".failed.svg")
						_ = os.WriteFile(failedSvgPath, []byte(svgContent), 0644)
					}
				}
				errChan <- fmt.Errorf("failed to convert SVG to PNG for '%s': %w", path, err)
				return
			}

			// Save PNG file
			fullPath := filepath.Join(outputDir, path)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				errChan <- fmt.Errorf("failed to create directory for '%s': %w", path, err)
				return
			}

			if err := os.WriteFile(fullPath, pngData, 0644); err != nil {
				errChan <- fmt.Errorf("failed to write PNG file '%s': %w", path, err)
				return
			}

			resultsMu.Lock()
			generatedFiles = append(generatedFiles, path)
			resultsMu.Unlock()

			if e.debug {
				tflog.Debug(ctx, "Generated wireframe image", map[string]interface{}{
					"path": path,
					"size": len(pngData),
				})
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	if err := <-errChan; err != nil {
		return nil, err
	}

	return &llm.ExecutionStatus{
		State:       "completed",
		ProjectPath: outputDir,
	}, nil
}

// extractPromptTextFromFile extracts the prompt from a file specification
func extractPromptTextFromFile(fileMap map[string]interface{}) string {
	instructions, ok := fileMap["instructions"].([]interface{})
	if !ok || len(instructions) == 0 {
		return ""
	}

	firstInst, ok := instructions[0].(map[string]interface{})
	if !ok {
		return ""
	}

	prompt, _ := firstInst["prompt"].(string)
	return prompt
}

// extractImageSize extracts width and height from file specification
func extractImageSize(fileMap map[string]interface{}) (int, int) {
	// Default size
	width, height := 1024, 1024

	// Check for image block
	if imageBlock, ok := fileMap["image"].(map[string]interface{}); ok {
		if size, ok := imageBlock["size"].(string); ok {
			// Parse "WIDTHxHEIGHT" format
			var w, h int
			if _, err := fmt.Sscanf(size, "%dx%d", &w, &h); err == nil {
				width, height = w, h
			}
		}
	}

	// Check for constraints
	if instructions, ok := fileMap["instructions"].([]interface{}); ok && len(instructions) > 0 {
		if firstInst, ok := instructions[0].(map[string]interface{}); ok {
			if constraints, ok := firstInst["constraints"].([]interface{}); ok {
				for _, c := range constraints {
					if str, ok := c.(string); ok {
						lower := strings.ToLower(str)
						if strings.HasPrefix(lower, "size:") {
							var w, h int
							sizeStr := strings.TrimPrefix(lower, "size:")
							sizeStr = strings.TrimSpace(sizeStr)
							if _, err := fmt.Sscanf(sizeStr, "%dx%d", &w, &h); err == nil {
								width, height = w, h
							}
						}
					}
				}
			}
		}
	}

	return width, height
}

// ExecuteWithPromptJSON executes SVG generation with a pre-built prompt JSON
func (e *SVGExecutor) ExecuteWithPromptJSON(
	ctx context.Context,
	promptJSON string,
	outputDir string,
	filesToVerify []schemas.FileModelWithPath,
	maxRetries int,
) (*llm.ExecutionStatus, *files.VerificationReport, error) {
	// Parse prompt JSON
	var promptData map[string]interface{}
	if err := parseJSON(promptJSON, &promptData); err != nil {
		return nil, nil, fmt.Errorf("failed to parse prompt JSON: %w", err)
	}

	// Extract project spec
	var projectSpec map[string]interface{}
	if request, ok := promptData["request"].(map[string]interface{}); ok {
		projectSpec = request
	} else {
		projectSpec = promptData
	}

	// Execute
	status, err := e.Execute(ctx, projectSpec, outputDir)
	if err != nil {
		return nil, nil, err
	}

	// Return simple verification report (SVG mode doesn't run verifications)
	report := &files.VerificationReport{
		AllPassed:   true,
		FailedCount: 0,
		PassedCount: 0,
		Results:     []files.VerificationResult{},
	}

	return status, report, nil
}

// Query executes a simple query (delegates to Claude)
func (e *SVGExecutor) Query(ctx context.Context, instructions []string, model string) (string, error) {
	return e.claudeExecutor.Query(ctx, instructions, model)
}

// Validate checks if the executor is properly configured
func (e *SVGExecutor) Validate(ctx context.Context) error {
	return e.claudeExecutor.client.ValidateClaudeCodeAvailability(ctx)
}

// RetryExecution retries execution (simple re-execution for SVG mode)
func (e *SVGExecutor) RetryExecution(ctx context.Context, projectSpec map[string]interface{}, outputDir string, maxRetries int) (*llm.ExecutionStatus, error) {
	return e.Execute(ctx, projectSpec, outputDir)
}

// IsProjectGenerated checks if the output directory exists
func (e *SVGExecutor) IsProjectGenerated(ctx context.Context, projectPath string) bool {
	info, err := os.Stat(projectPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// CleanupProject removes the project directory
func (e *SVGExecutor) CleanupProject(ctx context.Context, projectPath string) error {
	return os.RemoveAll(projectPath)
}

// saveDebugInfo saves debug information if debug mode is enabled
func (e *SVGExecutor) saveDebugInfo(ctx context.Context, outputDir string, data map[string]interface{}) error {
	if !e.debug {
		return nil
	}

	debugDir := filepath.Join(outputDir, ".debug")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Unix()
	debugPath := filepath.Join(debugDir, fmt.Sprintf("svg-executor-%d.json", timestamp))

	jsonData, err := marshalJSON(data)
	if err != nil {
		return err
	}

	return os.WriteFile(debugPath, jsonData, 0644)
}

// Helper functions for JSON operations (to avoid circular imports)
func parseJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

func marshalJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
