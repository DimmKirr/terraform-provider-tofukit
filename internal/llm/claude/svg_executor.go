package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

// SVGQueryFunc is a function type for making LLM queries (used for dependency injection in tests)
type SVGQueryFunc func(ctx context.Context, prompt string) (string, error)

// SVGExecutor generates images using Claude to create SVG wireframes, then converts to PNG
// Uses per-call isolated Claude homes and semaphore to prevent config file corruption
type SVGExecutor struct {
	originalClaudeHome       string // Original Claude home to copy from (e.g., ~/.claude)
	sessionID                string // Session ID for isolation (shared prefix for all calls)
	dangerouslySkipPerms     bool   // Skip permission prompts
	maxTurns                 int    // Max turns for Claude CLI
	debug                    bool
	outputPath               string
	semaphore                chan struct{} // Limits concurrent Claude calls
	maxConcurrentClaudeCalls int           // Configured limit for parallel calls
	queryFunc                SVGQueryFunc  // Injectable query function (nil = use real Claude)
}

// NewSVGExecutor creates a new SVG executor that uses Claude for wireframe generation
// Each Claude call gets its own isolated home directory for safe parallel execution
// maxConcurrentClaudeCalls controls the semaphore size (default: 4 if <= 0)
func NewSVGExecutor(claudeHomeDir string, dangerouslySkipPermissions bool, maxTurns int, maxConcurrentClaudeCalls int) *SVGExecutor {
	// Default to 4 if not configured
	if maxConcurrentClaudeCalls <= 0 {
		maxConcurrentClaudeCalls = 4
	}

	// Expand ~ in claude home path
	if strings.HasPrefix(claudeHomeDir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			claudeHomeDir = filepath.Join(home, claudeHomeDir[2:])
		}
	}

	// Generate session ID once for all calls in this executor instance
	sessionID := GenerateSessionID()

	log.Printf("[INFO] SVGExecutor created with sessionID=%s, max_concurrent_claude_calls=%d (per-call isolation enabled)", sessionID, maxConcurrentClaudeCalls)

	return &SVGExecutor{
		originalClaudeHome:       claudeHomeDir,
		sessionID:                sessionID,
		dangerouslySkipPerms:     dangerouslySkipPermissions,
		maxTurns:                 maxTurns,
		debug:                    false,
		semaphore:                make(chan struct{}, maxConcurrentClaudeCalls),
		maxConcurrentClaudeCalls: maxConcurrentClaudeCalls,
	}
}

// Cleanup releases resources (no-op for SVGExecutor since each call cleans up its own isolation)
func (e *SVGExecutor) Cleanup() error {
	// Per-call isolation means each goroutine cleans up its own isolated home
	// Nothing to clean up at executor level
	return nil
}

// SetDebug enables or disables debug mode
func (e *SVGExecutor) SetDebug(debug bool) {
	e.debug = debug
}

// SetOutputPath sets the output path for debug files
func (e *SVGExecutor) SetOutputPath(outputPath string) {
	e.outputPath = outputPath
}

// SetSystemPrompt sets a custom system prompt (not used for SVG mode)
func (e *SVGExecutor) SetSystemPrompt(systemPrompt string) {
	// SVG executor uses its own system prompt
}

// SetModel sets the model (not used, always uses Claude haiku for SVG)
func (e *SVGExecutor) SetModel(model string) {
	// SVG mode uses haiku by default for speed
}

// SetQueryFunc sets a custom query function (for testing/mocking)
func (e *SVGExecutor) SetQueryFunc(fn SVGQueryFunc) {
	e.queryFunc = fn
}

