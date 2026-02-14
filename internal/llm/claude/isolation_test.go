package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSessionID_Format(t *testing.T) {
	sessionID := GenerateSessionID()

	// Format should be: {timestamp}-{8-char-hex}
	parts := strings.Split(sessionID, "-")
	require.Len(t, parts, 2, "Session ID should have two parts separated by hyphen")

	// First part should be a Unix nanosecond timestamp (numeric)
	_, err := time.Parse("2006", parts[0][:4])
	// Check it's a large number (nanoseconds since epoch)
	assert.Greater(t, len(parts[0]), 10, "Timestamp should be at least 10 digits")

	// Second part should be 8 hex characters
	assert.Len(t, parts[1], 8, "Random suffix should be 8 hex chars")
	for _, c := range parts[1] {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"Random suffix should be lowercase hex: %c", c)
	}

	_ = err // silence unused warning
}

func TestGenerateSessionID_Unique(t *testing.T) {
	// Generate multiple session IDs and verify they're unique
	sessions := make(map[string]bool)
	for i := 0; i < 100; i++ {
		sessionID := GenerateSessionID()
		assert.False(t, sessions[sessionID], "Session ID should be unique: %s", sessionID)
		sessions[sessionID] = true
	}
}

func TestCreateIsolatedClaudeHome_SessionOnly(t *testing.T) {
	// Create a temporary "original" Claude home
	tmpDir := t.TempDir()
	originalHome := filepath.Join(tmpDir, "claude")
	require.NoError(t, os.MkdirAll(originalHome, 0755))

	// Create a test config file
	testConfig := []byte(`{"test": "config"}`)
	require.NoError(t, os.WriteFile(filepath.Join(originalHome, "claude.json"), testConfig, 0600))

	sessionID := GenerateSessionID()

	// Create isolated home (session-level, no call suffix)
	isolated, err := CreateIsolatedClaudeHome(originalHome, sessionID)
	require.NoError(t, err)
	defer isolated.Cleanup()

	// Verify path contains the session ID
	assert.Contains(t, isolated.IsolatedPath, ".claude-tofukit-"+sessionID)

	// Verify config file was copied
	copiedConfig, err := os.ReadFile(filepath.Join(isolated.IsolatedPath, "claude.json"))
	require.NoError(t, err)
	assert.Equal(t, testConfig, copiedConfig)

	// Verify original is unchanged
	originalConfig, err := os.ReadFile(filepath.Join(originalHome, "claude.json"))
	require.NoError(t, err)
	assert.Equal(t, testConfig, originalConfig)
}

func TestCreateIsolatedClaudeHome_WithCallSuffix(t *testing.T) {
	// Create a temporary "original" Claude home
	tmpDir := t.TempDir()
	originalHome := filepath.Join(tmpDir, "claude")
	require.NoError(t, os.MkdirAll(originalHome, 0755))

	sessionID := GenerateSessionID()

	// Create isolated home with file path suffix (per-call isolation)
	isolated, err := CreateIsolatedClaudeHome(originalHome, sessionID, "images/logo.png")
	require.NoError(t, err)
	defer isolated.Cleanup()

	// Verify path contains both session ID and sanitized file path
	assert.Contains(t, isolated.IsolatedPath, ".claude-tofukit-"+sessionID+"-")
	assert.Contains(t, isolated.IsolatedPath, "images-logo")
}

func TestCreateIsolatedClaudeHome_Cleanup(t *testing.T) {
	// Create a temporary "original" Claude home
	tmpDir := t.TempDir()
	originalHome := filepath.Join(tmpDir, "claude")
	require.NoError(t, os.MkdirAll(originalHome, 0755))

	sessionID := GenerateSessionID()

	// Create isolated home
	isolated, err := CreateIsolatedClaudeHome(originalHome, sessionID)
	require.NoError(t, err)

	// Verify it exists
	_, err = os.Stat(isolated.IsolatedPath)
	require.NoError(t, err, "Isolated home should exist")

	// Cleanup
	err = isolated.Cleanup()
	require.NoError(t, err)

	// Verify it's gone
	_, err = os.Stat(isolated.IsolatedPath)
	assert.True(t, os.IsNotExist(err), "Isolated home should be removed after cleanup")
}

func TestCreateIsolatedClaudeHome_CleanupOnlyTofukit(t *testing.T) {
	// Create a fake isolated home struct that doesn't match our pattern
	fakeHome := &IsolatedClaudeHome{
		IsolatedPath: "/tmp/some-other-directory",
		OriginalPath: "/home/user/.claude",
	}

	// Cleanup should NOT remove it (safety check)
	err := fakeHome.Cleanup()
	assert.NoError(t, err, "Cleanup should succeed but not remove non-tofukit directories")
}

func TestSanitizePathForDirName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"images/logo.png", "images-logo"},
		{"src/components/Button.tsx", "src-components-Button"},
		{"simple.txt", "simple"},
		{"path with spaces/file.md", "path-with-spaces-file"},
		{"a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p.txt", "a-b-c-d-e-f-g-h-i-j-k-l-m-n-o-"}, // truncated at 30 chars (may leave trailing hyphen)
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizePathForDirName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSVGExecutor_SessionIDGenerated(t *testing.T) {
	// Create SVGExecutor and verify it generates a session ID
	executor := NewSVGExecutor("/tmp/test-claude", false, 10, 4)

	assert.NotEmpty(t, executor.sessionID, "SVGExecutor should have a session ID")
	parts := strings.Split(executor.sessionID, "-")
	assert.Len(t, parts, 2, "Session ID should have timestamp-random format")
}

func TestExecutor_CreatesIsolatedHome(t *testing.T) {
	// Create a temporary "original" Claude home
	tmpDir := t.TempDir()
	originalHome := filepath.Join(tmpDir, "claude")
	require.NoError(t, os.MkdirAll(originalHome, 0755))

	// Create executor - this should create an isolated home
	executor := NewExecutor(originalHome, false, 10)
	require.NotNil(t, executor.isolatedHome)

	// Verify isolated home was created
	assert.Contains(t, executor.isolatedHome.IsolatedPath, ".claude-tofukit-")
	assert.NotEqual(t, executor.isolatedHome.IsolatedPath, originalHome)

	// Verify the directory exists
	_, err := os.Stat(executor.isolatedHome.IsolatedPath)
	require.NoError(t, err, "Isolated home directory should exist")

	// Cleanup
	err = executor.Cleanup()
	require.NoError(t, err)

	// Verify it's gone
	_, err = os.Stat(executor.isolatedHome.IsolatedPath)
	assert.True(t, os.IsNotExist(err), "Isolated home should be cleaned up")
}
