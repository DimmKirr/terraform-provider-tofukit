# Implicit File Generation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable natural language multi-file generation where Claude decides file structure based on requirements without explicit file definitions.

**Architecture:** Dual-mode system - explicit mode (files defined, current behavior) and implicit mode (no files defined, Claude decides structure). In implicit mode, Claude outputs a JSON manifest declaring which files it created, provider validates manifest against filesystem, and tracks discovered files in state for drift detection.

**Tech Stack:** Go, Terraform Plugin Framework, JSON parsing, filesystem operations, SHA256 hashing

**Design Reference:** See `docs/plans/2025-11-14-implicit-file-generation-design.md` for complete architecture.

---

## Prerequisites

**Required Knowledge:**
- Go programming
- Terraform Plugin Framework basics
- JSON marshaling/unmarshaling
- Regular expressions
- File I/O operations

**Files You'll Need to Reference:**
- `internal/resources/project.go` - Main project resource (where mode detection happens)
- `internal/llm/claude/prompt_types.go` - Prompt construction
- `internal/llm/claude/executor.go` - Claude execution
- `internal/files/operations.go` - File operation logic (explicit mode reference)
- `test/resource_project_inline_file_test.go` - Existing test patterns

**Testing Pattern:**
This codebase uses table-driven tests with `testify`. Example:
```go
func TestSomething(t *testing.T) {
    require := require.New(t)
    assert := assert.New(t)

    result := doSomething()
    require.NoError(err)
    assert.Equal(expected, result)
}
```

**Commit Pattern:**
Frequent small commits following conventional commits:
- `test: add test for X`
- `feat: implement X`
- `refactor: extract Y`

---

## Phase 1: Core Manifest Support (Foundation)

### Task 1.1: Create Manifest Parser Skeleton

**Files:**
- Create: `internal/llm/claude/manifest_parser.go`
- Create: `internal/llm/claude/manifest_parser_test.go`

**Step 1: Write failing test for valid manifest extraction**

File: `internal/llm/claude/manifest_parser_test.go`

```go
package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractManifest_ValidJSON(t *testing.T) {
	response := `Some text before

` + "```json" + `
{
  "generated_files": ["app.py", "config.py"]
}
` + "```" + `

Some text after`

	manifest, err := ExtractManifest(response)
	require.NoError(t, err)
	assert.Equal(t, []string{"app.py", "config.py"}, manifest.GeneratedFiles)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/llm/claude -v -run TestExtractManifest_ValidJSON`

Expected: FAIL with "undefined: ExtractManifest"

**Step 3: Create minimal implementation**

File: `internal/llm/claude/manifest_parser.go`

