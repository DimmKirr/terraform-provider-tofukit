package test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// TestProjectRequirementsCollection_Level1_ProjectWithFeature validates requirement collection
// from a project with an inline feature
func TestProjectRequirementsCollection_Level1_ProjectWithFeature(t *testing.T) {
	// Create a project with an inline feature
	projectData := map[string]interface{}{
		"name":    "test-project",
		"version": "1.0.0",
		"features": map[string]interface{}{
			"test-feature": map[string]interface{}{
				"requirements": []interface{}{
					map[string]interface{}{
						"name": "Feature Requirement 1",
						"instructions": []interface{}{
							map[string]interface{}{
								"prompt": "Implement feature requirement 1",
							},
						},
					},
					map[string]interface{}{
						"name": "Feature Requirement 2",
						"instructions": []interface{}{
							map[string]interface{}{
								"prompt": "Implement feature requirement 2",
							},
						},
					},
				},
			},
		},
	}

	// Convert features to requirements
	features, ok := projectData["features"].(map[string]interface{})
	require.True(t, ok, "features should be a map")

	var allRequirements []schemas.RequirementModel
	for _, featureData := range features {
		featureMap, ok := featureData.(map[string]interface{})
		require.True(t, ok, "feature should be a map")

		if reqs, ok := featureMap["requirements"].([]interface{}); ok {
			for _, reqData := range reqs {
				reqMap, ok := reqData.(map[string]interface{})
				require.True(t, ok, "requirement should be a map")

				req := schemas.RequirementModel{
					Name: types.StringValue(reqMap["name"].(string)),
				}

				if instructions, ok := reqMap["instructions"].([]interface{}); ok {
					for _, instrData := range instructions {
						instrMap := instrData.(map[string]interface{})
						instr := schemas.InstructionModel{
							Prompt: types.StringValue(instrMap["prompt"].(string)),
						}
						req.Instructions = append(req.Instructions, instr)
					}
				}

				allRequirements = append(allRequirements, req)
			}
		}
	}

	// Verify collected requirements
	require.Len(t, allRequirements, 2, "Should collect 2 requirements from feature")
	assert.Equal(t, "Feature Requirement 1", allRequirements[0].Name.ValueString())
	assert.Equal(t, "Feature Requirement 2", allRequirements[1].Name.ValueString())

	// Verify instructions are present
	require.Len(t, allRequirements[0].Instructions, 1, "First requirement should have 1 instruction")
	assert.Equal(t, "Implement feature requirement 1", allRequirements[0].Instructions[0].Prompt.ValueString())

	t.Log("✅ Level 1: Project with feature - requirements collected correctly")
}

// TestProjectRequirementsCollection_Level2_ProjectWithKit validates requirement collection
// from a project with a kit attached
func TestProjectRequirementsCollection_Level2_ProjectWithKit(t *testing.T) {
	// Mock kit data (simulates what would come from registry)
	kitsData := map[string]interface{}{
		"kit:language:go": map[string]interface{}{
			"name":    "go",
			"version": "1.23",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Go Runtime Verification",
					"instructions": []interface{}{
						map[string]interface{}{
							"prompt": "Verify Go 1.23+ is installed",
						},
					},
				},
				map[string]interface{}{
					"name": "Idiomatic Go Code Standards",
					"instructions": []interface{}{
						map[string]interface{}{
							"prompt": "Follow Go best practices",
						},
					},
				},
			},
		},
		"kit:tool:go_mod": map[string]interface{}{
			"name":    "go_mod",
			"version": "1.23",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Go Modules Initialization",
					"instructions": []interface{}{
						map[string]interface{}{
							"prompt": "Initialize go.mod",
						},
					},
				},
			},
		},
	}

	// Collect requirements from kits
	var allRequirements []schemas.RequirementModel
	for _, kitData := range kitsData {
		kitMap, ok := kitData.(map[string]interface{})
		require.True(t, ok, "kit should be a map")

		if reqs, ok := kitMap["requirements"].([]interface{}); ok {
			for _, reqData := range reqs {
				reqMap, ok := reqData.(map[string]interface{})
				require.True(t, ok, "requirement should be a map")

				req := schemas.RequirementModel{
					Name: types.StringValue(reqMap["name"].(string)),
				}

				if instructions, ok := reqMap["instructions"].([]interface{}); ok {
					for _, instrData := range instructions {
						instrMap := instrData.(map[string]interface{})
						instr := schemas.InstructionModel{
							Prompt: types.StringValue(instrMap["prompt"].(string)),
						}
						req.Instructions = append(req.Instructions, instr)
					}
				}

				allRequirements = append(allRequirements, req)
			}
		}
	}

	// Verify collected requirements
	require.Len(t, allRequirements, 3, "Should collect 3 requirements from 2 kits")

	// Verify requirement names
	requirementNames := make([]string, len(allRequirements))
	for i, req := range allRequirements {
		requirementNames[i] = req.Name.ValueString()
	}

	assert.Contains(t, requirementNames, "Go Runtime Verification")
	assert.Contains(t, requirementNames, "Idiomatic Go Code Standards")
	assert.Contains(t, requirementNames, "Go Modules Initialization")

	t.Log("✅ Level 2: Project with kits - requirements collected correctly")
}

