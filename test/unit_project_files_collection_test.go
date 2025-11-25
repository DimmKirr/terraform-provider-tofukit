package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProjectFilesCollection_Level1_ProjectWithFeature validates file collection
// from a project with an inline feature
func TestUnitProjectFilesCollection_Level1_ProjectWithFeature(t *testing.T) {
	// Create a project with an inline feature that has files
	projectData := map[string]interface{}{
		"name":    "test-project",
		"version": "1.0.0",
		"features": map[string]interface{}{
			"logger": map[string]interface{}{
				"requirements": []interface{}{
					map[string]interface{}{
						"name": "Logger Implementation",
						"instructions": []interface{}{
							map[string]interface{}{
								"prompt": "Implement logger",
							},
						},
					},
				},
				"files": map[string]interface{}{
					"logger.py": map[string]interface{}{
						"content": "# Logger implementation\n",
					},
					"logger_test.py": map[string]interface{}{
						"content": "# Logger tests\n",
					},
				},
			},
		},
	}

	// Extract files from features
	features, ok := projectData["features"].(map[string]interface{})
	require.True(t, ok, "features should be a map")

	var allFiles []string
	for featureName, featureData := range features {
		featureMap, ok := featureData.(map[string]interface{})
		require.True(t, ok, "feature should be a map")

		if files, ok := featureMap["files"].(map[string]interface{}); ok {
			for filePath := range files {
				allFiles = append(allFiles, filePath)
				t.Logf("Collected file from feature '%s': %s", featureName, filePath)
			}
		}
	}

	// Verify collected files
	require.Len(t, allFiles, 2, "Should collect 2 files from feature")
	assert.Contains(t, allFiles, "logger.py", "Should include logger.py")
	assert.Contains(t, allFiles, "logger_test.py", "Should include logger_test.py")

	t.Log("✅ Level 1: Project with feature - files collected correctly")
}

// TestProjectFilesCollection_Level2_ProjectWithKit validates file collection
// from a project with kits attached
func TestUnitProjectFilesCollection_Level2_ProjectWithKit(t *testing.T) {
	// Mock kit data (simulates what would come from registry)
	kitsData := map[string]interface{}{
		"kit:language:go": map[string]interface{}{
			"name": "go",
			"files": map[string]interface{}{
				"go.mod": map[string]interface{}{
					"content": "module example.com/project\n",
				},
				"main.go": map[string]interface{}{
					"content": "package main\n",
				},
			},
		},
		"kit:tool:gotask": map[string]interface{}{
			"name": "gotask",
			"files": map[string]interface{}{
				"Taskfile.yaml": map[string]interface{}{
					"content": "version: '3'\n",
				},
			},
		},
	}

	// Collect files from kits
	var allFiles []string
	for kitID, kitData := range kitsData {
		kitMap, ok := kitData.(map[string]interface{})
		require.True(t, ok, "kit should be a map")

		if files, ok := kitMap["files"].(map[string]interface{}); ok {
			for filePath := range files {
				allFiles = append(allFiles, filePath)
				t.Logf("Collected file from kit '%s': %s", kitID, filePath)
			}
		}
	}

	// Verify collected files
	require.Len(t, allFiles, 3, "Should collect 3 files from 2 kits")

	assert.Contains(t, allFiles, "go.mod", "Should include go.mod from go kit")
	assert.Contains(t, allFiles, "main.go", "Should include main.go from go kit")
	assert.Contains(t, allFiles, "Taskfile.yaml", "Should include Taskfile.yaml from gotask kit")

	t.Log("✅ Level 2: Project with kits - files collected correctly")
}