// wrapPromptForSVG wraps the user's image prompt with instructions for SVG wireframe generation
func wrapPromptForSVG(originalPrompt string, width, height int) string {
	return fmt.Sprintf(`Generate a valid SVG 1.1 wireframe/schematic representing the following image composition.
This should be a visual layout guide as if you were creating a wireframe for an artist to follow.

Use simple shapes, lines, and labels to represent:
- Main subject placement and proportions
- Background elements and layers
- Key visual elements and their relationships
- Composition guidelines (rule of thirds, focal points)
- Approximate color regions (use solid fills with hex colors)

CRITICAL SVG 1.1 REQUIREMENTS:
- Output ONLY valid SVG 1.1 code, no explanations or markdown
- Start with <svg and end with </svg>
- ALL attributes MUST have quoted values (e.g., width="512" NOT width=512)
- Hex colors MUST be exactly 3 digits (#RGB) or 6 digits (#RRGGBB) - NEVER use 5, 7, or 8 digit hex colors
- Escape ampersands in text content as &amp; (e.g., "Tom &amp; Jerry" NOT "Tom & Jerry")
- Closing tags MUST be exactly </tagname> with NO extra characters after the tag name
- Text content must NOT contain unescaped quote characters - describe colors by name instead
- Use viewBox="0 0 %d %d"
- Use a clean, minimalist style

COMPLETE EXAMPLE SVG (person portrait wireframe):
<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <rect x="0" y="0" width="512" height="512" fill="#f5f5f5"/>
  <ellipse cx="256" cy="180" rx="80" ry="100" fill="#ffe0bd" stroke="#000000" stroke-width="2"/>
  <ellipse cx="256" cy="400" rx="120" ry="60" fill="#000000" opacity="0.1"/>
  <rect x="180" y="280" width="152" height="200" fill="#4a90d9"/>
  <text x="256" y="480" text-anchor="middle" font-size="14">Person &amp; shadow - centered composition</text>
</svg>

Your SVG must use width="%d" height="%d" viewBox="0 0 %d %d".

Image to create wireframe for:
%s`, width, height, width, height, width, height, originalPrompt)
}