```go
package claude

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// FileGenerationManifest represents Claude's declaration of generated files
type FileGenerationManifest struct {
	GeneratedFiles []string `json:"generated_files"`
}

// ExtractManifest parses file generation manifest from Claude's response
func ExtractManifest(response string) (*FileGenerationManifest, error) {
	// Pattern to match JSON in markdown code blocks
	jsonPattern := regexp.MustCompile("```json\\s*({[^`]+})\\s*```")
	matches := jsonPattern.FindStringSubmatch(response)

	if len(matches) < 2 {
		return nil, fmt.Errorf("no manifest found in response")
	}

	var manifest FileGenerationManifest
	if err := json.Unmarshal([]byte(matches[1]), &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}

	return &manifest, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/llm/claude -v -run TestExtractManifest_ValidJSON`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/llm/claude/manifest_parser.go internal/llm/claude/manifest_parser_test.go
git commit -m "test: add manifest extraction test

Add test for extracting file manifest from Claude response"
```

```bash
git add internal/llm/claude/manifest_parser.go
git commit -m "feat: implement basic manifest extraction

Extract JSON manifest from markdown code blocks in Claude responses"
```

---

### Task 1.2: Add Manifest Validation Tests

**Files:**
- Modify: `internal/llm/claude/manifest_parser_test.go`

**Step 1: Write test for missing manifest**

Add to `manifest_parser_test.go`:

```go
func TestExtractManifest_NoManifest(t *testing.T) {
	response := "Just regular text with no manifest"

	manifest, err := ExtractManifest(response)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no manifest found")
}
```

**Step 2: Run test**

Run: `go test ./internal/llm/claude -v -run TestExtractManifest_NoManifest`

Expected: PASS (already handled by implementation)

**Step 3: Write test for invalid JSON**

Add to `manifest_parser_test.go`:

```go
func TestExtractManifest_InvalidJSON(t *testing.T) {
	response := "```json\n{broken json}\n```"

	manifest, err := ExtractManifest(response)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid manifest JSON")
}
```

**Step 4: Run test**

Run: `go test ./internal/llm/claude -v -run TestExtractManifest_InvalidJSON`

Expected: PASS (already handled)

**Step 5: Write test for empty file list**

Add to `manifest_parser_test.go`:

```go
func TestExtractManifest_EmptyFileList(t *testing.T) {
	response := "```json\n{\"generated_files\": []}\n```"

	manifest, err := ExtractManifest(response)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "zero files")
}
```

**Step 6: Run test to verify it fails**

Run: `go test ./internal/llm/claude -v -run TestExtractManifest_EmptyFileList`

Expected: FAIL (not yet implemented)

**Step 7: Add validation to ExtractManifest**

Modify `ExtractManifest()` in `manifest_parser.go`:

```go
func ExtractManifest(response string) (*FileGenerationManifest, error) {
	// ... existing code ...

	if err := json.Unmarshal([]byte(matches[1]), &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}

	// Validate manifest has files
	if len(manifest.GeneratedFiles) == 0 {
		return nil, fmt.Errorf("manifest contains zero files - at least one file must be declared")
	}

	return &manifest, nil
}
```

**Step 8: Run test to verify it passes**

Run: `go test ./internal/llm/claude -v -run TestExtractManifest_EmptyFileList`

Expected: PASS

**Step 9: Commit**

```bash
git add internal/llm/claude/manifest_parser_test.go
git commit -m "test: add manifest validation tests

Test missing manifest, invalid JSON, and empty file list"
```

```bash
git add internal/llm/claude/manifest_parser.go
git commit -m "feat: validate manifest has files

Reject manifests with zero files"
```

---

### Task 1.3: Add Path Validation

**Files:**
- Modify: `internal/llm/claude/manifest_parser.go`
- Modify: `internal/llm/claude/manifest_parser_test.go`

**Step 1: Write test for path validation**

Add to `manifest_parser_test.go`:

```go
func TestValidateManifestPaths_AbsolutePath(t *testing.T) {
	manifest := &FileGenerationManifest{
		GeneratedFiles: []string{"/absolute/path.txt"},
	}

	err := ValidateManifestPaths(manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "absolute path")
}

func TestValidateManifestPaths_PathTraversal(t *testing.T) {
	manifest := &FileGenerationManifest{
		GeneratedFiles: []string{"../../../etc/passwd"},
	}

	err := ValidateManifestPaths(manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal")
}

func TestValidateManifestPaths_Valid(t *testing.T) {
	manifest := &FileGenerationManifest{
		GeneratedFiles: []string{"app.py", "src/config.py"},
	}

	err := ValidateManifestPaths(manifest)
	assert.NoError(t, err)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/llm/claude -v -run TestValidateManifestPaths`

Expected: FAIL with "undefined: ValidateManifestPaths"

**Step 3: Implement path validation**

Add to `manifest_parser.go`:

```go
import (
	"strings"
	// ... existing imports
)

// ValidateManifestPaths checks for invalid path patterns
func ValidateManifestPaths(manifest *FileGenerationManifest) error {
	for _, path := range manifest.GeneratedFiles {
		// Check for absolute paths
		if strings.HasPrefix(path, "/") || strings.Contains(path, ":") {
			return fmt.Errorf("manifest contains absolute path '%s' - only relative paths allowed", path)
		}

		// Check for path traversal
		if strings.Contains(path, "..") {
			return fmt.Errorf("manifest contains path traversal '%s' - paths must stay within project directory", path)
		}

		// Check for empty path
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("manifest contains empty path")
		}
	}

	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/llm/claude -v -run TestValidateManifestPaths`

Expected: PASS

**Step 5: Add path normalization to ExtractManifest**

Modify `ExtractManifest()` to normalize paths:

```go
func ExtractManifest(response string) (*FileGenerationManifest, error) {
	// ... existing code up to unmarshal ...

	if len(manifest.GeneratedFiles) == 0 {
		return nil, fmt.Errorf("manifest contains zero files - at least one file must be declared")
	}

	// Normalize paths (convert backslashes to forward slashes)
	for i, path := range manifest.GeneratedFiles {
		manifest.GeneratedFiles[i] = strings.ReplaceAll(path, "\\", "/")
	}

	return &manifest, nil
}
```

**Step 6: Commit**

```bash
git add internal/llm/claude/manifest_parser_test.go
git commit -m "test: add path validation tests

Test absolute paths, path traversal, and valid paths"
```

```bash
git add internal/llm/claude/manifest_parser.go
git commit -m "feat: add manifest path validation

Validate paths are relative and don't traverse outside project.
Normalize backslashes to forward slashes."
```

---

### Task 1.4: Add FileGenerationMode Type

**Files:**
- Modify: `internal/llm/claude/prompt_types.go`

**Step 1: Add FileGenerationMode enum**

Add to top of `prompt_types.go`:

```go
// FileGenerationMode determines how files are managed
type FileGenerationMode int

const (
	// ExplicitMode means user defined files explicitly
	ExplicitMode FileGenerationMode = iota
	// ImplicitMode means Claude decides file structure
	ImplicitMode
)

// String returns string representation of mode
func (m FileGenerationMode) String() string {
	switch m {
	case ExplicitMode:
		return "explicit"
	case ImplicitMode:
		return "implicit"
	default:
		return "unknown"
	}
}
```

**Step 2: Commit**

```bash
git add internal/llm/claude/prompt_types.go
git commit -m "feat: add FileGenerationMode type

Add enum for explicit vs implicit file generation modes"
```

---

### Task 1.5: Add Manifest Requirement to Prompt

**Files:**
- Modify: `internal/llm/claude/prompt_types.go`

**Step 1: Add manifest requirement function**

Add to `prompt_types.go`:

```go
// buildManifestRequirement returns the manifest requirement for implicit mode
func buildManifestRequirement() string {
	return `
IMPORTANT: File Generation Manifest

You have freedom to create any files needed for this project based on requirements.
After completing all file operations, output a JSON manifest listing every file you created.

Format:
` + "```json" + `
{
  "generated_files": [
    "path/to/file1.ext",
    "path/to/file2.ext",
    "path/to/file3.ext"
  ]
}
` + "```" + `

Requirements:
- List EVERY file you created (no omissions)
- Use relative paths from project root
- Use forward slashes for path separators
- Manifest must be valid JSON
- Wrap in triple-backtick code fence with "json" language tag

This manifest is required for the provider to track generated files.
`
}
```

**Step 2: Find where system prompt is built**

Check `prompt_types.go` for existing prompt building functions. Look for something like `buildSystemPrompt()` or similar.

**Step 3: Modify system prompt builder to include manifest**

You'll need to modify the existing prompt building function to conditionally add the manifest requirement. The exact location depends on your current code structure.

Example modification (adapt to your actual code):

```go
// Modify the existing prompt builder
func buildSystemPromptForMode(mode FileGenerationMode) string {
	basePrompt := getBaseSystemPrompt() // or however you currently get it

	if mode == ImplicitMode {
		return basePrompt + buildManifestRequirement()
	}

	return basePrompt
}
```

**Step 4: Commit**

```bash
git add internal/llm/claude/prompt_types.go
git commit -m "feat: add manifest requirement to prompt

Add manifest requirement instructions for implicit mode"
```

---

## Phase 2: File Discovery & Validation

### Task 2.1: Create File Discovery Skeleton

**Files:**
- Create: `internal/files/discovery.go`
- Create: `internal/files/discovery_test.go`

**Step 1: Write test for basic discovery**

File: `internal/files/discovery_test.go`

```go
package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
)

func TestDiscoverAndValidateFiles_Success(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	// Create test files
	err := os.WriteFile(filepath.Join(tempDir, "app.py"), []byte("print('hello')"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "config.py"), []byte("DEBUG=True"), 0644)
	require.NoError(t, err)

	// Create manifest
	manifest := &claude.FileGenerationManifest{
		GeneratedFiles: []string{"app.py", "config.py"},
	}

	// Discover and validate
	result, err := DiscoverAndValidateFiles(tempDir, manifest)
	require.NoError(t, err)
	assert.Len(t, result.Files, 2)
	assert.Empty(t, result.ExtraFiles)
	assert.Empty(t, result.ManifestMissing)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/files -v -run TestDiscoverAndValidateFiles_Success`

Expected: FAIL with "undefined: DiscoverAndValidateFiles"

**Step 3: Create minimal implementation**

File: `internal/files/discovery.go`

```go
package files

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
)

// DiscoveryResult contains validated files from implicit generation
type DiscoveryResult struct {
	Files           []FileModel
	ExtraFiles      []string
	ManifestMissing []string
}

// FileModel represents a discovered file (minimal version for now)
type FileModel struct {
	Path        string
	Content     string
	ContentHash string
	FileHash    string
	FileModTime time.Time
}

// DiscoverAndValidateFiles validates manifest against filesystem
func DiscoverAndValidateFiles(
	outputPath string,
	manifest *claude.FileGenerationManifest,
) (*DiscoveryResult, error) {
	result := &DiscoveryResult{
		Files:           make([]FileModel, 0),
		ExtraFiles:      make([]string, 0),
		ManifestMissing: make([]string, 0),
	}

	// For each manifest file, verify it exists and build FileModel
	for _, manifestPath := range manifest.GeneratedFiles {
		fullPath := filepath.Join(outputPath, manifestPath)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			result.ManifestMissing = append(result.ManifestMissing, manifestPath)
			continue
		}

		// Build FileModel
		fileModel, err := buildFileModelFromPath(fullPath, manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", manifestPath, err)
		}

		result.Files = append(result.Files, fileModel)
	}

	// Error if any manifest files are missing
	if len(result.ManifestMissing) > 0 {
		return nil, fmt.Errorf(
			"manifest declared %d file(s) that don't exist: %v",
			len(result.ManifestMissing),
			result.ManifestMissing,
		)
	}

	return result, nil
}

// buildFileModelFromPath creates FileModel from filesystem file
func buildFileModelFromPath(fullPath, relativePath string) (FileModel, error) {
	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return FileModel{}, err
	}

	// Compute hashes
	contentHash := computeHash(content)
	fileHash := contentHash

	// Get modification time
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return FileModel{}, err
	}

	return FileModel{
		Path:        relativePath,
		Content:     string(content),
		ContentHash: contentHash,
		FileHash:    fileHash,
		FileModTime: fileInfo.ModTime(),
	}, nil
}