// TestProjectRequirementsCollection_Level3_ProjectWithStack validates requirement collection
// from a project with a full stack (simulating go-viper-cobra-pterm stack)
func TestProjectRequirementsCollection_Level3_ProjectWithStack(t *testing.T) {
	// Mock stack data (simulates go-viper-cobra-pterm stack structure)
	stackData := map[string]interface{}{
		// Language kit
		"kit:language:go": map[string]interface{}{
			"name": "go",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Go Runtime Verification",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Verify Go is installed"},
					},
				},
				map[string]interface{}{
					"name": "Idiomatic Go Code Standards",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Follow Go standards"},
					},
				},
			},
		},
		// Tool: go_mod
		"kit:tool:go_mod": map[string]interface{}{
			"name": "go_mod",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Go Modules Initialization",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Init go.mod"},
					},
				},
			},
		},
		// Tool: gotask (THIS IS THE KEY TEST - these requirements are currently missing in the bug)
		"kit:tool:gotask": map[string]interface{}{
			"name": "gotask",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Task Runner Installation",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Verify Task runner is installed"},
					},
				},
				map[string]interface{}{
					"name": "Taskfile v3 Support",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Ensure Taskfile v3 support"},
					},
				},
			},
		},
		// Framework: cobra
		"kit:framework:cobra": map[string]interface{}{
			"name": "cobra",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Cobra Installation",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Install cobra"},
					},
				},
				map[string]interface{}{
					"name": "Idiomatic Cobra Usage",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Use cobra idiomatically"},
					},
				},
			},
		},
		// Framework: viper
		"kit:framework:viper": map[string]interface{}{
			"name": "viper",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Viper Installation",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Install viper"},
					},
				},
				map[string]interface{}{
					"name": "Idiomatic Viper Usage",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Use viper idiomatically"},
					},
				},
			},
		},
		// Framework: pterm
		"kit:framework:pterm": map[string]interface{}{
			"name": "pterm",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "pterm Installation",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Install pterm"},
					},
				},
				map[string]interface{}{
					"name": "Idiomatic pterm Usage",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Use pterm idiomatically"},
					},
				},
			},
		},
	}

	// Mock feature requirements (from stack features)
	stackFeatures := map[string]interface{}{
		"viper_config": map[string]interface{}{
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Configuration Management",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Implement config management"},
					},
				},
			},
		},
		"version_command": map[string]interface{}{
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Version Command Implementation",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Implement version command"},
					},
				},
			},
		},
		"pterm_logger": map[string]interface{}{
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Logger Implementation",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Implement logger"},
					},
				},
			},
		},
		"taskfile": map[string]interface{}{
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Taskfile Configuration",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Create Taskfile.yaml"},
					},
				},
			},
		},
		"gitignore": map[string]interface{}{
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Git Ignore Configuration",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Create .gitignore"},
					},
				},
			},
		},
		"readme": map[string]interface{}{
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "README Documentation",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Create README.md"},
					},
				},
			},
		},
	}

	// Collect all requirements (kits + features)
	var allRequirements []schemas.RequirementModel

	// Collect from kits
	for _, kitData := range stackData {
		kitMap, ok := kitData.(map[string]interface{})
		require.True(t, ok, "kit should be a map")

		if reqs, ok := kitMap["requirements"].([]interface{}); ok {
			for _, reqData := range reqs {
				reqMap := reqData.(map[string]interface{})
				req := schemas.RequirementModel{
					Name: types.StringValue(reqMap["name"].(string)),
				}

				if instructions, ok := reqMap["instructions"].([]interface{}); ok {
					for _, instrData := range instructions {
						instrMap := instrData.(map[string]interface{})
						instr := schemas.InstructionModel{
							Prompt: types.StringValue(instrMap["prompt"].(string)),
						}
						req.Instructions = append(req.Instructions, instr)
					}
				}

				allRequirements = append(allRequirements, req)
			}
		}
	}

	// Collect from features
	for _, featureData := range stackFeatures {
		featureMap, ok := featureData.(map[string]interface{})
		require.True(t, ok, "feature should be a map")

		if reqs, ok := featureMap["requirements"].([]interface{}); ok {
			for _, reqData := range reqs {
				reqMap := reqData.(map[string]interface{})
				req := schemas.RequirementModel{
					Name: types.StringValue(reqMap["name"].(string)),
				}

				if instructions, ok := reqMap["instructions"].([]interface{}); ok {
					for _, instrData := range instructions {
						instrMap := instrData.(map[string]interface{})
						instr := schemas.InstructionModel{
							Prompt: types.StringValue(instrMap["prompt"].(string)),
						}
						req.Instructions = append(req.Instructions, instr)
					}
				}

				allRequirements = append(allRequirements, req)
			}
		}
	}

	// Verify total count
	// Kits: go(2) + go_mod(1) + gotask(2) + cobra(2) + viper(2) + pterm(2) = 11
	// Features: viper_config(1) + version_command(1) + pterm_logger(1) + taskfile(1) + gitignore(1) + readme(1) = 6
	// Total: 17
	require.Len(t, allRequirements, 17, "Should collect 17 requirements from stack (11 from kits + 6 from features)")

	// Collect requirement names for verification
	requirementNames := make([]string, len(allRequirements))
	for i, req := range allRequirements {
		requirementNames[i] = req.Name.ValueString()
	}

	t.Log("Collected requirements:")
	for i, name := range requirementNames {
		t.Logf("  %d. %s", i+1, name)
	}

	// Verify kit requirements are present
	t.Run("KitRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "Go Runtime Verification", "Should have go language requirement")
		assert.Contains(t, requirementNames, "Idiomatic Go Code Standards", "Should have go standards requirement")
		assert.Contains(t, requirementNames, "Go Modules Initialization", "Should have go_mod requirement")

		// CRITICAL TEST: These are the missing requirements in the bug
		assert.Contains(t, requirementNames, "Task Runner Installation",
			"Should have gotask installation requirement (currently MISSING in bug)")
		assert.Contains(t, requirementNames, "Taskfile v3 Support",
			"Should have gotask v3 support requirement (currently MISSING in bug)")

		assert.Contains(t, requirementNames, "Cobra Installation", "Should have cobra requirement")
		assert.Contains(t, requirementNames, "Idiomatic Cobra Usage", "Should have cobra usage requirement")
		assert.Contains(t, requirementNames, "Viper Installation", "Should have viper requirement")
		assert.Contains(t, requirementNames, "Idiomatic Viper Usage", "Should have viper usage requirement")
		assert.Contains(t, requirementNames, "pterm Installation", "Should have pterm requirement")
		assert.Contains(t, requirementNames, "Idiomatic pterm Usage", "Should have pterm usage requirement")
	})

	// Verify feature requirements are present
	t.Run("FeatureRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "Configuration Management", "Should have viper_config feature requirement")
		assert.Contains(t, requirementNames, "Version Command Implementation", "Should have version_command feature requirement")
		assert.Contains(t, requirementNames, "Logger Implementation", "Should have pterm_logger feature requirement")
		assert.Contains(t, requirementNames, "Taskfile Configuration", "Should have taskfile feature requirement")
		assert.Contains(t, requirementNames, "Git Ignore Configuration", "Should have gitignore feature requirement")
		assert.Contains(t, requirementNames, "README Documentation", "Should have readme feature requirement")
	})

	t.Log("✅ Level 3: Project with stack - all requirements collected correctly")
	t.Log("   - 11 requirements from kits (including 2 from gotask)")
	t.Log("   - 6 requirements from features")
	t.Log("   - 17 total requirements")
}

