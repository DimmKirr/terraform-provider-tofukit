package claude

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// IsolatedClaudeHome manages a session-isolated copy of the Claude home directory
// This prevents race conditions when multiple parallel Claude CLI processes
// try to read/write the same config file simultaneously
type IsolatedClaudeHome struct {
	IsolatedPath string // Path to the isolated copy (e.g., ~/.claude-tofukit-123456)
	OriginalPath string // Path to the original Claude home (e.g., ~/.claude)
	SessionID    string // Session ID used for this isolation
}

// GenerateSessionID creates a unique session ID for isolation
// Format: {timestamp}-{random} (e.g., "1736031289123456789-a1b2c3d4")
func GenerateSessionID() string {
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), hex.EncodeToString(randomBytes))
}

// CreateIsolatedClaudeHome creates a session-specific copy of the Claude home directory
// The original Claude home is NEVER modified - only copied from
//
// Parameters:
//   - originalHome: Path to original Claude home (e.g., ~/.claude)
//   - sessionID: Session identifier (use GenerateSessionID())
//   - callSuffix: Optional suffix for per-call isolation (e.g., file path)
//
// Directory naming:
//   - Session only: ~/.claude-tofukit-{sessionID}
//   - With call suffix: ~/.claude-tofukit-{sessionID}-{callSuffix}
func CreateIsolatedClaudeHome(originalHome string, sessionID string, callSuffix ...string) (*IsolatedClaudeHome, error) {
	// Expand ~ if present
	if strings.HasPrefix(originalHome, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		originalHome = filepath.Join(home, originalHome[2:])
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	// Build directory name: session ID + optional call suffix
	dirName := sessionID
	if len(callSuffix) > 0 && callSuffix[0] != "" {
		sanitized := sanitizePathForDirName(callSuffix[0])
		if sanitized != "" {
			dirName = fmt.Sprintf("%s-%s", sessionID, sanitized)
		}
	}

	isolatedPath := filepath.Join(homeDir, fmt.Sprintf(".claude-tofukit-%s", dirName))

	// Create the isolated directory
	if err := os.MkdirAll(isolatedPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create isolated Claude home: %w", err)
	}

	// Copy essential config files from original Claude home
	// These are read-only copies - the original is never modified
	configFiles := []string{
		"claude.json",       // Main config
		"settings.json",     // User settings
		"credentials.json",  // API credentials
		".credentials.json", // Alternative credentials location
	}

	for _, configFile := range configFiles {
		srcPath := filepath.Join(originalHome, configFile)
		dstPath := filepath.Join(isolatedPath, configFile)

		// Check if source file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue // Skip non-existent files
		}

		// Read source file (READ-ONLY from original)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			log.Printf("[WARN] Failed to read %s for isolation copy: %v", srcPath, err)
			continue
		}

		// Write to isolated directory
		if err := os.WriteFile(dstPath, data, 0600); err != nil {
			log.Printf("[WARN] Failed to write %s to isolated home: %v", dstPath, err)
			continue
		}
	}

	log.Printf("[INFO] Created isolated Claude home: %s (copied from %s)", isolatedPath, originalHome)

	return &IsolatedClaudeHome{
		IsolatedPath: isolatedPath,
		OriginalPath: originalHome,
	}, nil
}

// Cleanup removes the isolated Claude home directory
// Safe to call multiple times - will only cleanup valid isolated directories
func (h *IsolatedClaudeHome) Cleanup() error {
	if h == nil || h.IsolatedPath == "" {
		return nil
	}

	// Safety check: only remove directories that match our naming pattern
	if !strings.Contains(h.IsolatedPath, ".claude-tofukit-") {
		log.Printf("[WARN] Refusing to cleanup non-tofukit directory: %s", h.IsolatedPath)
		return nil
	}

	// Don't cleanup if it's the same as original (fallback case)
	if h.IsolatedPath == h.OriginalPath {
		return nil
	}

	log.Printf("[INFO] Cleaning up isolated Claude home: %s", h.IsolatedPath)
	return os.RemoveAll(h.IsolatedPath)
}

// Path returns the path to use for Claude home
// Returns the isolated path if available, otherwise the original
func (h *IsolatedClaudeHome) Path() string {
	if h == nil || h.IsolatedPath == "" {
		return h.OriginalPath
	}
	return h.IsolatedPath
}

// sanitizePathForDirName converts a file path to a safe directory name suffix
// e.g., "images/logo.png" -> "images-logo"
func sanitizePathForDirName(path string) string {
	// Remove extension
	path = strings.TrimSuffix(path, filepath.Ext(path))

	// Replace path separators and special chars with hyphens
	re := regexp.MustCompile(`[/\\:*?"<>|.\s]+`)
	path = re.ReplaceAllString(path, "-")

	// Remove leading/trailing hyphens
	path = strings.Trim(path, "-")

	// Limit length to avoid filesystem issues
	if len(path) > 30 {
		path = path[:30]
	}

	return path
}
