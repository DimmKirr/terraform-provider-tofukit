package test

import (
	"context"
	"testing"

	"github.com/tofukit/opentofu-provider-tofukit/internal/resources"
	"github.com/stretchr/testify/require"
)

// TestCollectKitVerifications validates that kit verifications are properly extracted
func TestCollectKitVerifications(t *testing.T) {
	ctx := context.Background()

	// Create a ProjectResourceFinal instance (needed for method receiver)
	r := &resources.ProjectResourceFinal{}

	// Mock kit spec with verifications
	kitsData := map[string]interface{}{
		"kit-id-1": map[string]interface{}{
			"name":    "test-tool",
			"version": "1.0.0",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Install Test Tool",
					"verifications": []interface{}{
						map[string]interface{}{
							"command": "test-tool --version && echo 'OK'",
							"expect":  "OK",
						},
					},
				},
			},
		},
		"kit-id-2": map[string]interface{}{
			"name":    "python",
			"version": "3.11",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Python Installation",
					"verifications": []interface{}{
						map[string]interface{}{
							"command": "python3 --version && echo 'OK'",
							"expect":  "OK",
						},
						map[string]interface{}{
							"command": "which python3 && echo 'found'",
							"expect":  "found",
						},
					},
				},
				map[string]interface{}{
					"name": "Pip Installation",
					"verifications": []interface{}{
						map[string]interface{}{
							"command": "pip3 --version",
							"expect":  "",
						},
					},
				},
			},
		},
	}

	// Call collection function
	verifications := r.CollectKitVerifications(ctx, kitsData)

	// Assert we got all verifications
	require.Len(t, verifications, 4, "Should collect all 4 verifications from 2 kits")

	// Verify pseudo-path format
	require.Contains(t, verifications[0].Path, "kit:", "Path should start with 'kit:'")
	require.Contains(t, verifications[0].Path, "test-tool", "Path should contain kit name")
	require.Contains(t, verifications[0].Path, "Install Test Tool", "Path should contain requirement name")

	// Verify verification content is preserved
	foundTestTool := false
	foundPython1 := false
	foundPython2 := false
	foundPip := false

	for _, v := range verifications {
		if len(v.Verifications) > 0 {
			cmd := v.Verifications[0].Command.ValueString()
			expect := v.Verifications[0].Expect.ValueString()

			if cmd == "test-tool --version && echo 'OK'" && expect == "OK" {
				foundTestTool = true
			}
			if cmd == "python3 --version && echo 'OK'" && expect == "OK" {
				foundPython1 = true
			}
			if cmd == "which python3 && echo 'found'" && expect == "found" {
				foundPython2 = true
			}
			if cmd == "pip3 --version" {
				foundPip = true
			}
		}
	}

	require.True(t, foundTestTool, "Should find test-tool verification")
	require.True(t, foundPython1, "Should find python version verification")
	require.True(t, foundPython2, "Should find python which verification")
	require.True(t, foundPip, "Should find pip verification")
}

// TestCollectKitVerificationsEmpty validates handling of empty kits
func TestCollectKitVerificationsEmpty(t *testing.T) {
	ctx := context.Background()
	r := &resources.ProjectResourceFinal{}

	// Test with nil kits
	verifications := r.CollectKitVerifications(ctx, nil)
	require.Empty(t, verifications, "Should return empty for nil kits")

	// Test with empty kits map
	verifications = r.CollectKitVerifications(ctx, map[string]interface{}{})
	require.Empty(t, verifications, "Should return empty for empty kits map")

	// Test with kit that has no requirements
	kitsData := map[string]interface{}{
		"kit-id-1": map[string]interface{}{
			"name":    "test-tool",
			"version": "1.0.0",
		},
	}
	verifications = r.CollectKitVerifications(ctx, kitsData)
	require.Empty(t, verifications, "Should return empty for kit with no requirements")

	// Test with kit that has requirements but no verifications
	kitsData = map[string]interface{}{
		"kit-id-1": map[string]interface{}{
			"name": "test-tool",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Install Test Tool",
					"instructions": []interface{}{
						map[string]interface{}{
							"prompt": "Install the tool",
						},
					},
				},
			},
		},
	}
	verifications = r.CollectKitVerifications(ctx, kitsData)
	require.Empty(t, verifications, "Should return empty for kit with no verifications")
}

// TestCollectKitVerificationsPseudoPathFormat validates the pseudo-path format
func TestCollectKitVerificationsPseudoPathFormat(t *testing.T) {
	ctx := context.Background()
	r := &resources.ProjectResourceFinal{}

	kitsData := map[string]interface{}{
		"kit-id-1": map[string]interface{}{
			"name": "my-tool",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Tool Setup",
					"verifications": []interface{}{
						map[string]interface{}{
							"command": "my-tool --version",
							"expect":  "v1.0",
						},
						map[string]interface{}{
							"command": "my-tool check",
							"expect":  "OK",
						},
					},
				},
			},
		},
	}

	verifications := r.CollectKitVerifications(ctx, kitsData)
	require.Len(t, verifications, 2, "Should collect 2 verifications")

	// Check pseudo-path format: "kit:{kitName}:{reqName}:{idx}"
	require.Equal(t, "kit:my-tool:Tool Setup:0", verifications[0].Path, "First verification should have index 0")
	require.Equal(t, "kit:my-tool:Tool Setup:1", verifications[1].Path, "Second verification should have index 1")
}