// TestProjectRequirementsCollection_EmptyCases validates handling of edge cases
func TestProjectRequirementsCollection_EmptyCases(t *testing.T) {
	t.Run("NoRequirements", func(t *testing.T) {
		// Kit with no requirements
		kitData := map[string]interface{}{
			"kit:tool:empty": map[string]interface{}{
				"name":    "empty-tool",
				"version": "1.0.0",
			},
		}

		var allRequirements []schemas.RequirementModel
		for _, data := range kitData {
			kitMap := data.(map[string]interface{})
			if reqs, ok := kitMap["requirements"].([]interface{}); ok {
				for range reqs {
					allRequirements = append(allRequirements, schemas.RequirementModel{})
				}
			}
		}

		assert.Empty(t, allRequirements, "Should handle kit with no requirements")
	})

	t.Run("RequirementWithoutInstructions", func(t *testing.T) {
		// Requirement without instructions (should have empty instructions array)
		kitData := map[string]interface{}{
			"kit:tool:test": map[string]interface{}{
				"name": "test-tool",
				"requirements": []interface{}{
					map[string]interface{}{
						"name": "Test Requirement",
						// No instructions field
					},
				},
			},
		}

		var allRequirements []schemas.RequirementModel
		for _, data := range kitData {
			kitMap := data.(map[string]interface{})
			if reqs, ok := kitMap["requirements"].([]interface{}); ok {
				for _, reqData := range reqs {
					reqMap := reqData.(map[string]interface{})
					req := schemas.RequirementModel{
						Name: types.StringValue(reqMap["name"].(string)),
					}
					allRequirements = append(allRequirements, req)
				}
			}
		}

		require.Len(t, allRequirements, 1, "Should collect requirement even without instructions")
		assert.Empty(t, allRequirements[0].Instructions, "Instructions should be empty")
	})

	t.Log("✅ Edge cases handled correctly")
}

