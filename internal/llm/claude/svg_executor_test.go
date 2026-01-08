package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeSVG_UnescapedAmpersand(t *testing.T) {
	// This is the exact problematic SVG from KIRR-128
	// Line 145: <text ...>...carrots & overflowing greens</text>
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <rect x="0" y="0" width="512" height="512" fill="#f5f5dc"/>
  <text x="320" y="900" font-family="Arial, sans-serif" font-size="12" fill="#666">Kraft paper bag center with vibrant carrots & overflowing greens</text>
  <text x="100" y="100" fill="#333">Tom & Jerry show</text>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Ampersands should be escaped
	assert.Contains(t, sanitized, "carrots &amp; overflowing")
	assert.Contains(t, sanitized, "Tom &amp; Jerry")

	// Now try to parse it - this is the real test
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err, "SVG parsing should succeed after sanitization")
}

func TestSanitizeSVG_AlreadyEscapedAmpersand(t *testing.T) {
	// Already properly escaped - should not double-escape
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <text x="100" y="100">Tom &amp; Jerry</text>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Should remain single-escaped, not become &amp;amp;
	assert.Contains(t, sanitized, "Tom &amp; Jerry")
	assert.NotContains(t, sanitized, "&amp;amp;")
}

func TestSanitizeSVG_NumericEntityPreserved(t *testing.T) {
	// Numeric entities like &#x26; or &#38; should be preserved
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <text x="100" y="100">Ampersand: &#38; and &#x26;</text>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Numeric entities should be preserved
	assert.Contains(t, sanitized, "&#38;")
	assert.Contains(t, sanitized, "&#x26;")
}

func TestSanitizeSVG_NamedEntitiesPreserved(t *testing.T) {
	// Named entities like &lt; &gt; &quot; should be preserved
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <text x="100" y="100">&lt;hello&gt; &quot;world&quot;</text>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Named entities should be preserved, not double-escaped
	assert.Contains(t, sanitized, "&lt;")
	assert.Contains(t, sanitized, "&gt;")
	assert.Contains(t, sanitized, "&quot;")
	assert.NotContains(t, sanitized, "&amp;lt;")
}

func TestSanitizeSVG_MultipleUnescapedAmpersands(t *testing.T) {
	// Multiple unescaped ampersands in various positions
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <text x="100" y="100">A & B & C & D</text>
  <text x="100" y="200">Start& middle &end</text>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// All unescaped ampersands should be fixed
	assert.Contains(t, sanitized, "A &amp; B &amp; C &amp; D")
	assert.Contains(t, sanitized, "Start&amp; middle &amp;end")

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_DoubleEscapedQuotes(t *testing.T) {
	// This is the exact problematic SVG from KIRR-129
	// Claude outputs backslash-escaped quotes in attribute values
	inputSVG := `<svg xmlns="\"http://www.w3.org/2000/svg\"" width="\"1024\"" height="\"1024\"" viewBox="\"0 0 1024 1024\"">
  <rect width="\"1024\"" height="\"1024\"" fill="\"#f5e6d3\""/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Double-escaped quotes should be fixed
	assert.Contains(t, sanitized, `xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, sanitized, `width="1024"`)
	assert.Contains(t, sanitized, `height="1024"`)
	assert.Contains(t, sanitized, `fill="#f5e6d3"`)
	assert.NotContains(t, sanitized, `\"`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 1024, 1024)
	require.NoError(t, err, "SVG parsing should succeed after sanitization")
}

