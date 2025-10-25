package files

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// FileOperation represents an explicit action to perform on a file
type FileOperation struct {
	Action       string // "add", "remove", "rename", "modify", "unchanged"
	Path         string // New/current path
	OldPath      string // For renames only
	Content      string
	Instructions []schemas.InstructionModel
	Verification []schemas.VerificationModel
}

// OperationType constants for file operations
const (
	OpAdd       = "add"
	OpRemove    = "remove"
	OpRename    = "rename"
	OpModify    = "modify"
	OpUnchanged = "unchanged"
)

// EnrichFilesWithInstructions enriches file models with action-based instructions
// Returns new file models with instructions populated based on what changed
func EnrichFilesWithInstructions(oldFiles, newFiles []schemas.FileModel) []schemas.FileModel {
	// Compute what operations need to happen
	operations := ComputeFileOperations(oldFiles, newFiles)

	// Create a map of path -> operation for quick lookup
	operationsByPath := make(map[string]FileOperation)
	for _, op := range operations {
		operationsByPath[op.Path] = op
	}

	// Enrich new files with instructions from operations
	enrichedFiles := make([]schemas.FileModel, len(newFiles))
	copy(enrichedFiles, newFiles)

	for i := range enrichedFiles {
		path := enrichedFiles[i].Path.ValueString()
		if op, exists := operationsByPath[path]; exists {
			// Use instructions from the operation (which were smartly merged)
			enrichedFiles[i].Instructions = op.Instructions
		}
	}

	return enrichedFiles
}

// ComputeFileOperations compares old and new file lists and returns explicit operations
// If files don't have user-provided instructions, auto-generates them based on the action
func ComputeFileOperations(oldFiles, newFiles []schemas.FileModel) []FileOperation {
	operations := []FileOperation{}

	// Build maps for efficient lookup
	oldMap := make(map[string]schemas.FileModel)
	newMap := make(map[string]schemas.FileModel)

	for _, file := range oldFiles {
		path := file.Path.ValueString()
		if path != "" {
			oldMap[path] = file
		}
	}

	for _, file := range newFiles {
		path := file.Path.ValueString()
		if path != "" {
			newMap[path] = file
		}
	}

	// Track which new files are part of renames (to avoid treating them as adds)
	renamedNewPaths := make(map[string]bool)

	// Step 1: Detect REMOVALS and RENAMES
	for oldPath, oldFile := range oldMap {
		if _, stillExists := newMap[oldPath]; !stillExists {
			// File path removed - check if it's a rename
			if newPath, newFile, isRename := findRename(oldFile, newMap); isRename {
				op := FileOperation{
					Action:       OpRename,
					OldPath:      oldPath,
					Path:         newPath,
					Content:      newFile.Content.ValueString(),
					Instructions: newFile.Instructions,
					Verification: newFile.Verification,
				}
				// Merge/generate instructions based on action
				op.Instructions = mergeInstructions(op, newFile.Instructions)
				operations = append(operations, op)
				renamedNewPaths[newPath] = true
			} else {
				// Genuine removal
				op := FileOperation{
					Action: OpRemove,
					Path:   oldPath,
				}
				// Always generate instruction for removals (no user file to provide it)
				op.Instructions = mergeInstructions(op, []schemas.InstructionModel{})
				operations = append(operations, op)
			}
		}
	}

	// Step 2: Detect MODIFICATIONS and UNCHANGED files
	for newPath, newFile := range newMap {
		// Skip if this path is part of a rename
		if renamedNewPaths[newPath] {
			continue
		}

		if oldFile, existed := oldMap[newPath]; existed {
			// Same path exists in both old and new
			if hasContentChanged(oldFile, newFile) {
				op := FileOperation{
					Action:       OpModify,
					Path:         newPath,
					Content:      newFile.Content.ValueString(),
					Instructions: newFile.Instructions,
					Verification: newFile.Verification,
				}
				// Merge/generate instructions based on action
				op.Instructions = mergeInstructions(op, newFile.Instructions)
				operations = append(operations, op)
			} else {
				// Content unchanged
				op := FileOperation{
					Action:       OpUnchanged,
					Path:         newPath,
					Content:      newFile.Content.ValueString(),
					Instructions: newFile.Instructions,
					Verification: newFile.Verification,
				}
				// Merge/generate instructions based on action
				op.Instructions = mergeInstructions(op, newFile.Instructions)
				operations = append(operations, op)
			}
		} else {
			// New file (not part of rename)
			op := FileOperation{
				Action:       OpAdd,
				Path:         newPath,
				Content:      newFile.Content.ValueString(),
				Instructions: newFile.Instructions,
				Verification: newFile.Verification,
			}
			// Auto-generate instruction if not provided by user
			if len(op.Instructions) == 0 {
				op.Instructions = generateInstructionForAction(op)
			}
			operations = append(operations, op)
		}
	}

	return operations
}