// buildSVGFeedbackPrompt constructs a prompt that includes error feedback for Claude to fix
// TFK-11 Option B: Feedback loop for SVG validation errors
func buildSVGFeedbackPrompt(originalPrompt string, errorMessage string, failedSVG string) string {
	var feedback strings.Builder

	feedback.WriteString("IMPORTANT: Your previous SVG generation attempt FAILED with the following error:\n\n")
	feedback.WriteString("ERROR: ")
	feedback.WriteString(errorMessage)
	feedback.WriteString("\n\n")

	if failedSVG != "" {
		// Include a snippet of the failed SVG for context (first 500 chars)
		svgPreview := failedSVG
		if len(svgPreview) > 500 {
			svgPreview = svgPreview[:500] + "..."
		}
		feedback.WriteString("FAILED SVG (excerpt):\n")
		feedback.WriteString(svgPreview)
		feedback.WriteString("\n\n")
	}

	feedback.WriteString("COMMON SVG ISSUES TO AVOID:\n")
	feedback.WriteString("1. Inside <style> blocks, use CSS syntax (property: value;) NOT XML syntax (property=\"value\")\n")
	feedback.WriteString("   WRONG: .class { fill=\"none\" stroke=\"#FF0000\" }\n")
	feedback.WriteString("   RIGHT: .class { fill: none; stroke: #FF0000; }\n")
	feedback.WriteString("2. Hex colors must be 3 or 6 digits: #RGB or #RRGGBB (not 5 or 8 digits)\n")
	feedback.WriteString("3. Escape ampersands in text as &amp;\n")
	feedback.WriteString("4. All tags must be properly closed\n")
	feedback.WriteString("5. Attribute values must be quoted: width=\"100\" not width=100\n\n")

	feedback.WriteString("Please regenerate a VALID SVG that fixes the error above.\n\n")
	feedback.WriteString("---\n\n")
	feedback.WriteString("ORIGINAL REQUEST:\n")
	feedback.WriteString(originalPrompt)

	return feedback.String()
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
	// Debug: Log entry to verify function is being called (KIRR-129 debugging)
	inputPreview := svg
	if len(inputPreview) > 300 {
		inputPreview = inputPreview[:300]
	}
	log.Printf("[DEBUG sanitizeSVG] ENTRY - input length: %d, first 300 chars: %q", len(svg), inputPreview)

	// Fix backslash-escaped content (KIRR-129)
	// Claude sometimes outputs JSON-style escaping in SVG, including:
	// - Double-escaped quotes: =\"\\\"value\\\"\" (from double JSON encoding)
	// - Single-escaped quotes: ="\"value\""
	// - Literal newlines: \n instead of actual newline
	// In XML/SVG, backslash is NOT an escape character.
	//
	// We handle all levels of escaping by:
	// 1. Converting literal \n to actual newlines
	// 2. Removing all backslashes before quotes (handles any nesting level)
	// 3. Cleaning up any resulting double-quotes

	// Convert literal \n to actual newlines
	svg = strings.ReplaceAll(svg, `\n`, "\n")

	// Remove backslashes before quotes - repeat until no more found
	// This handles any level of escaping: \" → ", \\" → ", \\\" → ", etc.
	for strings.Contains(svg, `\"`) {
		svg = strings.ReplaceAll(svg, `\"`, `"`)
	}

	// Clean up any double-backslashes that might remain
	for strings.Contains(svg, `\\`) {
		svg = strings.ReplaceAll(svg, `\\`, `\`)
	}

	// Fix resulting double-quotes from pattern like ="\"value\""
	// After removing backslashes: ="" becomes ="
	svg = strings.ReplaceAll(svg, `=""`, `="`)
	// And at the end: "" becomes "
	svg = strings.ReplaceAll(svg, `""`, `"`)

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

	// Convert 8-digit hex colors to 6-digit + opacity (KIRR-132)
	// CSS Color Level 4 supports #RRGGBBAA but SVG 1.1 only supports #RGB or #RRGGBB
	// Claude sometimes generates 8-digit hex colors for transparency effects
	svg = convertEightDigitHexColors(svg)

	// Fix truncated 5-digit hex colors to valid 6-digit (TFK-11)
	// Claude sometimes outputs truncated hex colors like #00000 instead of #000000
	svg = convertFiveDigitHexColors(svg)

	// Fix invalid CSS syntax in <style> blocks (TFK-14)
	// OpenAI sometimes generates XML attribute syntax inside CSS: fill="none" instead of fill: none;
	svg = normalizeStyleBlockCSS(svg)

	// Fix malformed closing tags (KIRR-140) - FALLBACK SAFETY NET
	// When text content contains unescaped quotes like: <text>Color="#FF0000"</text">
	// Claude sometimes bleeds the quote into the closing tag: </text">
	// This is a last-resort fix; the prompt now instructs Claude to avoid this pattern.
	svg = fixMalformedClosingTags(svg)

	// Note: We don't fix empty tag pairs (<tag></tag> -> <tag/>) because Go's
	// regexp doesn't support backreferences. The SVG parser handles both forms.

	// Debug: Log exit
	outputPreview := svg
	if len(outputPreview) > 300 {
		outputPreview = outputPreview[:300]
	}
	log.Printf("[DEBUG sanitizeSVG] EXIT - output length: %d, first 300 chars: %q", len(svg), outputPreview)

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

// convertFiveDigitHexColors fixes truncated 5-digit hex colors to valid 6-digit (TFK-11)
// Converts: fill="#00000" -> fill="#000000" (prepends missing digit)
// Converts: stroke="#12345" -> stroke="#012345"
// SVG 1.1 only supports 3-digit (#RGB) or 6-digit (#RRGGBB) hex colors.
// Claude sometimes outputs truncated 5-digit hex colors which violate the spec.
func convertFiveDigitHexColors(svg string) string {
	// Match fill="#XXXXX" or stroke="#XXXXX" where XXXXX is exactly 5 hex digits
	// The regex captures: (1) fill or stroke, (2) 5-digit color
	re := regexp.MustCompile(`(fill|stroke)="#([0-9A-Fa-f]{5})"`)

	return re.ReplaceAllString(svg, `$1="#0$2"`)
}

// normalizeStyleBlockCSS fixes invalid CSS syntax in SVG <style> blocks (TFK-14)
// OpenAI GPT-image-1 sometimes generates XML attribute syntax inside CSS style blocks:
//
//	INVALID: .orbit { fill="none" stroke="#61DAFB" stroke-width="24" }
//	VALID:   .orbit { fill: none; stroke: #61DAFB; stroke-width: 24; }
//
// This function converts XML attribute syntax to proper CSS property syntax within <style> blocks.
func normalizeStyleBlockCSS(svg string) string {
	// Find all <style>...</style> blocks (case-insensitive, handles CDATA)
	styleRegex := regexp.MustCompile(`(?is)(<style[^>]*>)(.*?)(</style>)`)

	return styleRegex.ReplaceAllStringFunc(svg, func(match string) string {
		parts := styleRegex.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match // Safety: return original if no match
		}

		openTag := parts[1]      // <style> or <style type="text/css">
		styleContent := parts[2] // The CSS content
		closeTag := parts[3]     // </style>

		// Convert XML attribute syntax to CSS property syntax
		// Pattern: property="value" -> property: value;
		// This handles: fill="none", stroke="#61DAFB", font-family="Arial, sans-serif", etc.
		attrRegex := regexp.MustCompile(`([a-zA-Z-]+)="([^"]*)"`)
		normalizedCSS := attrRegex.ReplaceAllString(styleContent, `$1: $2;`)

		// Clean up any double semicolons that might result from already-semicolon-terminated values
		normalizedCSS = strings.ReplaceAll(normalizedCSS, ";;", ";")

		return openTag + normalizedCSS + closeTag
	})
}

// convertEightDigitHexColors converts CSS Color Level 4 8-digit hex colors to SVG 1.1 compatible format (KIRR-132)
// Converts: fill="#RRGGBBAA" -> fill="#RRGGBB" fill-opacity="0.XX"
// Converts: stroke="#RRGGBBAA" -> stroke="#RRGGBB" stroke-opacity="0.XX"
// SVG 1.1 only supports 3-digit (#RGB) or 6-digit (#RRGGBB) hex colors.
// CSS Color Level 4 adds optional alpha channel as 8-digit hex (#RRGGBBAA).
func convertEightDigitHexColors(svg string) string {
	// Match fill="#RRGGBBAA" or stroke="#RRGGBBAA" where AA is the alpha channel
	// The regex captures: (1) fill or stroke, (2) 6-digit color, (3) 2-digit alpha
	re := regexp.MustCompile(`(fill|stroke)="#([0-9A-Fa-f]{6})([0-9A-Fa-f]{2})"`)

	return re.ReplaceAllStringFunc(svg, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match // Safety: return original if no match
		}

		attr := parts[1]  // "fill" or "stroke"
		color := parts[2] // "RRGGBB"
		alpha := parts[3] // "AA"

		// Convert alpha hex to decimal opacity (0.00 to 1.00)
		alphaInt, err := strconv.ParseInt(alpha, 16, 64)
		if err != nil {
			return match // Safety: return original if parse fails
		}
		opacity := float64(alphaInt) / 255.0

		// Return the converted attribute with separate opacity
		// e.g., fill="#000000" fill-opacity="0.08"
		return fmt.Sprintf(`%s="#%s" %s-opacity="%.2f"`, attr, color, attr, opacity)
	})
}