// computeHash computes SHA256 hash of content
func computeHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/files -v -run TestDiscoverAndValidateFiles_Success`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/files/discovery_test.go
git commit -m "test: add file discovery test

Test basic discovery and validation of manifest files"
```

```bash
git add internal/files/discovery.go
git commit -m "feat: implement basic file discovery

Discover files from manifest and validate they exist on disk"
```

---

### Task 2.2: Add Extra File Detection

**Files:**
- Modify: `internal/files/discovery_test.go`
- Modify: `internal/files/discovery.go`

**Step 1: Write test for extra files**

Add to `discovery_test.go`:

```go
func TestDiscoverAndValidateFiles_ExtraFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create manifest files
	err := os.WriteFile(filepath.Join(tempDir, "app.py"), []byte("code"), 0644)
	require.NoError(t, err)

	// Create extra file NOT in manifest
	err = os.WriteFile(filepath.Join(tempDir, "debug.log"), []byte("logs"), 0644)
	require.NoError(t, err)

	manifest := &claude.FileGenerationManifest{
		GeneratedFiles: []string{"app.py"},
	}

	result, err := DiscoverAndValidateFiles(tempDir, manifest)
	require.NoError(t, err) // Should not error
	assert.Len(t, result.Files, 1)
	assert.Contains(t, result.ExtraFiles, "debug.log")
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/files -v -run TestDiscoverAndValidateFiles_ExtraFiles`

