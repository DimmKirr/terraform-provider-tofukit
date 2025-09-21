package test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// generateTestTimestamp creates a Unix timestamp for test directories
func generateTestTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}

// createTestDirectory creates a unique test directory in test-output
func createTestDirectory(t *testing.T, testName string) string {
	projectRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Failed to get project root: %v", err)
	}

	timestamp := generateTestTimestamp()
	testDir := filepath.Join(projectRoot, "test-output", fmt.Sprintf("%s-%s", testName, timestamp))

	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory %s: %v", testDir, err)
	}

	t.Logf("Created test directory: %s", testDir)

	// Register cleanup - disabled by default to keep test output
	t.Cleanup(func() {
		if os.Getenv("CLEANUP_TEST_OUTPUT") == "true" {
			os.RemoveAll(testDir)
			t.Logf("Cleaned up test directory: %s", testDir)
		} else {
			t.Logf("Keeping test directory: %s (set CLEANUP_TEST_OUTPUT=true to remove)", testDir)
		}
	})

	return testDir
}

// verifyFileContent checks if a file exists and contains expected content
func verifyFileContent(t *testing.T, path string, expectedContent string) {
	content, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("Failed to read %s: %v", path, err)
		return
	}

	// Normalize content by trimming trailing whitespace for comparison
	// This handles inconsistencies in trailing newlines
	actualContent := strings.TrimRight(string(content), "\n\r\t ")
	expectedNormalized := strings.TrimRight(expectedContent, "\n\r\t ")

	if actualContent != expectedNormalized {
		t.Errorf("Content mismatch in %s\nExpected: %q\nActual: %q",
			path, expectedContent, string(content))
	} else {
		t.Logf("✓ %s exists with correct content", filepath.Base(path))
	}
}

// verifyFileExists checks if a file exists
func verifyFileExists(t *testing.T, path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		t.Errorf("Error checking file %s: %v", path, err)
		return false
	}
	return true
}

// verifyFileDoesNotExist checks that a file does not exist
func verifyFileDoesNotExist(t *testing.T, path string) {
	if _, err := os.Stat(path); err == nil {
		t.Errorf("File %s should not exist but does", path)
	} else if !os.IsNotExist(err) {
		t.Errorf("Error checking file %s: %v", path, err)
	}
}