// mergeInstructions intelligently merges action-based instructions with user-provided instructions
// Logic:
// - DELETE/REMOVE: Use only action instruction (user instruction irrelevant when deleting)
// - RENAME: Use only action instruction (renaming preserves content, generation doesn't apply)
// - ADD: Combine action instruction + user instructions (create with specific content/generation)
// - MODIFY: Combine action instruction + user instructions (update with specific content/generation)
// - UNCHANGED: Use user instructions if any, otherwise no instruction needed
func mergeInstructions(op FileOperation, userInstructions []schemas.InstructionModel) []schemas.InstructionModel {
	hasUserInstructions := len(userInstructions) > 0

	switch op.Action {
	case OpRemove, OpRename:
		// For remove/rename, user instructions don't make sense
		// Example: "create a README" is wrong when we're deleting or renaming
		return generateInstructionForAction(op)

	case OpAdd, OpModify:
		if !hasUserInstructions {
			// No user instructions - just use action instruction
			return generateInstructionForAction(op)
		}
		// Combine: action instruction first, then user instructions
		actionInstructions := generateInstructionForAction(op)
		combined := make([]schemas.InstructionModel, 0, len(actionInstructions)+len(userInstructions))
		combined = append(combined, actionInstructions...)
		combined = append(combined, userInstructions...)
		return combined

	case OpUnchanged:
		if hasUserInstructions {
			// File unchanged but has user instructions - keep them for reference
			return userInstructions
		}
		// No user instructions and unchanged - minimal instruction
		return generateInstructionForAction(op)

	default:
		return generateInstructionForAction(op)
	}
}

// generateInstructionForAction creates an instruction based on the file operation action
// This provides explicit guidance to Claude about what to do with each file
func generateInstructionForAction(op FileOperation) []schemas.InstructionModel {
	var prompt string
	var constraints []types.String

	switch op.Action {
	case OpAdd:
		prompt = fmt.Sprintf("Create new file '%s' with the exact content specified in the files array", op.Path)
		constraints = []types.String{
			types.StringValue("Create parent directories if they don't exist (mkdir -p)"),
			types.StringValue("Use exact content without modification"),
			types.StringValue("Preserve trailing newlines as specified"),
		}

	case OpModify:
		prompt = fmt.Sprintf("Update existing file '%s' with new content from the files array", op.Path)
		constraints = []types.String{
			types.StringValue("Replace entire file content"),
			types.StringValue("Use exact content without modification"),
			types.StringValue("Preserve trailing newlines as specified"),
		}

	case OpRemove:
		prompt = fmt.Sprintf("Delete file '%s' from the project", op.Path)
		constraints = []types.String{
			types.StringValue("Remove the file completely"),
			types.StringValue("If this is the last file in its directory, remove the empty directory"),
			types.StringValue("Clean up parent directories recursively if they become empty"),
		}

	case OpRename:
		prompt = fmt.Sprintf("Rename file from '%s' to '%s'", op.OldPath, op.Path)
		constraints = []types.String{
			types.StringValue("Move/rename the file preserving its content"),
			types.StringValue("Create parent directories for new path if needed (mkdir -p)"),
			types.StringValue("Remove old file after successful move"),
			types.StringValue("Clean up old parent directories if they become empty"),
		}

	case OpUnchanged:
		prompt = fmt.Sprintf("File '%s' should remain unchanged", op.Path)
		constraints = []types.String{
			types.StringValue("No action needed - file is already correct"),
		}

	default:
		prompt = fmt.Sprintf("Process file '%s'", op.Path)
		constraints = []types.String{}
	}

	return []schemas.InstructionModel{
		{
			Prompt:      types.StringValue(prompt),
			Constraints: constraints,
		},
	}
}

// findRename detects if an old file was renamed by finding a matching new file
// Heuristics:
// 1. For static files: exact content match
// 2. For generated files: same instructions
func findRename(oldFile schemas.FileModel, newMap map[string]schemas.FileModel) (string, schemas.FileModel, bool) {
	oldContent := oldFile.Content.ValueString()
	oldHasInstructions := len(oldFile.Instructions) > 0

	for newPath, newFile := range newMap {
		newContent := newFile.Content.ValueString()
		newHasInstructions := len(newFile.Instructions) > 0

		// Case 1: Both are static files (no instructions)
		if !oldHasInstructions && !newHasInstructions {
			// Exact content match = likely rename
			if oldContent == newContent {
				return newPath, newFile, true
			}
		}

		// Case 2: Both have instructions (generated files)
		if oldHasInstructions && newHasInstructions {
			// Same instructions = likely rename
			if instructionsMatch(oldFile.Instructions, newFile.Instructions) {
				return newPath, newFile, true
			}
		}
	}

	return "", schemas.FileModel{}, false
}

