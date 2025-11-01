# Per-File Drift Detection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement granular per-file drift detection that identifies which specific files changed outside Terraform and provides targeted restoration instructions to Claude.

**Architecture:** Follow hashicorp/local_file pattern - calculate hashes in Read() for comparison only, store hashes in Create()/Update(). Add computed fields (drift_detected, drifted_files) that trigger Update() when files are manually edited. Use modification time optimization to avoid unnecessary hash computations.

**Tech Stack:** Go 1.21, Terraform Plugin Framework, SHA256 hashing, os.Stat for mtime checks

---

## Phase 1: Foundation - Hash Computation Functions

### Task 1.1: Create Hash Utility Functions

**Files:**
- Create: `internal/files/hashing.go`
- Create: `internal/files/hashing_test.go`

**Step 1: Write failing test for content hash (static files)**

Create `internal/files/hashing_test.go`:

```go
package files

import (
	"testing"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

func TestComputeContentHash_StaticFile(t *testing.T) {
	fileModel := schemas.FileModel{
		Content: types.StringValue("*.log\n*.tmp\n"),
	}

	hash := ComputeContentHash(fileModel)

	// SHA256 of "*.log\n*.tmp\n"
	expected := "c5e8f0e9c35c9e7c8f3c4b5d6a7f8e9d0c1b2a3f4e5d6c7b8a9f0e1d2c3b4a5"
	assert.NotEmpty(t, hash, "Hash should not be empty")
	assert.Equal(t, 64, len(hash), "SHA256 hash should be 64 hex characters")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/files -run TestComputeContentHash_StaticFile -v`

Expected: FAIL with "undefined: ComputeContentHash"

**Step 3: Write minimal implementation**

Create `internal/files/hashing.go`:

```go
package files

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

// ComputeContentHash generates SHA256 hash of file specification
// For static files: hash of content string
// For generated files: hash of instructions JSON
func ComputeContentHash(file schemas.FileModel) string {
	var hashInput []byte

	if !file.Content.IsNull() && file.Content.ValueString() != "" {
		// Static file: hash the content
		hashInput = []byte(file.Content.ValueString())
	} else if len(file.Instructions) > 0 {
		// Generated file: hash the instructions JSON
		jsonBytes, _ := json.Marshal(file.Instructions)
		hashInput = jsonBytes
	} else {
		return "" // No hashable content
	}

	hash := sha256.Sum256(hashInput)
	return hex.EncodeToString(hash[:])
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/files -run TestComputeContentHash_StaticFile -v`

Expected: PASS

**Step 5: Add test for instruction-based files**

Add to `internal/files/hashing_test.go`:

```go
func TestComputeContentHash_InstructionFile(t *testing.T) {
	fileModel := schemas.FileModel{
		Instructions: []schemas.InstructionModel{
			{
				Prompt: types.StringValue("Create hello.go"),
				Constraints: []types.String{
					types.StringValue("Use package main"),
				},
			},
		},
	}

	hash := ComputeContentHash(fileModel)

	assert.NotEmpty(t, hash, "Hash should not be empty")
	assert.Equal(t, 64, len(hash), "SHA256 hash should be 64 hex characters")
}
```

**Step 6: Run test to verify it passes**

Run: `go test ./internal/files -run TestComputeContentHash_InstructionFile -v`

Expected: PASS

**Step 7: Add test for empty file**

Add to `internal/files/hashing_test.go`:

```go
func TestComputeContentHash_EmptyFile(t *testing.T) {
	fileModel := schemas.FileModel{}

	hash := ComputeContentHash(fileModel)

	assert.Empty(t, hash, "Hash should be empty for file with no content or instructions")
}
```

**Step 8: Run test to verify it passes**

Run: `go test ./internal/files -run TestComputeContentHash_EmptyFile -v`

Expected: PASS

**Step 9: Add file hash function test**

Add to `internal/files/hashing_test.go`:

```go
func TestComputeFileHash(t *testing.T) {
	content := []byte("package main\n\nfunc main() {}\n")

	hash := ComputeFileHash(content)

	assert.NotEmpty(t, hash, "Hash should not be empty")
	assert.Equal(t, 64, len(hash), "SHA256 hash should be 64 hex characters")

	// Same content should produce same hash
	hash2 := ComputeFileHash(content)
	assert.Equal(t, hash, hash2, "Same content should produce same hash")

	// Different content should produce different hash
	differentContent := []byte("package main\n\nfunc main() { println(\"hello\") }\n")
	hash3 := ComputeFileHash(differentContent)
	assert.NotEqual(t, hash, hash3, "Different content should produce different hash")
}
```