Expected: FAIL (ExtraFiles not populated)

**Step 3: Add directory scanning to discovery**

Modify `discovery.go`:

```go
func DiscoverAndValidateFiles(
	outputPath string,
	manifest *claude.FileGenerationManifest,
) (*DiscoveryResult, error) {
	result := &DiscoveryResult{
		Files:           make([]FileModel, 0),
		ExtraFiles:      make([]string, 0),
		ManifestMissing: make([]string, 0),
	}

	// Scan output directory to get actual files
	actualFiles, err := scanDirectory(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan output directory: %w", err)
	}

	// Build set of actual files for quick lookup
	actualFileSet := make(map[string]bool)
	for _, path := range actualFiles {
		actualFileSet[path] = true
	}

	// Build set of manifest files
	manifestSet := make(map[string]bool)
	for _, manifestPath := range manifest.GeneratedFiles {
		manifestSet[manifestPath] = true

		fullPath := filepath.Join(outputPath, manifestPath)
		if !actualFileSet[manifestPath] {
			result.ManifestMissing = append(result.ManifestMissing, manifestPath)
			continue
		}

		fileModel, err := buildFileModelFromPath(fullPath, manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", manifestPath, err)
		}

		result.Files = append(result.Files, fileModel)
	}

	// Identify extra files (on disk but not in manifest)
	for _, actualPath := range actualFiles {
		if !manifestSet[actualPath] {
			result.ExtraFiles = append(result.ExtraFiles, actualPath)
		}
	}

	if len(result.ManifestMissing) > 0 {
		return nil, fmt.Errorf(
			"manifest declared %d file(s) that don't exist: %v",
			len(result.ManifestMissing),
			result.ManifestMissing,
		)
	}

	return result, nil
}

// scanDirectory recursively scans directory and returns relative paths
func scanDirectory(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip .debug directory
		if strings.Contains(path, ".debug") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Normalize path separators
		relPath = filepath.ToSlash(relPath)

		files = append(files, relPath)
		return nil
	})

	return files, err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/files -v -run TestDiscoverAndValidateFiles_ExtraFiles`

Expected: PASS

**Step 5: Commit**

```bash
git add internal/files/discovery_test.go
git commit -m "test: add extra file detection test

Test that files not in manifest are detected as extra"
```

```bash
git add internal/files/discovery.go
git commit -m "feat: detect extra files not in manifest

Scan directory and identify files not declared in manifest"
```

---

### Task 2.3: Add Missing File Test

**Files:**
- Modify: `internal/files/discovery_test.go`

**Step 1: Write test for missing manifest file**

Add to `discovery_test.go`:

```go
func TestDiscoverAndValidateFiles_MissingFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create only one file
	err := os.WriteFile(filepath.Join(tempDir, "app.py"), []byte("code"), 0644)
	require.NoError(t, err)

	// Manifest declares two files
	manifest := &claude.FileGenerationManifest{
		GeneratedFiles: []string{"app.py", "missing.py"},
	}

	result, err := DiscoverAndValidateFiles(tempDir, manifest)
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing.py")
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/files -v -run TestDiscoverAndValidateFiles_MissingFile`

Expected: PASS (already implemented)

**Step 3: Commit**

```bash
git add internal/files/discovery_test.go
git commit -m "test: add missing file test

Verify error when manifest file doesn't exist on disk"
```

---

### Task 2.4: Update Project Schema for Implicit Mode

**Files:**
- Modify: `internal/resources/project.go`

**Step 1: Add schema attributes for implicit mode**

Find the schema definition in `project.go` (likely in a `Schema()` method). Add these computed attributes:

