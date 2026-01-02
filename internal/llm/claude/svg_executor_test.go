package claude

import (
	"testing"

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