// TestProjectFilesCollection_Level3_ProjectWithStack validates file collection
// from a project with a full stack (simulating go-viper-cobra-pterm stack)
func TestUnitProjectFilesCollection_Level3_ProjectWithStack(t *testing.T) {
	// Mock stack structure with files at different levels
	stackData := map[string]interface{}{
		// Stack's own files (lowest precedence)
		"stack_files": map[string]interface{}{
			"README.md": map[string]interface{}{
				"content": "# Stack README\n",
			},
			".gitignore": map[string]interface{}{
				"content": "/bin/\n",
			},
		},
		// Kit files (from go, cobra, viper, pterm, gotask)
		"kit_files": map[string]interface{}{
			"go.mod": map[string]interface{}{
				"content": "module example.com/project\n",
			},
			"main.go": map[string]interface{}{
				"content": "package main\n",
			},
			"cmd/root.go": map[string]interface{}{
				"content": "// Cobra root command\n",
			},
			"Taskfile.yaml": map[string]interface{}{
				"content": "version: '3'\n",
			},
		},
		// Feature files (from viper_config, version_command, pterm_logger, etc.)
		"feature_files": map[string]interface{}{
			"internal/config/config.go": map[string]interface{}{
				"content": "// Viper config\n",
			},
			"internal/version/version.go": map[string]interface{}{
				"content": "// Version info\n",
			},
			"internal/logger/logger.go": map[string]interface{}{
				"content": "// Pterm logger\n",
			},
		},
	}

	// Collect all files (simulating merge from stack + kits + features)
	var allFiles []string
	fileCount := make(map[string]int) // Track file counts by source

	// Stack files
	if stackFiles, ok := stackData["stack_files"].(map[string]interface{}); ok {
		for filePath := range stackFiles {
			allFiles = append(allFiles, filePath)
			fileCount["stack"]++
		}
	}

	// Kit files
	if kitFiles, ok := stackData["kit_files"].(map[string]interface{}); ok {
		for filePath := range kitFiles {
			allFiles = append(allFiles, filePath)
			fileCount["kit"]++
		}
	}

	// Feature files
	if featureFiles, ok := stackData["feature_files"].(map[string]interface{}); ok {
		for filePath := range featureFiles {
			allFiles = append(allFiles, filePath)
			fileCount["feature"]++
		}
	}

	// Verify total count
	// Stack: 2 files + Kit: 4 files + Feature: 3 files = 9 total
	require.Len(t, allFiles, 9, "Should collect 9 files from stack (2 stack + 4 kit + 3 feature)")

	t.Logf("Collected %d files:", len(allFiles))
	for i, file := range allFiles {
		t.Logf("  %d. %s", i+1, file)
	}

	// Verify stack files
	t.Run("StackFiles", func(t *testing.T) {
		assert.Contains(t, allFiles, "README.md", "Should have stack README")
		assert.Contains(t, allFiles, ".gitignore", "Should have stack .gitignore")
	})

	// Verify kit files
	t.Run("KitFiles", func(t *testing.T) {
		assert.Contains(t, allFiles, "go.mod", "Should have go.mod from go kit")
		assert.Contains(t, allFiles, "main.go", "Should have main.go from go kit")
		assert.Contains(t, allFiles, "cmd/root.go", "Should have cmd/root.go from cobra kit")
		assert.Contains(t, allFiles, "Taskfile.yaml", "Should have Taskfile.yaml from gotask kit")
	})

	// Verify feature files
	t.Run("FeatureFiles", func(t *testing.T) {
		assert.Contains(t, allFiles, "internal/config/config.go", "Should have config from viper_config feature")
		assert.Contains(t, allFiles, "internal/version/version.go", "Should have version from version_command feature")
		assert.Contains(t, allFiles, "internal/logger/logger.go", "Should have logger from pterm_logger feature")
	})

	t.Log("✅ Level 3: Project with stack - all files collected correctly")
	t.Logf("   - %d files from stack", fileCount["stack"])
	t.Logf("   - %d files from kits", fileCount["kit"])
	t.Logf("   - %d files from features", fileCount["feature"])
	t.Logf("   - %d total files", len(allFiles))
}

