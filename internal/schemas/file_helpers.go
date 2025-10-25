package schemas

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// FileModelWithPath represents a file with its path as a field
// Used internally when working with the files map
type FileModelWithPath struct {
	Path          string
	Content       types.String
	Instructions  []InstructionModel
	Verifications []VerificationModel
}

// FilesMapToList converts a types.Map of files to a sorted list of FileModelWithPath
func FilesMapToList(ctx context.Context, filesMap types.Map) []FileModelWithPath {
	if filesMap.IsNull() || filesMap.IsUnknown() {
		return nil
	}

	elements := filesMap.Elements()
	result := make([]FileModelWithPath, 0, len(elements))

	for pathKey, fileValue := range elements {
		// Extract the file object
		fileObj, ok := fileValue.(types.Object)
		if !ok {
			tflog.Warn(ctx, "File value is not an object", map[string]interface{}{
				"path": pathKey,
				"type": fmt.Sprintf("%T", fileValue),
			})
			continue
		}

		attrs := fileObj.Attributes()

		// Extract content
		content := types.StringNull()
		if contentVal, exists := attrs["content"]; exists {
			if strVal, ok := contentVal.(types.String); ok {
				content = strVal
			}
		}

		// Extract instructions
		var instructions []InstructionModel
		if instructionsVal, exists := attrs["instructions"]; exists {
			if listVal, ok := instructionsVal.(types.List); ok && !listVal.IsNull() {
				for _, instVal := range listVal.Elements() {
					if instObj, ok := instVal.(types.Object); ok {
						instAttrs := instObj.Attributes()

						prompt := types.StringNull()
						if promptVal, exists := instAttrs["prompt"]; exists {
							if strVal, ok := promptVal.(types.String); ok {
								prompt = strVal
							}
						}

						var constraints []types.String
						if constraintsVal, exists := instAttrs["constraints"]; exists {
							if constraintsList, ok := constraintsVal.(types.List); ok && !constraintsList.IsNull() {
								for _, c := range constraintsList.Elements() {
									if strVal, ok := c.(types.String); ok {
										constraints = append(constraints, strVal)
									}
								}
							}
						}

						instructions = append(instructions, InstructionModel{
							Prompt:      prompt,
							Constraints: constraints,
						})
					}
				}
			}
		}

		// Extract verifications
		var verifications []VerificationModel
		if verificationsVal, exists := attrs["verifications"]; exists {
			if listVal, ok := verificationsVal.(types.List); ok && !listVal.IsNull() {
				for _, verifVal := range listVal.Elements() {
					if verifObj, ok := verifVal.(types.Object); ok {
						verifAttrs := verifObj.Attributes()

						command := types.StringNull()
						if cmdVal, exists := verifAttrs["command"]; exists {
							if strVal, ok := cmdVal.(types.String); ok {
								command = strVal
							}
						}

						expect := types.StringNull()
						if expectVal, exists := verifAttrs["expect"]; exists {
							if strVal, ok := expectVal.(types.String); ok {
								expect = strVal
							}
						}

						verifications = append(verifications, VerificationModel{
							Command: command,
							Expect:  expect,
						})
					}
				}
			}
		}

		// Create FileModelWithPath
		result = append(result, FileModelWithPath{
			Path:          pathKey,
			Content:       content,
			Instructions:  instructions,
			Verifications: verifications,
		})
	}

	// Sort by path for deterministic ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})

	tflog.Debug(ctx, "Converted files map to list", map[string]interface{}{
		"count": len(result),
	})

	return result
}

// FilesListToMap converts a list of FileModelWithPath to a types.Map
func FilesListToMap(ctx context.Context, files []FileModelWithPath) types.Map {
	if len(files) == 0 {
		return types.MapNull(types.ObjectType{
			AttrTypes: getFileAttrTypes(),
		})
	}

	elements := make(map[string]attr.Value, len(files))

	for _, file := range files {
		// Convert instructions to attribute value
		var instructionsValue attr.Value
		if len(file.Instructions) == 0 {
			instructionsValue = types.ListNull(types.ObjectType{
				AttrTypes: getInstructionAttrTypes(),
			})
		} else {
			instElems := make([]attr.Value, len(file.Instructions))
			for i, inst := range file.Instructions {
				var constraintsValue attr.Value
				if len(inst.Constraints) == 0 {
					constraintsValue = types.ListNull(types.StringType)
				} else {
					constraintElems := make([]attr.Value, len(inst.Constraints))
					for j, c := range inst.Constraints {
						constraintElems[j] = c
					}
					constraintsValue = types.ListValueMust(types.StringType, constraintElems)
				}

				instElems[i] = types.ObjectValueMust(
					getInstructionAttrTypes(),
					map[string]attr.Value{
						"prompt":      inst.Prompt,
						"constraints": constraintsValue,
					},
				)
			}
			instructionsValue = types.ListValueMust(
				types.ObjectType{AttrTypes: getInstructionAttrTypes()},
				instElems,
			)
		}

		// Convert verifications to attribute value
		var verificationsValue attr.Value
		if len(file.Verifications) == 0 {
			verificationsValue = types.ListNull(types.ObjectType{
				AttrTypes: getVerificationAttrTypes(),
			})
		} else {
			verifElems := make([]attr.Value, len(file.Verifications))
			for i, verif := range file.Verifications {
				verifElems[i] = types.ObjectValueMust(
					getVerificationAttrTypes(),
					map[string]attr.Value{
						"command": verif.Command,
						"expect":  verif.Expect,
					},
				)
			}
			verificationsValue = types.ListValueMust(
				types.ObjectType{AttrTypes: getVerificationAttrTypes()},
				verifElems,
			)
		}

		// Create file object
		fileObj := types.ObjectValueMust(
			getFileAttrTypes(),
			map[string]attr.Value{
				"content":       file.Content,
				"instructions":  instructionsValue,
				"verifications": verificationsValue,
			},
		)

		elements[file.Path] = fileObj
	}

	return types.MapValueMust(
		types.ObjectType{AttrTypes: getFileAttrTypes()},
		elements,
	)
}

// Helper functions for attribute types

func getFileAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"content": types.StringType,
		"instructions": types.ListType{
			ElemType: types.ObjectType{AttrTypes: getInstructionAttrTypes()},
		},
		"verifications": types.ListType{
			ElemType: types.ObjectType{AttrTypes: getVerificationAttrTypes()},
		},
	}
}

func getInstructionAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"prompt":      types.StringType,
		"constraints": types.ListType{ElemType: types.StringType},
	}
}

func getVerificationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"command": types.StringType,
		"expect":  types.StringType,
	}
}
