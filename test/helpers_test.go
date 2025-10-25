package test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
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

// setupIsolatedClaudeHome creates an isolated Claude home directory for testing
// It copies the necessary configuration files from the user's Claude home
func setupIsolatedClaudeHome(t *testing.T, testDir string) string {
	// Create isolated Claude home directory
	isolatedClaudeHome := filepath.Join(testDir, ".test-claude-home")
	if err := os.MkdirAll(isolatedClaudeHome, 0755); err != nil {
		t.Fatalf("Failed to create isolated Claude home: %v", err)
	}

	// Get user's Claude home directory
	userHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}
	userClaudeHome := filepath.Join(userHome, ".claude")

	// Copy .claude.json if it exists
	userClaudeJSON := filepath.Join(userHome, ".claude.json")
	if _, err := os.Stat(userClaudeJSON); err == nil {
		testClaudeJSON := filepath.Join(testDir, ".claude.json")
		if err := copyFile(userClaudeJSON, testClaudeJSON); err != nil {
			t.Logf("Warning: Failed to copy .claude.json: %v", err)
		} else {
			t.Logf("Copied .claude.json to test directory")
		}
	}

	// Copy essential Claude config directories and files
	essentialPaths := []string{
		".credentials.json",
		"settings.json",
	}

	for _, path := range essentialPaths {
		src := filepath.Join(userClaudeHome, path)
		dst := filepath.Join(isolatedClaudeHome, path)

		if info, err := os.Stat(src); err == nil {
			if info.IsDir() {
				if err := copyDir(src, dst); err != nil {
					t.Logf("Warning: Failed to copy directory %s: %v", path, err)
				} else {
					t.Logf("Copied directory %s to test Claude home", path)
				}
			} else {
				if err := copyFile(src, dst); err != nil {
					t.Logf("Warning: Failed to copy file %s: %v", path, err)
				} else {
					t.Logf("Copied file %s to test Claude home", path)
				}
			}
		}
	}

	t.Logf("Set up isolated Claude home at: %s", isolatedClaudeHome)
	return isolatedClaudeHome
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	// Preserve file permissions
	if info, err := os.Stat(src); err == nil {
		os.Chmod(dst, info.Mode())
	}

	_, err = io.Copy(destination, source)
	return err
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	// Get properties of source dir
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// removeQuarantineAttributes removes macOS quarantine attributes from provider binaries
// This is needed on macOS when OpenTofu copies binaries from filesystem_mirror
func removeQuarantineAttributes(t *testing.T, testDir string) {
	// Only run on macOS
	if fileExists("/usr/bin/xattr") {
		providerDir := filepath.Join(testDir, ".terraform", "providers")
		t.Logf("Looking for provider binaries in: %s", providerDir)

		if _, err := os.Stat(providerDir); err == nil {
			// Find all provider binaries and remove quarantine
			count := 0
			allFiles := 0
			filepath.Walk(providerDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					t.Logf("Walk error at %s: %v", path, err)
					return nil
				}

				// Skip directories
				if info.IsDir() {
					return nil
				}

				allFiles++

				// If it's a symlink, resolve it and check the target
				realPath := path
				if info.Mode()&os.ModeSymlink != 0 {
					t.Logf("Found symlink: %s", path)
					target, err := filepath.EvalSymlinks(path)
					if err != nil {
						t.Logf("Failed to resolve symlink %s: %v", path, err)
						return nil
					}
					t.Logf("Symlink resolves to: %s", target)
					realPath = target

					// Get info about the target
					targetInfo, err := os.Stat(target)
					if err != nil {
						t.Logf("Failed to stat symlink target %s: %v", target, err)
						return nil
					}
					info = targetInfo
				}

				t.Logf("Found file: %s (mode: %v, perm: %04o, executable: %v)",
					realPath, info.Mode(), info.Mode().Perm(), (info.Mode().Perm()&0111) != 0)

				// If the symlink target is a directory, walk it to find binaries
				if info.IsDir() {
					t.Logf("Symlink target is a directory, walking it: %s", realPath)
					filepath.Walk(realPath, func(binPath string, binInfo os.FileInfo, binErr error) error {
						if binErr != nil {
							return nil
						}
						if !binInfo.IsDir() && binInfo.Mode().IsRegular() && (binInfo.Mode().Perm()&0111) != 0 {
							t.Logf("Found executable in directory: %s", binPath)

							// Remove quarantine attribute
							cmd := exec.Command("xattr", "-d", "com.apple.quarantine", binPath)
							if err := cmd.Run(); err != nil {
								t.Logf("xattr returned: %v (ok if attribute doesn't exist)", err)
							}

							// Ad-hoc sign the binary
							cmd = exec.Command("codesign", "-s", "-", "-f", binPath)
							if output, err := cmd.CombinedOutput(); err != nil {
								t.Logf("codesign failed: %v, output: %s", err, string(output))
							} else {
								t.Logf("Successfully signed: %s", binPath)
							}
							count++
						}
						return nil
					})
				} else if info.Mode().IsRegular() && (info.Mode().Perm()&0111) != 0 {
					// This is an executable file, remove quarantine and codesign
					t.Logf("Processing executable binary: %s", realPath)

					// Remove quarantine attribute
					cmd := exec.Command("xattr", "-d", "com.apple.quarantine", realPath)
					if err := cmd.Run(); err != nil {
						t.Logf("xattr command returned: %v (this is ok if attribute doesn't exist)", err)
					}

					// Ad-hoc sign the binary
					cmd = exec.Command("codesign", "-s", "-", "-f", realPath)
					if output, err := cmd.CombinedOutput(); err != nil {
						t.Logf("codesign failed: %v, output: %s", err, string(output))
					} else {
						t.Logf("Successfully signed: %s", realPath)
					}

					count++
				}
				return nil
			})
			t.Logf("Found %d total files, processed %d executable binaries", allFiles, count)
		} else {
			t.Logf("Provider directory does not exist: %s", providerDir)
		}
	}
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Helper function to get minimum of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