// fixMalformedClosingTags fixes closing tags with extra characters (KIRR-140)
// Pattern: </tagname"> → </tagname>
// This happens when Claude's text content contains unescaped quotes that bleed into closing tags
func fixMalformedClosingTags(svg string) string {
	// Match closing tags with extra quote before >
	// e.g., </text"> → </text>
	// e.g., </tspan"> → </tspan>
	malformedCloseTagRegex := regexp.MustCompile(`</(\w+)">`)
	return malformedCloseTagRegex.ReplaceAllString(svg, `</$1>`)
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
// Each file gets its own isolated Claude home for safe parallel execution
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

			// Acquire semaphore to limit concurrent Claude calls
			e.semaphore <- struct{}{}
			defer func() { <-e.semaphore }()

			// Create per-call isolated Claude home: sessionID + file path suffix
			// Result: ~/.claude-tofukit-{sessionID}-{sanitized-path}
			isolatedHome, err := CreateIsolatedClaudeHome(e.originalClaudeHome, e.sessionID, path)
			if err != nil {
				errChan <- fmt.Errorf("failed to create isolated home for '%s': %w", path, err)
				return
			}
			// Cleanup this call's isolated home when done
			defer func() {
				if cleanupErr := isolatedHome.Cleanup(); cleanupErr != nil {
					log.Printf("[WARN] Failed to cleanup isolated home for '%s': %v", path, cleanupErr)
				}
			}()

			// Create executor for this call using the isolated home
			callExecutor := &Executor{
				client:       NewClient(isolatedHome.Path(), e.dangerouslySkipPerms, e.maxTurns),
				debug:        e.debug,
				outputPath:   e.outputPath,
				maxTurns:     e.maxTurns,
				isolatedHome: nil, // We manage isolation ourselves
			}

			// TFK-11 Option B: SVG generation with feedback loop
			// Retry up to maxRetries times, sending parse errors back to Claude for correction
			const maxSVGRetries = 3
			var lastError error
			var svgContent string
			currentPrompt := svgPrompt

			for attempt := 1; attempt <= maxSVGRetries; attempt++ {
				if e.debug && attempt > 1 {
					tflog.Info(ctx, "SVG feedback loop: retrying generation", map[string]interface{}{
						"path":    path,
						"attempt": attempt,
						"error":   lastError.Error(),
					})
				}

				// Save the prompt being sent (including feedback prompts) for debugging
				if e.debug {
					debugDir := filepath.Join(outputDir, ".debug")
					if err := os.MkdirAll(debugDir, 0755); err == nil {
						suffix := ""
						if attempt > 1 {
							suffix = fmt.Sprintf(".attempt%d", attempt)
						}
						promptPath := filepath.Join(debugDir, strings.ReplaceAll(path, "/", "-")+suffix+".prompt.txt")
						_ = os.WriteFile(promptPath, []byte(currentPrompt), 0644)
					}
				}

				// Use Claude to generate SVG (or mock if queryFunc is set)
				var response string
				var queryErr error
				if e.queryFunc != nil {
					// Use injected query function (for testing)
					response, queryErr = e.queryFunc(ctx, currentPrompt)
				} else {
					// Use real Claude executor
					response, queryErr = callExecutor.Query(ctx, []string{currentPrompt}, "")
				}
				if queryErr != nil {
					errChan <- fmt.Errorf("failed to generate SVG for '%s': %w", path, queryErr)
					return
				}

				// Save raw Claude response for debugging
				if e.debug {
					debugDir := filepath.Join(outputDir, ".debug")
					if err := os.MkdirAll(debugDir, 0755); err == nil {
						suffix := ""
						if attempt > 1 {
							suffix = fmt.Sprintf(".attempt%d", attempt)
						}
						rawPath := filepath.Join(debugDir, strings.ReplaceAll(path, "/", "-")+suffix+".claude-response.txt")
						_ = os.WriteFile(rawPath, []byte(response), 0644)
					}
				}

				// Extract SVG from response
				svgContent, err = extractSVGFromResponse(response)
				if err != nil {
					lastError = fmt.Errorf("failed to extract SVG: %w", err)
					currentPrompt = buildSVGFeedbackPrompt(svgPrompt, lastError.Error(), "")
					continue
				}

				// For non-SVG targets, validate by attempting PNG conversion
				ext := strings.ToLower(filepath.Ext(path))
				if ext != ".svg" {
					_, convErr := convertSVGToPNG(svgContent, width, height)
					if convErr != nil {
						// Save the problematic SVG for debugging
						if e.debug {
							debugDir := filepath.Join(outputDir, ".debug")
							if mkErr := os.MkdirAll(debugDir, 0755); mkErr == nil {
								failedSvgPath := filepath.Join(debugDir, strings.ReplaceAll(path, "/", "-")+fmt.Sprintf(".attempt%d.failed.svg", attempt))
								_ = os.WriteFile(failedSvgPath, []byte(svgContent), 0644)
							}
						}

						lastError = fmt.Errorf("SVG parsing/conversion failed: %w", convErr)
						currentPrompt = buildSVGFeedbackPrompt(svgPrompt, lastError.Error(), svgContent)
						continue
					}
				}

				// Success - SVG is valid
				lastError = nil
				break
			}

			// If all retries failed, report the error
			if lastError != nil {
				// Save final failed SVG
				if e.debug && svgContent != "" {
					debugDir := filepath.Join(outputDir, ".debug")
					if mkErr := os.MkdirAll(debugDir, 0755); mkErr == nil {
						failedSvgPath := filepath.Join(debugDir, strings.ReplaceAll(path, "/", "-")+".failed.svg")
						_ = os.WriteFile(failedSvgPath, []byte(svgContent), 0644)
					}
				}
				errChan <- fmt.Errorf("failed to generate valid SVG for '%s' after %d attempts: %w", path, maxSVGRetries, lastError)
				return
			}

			// Save sanitized SVG for debugging if enabled
			if e.debug {
				debugDir := filepath.Join(outputDir, ".debug")
				svgPath := filepath.Join(debugDir, strings.TrimSuffix(path, filepath.Ext(path))+".svg")
				if err := os.MkdirAll(filepath.Dir(svgPath), 0755); err == nil {
					_ = os.WriteFile(svgPath, []byte(svgContent), 0644)
				}
			}

			// Determine output format based on file extension (TFK-12)
			// If target is .svg, keep as SVG; otherwise convert to PNG
			fullPath := filepath.Join(outputDir, path)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				errChan <- fmt.Errorf("failed to create directory for '%s': %w", path, err)
				return
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".svg" {
				// Keep as SVG - no conversion needed
				if err := os.WriteFile(fullPath, []byte(svgContent), 0644); err != nil {
					errChan <- fmt.Errorf("failed to write SVG file '%s': %w", path, err)
					return
				}

				if e.debug {
					tflog.Debug(ctx, "Generated wireframe SVG (no conversion)", map[string]interface{}{
						"path": path,
						"size": len(svgContent),
					})
				}
			} else {
				// Convert SVG to PNG for .png, .jpg, .jpeg, etc.
				// Note: We already validated conversion in the retry loop, so this should succeed
				pngData, err := convertSVGToPNG(svgContent, width, height)
				if err != nil {
					errChan <- fmt.Errorf("failed to convert SVG to PNG for '%s': %w", path, err)
					return
				}

				if err := os.WriteFile(fullPath, pngData, 0644); err != nil {
					errChan <- fmt.Errorf("failed to write PNG file '%s': %w", path, err)
					return
				}

				if e.debug {
					tflog.Debug(ctx, "Generated wireframe PNG", map[string]interface{}{
						"path": path,
						"size": len(pngData),
					})
				}
			}

			resultsMu.Lock()
			generatedFiles = append(generatedFiles, path)
			resultsMu.Unlock()
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

// Query executes a simple query (creates temporary isolated home)
func (e *SVGExecutor) Query(ctx context.Context, instructions []string, model string) (string, error) {
	// Create temporary isolated home for this query: sessionID + "query" suffix
	isolatedHome, err := CreateIsolatedClaudeHome(e.originalClaudeHome, e.sessionID, "query")
	if err != nil {
		return "", fmt.Errorf("failed to create isolated home for query: %w", err)
	}
	defer isolatedHome.Cleanup()

	executor := &Executor{
		client:       NewClient(isolatedHome.Path(), e.dangerouslySkipPerms, e.maxTurns),
		debug:        e.debug,
		outputPath:   e.outputPath,
		maxTurns:     e.maxTurns,
		isolatedHome: nil,
	}
	return executor.Query(ctx, instructions, model)
}

// Validate checks if the executor is properly configured
func (e *SVGExecutor) Validate(ctx context.Context) error {
	// Create temporary client to validate (uses original home, read-only check)
	client := NewClient(e.originalClaudeHome, e.dangerouslySkipPerms, e.maxTurns)
	return client.ValidateClaudeCodeAvailability(ctx)
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