```go
// Add to resource schema
"file_generation_mode": schema.StringAttribute{
	Description: "Mode for file generation: 'explicit' (files defined) or 'implicit' (Claude decides)",
	Computed:    true,
},
"computed_files": schema.MapNestedAttribute{
	Description: "Files generated by Claude in implicit mode",
	Computed:    true,
	NestedObject: schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"content_hash": schema.StringAttribute{
				Description: "SHA256 hash of file specification",
				Computed:    true,
			},
			"file_hash": schema.StringAttribute{
				Description: "SHA256 hash of actual file content",
				Computed:    true,
			},
			"file_modtime": schema.StringAttribute{
				Description: "File modification time (RFC3339)",
				Computed:    true,
			},
		},
	},
},
```

**Step 2: Add fields to ProjectModel struct**

Find the ProjectModel struct and add:

```go
type ProjectModel struct {
	// ... existing fields ...

	FileGenerationMode types.String                       `tfsdk:"file_generation_mode"`
	ComputedFiles      map[string]ComputedFileModel       `tfsdk:"computed_files"`
}

type ComputedFileModel struct {
	ContentHash types.String `tfsdk:"content_hash"`
	FileHash    types.String `tfsdk:"file_hash"`
	FileModTime types.String `tfsdk:"file_modtime"`
}
```

**Step 3: Commit**

```bash
git add internal/resources/project.go
git commit -m "feat: add schema for implicit file generation

Add file_generation_mode and computed_files to project schema"
```

---

### Task 2.5: Add Mode Detection Function

**Files:**
- Modify: `internal/resources/project.go`

**Step 1: Add mode detection function**

Add near the top of `project.go` (after imports):

```go
import (
	// ... existing imports ...
	"github.com/tofukit/opentofu-provider-tofukit/internal/llm/claude"
)

// determineFileMode detects whether project uses explicit or implicit file generation
func determineFileMode(files map[string]interface{}) claude.FileGenerationMode {
	if len(files) > 0 {
		return claude.ExplicitMode
	}
	return claude.ImplicitMode
}
```

**Step 2: Commit**

```bash
git add internal/resources/project.go
git commit -m "feat: add file generation mode detection

Detect explicit vs implicit mode based on files map"
```

---

### Task 2.6: Integrate Discovery into Update Method

**Files:**
- Modify: `internal/resources/project.go`

**Step 1: Find the Update method**

Locate the `Update()` method in `project.go`. This is where we'll add implicit mode logic.

**Step 2: Add mode detection at start of Update**

Add after reading plan/state:

```go
func (r *ProjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ProjectModel
	var state ProjectModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine file generation mode
	filesMap := make(map[string]interface{})
	// Convert data.Files to map (adapt to your actual structure)
	// This depends on how Files is stored in your model

	mode := determineFileMode(filesMap)
	data.FileGenerationMode = types.StringValue(mode.String())

	// ... rest of Update logic
}
```

**Step 3: Add implicit mode execution branch**

Find where Claude execution happens (likely calling something like `executeClaudeCode()`). Add branching logic:

```go
	// Execute Claude based on mode
	if mode == claude.ImplicitMode {
		// Execute Claude and expect manifest
		response, err := r.executeClaudeWithManifest(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError("Execution failed", err.Error())
			return
		}

		// Parse manifest
		manifest, err := claude.ExtractManifest(response)
		if err != nil {
			resp.Diagnostics.AddError("Manifest parsing failed", err.Error())
			return
		}

		// Validate manifest paths
		if err := claude.ValidateManifestPaths(manifest); err != nil {
			resp.Diagnostics.AddError("Invalid manifest paths", err.Error())
			return
		}

		// Discover and validate files
		result, err := files.DiscoverAndValidateFiles(data.ProjectPath.ValueString(), manifest)
		if err != nil {
			resp.Diagnostics.AddError("File discovery failed", err.Error())
			return
		}

		// Warn about extra files
		for _, extraFile := range result.ExtraFiles {
			tflog.Warn(ctx, fmt.Sprintf("Found file '%s' not in manifest", extraFile))
		}

		// Convert discovered files to state format
		data.ComputedFiles = convertFilesToComputedState(result.Files)

	} else {
		// Explicit mode: existing behavior
		err := r.executeClaudeStandard(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError("Execution failed", err.Error())
			return
		}
	}
```

**Step 4: Add helper to convert discovered files to state**

Add helper function:

```go
func convertFilesToComputedState(discoveredFiles []files.FileModel) map[string]ComputedFileModel {
	computedFiles := make(map[string]ComputedFileModel)

	for _, file := range discoveredFiles {
		computedFiles[file.Path] = ComputedFileModel{
			ContentHash: types.StringValue(file.ContentHash),
			FileHash:    types.StringValue(file.FileHash),
			FileModTime: types.StringValue(file.FileModTime.Format(time.RFC3339)),
		}
	}

	return computedFiles
}
```

**Step 5: Create stub for executeClaudeWithManifest**