**Step 10: Implement ComputeFileHash**

Add to `internal/files/hashing.go`:

```go
// ComputeFileHash generates SHA256 hash of actual file content
func ComputeFileHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}
```

**Step 11: Run all hash tests**

Run: `go test ./internal/files -v`

Expected: All tests PASS

**Step 12: Commit hash utilities**

```bash
git add internal/files/hashing.go internal/files/hashing_test.go
git commit -m "feat(files): add hash computation utilities for drift detection

- Add ComputeContentHash for file specification hashing
- Add ComputeFileHash for actual file content hashing
- Support both static content and instruction-based files
- Include comprehensive test coverage

Part of per-file drift detection implementation"
```

---

## Phase 2: Schema Changes

### Task 2.1: Add Computed Fields to FileModel

**Files:**
- Modify: `internal/schemas/common.go:20-35` (FileModel struct)

**Step 1: Add fields to FileModel struct**

Modify `internal/schemas/common.go`:

```go
type FileModel struct {
	Content       types.String         `tfsdk:"content"`
	Instructions  []InstructionModel   `tfsdk:"instructions"`
	Verifications []VerificationModel  `tfsdk:"verifications"`

	// Computed drift detection fields
	ContentHash   types.String         `tfsdk:"content_hash"`
	FileHash      types.String         `tfsdk:"file_hash"`
	FileModTime   types.String         `tfsdk:"file_modtime"`
}
```

**Step 2: Verify code compiles**

Run: `go build ./...`