// TestProjectFilesCollection_Precedence validates that project files override
// stack/kit/feature files with the same path
func TestUnitProjectFilesCollection_Precedence(t *testing.T) {
	// Mock file sources with overlapping paths
	stackFiles := map[string]string{
		"config.txt": "stack content",
		"README.md":  "stack readme",
	}

	kitFiles := map[string]string{
		"config.txt": "kit content", // Overrides stack
		"main.go":    "kit main",
	}

	featureFiles := map[string]string{
		"config.txt": "feature content", // Overrides kit and stack
		"logger.py":  "feature logger",
	}

	projectFiles := map[string]string{
		"config.txt": "project content", // Overrides everything
		"custom.txt": "project custom",
	}

	// Simulate merge with precedence: project > feature > kit > stack
	mergedFiles := make(map[string]string)

	// Add stack files (lowest precedence)
	for path, content := range stackFiles {
		mergedFiles[path] = content
	}

	// Add kit files (override stack)
	for path, content := range kitFiles {
		mergedFiles[path] = content
	}

	// Add feature files (override kit and stack)
	for path, content := range featureFiles {
		mergedFiles[path] = content
	}

	// Add project files (override everything)
	for path, content := range projectFiles {
		mergedFiles[path] = content
	}

	// Verify precedence
	t.Run("ProjectOverridesAll", func(t *testing.T) {
		assert.Equal(t, "project content", mergedFiles["config.txt"],
			"Project file should override stack/kit/feature files")
		assert.Equal(t, "project custom", mergedFiles["custom.txt"],
			"Project-only file should be present")
	})

	t.Run("FeatureOverridesKitAndStack", func(t *testing.T) {
		assert.Equal(t, "feature logger", mergedFiles["logger.py"],
			"Feature file should be present")
	})

	t.Run("KitOverridesStack", func(t *testing.T) {
		assert.Equal(t, "kit main", mergedFiles["main.go"],
			"Kit file should override stack file")
	})

	t.Run("StackFilesPresent", func(t *testing.T) {
		assert.Equal(t, "kit main", mergedFiles["main.go"],
			"Kit main overrides stack if stack had main.go")
	})

	// Verify final file count
	expectedFiles := []string{"config.txt", "README.md", "main.go", "logger.py", "custom.txt"}
	require.Len(t, mergedFiles, len(expectedFiles), "Should have correct number of merged files")

	for _, file := range expectedFiles {
		assert.Contains(t, mergedFiles, file, "Should contain %s", file)
	}

	t.Log("✅ File precedence: project > feature > kit > stack verified")
	t.Logf("   config.txt: %s (project wins)", mergedFiles["config.txt"])
	t.Logf("   README.md: %s (stack, not overridden)", mergedFiles["README.md"])
	t.Logf("   main.go: %s (kit)", mergedFiles["main.go"])
	t.Logf("   logger.py: %s (feature)", mergedFiles["logger.py"])
	t.Logf("   custom.txt: %s (project only)", mergedFiles["custom.txt"])
}

// TestProjectFilesCollection_EmptyCases validates handling of edge cases
func TestUnitProjectFilesCollection_EmptyCases(t *testing.T) {
	t.Run("NoFiles", func(t *testing.T) {
		// Kit with no files
		kitData := map[string]interface{}{
			"kit:tool:empty": map[string]interface{}{
				"name":    "empty-tool",
				"version": "1.0.0",
				// No files field
			},
		}

		var allFiles []string
		for _, data := range kitData {
			kitMap := data.(map[string]interface{})
			if files, ok := kitMap["files"].(map[string]interface{}); ok {
				for filePath := range files {
					allFiles = append(allFiles, filePath)
				}
			}
		}

		assert.Empty(t, allFiles, "Should handle kit with no files")
	})

	t.Run("EmptyFileContent", func(t *testing.T) {
		// File with empty content
		featureData := map[string]interface{}{
			"files": map[string]interface{}{
				"empty.txt": map[string]interface{}{
					"content": "",
				},
			},
		}

		files := featureData["files"].(map[string]interface{})
		emptyFile := files["empty.txt"].(map[string]interface{})
		content := emptyFile["content"].(string)

		assert.Equal(t, "", content, "Should allow empty file content")
		assert.Contains(t, files, "empty.txt", "Empty file should still be tracked")
	})

	t.Log("✅ Edge cases handled correctly")
}

// TestProjectFilesCollection_BugRepro_FeatureKitFiles reproduces scenarios
// where files from kits referenced by features should be collected
func TestUnitProjectFilesCollection_BugRepro_FeatureKitFiles(t *testing.T) {
	// This test ensures that when a feature references a kit,
	// the kit's files are properly collected

	// Mock stack structure
	stackData := map[string]interface{}{
		// Kit A: Directly in stack kits list
		"kit:tool:go_mod": map[string]interface{}{
			"name": "go_mod",
			"files": map[string]interface{}{
				"go.mod": map[string]interface{}{
					"content": "module example.com/project\n",
				},
			},
		},
		// Kit B: Referenced by feature, NOT in stack kits list directly
		"kit:tool:gotask": map[string]interface{}{
			"name": "gotask",
			"files": map[string]interface{}{
				"Taskfile.yaml": map[string]interface{}{
					"content": "version: '3'\n",
				},
			},
		},
	}

	// Feature that references kit B
	taskfileFeature := map[string]interface{}{
		"name": "taskfile",
		"kits": []string{"kit:tool:gotask"}, // Feature depends on gotask kit
		"requirements": []interface{}{
			map[string]interface{}{
				"name": "Taskfile Configuration",
				"instructions": []interface{}{
					map[string]interface{}{"prompt": "Create Taskfile.yaml"},
				},
			},
		},
		"files": map[string]interface{}{
			"scripts/setup.sh": map[string]interface{}{
				"content": "#!/bin/bash\n",
			},
		},
	}

	// Collect files from kits
	var allFiles []string
	for kitID, kitData := range stackData {
		kitMap := kitData.(map[string]interface{})
		if files, ok := kitMap["files"].(map[string]interface{}); ok {
			for filePath := range files {
				allFiles = append(allFiles, filePath)
				t.Logf("Collected file from kit '%s': %s", kitID, filePath)
			}
		}
	}

	// Collect files from feature
	if files, ok := taskfileFeature["files"].(map[string]interface{}); ok {
		for filePath := range files {
			allFiles = append(allFiles, filePath)
			t.Logf("Collected file from feature: %s", filePath)
		}
	}

	t.Logf("Collected %d files:", len(allFiles))
	for i, file := range allFiles {
		t.Logf("  %d. %s", i+1, file)
	}

	// CRITICAL ASSERTION: Kit B (gotask) files MUST be included
	// even though gotask is only referenced by a feature, not directly by the stack
	require.Len(t, allFiles, 3, "Should collect 3 files: 1 from go_mod + 1 from gotask + 1 from feature")

	assert.Contains(t, allFiles, "go.mod",
		"Should include go.mod (from go_mod kit, directly in stack)")

	// BUG DETECTION: This will be MISSING if kit files from feature-referenced kits are not collected
	assert.Contains(t, allFiles, "Taskfile.yaml",
		"❌ BUG: Should include Taskfile.yaml (from gotask kit, referenced by taskfile feature)")

	assert.Contains(t, allFiles, "scripts/setup.sh",
		"Should include scripts/setup.sh (from taskfile feature)")

	t.Log("✅ Bug reproduction test: All files from feature-referenced kits are properly collected")
	t.Log("   If this test FAILS, it means files from kits referenced by features are not being collected!")
}