Add method (we'll implement fully later):

```go
func (r *ProjectResource) executeClaudeWithManifest(ctx context.Context, data *ProjectModel) (string, error) {
	// TODO: Implement actual execution
	// For now, this is a placeholder that will be implemented in Phase 3
	return "", fmt.Errorf("not yet implemented")
}
```

**Step 6: Commit**

```bash
git add internal/resources/project.go
git commit -m "feat: integrate implicit mode into Update

Add mode detection and discovery flow to Update method.
Stub executeClaudeWithManifest for Phase 3 implementation."
```

---

## Phase 3: Execution & Drift Detection

### Task 3.1: Implement executeClaudeWithManifest

**Files:**
- Modify: `internal/resources/project.go`

**Step 1: Find existing Claude execution code**

Look for the existing `executeClaudeStandard()` or similar method that executes Claude. We'll follow the same pattern but with manifest-aware prompting.

**Step 2: Implement executeClaudeWithManifest**

Replace the stub with actual implementation:

```go
func (r *ProjectResource) executeClaudeWithManifest(ctx context.Context, data *ProjectModel) (string, error) {
	// Build prompt with manifest requirement
	promptJSON, err := r.buildPromptForMode(ctx, data, claude.ImplicitMode)
	if err != nil {
		return "", fmt.Errorf("failed to build prompt: %w", err)
	}

	// Execute Claude
	executor := r.claudeAdapter // or however you access the executor
	response, err := executor.ExecuteWithPromptJSON(ctx, promptJSON, data.ProjectPath.ValueString())
	if err != nil {
		return "", fmt.Errorf("Claude execution failed: %w", err)
	}

	return response, nil
}
```

**Step 3: Update prompt building to include manifest requirement**

Find where prompts are built (likely `buildPromptForMode()` or similar). Ensure it includes manifest requirement for implicit mode:

```go
func (r *ProjectResource) buildPromptForMode(ctx context.Context, data *ProjectModel, mode claude.FileGenerationMode) (string, error) {
	// Build base prompt
	prompt := buildBasePrompt(data)

	// Add manifest requirement if implicit mode
	if mode == claude.ImplicitMode {
		systemPrompt := claude.buildManifestRequirement()
		// Incorporate into prompt structure
		// This depends on your current prompt structure
	}

	return prompt, nil
}
```

**Step 4: Commit**

```bash
git add internal/resources/project.go
git commit -m "feat: implement Claude execution with manifest

Execute Claude with manifest requirement for implicit mode"
```

---

### Task 3.2: Implement Drift Detection for Implicit Mode

**Files:**
- Modify: `internal/resources/project.go`

**Step 1: Find the Read method**

Locate the `Read()` method that handles drift detection.

**Step 2: Update Read to handle both modes**

Modify to check both `files` and `computed_files`:

```go
func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Determine which files to check based on mode
	var filesToCheck map[string]ComputedFileModel

	if state.FileGenerationMode.ValueString() == "implicit" {
		filesToCheck = state.ComputedFiles
	} else {
		// Convert explicit files to same format for consistent checking
		filesToCheck = convertExplicitFilesToComputedFormat(state.Files)
	}

	// Check each file for drift
	driftDetected := false
	driftedFiles := []string{}

	for path, fileState := range filesToCheck {
		fullPath := filepath.Join(state.ProjectPath.ValueString(), path)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			driftDetected = true
			driftedFiles = append(driftedFiles, path)
			continue
		}

		// Check modification time (optimization)
		fileInfo, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		storedModTime, _ := time.Parse(time.RFC3339, fileState.FileModTime.ValueString())
		if fileInfo.ModTime().Equal(storedModTime) {
			// No change in mtime, skip hash check
			continue
		}

		// Compute current hash
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		currentHash := computeHash(content)
		if currentHash != fileState.FileHash.ValueString() {
			driftDetected = true
			driftedFiles = append(driftedFiles, path)
		}
	}

	// Update drift fields
	state.DriftDetected = types.BoolValue(driftDetected)
	state.DriftedFiles = driftedFiles

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper to compute hash (may already exist)
func computeHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}
```

**Step 3: Commit**

```bash
git add internal/resources/project.go
git commit -m "feat: implement drift detection for implicit mode

Check computed_files for drift same as explicit files"
```

---

### Task 3.3: Add Integration Test for Implicit Mode

**Files:**
- Create: `test/resource_project_implicit_mode_test.go`

**Step 1: Create basic implicit mode test**

File: `test/resource_project_implicit_mode_test.go`

```go
package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectImplicitMode_CreateSuccess(t *testing.T) {
	testDir := createTestDirectory(t, "TestProjectImplicitMode_CreateSuccess")

	// Create config with NO files defined
	config := `
resource "tofukit_project" "test" {
  name = "implicit-test"
  version = "1.0.0"

  requirements = [{
    name = "Simple App"
    instructions = [{
      prompt = "Create a simple Python app with app.py and config.py"
    }]
  }]

  # NO files block - implicit mode
}
`

	configPath := filepath.Join(testDir, "main.tf")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	// Mock Claude response with manifest
	mockResponse := `
Created app.py and config.py.

` + "```json" + `
{
  "generated_files": ["app.py", "config.py"]
}
` + "```" + `
`

	// TODO: Set up mock to return mockResponse
	// This depends on your existing test infrastructure

	// Run terraform apply
	// TODO: Execute terraform/tofu apply
	// TODO: Verify computed_files populated in state
	// TODO: Verify file_generation_mode = "implicit"

	t.Log("Test implementation depends on existing test helpers")
}
```

**Step 2: Commit**

```bash
git add test/resource_project_implicit_mode_test.go
git commit -m "test: add implicit mode integration test skeleton

Add test structure for implicit mode validation.
Full implementation requires mock infrastructure."
```

---

### Task 3.4: Add E2E Test Configuration

**Files:**
- Create: `test/testdata/configs/project-implicit-mode-basic.tofu`

**Step 1: Create test config**

File: `test/testdata/configs/project-implicit-mode-basic.tofu`

```hcl
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  debug                 = true
  claude_home_directory = "~/.claude"
  output_path           = "output"
}

resource "tofukit_project" "implicit_test" {
  name    = "implicit-mode-test"
  version = "1.0.0"

  requirements = [{
    name = "Python Application"
    instructions = [{
      prompt = "Create a simple Python application with proper structure"
      constraints = [
        "Include app.py as main file",
        "Include config.py for configuration",
        "Follow Python best practices"
      ]
    }]
  }]

  # NO files block - this triggers implicit mode
}
```

**Step 2: Commit**

```bash
git add test/testdata/configs/project-implicit-mode-basic.tofu
git commit -m "test: add implicit mode test config

Add test configuration for implicit file generation"
```

---

### Task 3.5: Update Documentation

**Files:**
- Modify: `CLAUDE.md`

**Step 1: Add implicit mode section to CLAUDE.md**

Find the section about file generation and add:

```markdown
### Implicit File Generation (NEW)

**Mode Detection**:
- `files = {}` or omitted → Implicit mode (Claude decides structure)
- `files = { ... }` → Explicit mode (user defines all files)

**How It Works**:

1. **Explicit Mode** (current behavior):
   - User defines `files = { "app.py" = {...}, "config.py" = {...} }`
   - Claude must work within these constraints
   - Provider tracks exactly those files

2. **Implicit Mode** (new):
   - User omits `files` block or sets `files = {}`
   - Claude decides file structure based on requirements
   - Claude outputs JSON manifest declaring created files
   - Provider validates manifest against filesystem
   - Full drift detection applies to discovered files

**Example - Implicit Mode**:
```hcl
resource "tofukit_project" "app" {
  requirements = [{
    name = "Flask App"
    instructions = [{
      prompt = "Create production-ready Flask application"
    }]
  }]

  # No files block - Claude decides structure
}
```

Claude creates:
- `app.py`
- `config.py`
- `requirements.txt`
- `tests/test_app.py`
- Any other files following best practices

**File Manifest**:

Claude outputs a manifest in its response:
```json
{
  "generated_files": [
    "app.py",
    "config.py",
    "requirements.txt"
  ]
}
```

Provider validates:
- Every manifest file exists on disk
- Paths are relative (no absolute paths)
- No path traversal (`..`)
- Warns about extra files not in manifest

**State Storage**:

Implicit mode stores discovered files in `computed_files`:
```hcl
computed_files = {
  "app.py" = {
    content_hash = "abc123"
    file_hash    = "def456"
    file_modtime = "2025-11-14T20:00:00Z"
  }
}
```

**Drift Detection**:

Works identically for both modes - provider tracks content hashes and detects manual edits.
```

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add implicit file generation documentation

Document implicit mode, manifest system, and mode detection"
```

---

## Testing & Validation

### Task 4.1: Run All Unit Tests

**Step 1: Run manifest parser tests**

Run: `go test ./internal/llm/claude -v`

Expected: All tests PASS

**Step 2: Run file discovery tests**

Run: `go test ./internal/files -v`

Expected: All tests PASS

**Step 3: Fix any failures**

If any tests fail, debug and fix before proceeding.

**Step 4: Commit if changes made**

```bash
git add .
git commit -m "fix: resolve test failures"
```

---

### Task 4.2: Run Integration Tests

**Step 1: Run project resource tests**

Run: `go test ./test -v -run TestProject`

Expected: Existing tests still PASS (no regression)

**Step 2: Run implicit mode tests**

Run: `go test ./test -v -run TestProjectImplicitMode`

Expected: PASS (once fully implemented)

**Step 3: Fix any failures**

Debug and fix integration test failures.

---

### Task 4.3: Manual Testing

**Step 1: Test with real Claude (Flask example)**

Create test directory and config:

```bash
mkdir -p /tmp/tofukit-implicit-test
cd /tmp/tofukit-implicit-test
```

Create `main.tf`:
```hcl
resource "tofukit_project" "flask_app" {
  name = "test-flask"

  requirements = [{
    name = "Flask Application"
    instructions = [{
      prompt = "Create production-ready Flask application with proper structure"
    }]
  }]
}
```

Run:
```bash
terraform init
terraform plan
terraform apply
```

Verify:
- Files created (app.py, config.py, etc.)
- `terraform.tfstate` contains `computed_files`
- `file_generation_mode = "implicit"`

**Step 2: Test drift detection**

Edit a file manually:
```bash
echo "# modified" >> output/app.py
```

Run:
```bash
terraform plan
```

Verify:
- Plan shows drift detected
- Drift listed in output

Run:
```bash
terraform apply
```

Verify:
- File restored to original content

**Step 3: Test mode transition (Implicit → Explicit)**

Modify `main.tf` to add explicit files:
```hcl
resource "tofukit_project" "flask_app" {
  name = "test-flask"

  files = {
    "app.py" = {
      content = "print('hello')"
    }
  }
}
```

Run:
```bash
terraform plan
```

Verify:
- Plan shows mode transition
- Shows computed_files being replaced with explicit files

---

### Task 4.4: Create Example Project

**Files:**
- Create: `examples/projects/implicit-flask-app/project.tofu`
- Create: `examples/projects/implicit-flask-app/README.md`

**Step 1: Create example config**

File: `examples/projects/implicit-flask-app/project.tofu`

```hcl
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  debug                 = true
  claude_home_directory = "~/.claude"
}