// hasContentChanged checks if file content or instructions changed
func hasContentChanged(oldFile, newFile schemas.FileModel) bool {
	// Check content hash
	if computeContentHash(oldFile) != computeContentHash(newFile) {
		return true
	}

	// Check instructions
	if !instructionsMatch(oldFile.Instructions, newFile.Instructions) {
		return true
	}

	// Check verifications
	if !verificationsMatch(oldFile.Verification, newFile.Verification) {
		return true
	}

	return false
}

// computeContentHash creates a hash of file content
func computeContentHash(file schemas.FileModel) string {
	content := file.Content.ValueString()
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// instructionsMatch checks if two instruction lists are identical
func instructionsMatch(a, b []schemas.InstructionModel) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Prompt.ValueString() != b[i].Prompt.ValueString() {
			return false
		}

		// Check constraints
		aConstraints := make([]string, len(a[i].Constraints))
		bConstraints := make([]string, len(b[i].Constraints))

		for j, c := range a[i].Constraints {
			aConstraints[j] = c.ValueString()
		}
		for j, c := range b[i].Constraints {
			bConstraints[j] = c.ValueString()
		}

		if !stringSlicesEqual(aConstraints, bConstraints) {
			return false
		}
	}

	return true
}

// verificationsMatch checks if two verification lists are identical
func verificationsMatch(a, b []schemas.VerificationModel) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Command.ValueString() != b[i].Command.ValueString() {
			return false
		}
		if !a[i].Expect.Equal(b[i].Expect) {
			return false
		}
	}

	return true
}

// stringSlicesEqual compares two string slices
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// SerializeOperations converts FileOperations to JSON-serializable format
func SerializeOperations(ops []FileOperation) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(ops))

	for _, op := range ops {
		opData := map[string]interface{}{
			"action": op.Action,
			"path":   op.Path,
		}

		if op.Action == OpRename {
			opData["old_path"] = op.OldPath
		}

		if op.Content != "" {
			opData["content"] = op.Content
		}

		if len(op.Instructions) > 0 {
			instructions := make([]map[string]interface{}, 0, len(op.Instructions))
			for _, inst := range op.Instructions {
				instData := map[string]interface{}{
					"prompt": inst.Prompt.ValueString(),
				}
				if len(inst.Constraints) > 0 {
					constraints := make([]string, 0, len(inst.Constraints))
					for _, c := range inst.Constraints {
						if !c.IsNull() && !c.IsUnknown() {
							constraints = append(constraints, c.ValueString())
						}
					}
					if len(constraints) > 0 {
						instData["constraints"] = constraints
					}
				}
				instructions = append(instructions, instData)
			}
			opData["instructions"] = instructions
		}

		if len(op.Verification) > 0 {
			verifications := make([]map[string]interface{}, 0, len(op.Verification))
			for _, ver := range op.Verification {
				verData := map[string]interface{}{
					"command": ver.Command.ValueString(),
				}
				if !ver.Expect.IsNull() && !ver.Expect.IsUnknown() {
					verData["expect"] = ver.Expect.ValueString()
				}
				verifications = append(verifications, verData)
			}
			opData["verification"] = verifications
		}

		result = append(result, opData)
	}

	return result
}

// FormatOperationsSummary creates a human-readable summary of operations
func FormatOperationsSummary(ops []FileOperation) string {
	var adds, removes, renames, modifies, unchanged int

	for _, op := range ops {
		switch op.Action {
		case OpAdd:
			adds++
		case OpRemove:
			removes++
		case OpRename:
			renames++
		case OpModify:
			modifies++
		case OpUnchanged:
			unchanged++
		}
	}

	return fmt.Sprintf("Operations: +%d ~%d →%d -%d =%d",
		adds, modifies, renames, removes, unchanged)
}

// NormalizeFilesForState replaces action-based instructions with neutral ones for state storage
// After successful Claude execution, we store neutral instructions that won't change on subsequent applies
// This prevents Terraform from detecting drift when instructions are recomputed
func NormalizeFilesForState(files []schemas.FileModel) []schemas.FileModel {
	normalized := make([]schemas.FileModel, len(files))

	for i, file := range files {
		normalized[i] = file

		// Replace any instructions with a single neutral instruction
		// This instruction is stable and won't change between applies
		normalized[i].Instructions = []schemas.InstructionModel{
			{
				Prompt: types.StringValue("Ensure file is present with contents specified"),
				Constraints: []types.String{
					types.StringValue("File must exist at the specified path"),
					types.StringValue("Content must match exactly"),
				},
			},
		}
	}

	return normalized
}