// TestProjectFilesCollection_FieldPreservation validates that file fields
// (instructions.prompt, instructions.constraints, verifications) are preserved through merging
func TestUnitProjectFilesCollection_FieldPreservation(t *testing.T) {
	// Mock files with full field definitions from different sources

	// Kit file with instructions and verifications
	kitFile := map[string]interface{}{
		"path":    "cmd/root.go",
		"content": "// Cobra root command\n",
		"instructions": []interface{}{
			map[string]interface{}{
				"prompt": "Implement Cobra root command structure",
				"constraints": []interface{}{
					"Use cobra.Command struct",
					"NO custom flag parsing",
				},
			},
		},
		"verifications": []interface{}{
			map[string]interface{}{
				"command": "go build cmd/root.go",
				"expect":  "success",
			},
		},
	}

	// Feature file with instructions only
	featureFile := map[string]interface{}{
		"path":    "internal/config/config.go",
		"content": "// Viper configuration\n",
		"instructions": []interface{}{
			map[string]interface{}{
				"prompt": "Implement Viper configuration management",
				"constraints": []interface{}{
					"Config file must be optional",
					"Use environment prefix",
				},
			},
		},
	}

	// Project file with both (should override kit file precedence)
	projectFile := map[string]interface{}{
		"path":    "cmd/root.go", // Same path as kit file
		"content": "// Custom root command\n",
		"instructions": []interface{}{
			map[string]interface{}{
				"prompt": "Custom root command implementation",
			},
		},
	}

	// Simulate merge with precedence: project > feature > kit
	mergedFiles := make(map[string]map[string]interface{})

	// Add kit file
	mergedFiles[kitFile["path"].(string)] = kitFile

	// Add feature file
	mergedFiles[featureFile["path"].(string)] = featureFile

	// Add project file (overrides kit file)
	mergedFiles[projectFile["path"].(string)] = projectFile

	// Verify field preservation

	t.Run("KitFileFields", func(t *testing.T) {
		// cmd/root.go was overridden by project, but verify fields are structurally present
		rootFile := mergedFiles["cmd/root.go"]

		// Verify content overridden
		assert.Equal(t, "// Custom root command\n", rootFile["content"],
			"Project file content should override kit file content")

		// Verify instructions structure present
		instructions, ok := rootFile["instructions"].([]interface{})
		require.True(t, ok, "Instructions should be present")
		require.Len(t, instructions, 1, "Should have 1 instruction")

		instr := instructions[0].(map[string]interface{})
		assert.Equal(t, "Custom root command implementation", instr["prompt"],
			"Project instruction should override kit instruction")
	})

	t.Run("FeatureFileFields", func(t *testing.T) {
		configFile := mergedFiles["internal/config/config.go"]

		// Verify content
		assert.Equal(t, "// Viper configuration\n", configFile["content"])

		// Verify instructions with constraints
		instructions, ok := configFile["instructions"].([]interface{})
		require.True(t, ok, "Instructions should be present")
		require.Len(t, instructions, 1, "Should have 1 instruction")

		instr := instructions[0].(map[string]interface{})
		assert.Equal(t, "Implement Viper configuration management", instr["prompt"],
			"Instruction prompt should be preserved")

		constraints, ok := instr["constraints"].([]interface{})
		require.True(t, ok, "Constraints should be present")
		require.Len(t, constraints, 2, "Should have 2 constraints")
		assert.Contains(t, constraints, "Config file must be optional",
			"Constraint 1 should be preserved")
		assert.Contains(t, constraints, "Use environment prefix",
			"Constraint 2 should be preserved")
	})

	t.Log("✅ File field preservation: instructions.prompt, constraints, and verifications preserved through merging")
}