resource "tofukit_project" "flask_app" {
  name        = "implicit-flask-example"
  description = "Flask application with implicit file generation"
  version     = "1.0.0"

  requirements = [{
    name = "Flask Application"
    instructions = [{
      prompt = <<-EOF
        Create a production-ready Flask REST API application.

        Include:
        - Main application file
        - Configuration management
        - Environment variable handling
        - Basic error handling
        - Health check endpoint
      EOF

      constraints = [
        "Use Flask 3.0+",
        "Follow Python best practices",
        "Include proper logging"
      ]
    }]
  }]

  # NO files block - Claude decides structure
}
```

**Step 2: Create README**

File: `examples/projects/implicit-flask-app/README.md`

```markdown
# Implicit Flask Application Example

Example demonstrating **implicit file generation** where Claude decides the project structure.

## How It Works

This project has **no `files` block**, which triggers implicit mode. Claude:
1. Reads requirements
2. Decides optimal file structure
3. Creates files following best practices
4. Outputs manifest declaring created files

## Usage

```bash
cd examples/projects/implicit-flask-app
tofu init
tofu apply
```

Claude will create files like:
- `app.py` (main application)
- `config.py` (configuration)
- `requirements.txt` or `pyproject.toml`
- `.env.example`
- `README.md`
- Tests