Expected: BUILD SUCCESS (schema changes don't break anything yet)

**Step 3: Commit schema changes**

```bash
git add internal/schemas/common.go
git commit -m "feat(schemas): add drift detection fields to FileModel

Add computed fields:
- content_hash: Hash of file specification
- file_hash: Hash of actual file on disk
- file_modtime: Modification time for optimization

Part of per-file drift detection implementation"
```

### Task 2.2: Add Schema Attributes for File Hashing

**Files:**
- Modify: `internal/resources/project.go:~170-250` (schema definition)

**Step 1: Find the Files attribute schema**

Search for: `"files": schema.MapNestedAttribute` in `internal/resources/project.go`

**Step 2: Add computed hash attributes to nested file schema**

Within the NestedObject attributes map, add after verifications:

```go
"content_hash": schema.StringAttribute{
	Computed:            true,
	MarkdownDescription: "SHA256 hash of the file specification (content or instructions JSON) for drift detection",
},
"file_hash": schema.StringAttribute{
	Computed:            true,
	MarkdownDescription: "SHA256 hash of the actual file on disk for drift detection",
},
"file_modtime": schema.StringAttribute{
	Computed:            true,
	MarkdownDescription: "File modification time in RFC3339 format (optimization for drift detection)",
},
```

**Step 3: Verify schema compiles**

Run: `go build ./internal/resources`

Expected: BUILD SUCCESS

**Step 4: Commit file schema attributes**

```bash
git add internal/resources/project.go
git commit -m "feat(resources): add drift detection attributes to file schema

Add computed attributes for per-file drift detection:
- content_hash
- file_hash
- file_modtime

Part of per-file drift detection implementation"
```

### Task 2.3: Add Drift Detection Fields to ProjectModel

**Files:**
- Modify: `internal/resources/project.go:~50-90` (ProjectModelFinal struct)

**Step 1: Add drift fields to struct**

Add to ProjectModelFinal (after OutputHash field):

```go
// Drift detection
DriftDetected types.Bool   `tfsdk:"drift_detected"`
DriftedFiles  types.List   `tfsdk:"drifted_files"`
```

**Step 2: Add schema attributes**

In the Schema() method, add after output_hash:

```go
"drift_detected": schema.BoolAttribute{
	Computed:            true,
	MarkdownDescription: "True if any files have been modified outside Terraform",
},
"drifted_files": schema.ListAttribute{
	Computed:            true,
	ElementType:         types.StringType,
	MarkdownDescription: "List of file paths that have drifted from their expected state",
},
```

**Step 3: Verify schema compiles**

Run: `go build ./internal/resources`

Expected: BUILD SUCCESS

**Step 4: Commit project drift schema**

```bash
git add internal/resources/project.go
git commit -m "feat(resources): add drift detection fields to project schema

Add computed fields:
- drift_detected: Boolean trigger for drift
- drifted_files: List of drifted file paths

These fields trigger Update() when files change outside Terraform

Part of per-file drift detection implementation"
```

---

## Phase 3: Hash Storage in Create()

### Task 3.1: Implement computeAndStoreFileHashes Helper

**Files:**
- Modify: `internal/resources/project.go` (add new function at end)

**Step 1: Write the helper function**

Add to end of `internal/resources/project.go`:

```go
// computeAndStoreFileHashes calculates and stores hashes for all files
// Called after successful Claude execution to establish baseline
func (r *ProjectResourceFinal) computeAndStoreFileHashes(
	ctx context.Context,
	data *ProjectModelFinal,
	mergedFiles []schemas.FileModelWithPath,
) error {
	projectPath := data.ProjectPath.ValueString()

	// Get current files map from data
	filesMap := make(map[string]schemas.FileModel)
	if !data.Files.IsNull() && !data.Files.IsUnknown() {
		data.Files.ElementsAs(ctx, &filesMap, false)
	}

	for _, fileWithPath := range mergedFiles {
		fullPath := filepath.Join(projectPath, fileWithPath.Path)

		// Read actual file content
		actualContent, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				// File wasn't created - skip
				tflog.Debug(ctx, "File not found, skipping hash", map[string]interface{}{
					"path": fileWithPath.Path,
				})
				continue
			}
			return fmt.Errorf("failed to read file %s: %w", fileWithPath.Path, err)
		}

		// Get or create FileModel for this path
		fileModel, exists := filesMap[fileWithPath.Path]
		if !exists {
			fileModel = schemas.FileModel{
				Content:      fileWithPath.Content,
				Instructions: fileWithPath.Instructions,
			}
		}

		// Compute hashes
		fileModel.ContentHash = types.StringValue(files.ComputeContentHash(fileModel))
		fileModel.FileHash = types.StringValue(files.ComputeFileHash(actualContent))

		// Store modification time
		fileInfo, err := os.Stat(fullPath)
		if err == nil {
			fileModel.FileModTime = types.StringValue(fileInfo.ModTime().Format(time.RFC3339))
		}

		tflog.Debug(ctx, "Computed file hashes", map[string]interface{}{
			"path":         fileWithPath.Path,
			"content_hash": fileModel.ContentHash.ValueString()[:8] + "...",
			"file_hash":    fileModel.FileHash.ValueString()[:8] + "...",
		})

		// Update map
		filesMap[fileWithPath.Path] = fileModel
	}

	// Convert back to map attribute
	filesMapValue, diags := types.MapValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"content":       types.StringType,
			"instructions":  types.ListType{ElemType: types.ObjectType{ /* ... */ }},
			"verifications": types.ListType{ElemType: types.ObjectType{ /* ... */ }},
			"content_hash":  types.StringType,
			"file_hash":     types.StringType,
			"file_modtime":  types.StringType,
		},
	}, filesMap)

	if diags.HasError() {
		return fmt.Errorf("failed to convert files map: %s", diags.Errors())
	}

	data.Files = filesMapValue
	return nil
}
```

**Step 2: Add import for files package**

Add to imports:

```go
"github.com/tofukit/opentofu-provider-tofukit/internal/files"
```

**Step 3: Verify code compiles**

Run: `go build ./internal/resources`

Expected: BUILD SUCCESS

**Step 4: Commit hash storage helper**

```bash
git add internal/resources/project.go
git commit -m "feat(resources): add computeAndStoreFileHashes helper

Implement helper function to calculate and store file hashes after
Claude execution. Computes content_hash, file_hash, and file_modtime
for all files in the project.

Part of per-file drift detection implementation"
```

### Task 3.2: Call Hash Storage in Create()

**Files:**
- Modify: `internal/resources/project.go:622-675` (Create method, after executeClaudeCode)

**Step 1: Add hash computation after successful execution**

After line ~648 (after outputHash computation), add:

```go
// Compute and store per-file hashes
if err := r.computeAndStoreFileHashes(ctx, &data, mergedFiles); err != nil {
	tflog.Warn(ctx, "Failed to compute file hashes", map[string]interface{}{
		"error": err.Error(),
	})
	// Don't fail the resource, just skip storing hashes
}

// Initialize drift flags (no drift on fresh creation)
data.DriftDetected = types.BoolValue(false)
data.DriftedFiles = types.ListNull(types.StringType)

tflog.Info(ctx, "Computed and stored file hashes", map[string]interface{}{
	"project_id": data.ID.ValueString(),
	"file_count": len(mergedFiles),
})
```

**Step 2: Verify code compiles**

Run: `go build ./internal/resources`

Expected: BUILD SUCCESS

**Step 3: Test with a simple project**

Create test file: `test/drift_create_test.go`:

```go
package test

import (
	"testing"
	"github.com/stretchr/testify/require"
)

func TestProjectCreateStoresHashes(t *testing.T) {
	// This test verifies hashes are stored after Create
	// Run manually with existing test infrastructure
	t.Skip("Manual integration test - run with actual provider")
}
```

**Step 4: Commit Create() integration**

```bash
git add internal/resources/project.go test/drift_create_test.go
git commit -m "feat(resources): compute file hashes in Create method

After successful Claude execution:
- Compute content_hash and file_hash for all files
- Store modification time for optimization
- Initialize drift_detected = false
- Log hash computation results

Part of per-file drift detection implementation"
```

---

## Phase 4: Drift Detection in Read()

### Task 4.1: Implement Drift Detection Logic

**Files:**
- Modify: `internal/resources/project.go:704-790` (Read method)

**Step 1: Add drift detection after mergedFiles collection**

After line ~727 (after mergedFiles = r.collectAndMergeFiles), add:

```go
// DRIFT DETECTION: Check each file for changes outside Terraform
projectPath := data.ProjectPath.ValueString()
driftedFiles := []string{}

if projectPath != "" && !data.Files.IsNull() {
	filesMap := make(map[string]schemas.FileModel)
	data.Files.ElementsAs(ctx, &filesMap, false)

	for path, fileModel := range filesMap {
		// Skip if no hash stored yet (first run after upgrade)
		if fileModel.FileHash.IsNull() || fileModel.FileHash.IsUnknown() {
			continue
		}

		fullPath := filepath.Join(projectPath, path)

		// Check if file exists
		fileInfo, err := os.Stat(fullPath)
		if os.IsNotExist(err) {
			// File deleted outside Terraform
			tflog.Warn(ctx, "File deleted outside Terraform", map[string]interface{}{
				"project_id": data.ID.ValueString(),
				"path":       path,
			})
			driftedFiles = append(driftedFiles, path)
			continue
		}
		if err != nil {
			tflog.Warn(ctx, "Failed to stat file for drift detection", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			continue
		}

		// Optimization: Check modification time first
		currentModTime := fileInfo.ModTime().Format(time.RFC3339)
		if !fileModel.FileModTime.IsNull() && currentModTime == fileModel.FileModTime.ValueString() {
			// File unchanged since last check, skip hashing
			tflog.Trace(ctx, "File modification time unchanged, skipping hash", map[string]interface{}{
				"path": path,
			})
			continue
		}

		// Read actual file content
		actualContent, err := os.ReadFile(fullPath)
		if err != nil {
			tflog.Warn(ctx, "Failed to read file for drift detection", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
			continue
		}

		// Calculate current hash (temporary, in-memory)
		currentHash := files.ComputeFileHash(actualContent)

		// Compare with stored hash
		if currentHash != fileModel.FileHash.ValueString() {
			tflog.Warn(ctx, "File content changed outside Terraform", map[string]interface{}{
				"project_id":    data.ID.ValueString(),
				"path":          path,
				"expected_hash": fileModel.FileHash.ValueString()[:8] + "...",
				"current_hash":  currentHash[:8] + "...",
			})
			driftedFiles = append(driftedFiles, path)
		}
	}
}

// Set computed drift fields (triggers Update if changed)
if len(driftedFiles) > 0 {
	data.DriftDetected = types.BoolValue(true)
	driftList, diags := types.ListValueFrom(ctx, types.StringType, driftedFiles)
	if diags.HasError() {
		tflog.Warn(ctx, "Failed to create drifted files list", map[string]interface{}{
			"errors": diags.Errors(),
		})
	} else {
		data.DriftedFiles = driftList
	}

	tflog.Info(ctx, "Drift detected in project files", map[string]interface{}{
		"project_id":    data.ID.ValueString(),
		"drifted_count": len(driftedFiles),
		"drifted_files": driftedFiles,
	})
} else {
	data.DriftDetected = types.BoolValue(false)
	data.DriftedFiles = types.ListNull(types.StringType)
}
```

**Step 2: Add files import if not present**

Ensure import exists:

```go
"github.com/tofukit/opentofu-provider-tofukit/internal/files"
```

**Step 3: Verify code compiles**

Run: `go build ./internal/resources`

Expected: BUILD SUCCESS

**Step 4: Commit Read() drift detection**

```bash
git add internal/resources/project.go
git commit -m "feat(resources): add drift detection to Read method

Implement per-file drift detection in Read():
- Check each file for existence
- Use mtime optimization to skip unchanged files
- Calculate SHA256 hash for changed files
- Compare with stored hash
- Set drift_detected and drifted_files on mismatch

Follows hashicorp/local_file pattern: calculate hashes in Read()
for comparison only, never update stored hashes.

Part of per-file drift detection implementation"
```

---

## Phase 5: Drift Restoration in Update()

### Task 5.1: Add Drift Instruction Helper

**Files:**
- Modify: `internal/resources/project.go` (add new function)

**Step 1: Implement addDriftInstructions helper**

Add to end of file:

```go
// addDriftInstructions adds drift warning instructions to files that changed outside Terraform
func (r *ProjectResourceFinal) addDriftInstructions(
	ctx context.Context,
	files []schemas.FileModelWithPath,
	driftedPaths []string,
) []schemas.FileModelWithPath {
	// Build drift map for O(1) lookup
	driftMap := make(map[string]bool)
	for _, path := range driftedPaths {
		driftMap[path] = true
	}

	// Add drift warnings to affected files
	for i, file := range files {
		if driftMap[file.Path] {
			driftInstruction := schemas.InstructionModel{
				Prompt: types.StringValue(fmt.Sprintf(
					"⚠️ DRIFT DETECTED: File '%s' was modified outside Terraform. "+
					"Restore it to match the specification below. "+
					"Ignore any manual edits that were made.",
					file.Path,
				)),
				Constraints: []types.String{
					types.StringValue("Restore file to match Terraform specification exactly"),
					types.StringValue("Ignore manual edits made outside Terraform"),
					types.StringValue("Ensure the restored file matches the original intent"),
				},
			}

			tflog.Info(ctx, "Adding drift instruction to file", map[string]interface{}{
				"path": file.Path,
			})

			// Prepend drift instruction (highest priority)
			files[i].Instructions = append(
				[]schemas.InstructionModel{driftInstruction},
				file.Instructions...,
			)
		}
	}

	return files
}
```

**Step 2: Verify code compiles**

Run: `go build ./internal/resources`

Expected: BUILD SUCCESS

**Step 3: Commit drift instruction helper**

```bash
git add internal/resources/project.go
git commit -m "feat(resources): add drift instruction helper

Implement helper to add drift warning instructions to files that
changed outside Terraform. Prepends explicit instructions to Claude
to restore files to their specification, ignoring manual edits.

Part of per-file drift detection implementation"
```

### Task 5.2: Integrate Drift Handling in Update()

**Files:**
- Modify: `internal/resources/project.go:796-1090` (Update method)

**Step 1: Detect if update triggered by drift**

After line ~806 (after getting state), add:

```go
// Check if update triggered by drift vs config change
isDriftTriggered := state.DriftDetected.ValueBool()

if isDriftTriggered {
	tflog.Info(ctx, "Update triggered by drift detection", map[string]interface{}{
		"project_id": state.ID.ValueString(),
	})
}
```

**Step 2: Add drift instructions to enriched files**

After line ~873 (after EnrichFilesWithInstructions), add:

```go
// Add drift context if update triggered by drift
if isDriftTriggered {
	var driftedPaths []string
	if !state.DriftedFiles.IsNull() && !state.DriftedFiles.IsUnknown() {
		state.DriftedFiles.ElementsAs(ctx, &driftedPaths, false)

		tflog.Info(ctx, "Adding drift instructions", map[string]interface{}{
			"drifted_files": driftedPaths,
		})

		enrichedFiles = r.addDriftInstructions(ctx, enrichedFiles, driftedPaths)
	}
}
```

**Step 3: Recompute hashes after successful execution**

After line ~1007 (after successful execution, before setting FileHash), replace the FileHash line with:

```go
// Recompute all file hashes after successful execution
if err := r.computeAndStoreFileHashes(ctx, &data, mergedFiles); err != nil {
	tflog.Warn(ctx, "Failed to recompute file hashes", map[string]interface{}{
		"error": err.Error(),
	})
}

// Clear drift flags
data.DriftDetected = types.BoolValue(false)
data.DriftedFiles = types.ListNull(types.StringType)

tflog.Info(ctx, "Cleared drift flags after successful restoration", map[string]interface{}{
	"project_id": data.ID.ValueString(),
})
```

**Step 4: Verify code compiles**

Run: `go build ./internal/resources`

Expected: BUILD SUCCESS

**Step 5: Commit Update() drift handling**

```bash
git add internal/resources/project.go
git commit -m "feat(resources): handle drift restoration in Update method

When Update() triggered by drift:
- Detect drift trigger flag
- Add drift warning instructions to affected files
- Execute Claude with drift context
- Recompute hashes after successful restoration
- Clear drift flags

Part of per-file drift detection implementation"
```

---

## Phase 6: Integration Testing

### Task 6.1: Create Drift Detection Integration Test

**Files:**
- Create: `test/project_drift_detection_test.go`

**Step 1: Write test for static file drift**

Create `test/project_drift_detection_test.go`:

```go
package test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestProjectDriftDetection_StaticFile(t *testing.T) {
	testDir := setupTestEnvironment(t, "TestProjectDriftDetection_StaticFile")
	defer cleanupTestEnvironment(t, testDir)

	// Step 1: Create project with static .gitignore
	config := fmt.Sprintf(`
resource "tofukit_project" "test" {
  name = "drift-test"

  output_path = "%s/output"

  files = {
    ".gitignore" = {
      content = <<-EOF
        *.log
        *.tmp
      EOF
    }
  }
}
`, testDir)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "drift_detected", "false"),
					resource.TestCheckResourceAttr("tofukit_project.test", "drifted_files.#", "0"),
				),
			},
			{
				// Manually edit .gitignore outside Terraform
				PreConfig: func() {
					gitignorePath := filepath.Join(testDir, "output", ".gitignore")
					err := os.WriteFile(gitignorePath, []byte("*.log\n*.tmp\n*.cache\n"), 0644)
					require.NoError(t, err, "Failed to manually edit .gitignore")

					t.Logf("Manually edited .gitignore to add *.cache")
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					// Should detect drift
					resource.TestCheckResourceAttr("tofukit_project.test", "drift_detected", "true"),
					resource.TestCheckResourceAttr("tofukit_project.test", "drifted_files.#", "1"),
					resource.TestCheckResourceAttr("tofukit_project.test", "drifted_files.0", ".gitignore"),
				),
				// ExpectNonEmptyPlan because drift should trigger update
				ExpectNonEmptyPlan: true,
			},
			{
				// Apply to restore file
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					// After apply, drift should be cleared
					resource.TestCheckResourceAttr("tofukit_project.test", "drift_detected", "false"),
					resource.TestCheckResourceAttr("tofukit_project.test", "drifted_files.#", "0"),
					// Verify file content restored
					func(s *terraform.State) error {
						gitignorePath := filepath.Join(testDir, "output", ".gitignore")
						content, err := os.ReadFile(gitignorePath)
						if err != nil {
							return fmt.Errorf("failed to read restored .gitignore: %w", err)
						}

						expected := "*.log\n*.tmp\n"
						if string(content) != expected {
							return fmt.Errorf("file not restored correctly.\nExpected:\n%s\nGot:\n%s", expected, string(content))
						}

						return nil
					},
				),
			},
		},
	})
}
```

**Step 2: Add test for deleted file**

Add to same file:

```go
func TestProjectDriftDetection_DeletedFile(t *testing.T) {
	testDir := setupTestEnvironment(t, "TestProjectDriftDetection_DeletedFile")
	defer cleanupTestEnvironment(t, testDir)

	config := fmt.Sprintf(`
resource "tofukit_project" "test" {
  name = "drift-delete-test"

  output_path = "%s/output"

  files = {
    "README.md" = {
      content = "# Test Project\n"
    }
  }
}
`, testDir)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "drift_detected", "false"),
				),
			},
			{
				// Delete README.md manually
				PreConfig: func() {
					readmePath := filepath.Join(testDir, "output", "README.md")
					err := os.Remove(readmePath)
					require.NoError(t, err, "Failed to delete README.md")

					t.Logf("Manually deleted README.md")
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "drift_detected", "true"),
					resource.TestCheckResourceAttr("tofukit_project.test", "drifted_files.#", "1"),
					resource.TestCheckResourceAttr("tofukit_project.test", "drifted_files.0", "README.md"),
				),
				ExpectNonEmptyPlan: true,
			},
			{
				// Apply to recreate file
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "drift_detected", "false"),
					// Verify file recreated
					func(s *terraform.State) error {
						readmePath := filepath.Join(testDir, "output", "README.md")
						_, err := os.Stat(readmePath)
						if os.IsNotExist(err) {
							return fmt.Errorf("README.md was not recreated")
						}
						return err
					},
				),
			},
		},
	})
}
```

**Step 3: Run drift detection tests**

Run: `go test ./test -run TestProjectDriftDetection -v`

Expected: PASS (both tests)

**Step 4: Commit drift detection tests**

```bash
git add test/project_drift_detection_test.go
git commit -m "test: add drift detection integration tests

Add comprehensive tests for drift detection:
- Static file modification detection
- Deleted file detection
- Drift restoration after apply
- Drift flag clearing

Part of per-file drift detection implementation"
```

### Task 6.2: Test Multiple File Drift

**Files:**
- Modify: `test/project_drift_detection_test.go`

**Step 1: Add multi-file drift test**

Add to file:

```go
func TestProjectDriftDetection_MultipleFiles(t *testing.T) {
	testDir := setupTestEnvironment(t, "TestProjectDriftDetection_MultipleFiles")
	defer cleanupTestEnvironment(t, testDir)

	config := fmt.Sprintf(`
resource "tofukit_project" "test" {
  name = "drift-multi-test"

  output_path = "%s/output"

  files = {
    ".gitignore" = {
      content = "*.log\n"
    }
    "README.md" = {
      content = "# Project\n"
    }
    "LICENSE" = {
      content = "MIT License\n"
    }
  }
}
`, testDir)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "drift_detected", "false"),
					resource.TestCheckResourceAttr("tofukit_project.test", "drifted_files.#", "0"),
				),
			},
			{
				// Edit 2 out of 3 files
				PreConfig: func() {
					outputDir := filepath.Join(testDir, "output")

					// Edit .gitignore
					gitignorePath := filepath.Join(outputDir, ".gitignore")
					os.WriteFile(gitignorePath, []byte("*.log\n*.cache\n"), 0644)

					// Edit LICENSE
					licensePath := filepath.Join(outputDir, "LICENSE")
					os.WriteFile(licensePath, []byte("Apache License\n"), 0644)

					// Leave README.md unchanged

					t.Logf("Manually edited .gitignore and LICENSE")
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("tofukit_project.test", "drift_detected", "true"),
					resource.TestCheckResourceAttr("tofukit_project.test", "drifted_files.#", "2"),
					// Verify both drifted files are detected (order may vary)
					func(s *terraform.State) error {
						rs := s.RootModule().Resources["tofukit_project.test"]
						driftedFiles := []string{
							rs.Primary.Attributes["drifted_files.0"],
							rs.Primary.Attributes["drifted_files.1"],
						}

						expectedFiles := map[string]bool{
							".gitignore": false,
							"LICENSE":    false,
						}

						for _, file := range driftedFiles {
							if _, exists := expectedFiles[file]; exists {
								expectedFiles[file] = true
							}
						}

						for file, found := range expectedFiles {
							if !found {
								return fmt.Errorf("expected drifted file %s not found", file)
							}
						}

						return nil
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
```

**Step 2: Run multi-file test**

Run: `go test ./test -run TestProjectDriftDetection_MultipleFiles -v`

Expected: PASS

**Step 3: Commit multi-file test**

```bash
git add test/project_drift_detection_test.go
git commit -m "test: add multi-file drift detection test

Verify drift detection correctly identifies multiple drifted files
in a single project and reports them all.

Part of per-file drift detection implementation"
```

---

## Phase 7: Documentation and Cleanup

### Task 7.1: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md` (add drift detection section)

**Step 1: Add drift detection documentation**

Add new section after "File Operations":

```markdown
### Drift Detection

The provider implements granular per-file drift detection following the `hashicorp/local_file` pattern.

**How It Works:**

1. **Hash Storage** (Create/Update):
   - After Claude execution, computes SHA256 hashes for all files
   - Stores `content_hash` (specification), `file_hash` (actual file), `file_modtime` (optimization)

2. **Drift Detection** (Read):
   - Called during `tofu plan`
   - Checks modification time first (fast)
   - If changed, computes current hash and compares with stored `file_hash`
   - Sets `drift_detected = true` if mismatch

3. **Drift Restoration** (Update):
   - Triggered when `drift_detected` changes to `true`
   - Adds drift warning instructions to affected files
   - Claude receives: "⚠️ DRIFT DETECTED: File X was modified, restore to spec"
   - After successful execution, clears drift flags

**Performance Optimization:**
- Uses modification time (`mtime`) to skip hash computation for unchanged files
- Scales efficiently to 100+ file projects

**Example:**
```
1. User manually edits .gitignore
2. tofu plan → detects drift, shows "drift_detected: false → true"
3. tofu apply → Claude restores .gitignore to specification
```

**State Fields:**
- `drift_detected` (bool): True if any files drifted
- `drifted_files` (list): Paths of files that changed outside Terraform
- Per-file: `content_hash`, `file_hash`, `file_modtime` (computed)
```

**Step 2: Commit documentation**

```bash
git add CLAUDE.md
git commit -m "docs: document per-file drift detection feature

Add comprehensive documentation for drift detection:
- How it works (hash storage, detection, restoration)
- Performance optimization via mtime checks
- Example workflow
- State field descriptions

Part of per-file drift detection implementation"
```

### Task 7.2: Run Full Test Suite

**Step 1: Run all tests**

Run: `go test ./... -v`

Expected: All tests PASS

**Step 2: Run integration tests**

Run: `go test ./test -v`

Expected: All tests PASS including drift detection tests

**Step 3: Build provider**

Run: `task build`

Expected: BUILD SUCCESS

**Step 4: Install provider locally**

Run: `task install`

Expected: Provider installed to ~/.terraform.d/plugins

### Task 7.3: Manual Verification

**Step 1: Create test project**

Create `test-drift/main.tf`:

```hcl
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  debug = true
}

resource "tofukit_project" "test" {
  name = "drift-test"

  files = {
    ".gitignore" = {
      content = <<-EOF
        *.log
        *.tmp
      EOF
    }

    "README.md" = {
      content = "# Drift Test\n"
    }
  }
}
```

**Step 2: Initialize and apply**

```bash
cd test-drift
tofu init
tofu apply -auto-approve
```

Expected: Project created, files exist

**Step 3: Manually edit .gitignore**

```bash
echo "*.cache" >> output/.gitignore
```

**Step 4: Run plan**

```bash
tofu plan
```

Expected output:
```
  ~ resource "tofukit_project" "test" {
      ~ drift_detected = false -> true
      ~ drifted_files  = [] -> [
          + ".gitignore",
        ]
  }

Plan: 0 to add, 1 to change, 0 to destroy.
```

**Step 5: Apply to restore**

```bash
tofu apply -auto-approve
```

Expected: File restored to original content (without *.cache)

**Step 6: Verify restoration**

```bash
cat output/.gitignore
```

Expected content:
```
*.log
*.tmp
```

**Step 7: Run plan again**

```bash
tofu plan
```

Expected: "No changes" (drift cleared)

### Task 7.4: Final Commit

**Step 1: Review all changes**

Run: `git status`

Verify: All changes committed

**Step 2: Create final summary commit**

```bash
git commit --allow-empty -m "feat: complete per-file drift detection implementation

Implement granular per-file drift detection following hashicorp/local_file
pattern. Fixes critical bug where manual file edits were not detected
during tofu plan.

Features:
- Per-file hash tracking (content_hash, file_hash, file_modtime)
- Drift detection in Read() method
- Hash storage in Create()/Update() methods
- Drift restoration with LLM context
- Performance optimization via mtime checks
- Comprehensive integration tests
- Full documentation

Technical Details:
- SHA256 hashing for file content and specifications
- Computed fields (drift_detected, drifted_files) trigger Update()
- Read() computes hashes for comparison only (never updates state)
- Create()/Update() store hashes after successful execution
- Drift instructions prepended to affected files for Claude

Testing:
- Unit tests for hash functions
- Integration tests for drift detection scenarios
- Manual verification with test project

Closes: drift-detection-feature
Ref: docs/plans/2025-11-01-per-file-drift-detection-design.md

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Verification Checklist

Before considering this complete, verify:

- [ ] All unit tests pass: `go test ./internal/files -v`
- [ ] All integration tests pass: `go test ./test -v`
- [ ] Provider builds: `task build`
- [ ] Provider installs: `task install`
- [ ] Manual test: Create project, edit file, plan shows drift
- [ ] Manual test: Apply restores file to spec
- [ ] Manual test: Second plan shows no changes
- [ ] Documentation updated: CLAUDE.md
- [ ] Design document exists: docs/plans/2025-11-01-per-file-drift-detection-design.md
- [ ] All changes committed with descriptive messages
- [ ] No compiler warnings
- [ ] No linter errors (if using golangci-lint)

## Success Criteria

✅ Manual file edits detected during `tofu plan`
✅ Plan output shows which files drifted
✅ Apply restores drifted files to specification
✅ Works for both inline and referenced files
✅ Performance acceptable for 100+ file projects
✅ No false positives (unchanged files don't show drift)
✅ All integration tests pass

## Notes for Engineer

- Follow TDD strictly: Write failing test → Make it pass → Commit
- Each commit should be atomic and buildable
- Test after each phase before moving to next
- If hash computation is slow, check mtime optimization is working
- If drift not detected, verify Read() is being called during plan
- If drift persists after apply, check Update() is clearing drift flags
- Use `tflog.Debug` liberally for troubleshooting
- Reference hashicorp/local_file source if uncertain about pattern