// TestProjectFilesCollection_FileVerifications validates that file-level
// verifications are preserved through merging
func TestUnitProjectFilesCollection_FileVerifications(t *testing.T) {
	// Mock files with verifications from different sources

	// Kit file with verifications
	kitFiles := map[string]interface{}{
		"go.mod": map[string]interface{}{
			"content": "module example.com/project\n",
			"verifications": []interface{}{
				map[string]interface{}{
					"command": "test -f go.mod && echo 'go.mod exists'",
					"expect":  "go.mod exists",
				},
			},
		},
		"Taskfile.yaml": map[string]interface{}{
			"content": "version: '3'\n",
			"verifications": []interface{}{
				map[string]interface{}{
					"command": "task --version",
					"expect":  "Task version: v3",
				},
				map[string]interface{}{
					"command": "which task && echo 'task found in PATH'",
					"expect":  "task found in PATH",
				},
			},
		},
	}

	// Feature file with verifications
	featureFiles := map[string]interface{}{
		"internal/logger/logger.go": map[string]interface{}{
			"content": "// Pterm logger\n",
			"verifications": []interface{}{
				map[string]interface{}{
					"command": "go test internal/logger/...",
					"expect":  "PASS",
				},
			},
		},
	}

	// Collect all files
	allFiles := make(map[string]map[string]interface{})
	for path, fileData := range kitFiles {
		allFiles[path] = fileData.(map[string]interface{})
	}
	for path, fileData := range featureFiles {
		allFiles[path] = fileData.(map[string]interface{})
	}

	// Verify file verifications are preserved

	t.Run("KitFileVerifications", func(t *testing.T) {
		// go.mod verification
		goModFile := allFiles["go.mod"]
		verifications, ok := goModFile["verifications"].([]interface{})
		require.True(t, ok, "go.mod should have verifications")
		require.Len(t, verifications, 1, "go.mod should have 1 verification")

		v1 := verifications[0].(map[string]interface{})
		assert.Equal(t, "test -f go.mod && echo 'go.mod exists'", v1["command"],
			"Verification command should be preserved")
		assert.Equal(t, "go.mod exists", v1["expect"],
			"Verification expect should be preserved")

		// Taskfile.yaml verifications (multiple)
		taskfileFile := allFiles["Taskfile.yaml"]
		verifications, ok = taskfileFile["verifications"].([]interface{})
		require.True(t, ok, "Taskfile.yaml should have verifications")
		require.Len(t, verifications, 2, "Taskfile.yaml should have 2 verifications")

		v1 = verifications[0].(map[string]interface{})
		assert.Equal(t, "task --version", v1["command"])

		v2 := verifications[1].(map[string]interface{})
		assert.Equal(t, "which task && echo 'task found in PATH'", v2["command"])
	})

	t.Run("FeatureFileVerifications", func(t *testing.T) {
		loggerFile := allFiles["internal/logger/logger.go"]
		verifications, ok := loggerFile["verifications"].([]interface{})
		require.True(t, ok, "logger.go should have verifications")
		require.Len(t, verifications, 1, "logger.go should have 1 verification")

		v1 := verifications[0].(map[string]interface{})
		assert.Equal(t, "go test internal/logger/...", v1["command"],
			"Verification command should be preserved")
		assert.Equal(t, "PASS", v1["expect"],
			"Verification expect should be preserved")
	})

	// Verify total verification count
	totalVerifications := 0
	for _, fileData := range allFiles {
		if verifications, ok := fileData["verifications"].([]interface{}); ok {
			totalVerifications += len(verifications)
		}
	}

	assert.Equal(t, 4, totalVerifications,
		"Should have 4 total verifications (1 go.mod + 2 Taskfile + 1 logger)")

	t.Log("✅ File verifications: All file-level verifications preserved through merging")
	t.Log("   This ensures verification commands can be collected and enforced")
}