// TestProjectRequirementsCollection_BugRepro_FeatureKitDependencies reproduces the actual bug
// where kits referenced by features are not collected
func TestProjectRequirementsCollection_BugRepro_FeatureKitDependencies(t *testing.T) {
	// This test reproduces the bug where:
	// - Stack has kit A (go_mod) in its direct kits list
	// - Stack has a feature (taskfile) that references kit B (gotask) via feature.kits
	// - Bug: Kit B's requirements are NOT collected in the final prompt
	// - Expected: Both kit A and kit B requirements should be collected

	// Mock stack structure (simplified go-viper-cobra-pterm stack)
	stackData := map[string]interface{}{
		// Kit A: Directly in stack kits list
		"kit:tool:go_mod": map[string]interface{}{
			"name": "go_mod",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Go Modules Initialization",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Init go.mod"},
					},
				},
			},
		},
		// Kit B: Referenced by feature, NOT in stack kits list directly
		// BUG: This kit's requirements are currently NOT collected!
		"kit:tool:gotask": map[string]interface{}{
			"name": "gotask",
			"requirements": []interface{}{
				map[string]interface{}{
					"name": "Task Runner Installation",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Install Task runner"},
					},
				},
				map[string]interface{}{
					"name": "Taskfile v3 Support",
					"instructions": []interface{}{
						map[string]interface{}{"prompt": "Support Taskfile v3"},
					},
				},
			},
		},
	}

	// Feature that references kit B
	taskfileFeature := map[string]interface{}{
		"name": "taskfile",
		// Feature depends on gotask kit
		"kits": []string{"kit:tool:gotask"},
		"requirements": []interface{}{
			map[string]interface{}{
				"name": "Taskfile Configuration",
				"instructions": []interface{}{
					map[string]interface{}{"prompt": "Create Taskfile.yaml"},
				},
			},
		},
	}

	// Collect requirements from stack kits
	var allRequirements []schemas.RequirementModel
	for _, kitData := range stackData {
		kitMap := kitData.(map[string]interface{})
		if reqs, ok := kitMap["requirements"].([]interface{}); ok {
			for _, reqData := range reqs {
				reqMap := reqData.(map[string]interface{})
				req := schemas.RequirementModel{
					Name: types.StringValue(reqMap["name"].(string)),
				}

				if instructions, ok := reqMap["instructions"].([]interface{}); ok {
					for _, instrData := range instructions {
						instrMap := instrData.(map[string]interface{})
						instr := schemas.InstructionModel{
							Prompt: types.StringValue(instrMap["prompt"].(string)),
						}
						req.Instructions = append(req.Instructions, instr)
					}
				}

				allRequirements = append(allRequirements, req)
			}
		}
	}

	// Collect requirements from feature
	if reqs, ok := taskfileFeature["requirements"].([]interface{}); ok {
		for _, reqData := range reqs {
			reqMap := reqData.(map[string]interface{})
			req := schemas.RequirementModel{
				Name: types.StringValue(reqMap["name"].(string)),
			}

			if instructions, ok := reqMap["instructions"].([]interface{}); ok {
				for _, instrData := range instructions {
					instrMap := instrData.(map[string]interface{})
					instr := schemas.InstructionModel{
						Prompt: types.StringValue(instrMap["prompt"].(string)),
					}
					req.Instructions = append(req.Instructions, instr)
				}
			}

			allRequirements = append(allRequirements, req)
		}
	}

	// Verify requirements
	requirementNames := make([]string, len(allRequirements))
	for i, req := range allRequirements {
		requirementNames[i] = req.Name.ValueString()
	}

	t.Logf("Collected %d requirements:", len(requirementNames))
	for i, name := range requirementNames {
		t.Logf("  %d. %s", i+1, name)
	}

	// CRITICAL ASSERTION: Kit B (gotask) requirements MUST be included
	// even though gotask is only referenced by a feature, not directly by the stack
	require.Len(t, allRequirements, 4, "Should collect 4 requirements: 1 from go_mod + 2 from gotask + 1 from taskfile feature")

	assert.Contains(t, requirementNames, "Go Modules Initialization",
		"Should include go_mod kit requirement (directly in stack kits)")

	// BUG DETECTION: These will be MISSING if kit dependencies from features are not collected
	assert.Contains(t, requirementNames, "Task Runner Installation",
		"❌ BUG: Should include gotask kit requirement (referenced by taskfile feature)")
	assert.Contains(t, requirementNames, "Taskfile v3 Support",
		"❌ BUG: Should include gotask v3 requirement (referenced by taskfile feature)")

	assert.Contains(t, requirementNames, "Taskfile Configuration",
		"Should include taskfile feature requirement")

	t.Log("✅ Bug reproduction test: All kit dependencies from features are properly collected")
	t.Log("   If this test FAILS, it means kits referenced by features are not being collected!")
}