## Verification

After apply, check state:
```bash
tofu show | grep file_generation_mode
# Should show: file_generation_mode = "implicit"

tofu show | grep -A 5 computed_files
# Shows discovered files with hashes
```

## Drift Detection

Edit a file manually:
```bash
echo "# modified" >> app.py
```

Run plan:
```bash
tofu plan
# Shows drift detected
```

Run apply to restore:
```bash
tofu apply
# Restores file to original content
```

## Compare to Explicit Mode

See `examples/projects/flask-api-nyc-weather` for explicit mode example where every file is defined.
```

**Step 3: Commit**

```bash
git add examples/projects/implicit-flask-app/
git commit -m "docs: add implicit Flask example

Add example project demonstrating implicit file generation"
```

---

## Final Steps

### Task 5.1: Clean Build & Test

**Step 1: Clean build**

Run: `go clean -cache`

**Step 2: Full build**

Run: `go build .`

Expected: No errors

**Step 3: Full test suite**

Run: `go test ./... -v`

Expected: All tests PASS

---

### Task 5.2: Update CHANGELOG

**Files:**
- Modify: `CHANGELOG.md` (if exists, else create)

**Step 1: Add entry**

```markdown
## [Unreleased]

### Added
- **Implicit file generation mode**: Projects can now omit the `files` block and let Claude decide file structure based on requirements
- File manifest system: Claude declares created files via JSON manifest
- Full drift detection for implicitly generated files
- Mode detection: Automatic detection of explicit vs implicit mode
- Path validation: Validates manifest paths are relative and safe
```

**Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: update CHANGELOG for implicit file generation"
```

---

### Task 5.3: Create Pull Request Prep

**Step 1: Squash commits if needed**

Review commit history:
```bash
git log --oneline
```

If you have many small commits, consider squashing:
```bash
git rebase -i HEAD~N  # where N = number of commits
```

**Step 2: Final commit message**

Create final summary commit if needed:
```bash
git commit --allow-empty -m "feat: implement implicit file generation

Complete implementation of implicit file generation mode.

Features:
- Dual-mode operation (explicit/implicit)
- JSON manifest parsing and validation
- File discovery and validation
- Full drift detection for both modes
- Path security validation
- Comprehensive test coverage

Closes #XXX (if applicable)"
```

**Step 3: Push branch**

```bash
git push origin feature/implicit-file-generation
```

---

## Plan Complete

Plan saved to: `docs/plans/2025-11-14-implicit-file-generation-implementation.md`

**Implementation Summary:**
- **Phase 1**: Manifest parsing infrastructure (6 tasks)
- **Phase 2**: File discovery and schema updates (6 tasks)
- **Phase 3**: Execution and drift detection (5 tasks)
- **Phase 4**: Testing and documentation (5 tasks)
- **Total**: 22 bite-sized tasks

**Estimated Time**: 28-36 hours total

**Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration with quality gates

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints between phases

**Which approach would you prefer?**