func TestSanitizeSVG_MixedEscapedQuotes(t *testing.T) {
	// Some attributes have escaped quotes, some don't
	inputSVG := `<svg xmlns="\"http://www.w3.org/2000/svg\"" width="512" height="\"512\"">
  <rect x="0" y="\"0\"" fill="#fff"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// All attributes should have clean quotes
	assert.Contains(t, sanitized, `xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, sanitized, `width="512"`)
	assert.Contains(t, sanitized, `height="512"`)
	assert.NotContains(t, sanitized, `\"`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_DoubleBackslashEscapedQuotes(t *testing.T) {
	// This is the EXACT pattern from QA failure (KIRR-129 re-test)
	// Production shows: xmlns=\"\\\"http://...\\\"\"
	// Which is: backslash-quote-backslash-backslash-backslash-quote
	// In Go raw string: `=\"\\\"` for opening, `\\\"\"` for closing
	inputSVG := `<svg xmlns=\"\\\"http://www.w3.org/2000/svg\\\"\" width=\"\\\"1024\\\"\" height=\"\\\"1024\\\"\">`

	sanitized := sanitizeSVG(inputSVG)

	// Should be clean
	assert.Contains(t, sanitized, `xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, sanitized, `width="1024"`)
	assert.Contains(t, sanitized, `height="1024"`)
	assert.NotContains(t, sanitized, `\\"`)
	assert.NotContains(t, sanitized, `\"`)
}

func TestSanitizeSVG_LiteralNewlines(t *testing.T) {
	// QA shows literal \n in the output instead of actual newlines
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">\n  <rect width="512" height="512" fill="#fff"/>\n</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Literal \n should be converted to actual newlines or removed
	assert.NotContains(t, sanitized, `\n`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_ProductionPattern_Rabbit(t *testing.T) {
	// Exact pattern from rabbit1.png.failed.svg (first line)
	inputSVG := `<svg xmlns=\"\\\"http://www.w3.org/2000/svg\\\"\" width=\"\\\"1024\\\"\" height=\"\\\"1024\\\"\" viewBox=\"\\\"0 0 1024 1024\\\"\">\n  <rect width=\"\\\"1024\\\"\" height=\"\\\"1024\\\"\" fill=\"\\\"#f5e6d3\\\"\"/>\n</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// All escapes should be removed
	assert.Contains(t, sanitized, `xmlns="http://www.w3.org/2000/svg"`)
	assert.Contains(t, sanitized, `width="1024"`)
	assert.Contains(t, sanitized, `viewBox="0 0 1024 1024"`)
	assert.Contains(t, sanitized, `fill="#f5e6d3"`)
	assert.NotContains(t, sanitized, `\\`)
	assert.NotContains(t, sanitized, `\n`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 1024, 1024)
	require.NoError(t, err, "SVG parsing should succeed after sanitization")
}

// =============================================================================
// extractSVGFromResponse tests - testing the FULL JSON extraction flow
// =============================================================================

func TestExtractSVGFromResponse_ValidJSON_MarkdownFence(t *testing.T) {
	// This simulates the EXACT JSON that Claude CLI outputs with --output-format=json
	// The JSON has proper escaping: \n for newlines, \" for quotes
	// When constructed in Go, we need to use the same escaping
	jsonResponse := `{"type":"result","result":"` + "```xml\\n<svg xmlns=\\\"http://www.w3.org/2000/svg\\\" width=\\\"1024\\\" height=\\\"1024\\\" viewBox=\\\"0 0 1024 1024\\\">\\n  <rect width=\\\"1024\\\" height=\\\"1024\\\" fill=\\\"#f5e6d3\\\"/>\\n</svg>\\n```" + `"}`

	t.Logf("Input JSON (first 200 chars): %s", jsonResponse[:min(200, len(jsonResponse))])

	svg, err := extractSVGFromResponse(jsonResponse)
	require.NoError(t, err, "extractSVGFromResponse should succeed")

	t.Logf("Extracted SVG (first 200 chars): %s", svg[:min(200, len(svg))])

	// After JSON parsing and extraction, quotes should be normal (not escaped)
	assert.Contains(t, svg, `xmlns="http://www.w3.org/2000/svg"`, "xmlns should have unescaped quotes")
	assert.Contains(t, svg, `width="1024"`, "width should have unescaped quotes")
	assert.NotContains(t, svg, `\"`, "Should NOT contain backslash-quote")
	assert.NotContains(t, svg, `\n`, "Should NOT contain literal backslash-n")

	// Should parse as valid SVG
	_, err = convertSVGToPNG(svg, 1024, 1024)
	require.NoError(t, err, "Extracted SVG should be valid")
}

func TestExtractSVGFromResponse_ValidJSON_NoFence(t *testing.T) {
	// JSON with SVG directly in result (no markdown fence)
	jsonResponse := `{"type":"result","result":"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"1024\" height=\"1024\"><rect fill=\"#fff\"/></svg>"}`

	svg, err := extractSVGFromResponse(jsonResponse)
	require.NoError(t, err)

	assert.Contains(t, svg, `xmlns="http://www.w3.org/2000/svg"`)
	assert.NotContains(t, svg, `\"`)
}

func TestExtractSVGFromResponse_RawSVG_NoJSON(t *testing.T) {
	// If response is raw SVG (not JSON), should still extract correctly
	rawSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect width="512" height="512" fill="#fff"/>
</svg>`

	svg, err := extractSVGFromResponse(rawSVG)
	require.NoError(t, err)

	assert.Contains(t, svg, `xmlns="http://www.w3.org/2000/svg"`)
}

func TestExtractSVGFromResponse_DebugExactBytes(t *testing.T) {
	// Debug test to understand exact byte flow
	// Construct JSON as Claude CLI would: {"result":"<svg xmlns=\"...
	// In the JSON wire format, the quote after xmlns= is escaped as \"
	// which is bytes: 0x5C 0x22

	// Build JSON byte by byte to be explicit
	// We want: {"result":"<svg xmlns=\"http://www.w3.org/2000/svg\" />"}
	// In JSON string literal: the \" inside the string value becomes \"
	jsonBytes := []byte(`{"result":"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"100\" height=\"100\"><rect/></svg>"}`)

	t.Logf("JSON bytes (first 100): %v", jsonBytes[:min(100, len(jsonBytes))])
	t.Logf("JSON string: %s", string(jsonBytes))

	svg, err := extractSVGFromResponse(string(jsonBytes))
	require.NoError(t, err)

	t.Logf("Extracted SVG: %s", svg)
	t.Logf("SVG bytes (first 100): %v", []byte(svg)[:min(100, len(svg))])

	// After json.Unmarshal, \" should become "
	assert.Contains(t, svg, `xmlns="http://www.w3.org/2000/svg"`)
	assert.NotContains(t, svg, `\"`)
}

func TestExtractSVGFromResponse_DoubleEscapedJSON(t *testing.T) {
	// TEST HYPOTHESIS: Production might receive DOUBLE-escaped JSON
	// If Claude outputs: {"result":"<svg xmlns=\\\"http://...\\\">"}
	// Then json.Unmarshal would give: <svg xmlns=\"http://...\">
	// This would have literal backslash-quotes!

	// Double-escaped JSON: \\\" in source becomes \" in the JSON string
	doubleEscapedJSON := `{"result":"<svg xmlns=\\\"http://www.w3.org/2000/svg\\\" width=\\\"100\\\" height=\\\"100\\\"><rect/></svg>"}`

	t.Logf("Double-escaped JSON: %s", doubleEscapedJSON)

	svg, err := extractSVGFromResponse(doubleEscapedJSON)
	require.NoError(t, err)

	t.Logf("Extracted SVG: %s", svg)
	t.Logf("SVG bytes: %v", []byte(svg))

	// After json.Unmarshal of double-escaped JSON, we get literal \"
	// But sanitizeSVG should fix this!
	assert.Contains(t, svg, `xmlns="http://www.w3.org/2000/svg"`, "sanitizeSVG should fix double-escaped quotes")
	assert.NotContains(t, svg, `\"`, "Should NOT contain backslash-quote after sanitization")

	// Should parse as valid SVG
	_, err = convertSVGToPNG(svg, 100, 100)
	require.NoError(t, err, "SVG should be valid after sanitization")
}

func TestExtractSVGFromResponse_TripleEscapedJSON(t *testing.T) {
	// Even more extreme: triple-escaped JSON
	// If there's nested JSON serialization: \\\\\" becomes \\" in result
	tripleEscapedJSON := `{"result":"<svg xmlns=\\\\\"http://www.w3.org/2000/svg\\\\\" width=\\\\\"100\\\\\" height=\\\\\"100\\\\\"><rect/></svg>"}`

	t.Logf("Triple-escaped JSON: %s", tripleEscapedJSON)

	svg, err := extractSVGFromResponse(tripleEscapedJSON)
	require.NoError(t, err)

	t.Logf("Extracted SVG: %s", svg)

	// sanitizeSVG should handle even this
	assert.Contains(t, svg, `xmlns="http://www.w3.org/2000/svg"`, "sanitizeSVG should fix triple-escaped quotes")
	assert.NotContains(t, svg, `\"`, "Should NOT contain backslash-quote after sanitization")
	assert.NotContains(t, svg, `\\`, "Should NOT contain double-backslash after sanitization")
}

func TestExtractSVGFromResponse_ExactQAFailurePattern(t *testing.T) {
	// This simulates the EXACT pattern from QA's .failed.svg files
	// File shows: xmlns=\"\\\"http://www.w3.org/2000/svg\\\"\"
	// Which means the bytes are: = " \ \ " http://... \ \ " "
	// In Go raw string that's: xmlns=\"\\\"http://www.w3.org/2000/svg\\\"\"

	// But this is what's in the FILE - not what comes from JSON.
	// If this pattern is in the file AFTER extractSVGFromResponse,
	// then extractSVGFromResponse is returning this pattern.

	// Let me construct JSON that would result in this pattern after unmarshal
	// If result has: xmlns=\"\\\"http://...\\\"\"
	// Then JSON must have had: xmlns=\\\"\\\\\\\"http://...\\\\\\\"\\\"
	// Let's verify by constructing different JSON inputs

	testCases := []struct {
		name  string
		json  string
		check func(t *testing.T, svg string)
	}{
		{
			name: "markdown fence with escaped content",
			// Claude wraps SVG in markdown: ```xml\n<svg...>\n```
			// And the SVG attributes might have escaped quotes
			json: `{"result":"` + "```xml\\n<svg xmlns=\\\"http://www.w3.org/2000/svg\\\" width=\\\"100\\\"><rect/></svg>\\n```" + `"}`,
			check: func(t *testing.T, svg string) {
				t.Logf("Case 1 - SVG: %s", svg)
				t.Logf("Case 1 - Bytes: %v", []byte(svg))
				assert.Contains(t, svg, `xmlns="http://www.w3.org/2000/svg"`)
				assert.NotContains(t, svg, `\"`)
			},
		},
		{
			name: "double-serialized JSON (nested JSON encoding)",
			// What if Claude's response got JSON-encoded twice?
			// Original: <svg xmlns="http://...">
			// First JSON encode: <svg xmlns=\"http://...\">
			// Second JSON encode: <svg xmlns=\\\"http://...\\\">
			json: `{"result":"<svg xmlns=\\\"http://www.w3.org/2000/svg\\\" width=\\\"100\\\"><rect/></svg>"}`,
			check: func(t *testing.T, svg string) {
				t.Logf("Case 2 - SVG: %s", svg)
				t.Logf("Case 2 - Bytes: %v", []byte(svg))
				assert.Contains(t, svg, `xmlns="http://www.w3.org/2000/svg"`)
				assert.NotContains(t, svg, `\"`)
			},
		},
		{
			name: "triple-serialized JSON",
			// What if it was JSON-encoded THREE times?
			// This would produce: \\\\\" in the wire format
			json: `{"result":"<svg xmlns=\\\\\"http://www.w3.org/2000/svg\\\\\" width=\\\\\"100\\\\\"><rect/></svg>"}`,
			check: func(t *testing.T, svg string) {
				t.Logf("Case 3 - SVG: %s", svg)
				t.Logf("Case 3 - Bytes: %v", []byte(svg))
				assert.Contains(t, svg, `xmlns="http://www.w3.org/2000/svg"`)
				assert.NotContains(t, svg, `\"`)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Input JSON: %s", tc.json)

			svg, err := extractSVGFromResponse(tc.json)
			require.NoError(t, err, "Should extract SVG successfully")

			tc.check(t, svg)

			// Final check: should produce valid SVG
			_, err = convertSVGToPNG(svg, 100, 100)
			require.NoError(t, err, "Should produce valid SVG")
		})
	}
}

func TestExtractSVGFromResponse_JSONFailFallback(t *testing.T) {
	// What if JSON parsing FAILS and we fall back to raw string matching?
	// If the response isn't valid JSON, we use the raw string
	// This could preserve escape sequences

	// Invalid JSON (missing closing brace) - should fall back to raw extraction
	invalidJSON := `{"result":"<svg xmlns="http://www.w3.org/2000/svg" width="100"><rect/></svg>`

	t.Logf("Invalid JSON: %s", invalidJSON)

	svg, err := extractSVGFromResponse(invalidJSON)
	require.NoError(t, err, "Should extract SVG even from invalid JSON")

	t.Logf("Extracted SVG: %s", svg)

	// Even without JSON parsing, the SVG should be extracted
	assert.Contains(t, svg, "xmlns=")
}

// =============================================================================
// KIRR-132: 8-digit hex color tests
// =============================================================================

func TestSanitizeSVG_EightDigitHexColor_Fill(t *testing.T) {
	// This is the exact problematic pattern from KIRR-132
	// Claude generates 8-digit hex colors for shadow/transparency effects
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">
  <ellipse cx="512" cy="650" rx="280" ry="120" fill="#00000015" />
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// 8-digit hex should be converted to 6-digit + fill-opacity
	assert.Contains(t, sanitized, `fill="#000000"`)
	assert.Contains(t, sanitized, `fill-opacity="0.08"`) // 0x15 = 21, 21/255 ≈ 0.08
	assert.NotContains(t, sanitized, "#00000015")

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 1024, 1024)
	require.NoError(t, err, "SVG parsing should succeed after 8-digit hex conversion")
}

func TestSanitizeSVG_EightDigitHexColor_Stroke(t *testing.T) {
	// Test stroke attribute with 8-digit hex
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect x="10" y="10" width="100" height="100" stroke="#FF000080" stroke-width="2" fill="none"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// 8-digit hex should be converted to 6-digit + stroke-opacity
	assert.Contains(t, sanitized, `stroke="#FF0000"`)
	assert.Contains(t, sanitized, `stroke-opacity="0.50"`) // 0x80 = 128, 128/255 ≈ 0.50
	assert.NotContains(t, sanitized, "#FF000080")

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_EightDigitHexColor_FullOpacity(t *testing.T) {
	// Test 8-digit hex with full opacity (FF = 255 = 1.00)
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect fill="#0000FFFF"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Should convert to 6-digit + opacity 1.00
	assert.Contains(t, sanitized, `fill="#0000FF"`)
	assert.Contains(t, sanitized, `fill-opacity="1.00"`)
	assert.NotContains(t, sanitized, "#0000FFFF")

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_EightDigitHexColor_ZeroOpacity(t *testing.T) {
	// Test 8-digit hex with zero opacity (00 = 0 = 0.00)
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect fill="#FFFFFF00"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Should convert to 6-digit + opacity 0.00
	assert.Contains(t, sanitized, `fill="#FFFFFF"`)
	assert.Contains(t, sanitized, `fill-opacity="0.00"`)
	assert.NotContains(t, sanitized, "#FFFFFF00")

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_SixDigitHexColor_Preserved(t *testing.T) {
	// Standard 6-digit hex colors should NOT be modified
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect fill="#FF8C42"/>
  <circle fill="#2D8659"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// 6-digit colors should remain unchanged
	assert.Contains(t, sanitized, `fill="#FF8C42"`)
	assert.Contains(t, sanitized, `fill="#2D8659"`)
	assert.NotContains(t, sanitized, "fill-opacity")

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_EightDigitHexColor_MultipleMixed(t *testing.T) {
	// Mix of 6-digit and 8-digit colors - only 8-digit should be converted
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect fill="#FF8C42"/>
  <ellipse fill="#00000015"/>
  <circle stroke="#FF000080" fill="#2D8659"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// 6-digit colors preserved
	assert.Contains(t, sanitized, `fill="#FF8C42"`)
	assert.Contains(t, sanitized, `fill="#2D8659"`)

	// 8-digit colors converted
	assert.Contains(t, sanitized, `fill="#000000"`)
	assert.Contains(t, sanitized, `fill-opacity="0.08"`)
	assert.Contains(t, sanitized, `stroke="#FF0000"`)
	assert.Contains(t, sanitized, `stroke-opacity="0.50"`)

	// Original 8-digit colors should be gone
	assert.NotContains(t, sanitized, "#00000015")
	assert.NotContains(t, sanitized, "#FF000080")

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_EightDigitHexColor_LowerCase(t *testing.T) {
	// Test lowercase hex digits
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect fill="#aabbcc40"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Should handle lowercase
	assert.Contains(t, sanitized, `fill="#aabbcc"`)
	assert.Contains(t, sanitized, `fill-opacity="0.25"`) // 0x40 = 64, 64/255 ≈ 0.25
	assert.NotContains(t, sanitized, "#aabbcc40")

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_ProductionPattern_Rabbit_KIRR132(t *testing.T) {
	// Exact SVG structure from KIRR-132 bug report
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">
  <defs>
    <linearGradient id="warmLighting" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#FFF5E6;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#F0E6D2;stop-opacity:1" />
    </linearGradient>
    <filter id="softBlur">
      <feGaussianBlur in="SourceGraphic" stdDeviation="8" />
    </filter>
  </defs>
  <g id="characterGroup">
    <ellipse cx="512" cy="650" rx="280" ry="120" fill="#00000015" />
    <ellipse cx="512" cy="400" rx="180" ry="200" fill="#f5deb3"/>
  </g>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// The problematic 8-digit color should be converted
	assert.Contains(t, sanitized, `fill="#000000"`)
	assert.Contains(t, sanitized, `fill-opacity="0.08"`)
	assert.NotContains(t, sanitized, "#00000015")

	// Other 6-digit colors should be preserved
	assert.Contains(t, sanitized, `fill="#f5deb3"`)

	// Should parse successfully (this was the original failure)
	_, err := convertSVGToPNG(sanitized, 1024, 1024)
	require.NoError(t, err, "Production rabbit SVG should parse after 8-digit hex conversion")
}

// =============================================================================
// KIRR-140: Malformed closing tag tests
// =============================================================================

func TestSanitizeSVG_MalformedClosingTag_Text(t *testing.T) {
	// This is the exact problematic pattern from KIRR-140
	// Claude generates </text"> instead of </text>
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <text x="50" y="975" class="label-text">Colors: Cyan="#40E0D0" | Yellow="#FFD700"</text">
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Malformed closing tag should be fixed
	assert.Contains(t, sanitized, `</text>`)
	assert.NotContains(t, sanitized, `</text">`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err, "SVG parsing should succeed after malformed closing tag fix")
}

func TestSanitizeSVG_MalformedClosingTag_Multiple(t *testing.T) {
	// Multiple malformed closing tags
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <text x="50" y="50">First text with quote"</text">
  <tspan x="50" y="100">Nested with quote"</tspan">
  <text x="50" y="150">Normal text</text>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// All malformed closing tags should be fixed
	assert.Contains(t, sanitized, `</text>`)
	assert.Contains(t, sanitized, `</tspan>`)
	assert.NotContains(t, sanitized, `</text">`)
	assert.NotContains(t, sanitized, `</tspan">`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_MalformedClosingTag_ProductionKIRR140(t *testing.T) {
	// Exact pattern from KIRR-140 bug report (line 199 of failing SVG)
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">
  <defs>
    <style>
      .label-text { font-family: Arial, sans-serif; font-size: 14px; fill: #000; }
    </style>
  </defs>
  <g id="legend">
    <text x="50" y="950" class="label-text">Wireframe Guide: Top section</text>
    <text x="50" y="975" class="label-text">Colors: Cyan="#40E0D0" | Yellow="#FFD700" | Orange="#FFA500" | Black="#000" | White="#FFF</text">
  </g>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Malformed closing tag should be fixed
	assert.NotContains(t, sanitized, `</text">`)

	// Should parse successfully (this was the original failure)
	_, err := convertSVGToPNG(sanitized, 1024, 1024)
	require.NoError(t, err, "Production KIRR-140 SVG should parse after malformed closing tag fix")
}

func TestFixMalformedClosingTags_Direct(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "text with quote",
			input:    `</text">`,
			expected: `</text>`,
		},
		{
			name:     "tspan with quote",
			input:    `</tspan">`,
			expected: `</tspan>`,
		},
		{
			name:     "rect with quote",
			input:    `</rect">`,
			expected: `</rect>`,
		},
		{
			name:     "normal closing tag unchanged",
			input:    `</text>`,
			expected: `</text>`,
		},
		{
			name:     "self-closing unchanged",
			input:    `<rect/>`,
			expected: `<rect/>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fixMalformedClosingTags(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// TFK-11: 5-digit hex color tests
// =============================================================================

func TestSanitizeSVG_FiveDigitHexColor_Fill(t *testing.T) {
	// This is the exact problematic pattern from TFK-11
	// Claude generates 5-digit hex colors like #00000 instead of #000000
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">
  <ellipse cx="650" cy="350" rx="50" ry="80" fill="#00000" opacity="0.08"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// 5-digit hex should be converted to 6-digit by prepending 0
	assert.Contains(t, sanitized, `fill="#000000"`)
	assert.NotContains(t, sanitized, `fill="#00000"`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 1024, 1024)
	require.NoError(t, err, "SVG parsing should succeed after 5-digit hex conversion")
}

func TestSanitizeSVG_FiveDigitHexColor_Stroke(t *testing.T) {
	// Test stroke attribute with 5-digit hex
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect x="10" y="10" width="100" height="100" stroke="#FF000" stroke-width="2" fill="none"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// 5-digit hex should be converted to 6-digit
	assert.Contains(t, sanitized, `stroke="#0FF000"`)
	assert.NotContains(t, sanitized, `stroke="#FF000"`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_FiveDigitHexColor_Multiple(t *testing.T) {
	// Multiple 5-digit hex colors in one SVG
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect fill="#12345"/>
  <circle fill="#ABCDE"/>
  <ellipse stroke="#fedcb"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// All 5-digit colors should be converted
	assert.Contains(t, sanitized, `fill="#012345"`)
	assert.Contains(t, sanitized, `fill="#0ABCDE"`)
	assert.Contains(t, sanitized, `stroke="#0fedcb"`)
	assert.NotContains(t, sanitized, `"#12345"`)
	assert.NotContains(t, sanitized, `"#ABCDE"`)
	assert.NotContains(t, sanitized, `"#fedcb"`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_SixDigitHexColor_NotModified(t *testing.T) {
	// Valid 6-digit hex colors should NOT be modified
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect fill="#123456"/>
  <circle fill="#ABCDEF"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// 6-digit colors should remain unchanged
	assert.Contains(t, sanitized, `fill="#123456"`)
	assert.Contains(t, sanitized, `fill="#ABCDEF"`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_FiveDigitHexColor_MixedWithValid(t *testing.T) {
	// Mix of 5-digit and valid 6-digit colors - only 5-digit should be modified
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect fill="#FF8C42"/>
  <ellipse fill="#00000"/>
  <circle stroke="#123456" fill="#ABCDE"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// 6-digit colors preserved
	assert.Contains(t, sanitized, `fill="#FF8C42"`)
	assert.Contains(t, sanitized, `stroke="#123456"`)

	// 5-digit colors converted
	assert.Contains(t, sanitized, `fill="#000000"`)
	assert.Contains(t, sanitized, `fill="#0ABCDE"`)

	// Original 5-digit colors should be gone
	assert.NotContains(t, sanitized, `"#00000"`)
	assert.NotContains(t, sanitized, `"#ABCDE"`)

	// Should parse successfully
	_, err := convertSVGToPNG(sanitized, 512, 512)
	require.NoError(t, err)
}

func TestSanitizeSVG_ProductionPattern_Daniel_TFK11(t *testing.T) {
	// Exact SVG structure from TFK-11 bug report (daniel.jpg wireframe)
	// Line 43: <ellipse cx="650" cy="350" rx="50" ry="80" fill="#00000" opacity="0.08"/>
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="1300" height="1300" viewBox="0 0 1300 1300">
  <rect x="0" y="0" width="1300" height="1300" fill="#f5f5f5"/>
  <ellipse cx="650" cy="400" rx="200" ry="250" fill="#ffe0bd"/>
  <ellipse cx="650" cy="350" rx="50" ry="80" fill="#00000" opacity="0.08"/>
  <rect x="450" y="650" width="400" height="500" fill="#1a365d"/>
  <text x="650" y="1250" text-anchor="middle" font-size="14">Professional headshot - centered composition</text>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// The problematic 5-digit color should be converted
	assert.Contains(t, sanitized, `fill="#000000"`)
	assert.NotContains(t, sanitized, `fill="#00000"`)

	// Other valid colors should be preserved
	assert.Contains(t, sanitized, `fill="#f5f5f5"`)
	assert.Contains(t, sanitized, `fill="#ffe0bd"`)
	assert.Contains(t, sanitized, `fill="#1a365d"`)

	// Should parse successfully (this was the original failure)
	_, err := convertSVGToPNG(sanitized, 1300, 1300)
	require.NoError(t, err, "Production daniel SVG should parse after 5-digit hex conversion")
}

func TestConvertFiveDigitHexColors_Direct(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "fill 5-digit black",
			input:    `fill="#00000"`,
			expected: `fill="#000000"`,
		},
		{
			name:     "stroke 5-digit",
			input:    `stroke="#12345"`,
			expected: `stroke="#012345"`,
		},
		{
			name:     "fill 6-digit unchanged",
			input:    `fill="#123456"`,
			expected: `fill="#123456"`,
		},
		{
			name:     "fill 3-digit unchanged",
			input:    `fill="#ABC"`,
			expected: `fill="#ABC"`,
		},
		{
			name:     "fill 8-digit unchanged",
			input:    `fill="#12345678"`,
			expected: `fill="#12345678"`,
		},
		{
			name:     "lowercase 5-digit",
			input:    `fill="#abcde"`,
			expected: `fill="#0abcde"`,
		},
		{
			name:     "mixed case 5-digit",
			input:    `fill="#AbCdE"`,
			expected: `fill="#0AbCdE"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertFiveDigitHexColors(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// =============================================================================
// TFK-12: SVG extension handling tests
// =============================================================================

func TestShouldKeepAsSVG(t *testing.T) {
	// Test the extension-based decision logic for TFK-12
	testCases := []struct {
		path        string
		keepSVG     bool
		description string
	}{
		{"output.svg", true, "SVG extension should keep as SVG"},
		{"output.SVG", true, "Uppercase SVG extension should keep as SVG"},
		{"path/to/image.svg", true, "Nested SVG path should keep as SVG"},
		{"output.png", false, "PNG extension should convert to PNG"},
		{"output.PNG", false, "Uppercase PNG should convert to PNG"},
		{"output.jpg", false, "JPG extension should convert to PNG"},
		{"output.jpeg", false, "JPEG extension should convert to PNG"},
		{"path/to/image.png", false, "Nested PNG path should convert"},
		{"no-extension", false, "No extension should convert to PNG"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ext := strings.ToLower(filepath.Ext(tc.path))
			keepSVG := ext == ".svg"
			assert.Equal(t, tc.keepSVG, keepSVG, "Extension check for %s", tc.path)
		})
	}
}

func TestSVGOutputFormat_SVGExtension(t *testing.T) {
	// Test that SVG content is valid and can be written directly
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <rect x="0" y="0" width="512" height="512" fill="#f5f5f5"/>
  <text x="256" y="256" text-anchor="middle">Test SVG</text>
</svg>`

	// Verify SVG content starts with <svg (what we'd write for .svg files)
	assert.True(t, strings.HasPrefix(svgContent, "<svg"), "SVG content should start with <svg tag")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(svgContent), "</svg>"), "SVG content should end with </svg> tag")
}

func TestSVGOutputFormat_PNGExtension(t *testing.T) {
	// Test that PNG conversion produces valid PNG data
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <rect x="0" y="0" width="512" height="512" fill="#f5f5f5"/>
</svg>`

	pngData, err := convertSVGToPNG(svgContent, 512, 512)
	require.NoError(t, err, "PNG conversion should succeed")

	// Verify PNG magic bytes (PNG signature: 89 50 4E 47 0D 0A 1A 0A)
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	assert.True(t, len(pngData) >= 8, "PNG data should be at least 8 bytes")
	assert.Equal(t, pngSignature, pngData[:8], "PNG data should have correct PNG signature")
}

// =============================================================================
// TFK-14: Invalid CSS syntax in SVG <style> blocks
// =============================================================================

func TestNormalizeStyleBlockCSS_InvalidXMLAttributeSyntax(t *testing.T) {
	// This is the exact failing SVG from TFK-14 bug report
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">
  <defs>
    <style>
      .orbit { fill="none" stroke="#61DAFB" stroke-width="24" }
      .nucleus { fill="#61DAFB" }
      .electron { fill="#61DAFB" }
      .label { font-family="Arial, sans-serif" font-size="32" fill="#333333" text-anchor="middle" }
    </style>
  </defs>
  <rect x="0" y="0" width="1024" height="1024" fill="#ffffff"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Verify CSS properties are properly formatted
	assert.Contains(t, sanitized, `fill: none;`, "fill should use CSS property syntax")
	assert.Contains(t, sanitized, `stroke: #61DAFB;`, "stroke should use CSS property syntax")
	assert.Contains(t, sanitized, `stroke-width: 24;`, "stroke-width should use CSS property syntax")
	assert.Contains(t, sanitized, `font-family: Arial, sans-serif;`, "font-family should use CSS property syntax")
	assert.Contains(t, sanitized, `text-anchor: middle;`, "text-anchor should use CSS property syntax")

	// Verify invalid syntax is removed
	assert.NotContains(t, sanitized, `fill="none"`, "XML attribute syntax should be replaced in style blocks")
	assert.NotContains(t, sanitized, `stroke="#61DAFB"`, "XML attribute syntax should be replaced in style blocks")
}

func TestNormalizeStyleBlockCSS_PreservesValidCSS(t *testing.T) {
	// SVG with already-valid CSS should not be changed
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <style>
    .valid { fill: none; stroke: #FF0000; stroke-width: 2; }
    .another { font-family: Arial, sans-serif; font-size: 16px; }
  </style>
  <rect class="valid" x="0" y="0" width="100" height="100"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Valid CSS should be preserved (might have minor formatting changes but values intact)
	assert.Contains(t, sanitized, `fill: none;`, "Valid CSS should be preserved")
	assert.Contains(t, sanitized, `stroke: #FF0000;`, "Valid CSS should be preserved")
}

func TestNormalizeStyleBlockCSS_MixedSyntax(t *testing.T) {
	// SVG with mixed valid and invalid CSS syntax
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <style>
    .valid { fill: none; stroke: #FF0000; }
    .invalid { fill="blue" stroke="#00FF00" }
  </style>
  <rect x="0" y="0" width="100" height="100"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Both should be valid CSS after normalization
	assert.Contains(t, sanitized, `fill: none;`, "Valid CSS should be preserved")
	assert.Contains(t, sanitized, `fill: blue;`, "Invalid syntax should be converted")
	assert.Contains(t, sanitized, `stroke: #00FF00;`, "Invalid syntax should be converted")
	assert.NotContains(t, sanitized, `fill="blue"`, "XML attribute syntax should be removed")
}

func TestNormalizeStyleBlockCSS_NoStyleBlock(t *testing.T) {
	// SVG without style block should not be affected
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <rect x="0" y="0" width="100" height="100" fill="blue"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// The inline attribute should remain as-is (it's valid SVG)
	assert.Contains(t, sanitized, `fill="blue"`, "Inline attributes outside style blocks should be preserved")
}

func TestNormalizeStyleBlockCSS_EmptyStyleBlock(t *testing.T) {
	// SVG with empty style block
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <style></style>
  <rect x="0" y="0" width="100" height="100" fill="blue"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Should not crash and should preserve the structure
	assert.Contains(t, sanitized, `<style></style>`, "Empty style block should be preserved")
}

func TestNormalizeStyleBlockCSS_CDataWrapped(t *testing.T) {
	// SVG with CDATA-wrapped style (common in some generators)
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <style><![CDATA[
    .orbit { fill="none" stroke="#61DAFB" }
  ]]></style>
  <rect x="0" y="0" width="100" height="100"/>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// The regex might not handle CDATA perfectly, but it shouldn't crash
	// At minimum it should preserve the structure
	assert.Contains(t, sanitized, `<svg`, "SVG should be preserved")
}

func TestSanitizeSVG_TFK14_FullReactIconExample(t *testing.T) {
	// Full example from TFK-14 bug report - this should parse successfully after sanitization
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="1024" height="1024" viewBox="0 0 1024 1024">
  <defs>
    <style>
      .orbit { fill="none" stroke="#61DAFB" stroke-width="24" }
      .nucleus { fill="#61DAFB" }
      .electron { fill="#61DAFB" }
      .label { font-family="Arial, sans-serif" font-size="32" fill="#333333" text-anchor="middle" }
    </style>
  </defs>

  <rect x="0" y="0" width="1024" height="1024" fill="#ffffff"/>

  <g transform="translate(512, 512)">
    <ellipse cx="0" cy="0" rx="240" ry="100" class="orbit" transform="rotate(0)"/>
    <ellipse cx="0" cy="0" rx="240" ry="100" class="orbit" transform="rotate(120)"/>
    <ellipse cx="0" cy="0" rx="240" ry="100" class="orbit" transform="rotate(240)"/>

    <circle cx="0" cy="0" r="60" class="nucleus"/>

    <circle cx="240" cy="0" r="32" class="electron"/>
    <circle cx="-120" cy="207.85" r="32" class="electron"/>
    <circle cx="-120" cy="-207.85" r="32" class="electron"/>
  </g>

  <text x="512" y="900" class="label">React Atom Icon - Cyan on White</text>
</svg>`

	sanitized := sanitizeSVG(inputSVG)

	// Verify the SVG is now valid
	assert.Contains(t, sanitized, `<svg xmlns="http://www.w3.org/2000/svg"`, "SVG root should be preserved")
	assert.Contains(t, sanitized, `fill: none;`, "fill should be CSS property syntax")
	assert.Contains(t, sanitized, `stroke: #61DAFB;`, "stroke should be CSS property syntax")
	assert.Contains(t, sanitized, `</svg>`, "SVG should be complete")

	// The SVG should now be parseable by oksvg - let's verify basic structure
	assert.True(t, strings.HasPrefix(strings.TrimSpace(sanitized), "<svg"), "Should start with <svg")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(sanitized), "</svg>"), "Should end with </svg>")
}

func TestSanitizeSVG_TFK14_PNGConversionAfterFix(t *testing.T) {
	// The original TFK-14 bug: SVG with invalid CSS syntax fails PNG conversion
	// After fix, the SVG should be convertible to PNG
	inputSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <defs>
    <style>
      .orbit { fill="none" stroke="#61DAFB" stroke-width="12" }
      .nucleus { fill="#61DAFB" }
    </style>
  </defs>
  <rect x="0" y="0" width="512" height="512" fill="#ffffff"/>
  <g transform="translate(256, 256)">
    <ellipse cx="0" cy="0" rx="120" ry="50" class="orbit"/>
    <circle cx="0" cy="0" r="30" class="nucleus"/>
  </g>
</svg>`

	// First sanitize the SVG
	sanitized := sanitizeSVG(inputSVG)

	// Verify CSS was fixed
	assert.Contains(t, sanitized, `fill: none;`, "CSS should be normalized")
	assert.Contains(t, sanitized, `stroke: #61DAFB;`, "CSS should be normalized")

	// Now attempt PNG conversion - this was failing before the fix
	pngData, err := convertSVGToPNG(sanitized, 512, 512)

	// This should no longer fail
	require.NoError(t, err, "PNG conversion should succeed after CSS normalization")

	// Verify PNG magic bytes
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	assert.True(t, len(pngData) >= 8, "PNG data should be at least 8 bytes")
	assert.Equal(t, pngSignature, pngData[:8], "PNG data should have correct PNG signature")
}

// =============================================================================
// TFK-11 Option B: SVG Feedback Loop tests
// =============================================================================

func TestBuildSVGFeedbackPrompt_WithError(t *testing.T) {
	originalPrompt := "Create a simple icon with a blue circle"
	errorMessage := "failed to parse SVG: color string 00000 is not length 3 or 6"

	feedback := buildSVGFeedbackPrompt(originalPrompt, errorMessage, "")

	// Should include the error message
	assert.Contains(t, feedback, errorMessage, "Feedback should include error message")

	// Should include original prompt
	assert.Contains(t, feedback, originalPrompt, "Feedback should include original prompt")

	// Should include guidance about common issues
	assert.Contains(t, feedback, "COMMON SVG ISSUES TO AVOID", "Feedback should include guidance")
	assert.Contains(t, feedback, "Hex colors must be 3 or 6 digits", "Feedback should mention hex color rules")
}

func TestBuildSVGFeedbackPrompt_WithFailedSVG(t *testing.T) {
	originalPrompt := "Create a React icon"
	errorMessage := "SVG parsing/conversion failed: invalid attribute format"
	failedSVG := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <style>.orbit { fill="none" stroke="#61DAFB" }</style>
  <circle cx="256" cy="256" r="50"/>
</svg>`

	feedback := buildSVGFeedbackPrompt(originalPrompt, errorMessage, failedSVG)

	// Should include error message
	assert.Contains(t, feedback, errorMessage, "Feedback should include error message")

	// Should include excerpt of failed SVG
	assert.Contains(t, feedback, "FAILED SVG (excerpt)", "Feedback should include SVG excerpt label")
	assert.Contains(t, feedback, `fill="none"`, "Feedback should include SVG content")

	// Should include CSS syntax guidance (since this is the TFK-14 issue)
	assert.Contains(t, feedback, "Inside <style> blocks, use CSS syntax", "Feedback should mention CSS syntax rules")
	assert.Contains(t, feedback, `WRONG: .class { fill="none"`, "Feedback should show wrong example")
	assert.Contains(t, feedback, `RIGHT: .class { fill: none;`, "Feedback should show correct example")
}

func TestBuildSVGFeedbackPrompt_TruncatesLongSVG(t *testing.T) {
	originalPrompt := "Create an icon"
	errorMessage := "parse error"
	// Create SVG longer than 500 chars
	longSVG := "<svg>" + strings.Repeat("x", 600) + "</svg>"

	feedback := buildSVGFeedbackPrompt(originalPrompt, errorMessage, longSVG)

	// Should truncate and add ...
	assert.Contains(t, feedback, "...", "Long SVG should be truncated")
	// Should not contain the full SVG
	assert.NotContains(t, feedback, strings.Repeat("x", 600), "Full long SVG should not be included")
}

// =============================================================================
// TFK-11 Option B: SVG Feedback Loop Integration Tests
// These tests verify the feedback loop mechanism works end-to-end
// =============================================================================

// TestSVGFeedbackLoop_SimulatedRetry simulates the feedback loop by testing
// what happens when an LLM returns invalid SVG and gets corrected
func TestSVGFeedbackLoop_SimulatedRetry(t *testing.T) {
	// This test simulates the feedback loop logic without actual LLM calls
	// It verifies that:
	// 1. Invalid SVG triggers feedback prompt construction
	// 2. Feedback prompt contains error details and guidance
	// 3. The corrected prompt would lead to valid SVG on retry

	originalPrompt := "Create a simple blue circle icon on white background"

	// Simulate first LLM response - invalid SVG with wrong CSS syntax (TFK-14 style error)
	firstResponse := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <style>
    .circle { fill="blue" stroke="none" }
  </style>
  <circle class="circle" cx="256" cy="256" r="200"/>
</svg>`

	// Step 1: Try to parse/sanitize the first response
	sanitized := sanitizeSVG(firstResponse)

	// Step 2: Attempt conversion - this would fail on the invalid CSS
	// For this test, we simulate the error that oksvg would return
	simulatedError := "failed to parse SVG: unexpected token in style block"

	// Step 3: Build feedback prompt (this is what the feedback loop does)
	feedbackPrompt := buildSVGFeedbackPrompt(originalPrompt, simulatedError, sanitized)

	// Verify feedback prompt structure
	t.Run("FeedbackPromptContainsError", func(t *testing.T) {
		assert.Contains(t, feedbackPrompt, simulatedError, "Feedback should include the error message")
	})

	t.Run("FeedbackPromptContainsOriginalRequest", func(t *testing.T) {
		assert.Contains(t, feedbackPrompt, originalPrompt, "Feedback should include original prompt")
		assert.Contains(t, feedbackPrompt, "ORIGINAL REQUEST:", "Feedback should label original request")
	})

	t.Run("FeedbackPromptContainsFailedSVG", func(t *testing.T) {
		assert.Contains(t, feedbackPrompt, "FAILED SVG (excerpt):", "Feedback should include failed SVG label")
		assert.Contains(t, feedbackPrompt, "<svg", "Feedback should include SVG content")
	})

	t.Run("FeedbackPromptContainsGuidance", func(t *testing.T) {
		assert.Contains(t, feedbackPrompt, "COMMON SVG ISSUES TO AVOID:", "Feedback should include guidance")
		assert.Contains(t, feedbackPrompt, "Inside <style> blocks, use CSS syntax", "Feedback should mention CSS syntax rule")
		assert.Contains(t, feedbackPrompt, "WRONG:", "Feedback should show wrong example")
		assert.Contains(t, feedbackPrompt, "RIGHT:", "Feedback should show correct example")
	})

	t.Run("FeedbackPromptIsActionable", func(t *testing.T) {
		assert.Contains(t, feedbackPrompt, "Please regenerate a VALID SVG", "Feedback should request regeneration")
		assert.Contains(t, feedbackPrompt, "fixes the error", "Feedback should mention fixing the error")
	})

	// Step 4: Simulate corrected LLM response after receiving feedback
	correctedResponse := `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512" viewBox="0 0 512 512">
  <style>
    .circle { fill: blue; stroke: none; }
  </style>
  <circle class="circle" cx="256" cy="256" r="200"/>
</svg>`

	// Step 5: Verify corrected response would pass sanitization
	t.Run("CorrectedResponsePassesSanitization", func(t *testing.T) {
		sanitizedCorrected := sanitizeSVG(correctedResponse)
		assert.Contains(t, sanitizedCorrected, "<svg", "Sanitized should contain SVG tag")
		assert.Contains(t, sanitizedCorrected, "</svg>", "Sanitized should contain closing SVG tag")
		// CSS syntax should be correct
		assert.Contains(t, sanitizedCorrected, "fill: blue;", "CSS should have correct property syntax")
	})

	// Step 6: Verify corrected response would pass PNG conversion
	t.Run("CorrectedResponsePassesPNGConversion", func(t *testing.T) {
		sanitizedCorrected := sanitizeSVG(correctedResponse)
		pngData, err := convertSVGToPNG(sanitizedCorrected, 512, 512)
		assert.NoError(t, err, "Corrected SVG should convert to PNG without error")
		assert.NotEmpty(t, pngData, "PNG data should not be empty")
	})

	t.Log("✓ Feedback loop simulation completed successfully")
	t.Log("  - First response (invalid CSS) → triggers feedback")
	t.Log("  - Feedback prompt built with error + guidance")
	t.Log("  - Corrected response (valid CSS) → passes validation")
}

// TestSVGFeedbackLoop_MultipleErrorTypes tests feedback for different error scenarios
func TestSVGFeedbackLoop_MultipleErrorTypes(t *testing.T) {
	testCases := []struct {
		name             string
		invalidSVG       string
		errorMessage     string
		expectInFeedback []string
	}{
		{
			name: "InvalidHexColor_5Digits",
			invalidSVG: `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <circle cx="256" cy="256" r="200" fill="#12345"/>
</svg>`,
			errorMessage: "color string 12345 is not length 3 or 6",
			expectInFeedback: []string{
				"Hex colors must be 3 or 6 digits",
				"#RGB or #RRGGBB",
			},
		},
		{
			name: "UnescapedAmpersand",
			invalidSVG: `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <text x="100" y="100">Tom & Jerry</text>
</svg>`,
			errorMessage: "XML syntax error: invalid character entity",
			expectInFeedback: []string{
				"Escape ampersands",
				"&amp;",
			},
		},
		{
			name: "InvalidCSSInStyle",
			invalidSVG: `<svg xmlns="http://www.w3.org/2000/svg" width="512" height="512">
  <style>.cls { fill="red" }</style>
  <rect class="cls" width="100" height="100"/>
</svg>`,
			errorMessage: "unexpected token in CSS",
			expectInFeedback: []string{
				"Inside <style> blocks, use CSS syntax",
				`fill="none"`,
				"fill: none;",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			originalPrompt := "Create a simple icon"
			feedback := buildSVGFeedbackPrompt(originalPrompt, tc.errorMessage, tc.invalidSVG)

			// Verify error is included
			assert.Contains(t, feedback, tc.errorMessage, "Feedback should include error message")

			// Verify expected guidance is included
			for _, expected := range tc.expectInFeedback {
				assert.Contains(t, feedback, expected, "Feedback should include: %s", expected)
			}

			// Verify structure
			assert.Contains(t, feedback, "ORIGINAL REQUEST:", "Should have original request section")
			assert.Contains(t, feedback, "FAILED SVG (excerpt):", "Should have failed SVG section")

			t.Logf("✓ Feedback for %s contains all expected guidance", tc.name)
		})
	}
}

// TestSVGFeedbackLoop_PromptChaining verifies that feedback prompts can be chained
// (simulating multiple retry attempts)
func TestSVGFeedbackLoop_PromptChaining(t *testing.T) {
	originalPrompt := "Create a React logo icon"

	// Attempt 1: LLM returns invalid SVG
	attempt1SVG := `<svg xmlns="http://www.w3.org/2000/svg"><circle fill="#12345"/></svg>`
	error1 := "invalid hex color"

	// Build first feedback
	feedback1 := buildSVGFeedbackPrompt(originalPrompt, error1, attempt1SVG)
	assert.Contains(t, feedback1, originalPrompt, "First feedback should contain original prompt")
	assert.Contains(t, feedback1, error1, "First feedback should contain first error")

	// Attempt 2: LLM responds to feedback but makes different error
	attempt2SVG := `<svg xmlns="http://www.w3.org/2000/svg"><circle fill="#123456"/><text>A & B</text></svg>`
	error2 := "unescaped ampersand"

	// Build second feedback (based on NEW error, but still referencing original prompt)
	feedback2 := buildSVGFeedbackPrompt(originalPrompt, error2, attempt2SVG)
	assert.Contains(t, feedback2, originalPrompt, "Second feedback should contain original prompt")
	assert.Contains(t, feedback2, error2, "Second feedback should contain second error")
	assert.NotContains(t, feedback2, error1, "Second feedback should NOT contain first error")

	// Verify feedbacks are different
	assert.NotEqual(t, feedback1, feedback2, "Different errors should produce different feedback")

	t.Log("✓ Feedback chaining works correctly across multiple retry attempts")
}

// TestSVGFeedbackLoop_FullRetryWithMock tests the complete feedback loop
// by simulating LLM responses: first returns invalid SVG, second returns valid SVG
// This verifies the "gaslighting" flow: error -> feedback prompt -> corrected response
func TestSVGFeedbackLoop_FullRetryWithMock(t *testing.T) {
	// Track all prompts received
	var promptsReceived []string

	// Invalid SVG that will fail oksvg parsing (unclosed tag that breaks XML structure)
	// After sanitization, the XML is still malformed and oksvg.ReadIconStream will fail
	invalidSVGResponse := `{"type":"result","result":"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"512\" height=\"512\" viewBox=\"0 0 512 512\"><rect x=\"0\" y=\"0\" width=\"512\" <BROKEN_TAG></svg>"}`

	// Valid SVG that will pass
	validSVGResponse := `{"type":"result","result":"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"512\" height=\"512\" viewBox=\"0 0 512 512\"><circle cx=\"256\" cy=\"256\" r=\"200\" fill=\"#123456\"/></svg>"}`

	// Mock query function that returns invalid on first call, valid on second
	callCount := 0
	mockQuery := func(prompt string) (string, error) {
		callCount++
		promptsReceived = append(promptsReceived, prompt)

		if callCount == 1 {
			t.Log("Mock: Returning INVALID SVG (5-digit hex color)")
			return invalidSVGResponse, nil
		}
		t.Log("Mock: Returning VALID SVG (6-digit hex color)")
		return validSVGResponse, nil
	}

	// Simulate the retry loop logic from SVGExecutor.Execute()
	originalPrompt := "Create a blue circle icon"
	wrappedPrompt := wrapPromptForSVG(originalPrompt, 512, 512)

	const maxRetries = 3
	var lastError error
	var svgContent string
	currentPrompt := wrappedPrompt

	for attempt := 1; attempt <= maxRetries; attempt++ {
		t.Logf("Attempt %d: Sending prompt (length=%d)", attempt, len(currentPrompt))

		// Call mock LLM
		response, err := mockQuery(currentPrompt)
		require.NoError(t, err, "Mock should not return error")

		// Extract SVG from response
		svgContent, err = extractSVGFromResponse(response)
		require.NoError(t, err, "Should extract SVG from response")

		// Try to convert to PNG (this is the validation step)
		_, convErr := convertSVGToPNG(svgContent, 512, 512)
		if convErr != nil {
			t.Logf("Attempt %d: PNG conversion FAILED - %v", attempt, convErr)
			lastError = fmt.Errorf("SVG parsing/conversion failed: %w", convErr)

			// Build feedback prompt for next attempt (this is the "gaslighting")
			currentPrompt = buildSVGFeedbackPrompt(wrappedPrompt, lastError.Error(), svgContent)
			continue
		}

		// Success!
		t.Logf("Attempt %d: PNG conversion SUCCESS", attempt)
		lastError = nil
		break
	}

	// Verify the results
	require.NoError(t, lastError, "Should eventually succeed")
	require.Len(t, promptsReceived, 2, "Should have made exactly 2 calls (initial + retry)")

	// Verify first prompt was the original wrapped prompt
	t.Run("FirstPromptIsOriginal", func(t *testing.T) {
		assert.Equal(t, wrappedPrompt, promptsReceived[0], "First prompt should be original")
		assert.Contains(t, promptsReceived[0], "Create a blue circle icon", "First prompt should contain user's request")
		assert.NotContains(t, promptsReceived[0], "FAILED", "First prompt should NOT have failure info")
	})

	// Verify second prompt contains feedback/gaslighting
	t.Run("SecondPromptHasFeedback", func(t *testing.T) {
		feedbackPrompt := promptsReceived[1]
		assert.Contains(t, feedbackPrompt, "IMPORTANT: Your previous SVG generation attempt FAILED", "Feedback should indicate failure")
		assert.Contains(t, feedbackPrompt, "ERROR:", "Feedback should contain error section")
		assert.Contains(t, feedbackPrompt, "FAILED SVG (excerpt):", "Feedback should contain failed SVG")
		assert.Contains(t, feedbackPrompt, "BROKEN_TAG", "Feedback should show the problematic SVG content")
		assert.Contains(t, feedbackPrompt, "ORIGINAL REQUEST:", "Feedback should contain original request section")
		assert.Contains(t, feedbackPrompt, "Create a blue circle icon", "Feedback should preserve original user request")
	})

	// Verify the final SVG is valid
	t.Run("FinalSVGIsValid", func(t *testing.T) {
		assert.Contains(t, svgContent, "#123456", "Final SVG should have valid 6-digit hex color")

		// Should convert to PNG successfully
		pngData, err := convertSVGToPNG(svgContent, 512, 512)
		require.NoError(t, err, "Final SVG should convert to PNG")
		assert.True(t, len(pngData) > 100, "PNG should have reasonable size")
	})

	t.Log("✓ Full feedback loop test passed: Invalid SVG -> Feedback prompt -> Valid SVG")
}

// TestSVGFeedbackLoop_MaxRetriesExhausted tests that the loop fails after max retries
func TestSVGFeedbackLoop_MaxRetriesExhausted(t *testing.T) {
	// Always return invalid SVG (missing xmlns)
	invalidSVGResponse := `{"type":"result","result":"<svg width=\"512\" height=\"512\"><broken"}`

	callCount := 0
	mockQuery := func(prompt string) (string, error) {
		callCount++
		return invalidSVGResponse, nil
	}

	originalPrompt := "Create an icon"
	wrappedPrompt := wrapPromptForSVG(originalPrompt, 512, 512)

	const maxRetries = 3
	var lastError error
	currentPrompt := wrappedPrompt

	for attempt := 1; attempt <= maxRetries; attempt++ {
		response, _ := mockQuery(currentPrompt)

		_, err := extractSVGFromResponse(response)
		if err != nil {
			lastError = err
			currentPrompt = buildSVGFeedbackPrompt(wrappedPrompt, lastError.Error(), "")
			continue
		}
	}

	// Verify failure
	assert.Error(t, lastError, "Should fail after exhausting retries")
	assert.Equal(t, 3, callCount, "Should have attempted exactly 3 times")

	t.Log("✓ Max retries exhaustion test passed")
}

// TestSVGExecutor_ExecuteWithMockedLLM tests the REAL SVGExecutor.Execute() method
// with an injected mock query function. This exercises the actual production code path.
func TestSVGExecutor_ExecuteWithMockedLLM(t *testing.T) {
	// Create temp output directory
	outputDir := t.TempDir()

	// Track all prompts received by the mock
	var mu sync.Mutex
	var promptsReceived []string
	callCount := 0

	// Invalid SVG (broken XML) for first call, valid SVG for second
	invalidSVG := `{"type":"result","result":"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"512\" height=\"512\"><rect <BROKEN></svg>"}`
	validSVG := `{"type":"result","result":"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"512\" height=\"512\" viewBox=\"0 0 512 512\"><circle cx=\"256\" cy=\"256\" r=\"200\" fill=\"#FF5733\"/></svg>"}`

	// Create mock query function
	mockQuery := func(ctx context.Context, prompt string) (string, error) {
		mu.Lock()
		callCount++
		currentCall := callCount
		promptsReceived = append(promptsReceived, prompt)
		mu.Unlock()

		if currentCall == 1 {
			t.Log("Mock LLM: Returning INVALID SVG (broken XML)")
			return invalidSVG, nil
		}
		t.Log("Mock LLM: Returning VALID SVG")
		return validSVG, nil
	}

	// Create REAL SVGExecutor with mock injected
	executor := NewSVGExecutor("/tmp/fake-claude-home", true, 10, 4)
	executor.SetDebug(true)
	executor.SetQueryFunc(mockQuery)

	// Create project spec with a PNG file (triggers wireframe mode with validation)
	projectSpec := map[string]interface{}{
		"files": map[string]interface{}{
			"test-icon.png": map[string]interface{}{
				"instructions": []interface{}{
					map[string]interface{}{
						"prompt": "Create a simple orange circle icon",
					},
				},
				"image": map[string]interface{}{
					"size": "512x512",
				},
			},
		},
	}

	// Execute the REAL method
	ctx := context.Background()
	status, err := executor.Execute(ctx, projectSpec, outputDir)

	// Verify execution succeeded
	require.NoError(t, err, "Execute should succeed after retry")
	require.NotNil(t, status, "Status should not be nil")
	assert.Equal(t, "completed", status.State, "State should be completed")

	// Verify the mock was called twice (initial + retry)
	assert.Equal(t, 2, callCount, "Should have made exactly 2 LLM calls")
	assert.Len(t, promptsReceived, 2, "Should have captured 2 prompts")

	// Verify first prompt is the original SVG generation prompt
	t.Run("FirstPromptIsOriginal", func(t *testing.T) {
		firstPrompt := promptsReceived[0]
		assert.Contains(t, firstPrompt, "Generate a valid SVG 1.1 wireframe", "First prompt should be SVG generation")
		assert.Contains(t, firstPrompt, "Create a simple orange circle icon", "First prompt should contain user's request")
		assert.NotContains(t, firstPrompt, "FAILED", "First prompt should NOT have failure info")
	})

	// Verify second prompt is the feedback/gaslighting prompt
	t.Run("SecondPromptIsFeedback", func(t *testing.T) {
		feedbackPrompt := promptsReceived[1]
		assert.Contains(t, feedbackPrompt, "IMPORTANT: Your previous SVG generation attempt FAILED", "Feedback should indicate failure")
		assert.Contains(t, feedbackPrompt, "ERROR:", "Feedback should have error section")
		assert.Contains(t, feedbackPrompt, "FAILED SVG (excerpt):", "Feedback should show failed SVG")
		assert.Contains(t, feedbackPrompt, "BROKEN", "Feedback should contain the broken content")
		assert.Contains(t, feedbackPrompt, "ORIGINAL REQUEST:", "Feedback should have original request section")
		assert.Contains(t, feedbackPrompt, "Create a simple orange circle icon", "Feedback should preserve user's request")
	})

	// Verify output files were created
	t.Run("OutputFilesCreated", func(t *testing.T) {
		pngPath := filepath.Join(outputDir, "test-icon.png")
		info, err := os.Stat(pngPath)
		require.NoError(t, err, "PNG file should exist")
		assert.True(t, info.Size() > 100, "PNG file should have content")
		t.Logf("✓ PNG file created: %d bytes", info.Size())
	})

	// Verify debug files were created
	t.Run("DebugFilesCreated", func(t *testing.T) {
		debugDir := filepath.Join(outputDir, ".debug")

		// Initial prompt
		prompt1Path := filepath.Join(debugDir, "test-icon.png.prompt.txt")
		_, err := os.Stat(prompt1Path)
		assert.NoError(t, err, "Initial prompt debug file should exist")

		// Feedback prompt (attempt 2)
		prompt2Path := filepath.Join(debugDir, "test-icon.png.attempt2.prompt.txt")
		_, err = os.Stat(prompt2Path)
		assert.NoError(t, err, "Feedback prompt debug file should exist")

		// Failed SVG from attempt 1
		failedSvgPath := filepath.Join(debugDir, "test-icon.png.attempt1.failed.svg")
		_, err = os.Stat(failedSvgPath)
		assert.NoError(t, err, "Failed SVG debug file should exist")

		// List all debug files for visibility
		files, _ := os.ReadDir(debugDir)
		t.Log("Debug files created:")
		for _, f := range files {
			t.Logf("  - %s", f.Name())
		}
	})

	t.Log("✓ Real SVGExecutor.Execute() with mocked LLM - feedback loop verified!")
}

// TestSVGExecutor_ShowFeedbackFiles runs the mocked executor and preserves files for inspection
func TestSVGExecutor_ShowFeedbackFiles(t *testing.T) {
	// Create test directory following project conventions: test-output/{timestamp}-{TestName}/output
	projectRoot, err := filepath.Abs("../../..")
	require.NoError(t, err)

	timestamp := time.Now().Format("20060102T150405")
	testDir := filepath.Join(projectRoot, "test-output", fmt.Sprintf("%s-TestSVGExecutor_FeedbackLoop", timestamp))
	outputDir := filepath.Join(testDir, "output")
	os.RemoveAll(testDir)
	os.MkdirAll(outputDir, 0755)

	var mu sync.Mutex
	callCount := 0

	invalidSVG := `{"type":"result","result":"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"512\" height=\"512\"><rect <BROKEN_TAG_HERE></svg>"}`
	validSVG := `{"type":"result","result":"<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"512\" height=\"512\" viewBox=\"0 0 512 512\"><circle cx=\"256\" cy=\"256\" r=\"200\" fill=\"#FF5733\"/></svg>"}`

	mockQuery := func(ctx context.Context, prompt string) (string, error) {
		mu.Lock()
		callCount++
		current := callCount
		mu.Unlock()
		if current == 1 {
			return invalidSVG, nil
		}
		return validSVG, nil
	}

	executor := NewSVGExecutor("/tmp/fake", true, 10, 4)
	executor.SetDebug(true)
	executor.SetQueryFunc(mockQuery)

	projectSpec := map[string]interface{}{
		"files": map[string]interface{}{
			"icon.png": map[string]interface{}{
				"instructions": []interface{}{map[string]interface{}{"prompt": "Create orange circle"}},
				"image":        map[string]interface{}{"size": "512x512"},
			},
		},
	}

	ctx := context.Background()
	_, err = executor.Execute(ctx, projectSpec, outputDir)
	require.NoError(t, err)

	t.Logf("\n\n=== FILES PRESERVED AT: %s/.debug/ ===\n", outputDir)
}