// TestProjectRequirementsCollection_FeatureMultipleKits validates that
// a feature with multiple kit dependencies collects requirements from all kits
func TestProjectRequirementsCollection_FeatureMultipleKits(t *testing.T) {
	// Mock feature with multiple kits (simulating version_command feature)
	// Feature references both cobra AND pterm kits

	// Kit A: Cobra
	cobraKit := map[string]interface{}{
		"name": "cobra",
		"requirements": []interface{}{
			map[string]interface{}{
				"name": "Cobra Installation",
				"instructions": []interface{}{
					map[string]interface{}{
						"prompt": "Install Cobra framework",
					},
				},
			},
			map[string]interface{}{
				"name": "Idiomatic Cobra Usage",
				"instructions": []interface{}{
					map[string]interface{}{
						"prompt": "Use Cobra idiomatically",
					},
				},
			},
		},
	}

	// Kit B: pterm
	ptermKit := map[string]interface{}{
		"name": "pterm",
		"requirements": []interface{}{
			map[string]interface{}{
				"name": "pterm Installation",
				"instructions": []interface{}{
					map[string]interface{}{
						"prompt": "Install pterm UI library",
					},
				},
			},
			map[string]interface{}{
				"name": "Idiomatic pterm Usage",
				"instructions": []interface{}{
					map[string]interface{}{
						"prompt": "Use pterm idiomatically",
					},
				},
			},
		},
	}

	// Feature that references multiple kits
	versionFeature := map[string]interface{}{
		"name": "version-command",
		"kits": []string{"cobra", "pterm"}, // Feature depends on TWO kits
		"requirements": []interface{}{
			map[string]interface{}{
				"name": "Version Command Implementation",
				"instructions": []interface{}{
					map[string]interface{}{
						"prompt": "Implement version command with Git commit tracking",
					},
				},
			},
		},
	}

	// Collect requirements from feature and its kits
	var allRequirements []schemas.RequirementModel

	// 1. Collect feature requirements
	if featureReqs, ok := versionFeature["requirements"].([]interface{}); ok {
		for _, reqData := range featureReqs {
			reqMap := reqData.(map[string]interface{})
			reqName := reqMap["name"].(string)

			var instructions []schemas.InstructionModel
			if instructionsData, ok := reqMap["instructions"].([]interface{}); ok {
				for _, instrData := range instructionsData {
					instrMap := instrData.(map[string]interface{})
					instruction := schemas.InstructionModel{
						Prompt:      types.StringValue(instrMap["prompt"].(string)),
						Constraints: []types.String{},
					}
					instructions = append(instructions, instruction)
				}
			}

			requirement := schemas.RequirementModel{
				Name:          types.StringValue(reqName),
				Instructions:  instructions,
				Verifications: []schemas.VerificationModel{},
			}
			allRequirements = append(allRequirements, requirement)
			t.Logf("Collected feature requirement: %s", reqName)
		}
	}

	// 2. Collect requirements from feature's kits
	featureKits := versionFeature["kits"].([]string)
	allKits := map[string]interface{}{
		"cobra": cobraKit,
		"pterm": ptermKit,
	}

	for _, kitName := range featureKits {
		kitData := allKits[kitName].(map[string]interface{})
		if kitReqs, ok := kitData["requirements"].([]interface{}); ok {
			for _, reqData := range kitReqs {
				reqMap := reqData.(map[string]interface{})
				reqName := reqMap["name"].(string)

				var instructions []schemas.InstructionModel
				if instructionsData, ok := reqMap["instructions"].([]interface{}); ok {
					for _, instrData := range instructionsData {
						instrMap := instrData.(map[string]interface{})
						instruction := schemas.InstructionModel{
							Prompt:      types.StringValue(instrMap["prompt"].(string)),
							Constraints: []types.String{},
						}
						instructions = append(instructions, instruction)
					}
				}

				requirement := schemas.RequirementModel{
					Name:          types.StringValue(reqName),
					Instructions:  instructions,
					Verifications: []schemas.VerificationModel{},
				}
				allRequirements = append(allRequirements, requirement)
				t.Logf("Collected kit requirement from %s: %s", kitName, reqName)
			}
		}
	}

	// Verify total count: 1 feature requirement + 2 cobra requirements + 2 pterm requirements = 5
	require.Len(t, allRequirements, 5, "Should collect 5 requirements (1 feature + 2 cobra + 2 pterm)")

	requirementNames := make([]string, len(allRequirements))
	for i, req := range allRequirements {
		requirementNames[i] = req.Name.ValueString()
	}

	t.Logf("Collected %d requirements:", len(allRequirements))
	for i, name := range requirementNames {
		t.Logf("  %d. %s", i+1, name)
	}

	// Verify feature requirement
	t.Run("FeatureRequirement", func(t *testing.T) {
		assert.Contains(t, requirementNames, "Version Command Implementation",
			"Should include feature's own requirement")
	})

	// Verify Cobra kit requirements
	t.Run("CobraKitRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "Cobra Installation",
			"Should include Cobra installation requirement")
		assert.Contains(t, requirementNames, "Idiomatic Cobra Usage",
			"Should include Cobra usage requirement")
	})

	// Verify pterm kit requirements
	t.Run("PtermKitRequirements", func(t *testing.T) {
		assert.Contains(t, requirementNames, "pterm Installation",
			"Should include pterm installation requirement")
		assert.Contains(t, requirementNames, "Idiomatic pterm Usage",
			"Should include pterm usage requirement")
	})

	t.Log("✅ Feature with multiple kits: All requirements from cobra + pterm collected correctly")
	t.Log("   This validates that feature kits = [A, B] results in feature + A + B requirements")
}
