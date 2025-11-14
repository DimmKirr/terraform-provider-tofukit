# Implicit Multi-File Generation Design

**Date**: 2025-11-14
**Status**: Design
**Author**: Claude (via brainstorming session)

## Overview

Enable natural language multi-file generation where users can specify requirements like "Create a production-ready Flask application" without explicitly defining every file. Claude decides the optimal file structure based on best practices and conventions.

### Current Limitation

Today, users must explicitly define every file:

```hcl
resource "tofukit_project" "app" {
  requirements = [{
    name = "Flask App"
    instructions = [{
      prompt = "Create production-ready Flask application"
    }]
  }]

  # REQUIRED: Must list every file explicitly
  files = {
    "app.py" = { instructions = [...] }
    "config.py" = { instructions = [...] }
    "requirements.txt" = { instructions = [...] }
    "tests/test_app.py" = { instructions = [...] }
  }
}
```

This violates DRY principles and limits Claude's ability to apply framework conventions.

### Desired Behavior

Users should be able to omit the `files` block and let Claude decide:

```hcl
resource "tofukit_project" "app" {
  requirements = [{
    name = "Flask App"
    instructions = [{
      prompt = "Create production-ready Flask application"
    }]
  }]

  # NO files block - Claude decides structure
}
```

Claude creates:
- `app.py` (main application)
- `config.py` (configuration)
- `requirements.txt` (dependencies)
- `tests/test_app.py` (tests)
- `.env.example` (environment template)
- Any other files following best practices

## Requirements

### Functional Requirements

**FR1**: Dual-mode operation
- **Explicit mode**: `files` defined → Claude works within constraints (current behavior)
- **Implicit mode**: `files` empty/omitted → Claude decides file structure

**FR2**: Mode boundary enforcement
- Explicit files are hard constraints - Claude MUST comply even if it thinks structure is suboptimal
- Implicit mode gives Claude complete freedom
- Mode determined at plan time based on presence of file definitions

**FR3**: File discovery mechanism
- Claude declares generated files via JSON manifest in response
- Provider validates manifest against actual filesystem
- Hybrid approach: declaration + verification

**FR4**: Full drift detection
- Both explicit and implicit modes track file content hashes
- Manual edits detected and restored on next apply
- Consistent behavior across modes

**FR5**: State management
- Explicit mode: stores user-defined files
- Implicit mode: stores discovered `computed_files`
- Mode stored in state to detect transitions

### Non-Functional Requirements

**NFR1**: Backward compatibility
- Existing projects with explicit files continue working unchanged
- No breaking changes to current API

**NFR2**: Error handling
- Strict validation of manifests (Claude must declare intent)
- Clear error messages when manifest missing/invalid
- Lenient on extra files (warnings, not errors)

**NFR3**: Performance
- Manifest parsing adds negligible overhead
- File discovery comparable to current explicit mode
- Directory scanning optimized for large projects

**NFR4**: Debuggability
- Debug mode shows manifest in output
- Clear distinction between "Claude intended" vs "found on disk"
- Helpful error messages for common issues

## Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────┐
│                    User Configuration                        │
│                                                              │
│  files = {} or omitted    →    Implicit Mode                │
│  files = { ... }          →    Explicit Mode                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                     Prompt Builder                           │
│                                                              │
│  Implicit Mode: Add manifest requirement to system prompt   │
│  Explicit Mode: Standard prompt (no changes)                │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    Claude Execution                          │
│                                                              │
│  Implicit: Creates files + outputs manifest                 │
│  Explicit: Creates defined files only                       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│              Post-Execution Processing                       │
│                                                              │
│  Implicit: Parse manifest → Validate → Store computed_files │
│  Explicit: Validate explicit files (current behavior)       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                      State Storage                           │
│                                                              │
│  file_generation_mode: "explicit" | "implicit"              │
│  files: {...}              (explicit mode)                   │
│  computed_files: {...}     (implicit mode)                   │
└─────────────────────────────────────────────────────────────┘
```

### Mode Detection

```go
type FileGenerationMode int

const (
    ExplicitMode FileGenerationMode = iota
    ImplicitMode
)

func determineFileMode(files map[string]FileModel) FileGenerationMode {
    if len(files) > 0 {
        return ExplicitMode  // User defined files explicitly
    }
    return ImplicitMode      // No files defined - Claude decides
}
```

**Critical constraint**: Once mode is detected, it becomes the contract for that apply cycle.

### File Manifest Structure

```json
{
  "generated_files": [
    "app.py",
    "config.py",
    "requirements.txt",
    "tests/test_app.py",
    ".env.example"
  ]
}
```

**Manifest rules**:
- Must be valid JSON
- Must be wrapped in markdown code fence: ` ```json ... ``` `
- Paths are relative to project output directory
- Forward slashes for all paths (normalized by provider)

### State Schema Changes

**Current state (explicit mode)**:
```hcl
resource "tofukit_project" "app" {
  files = {
    "app.py" = {
      content_hash = "abc123"
      file_hash    = "def456"
      file_modtime = "2025-11-14T20:00:00Z"
    }
  }
}
```

**New state (implicit mode)**:
```hcl
resource "tofukit_project" "app" {
  file_generation_mode = "implicit"

  computed_files = {
    "app.py" = {
      content_hash = "abc123"
      file_hash    = "def456"
      file_modtime = "2025-11-14T20:00:00Z"
    }
    "config.py" = {
      content_hash = "xyz789"
      file_hash    = "uvw012"
      file_modtime = "2025-11-14T20:05:00Z"
    }
  }
}
```

**Rationale for separate fields**:
- Clear distinction between user-defined vs Claude-generated
- Prevents confusion when debugging
- Allows future enhancements (e.g., mixing modes)

## Component Design

### 1. Prompt Builder Enhancement

**File**: `internal/llm/claude/prompt_types.go`

**Changes**:

```go
// Add to ProjectPromptRequest or similar structure
type FileGenerationManifest struct {
    GeneratedFiles []string `json:"generated_files"`
}

// New function to build manifest requirement
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

// Modify existing system prompt builder
func buildSystemPrompt(mode FileGenerationMode) string {
    basePrompt := getBaseSystemPrompt()

    if mode == ImplicitMode {
        return basePrompt + buildManifestRequirement()
    }

    return basePrompt
}
```

**Integration point**: Called during prompt construction in `buildProjectPrompt()` or equivalent.

### 2. Manifest Parser

**New file**: `internal/llm/claude/manifest_parser.go`

```go
package claude

import (
    "encoding/json"
    "fmt"
    "regexp"
    "strings"
)

// FileGenerationManifest represents Claude's declaration of generated files
type FileGenerationManifest struct {
    GeneratedFiles []string `json:"generated_files"`
}

// ExtractManifest parses file generation manifest from Claude's response
func ExtractManifest(response string) (*FileGenerationManifest, error) {
    // Pattern to match JSON in markdown code blocks
    // Matches: ```json\n{...}\n```
    jsonPattern := regexp.MustCompile("```json\\s*({[^`]+})\\s*```")
    matches := jsonPattern.FindStringSubmatch(response)

    if len(matches) < 2 {
        return nil, fmt.Errorf("no manifest found in response - ensure Claude outputs manifest in ```json code fence")
    }

    // Parse JSON
    var manifest FileGenerationManifest
    if err := json.Unmarshal([]byte(matches[1]), &manifest); err != nil {
        return nil, fmt.Errorf("invalid manifest JSON: %w", err)
    }

    // Validate manifest has files
    if len(manifest.GeneratedFiles) == 0 {
        return nil, fmt.Errorf("manifest contains zero files - at least one file must be declared")
    }

    // Normalize paths (convert backslashes to forward slashes)
    for i, path := range manifest.GeneratedFiles {
        manifest.GeneratedFiles[i] = strings.ReplaceAll(path, "\\", "/")
    }

    return &manifest, nil
}

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

**Test coverage**:
- Valid manifest extraction
- Invalid JSON handling
- Missing manifest handling
- Multiple code blocks (should use first)
- Path normalization (backslash → forward slash)
- Absolute path rejection
- Path traversal rejection

### 3. File Discovery & Validation

**New file**: `internal/files/discovery.go`

```go
package files

import (
    "crypto/sha256"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
)

// DiscoveryResult contains validated files from implicit generation
type DiscoveryResult struct {
    Files           []FileModel
    ExtraFiles      []string  // Files on disk not in manifest (warnings)
    ManifestMissing []string  // Files in manifest but not on disk (errors)
}

// DiscoverAndValidateFiles validates manifest against filesystem
func DiscoverAndValidateFiles(
    outputPath string,
    manifest *FileGenerationManifest,
) (*DiscoveryResult, error) {
    result := &DiscoveryResult{
        Files:           make([]FileModel, 0, len(manifest.GeneratedFiles)),
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

    // Validate each manifest file exists
    manifestSet := make(map[string]bool)
    for _, manifestPath := range manifest.GeneratedFiles {
        manifestSet[manifestPath] = true

        fullPath := filepath.Join(outputPath, manifestPath)
        if !actualFileSet[manifestPath] {
            result.ManifestMissing = append(result.ManifestMissing, manifestPath)
            continue
        }

        // Build FileModel from actual file
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

// buildFileModelFromPath creates FileModel from filesystem file
func buildFileModelFromPath(fullPath, relativePath string) (FileModel, error) {
    // Read file content
    content, err := os.ReadFile(fullPath)
    if err != nil {
        return FileModel{}, err
    }

    // Compute hashes
    contentHash := computeHash(content)
    fileHash := contentHash  // Initially same as content

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

**Test coverage**:
- All manifest files exist → success
- Manifest file missing → error
- Extra files on disk → warning (no error)
- Directory scanning excludes .debug
- Path normalization works correctly
- Hash computation accuracy

### 4. Project Resource Changes

**File**: `internal/resources/project.go`

**Schema additions**:

```go
// Add to ProjectModel or schema definition
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

**Update() method changes**:

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
    mode := determineFileMode(data.Files)
    data.FileGenerationMode = modeToString(mode)

    // Execute Claude with mode-aware prompt
    var discoveredFiles []FileModel
    var err error

    if mode == ImplicitMode {
        // Implicit mode: expect manifest in response
        response, err := r.executeClaudeWithManifest(ctx, data)
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
        result, err := files.DiscoverAndValidateFiles(data.ProjectPath, manifest)
        if err != nil {
            resp.Diagnostics.AddError("File discovery failed", err.Error())
            return
        }

        // Warn about extra files
        for _, extraFile := range result.ExtraFiles {
            tflog.Warn(ctx, fmt.Sprintf("Found file '%s' not in manifest", extraFile))
        }

        discoveredFiles = result.Files

        // Store in computed_files
        data.ComputedFiles = convertFilesToState(discoveredFiles)

    } else {
        // Explicit mode: existing behavior
        err = r.executeClaudeStandard(ctx, data)
        if err != nil {
            resp.Diagnostics.AddError("Execution failed", err.Error())
            return
        }
    }

    // Save state
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

### 5. Drift Detection Integration

**File**: `internal/resources/project.go` (Read method)

```go
func (r *ProjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state ProjectModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

    if resp.Diagnostics.HasError() {
        return
    }

    // Determine which files to check based on mode
    var filesToCheck map[string]FileModel

    if state.FileGenerationMode == "implicit" {
        filesToCheck = state.ComputedFiles
    } else {
        filesToCheck = state.Files
    }

    // Check each file for drift
    driftDetected := false
    driftedFiles := []string{}

    for path, fileState := range filesToCheck {
        fullPath := filepath.Join(state.ProjectPath, path)

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

        storedModTime, _ := time.Parse(time.RFC3339, fileState.FileModTime)
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
        if currentHash != fileState.FileHash {
            driftDetected = true
            driftedFiles = append(driftedFiles, path)
        }
    }

    // Update drift fields
    state.DriftDetected = driftDetected
    state.DriftedFiles = driftedFiles

    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

**Key point**: Drift detection is identical for both modes - we just check different field (`files` vs `computed_files`).

## Data Flow

### Complete Apply Cycle (Implicit Mode)

```
┌──────────────────────────────────────────────────────────────┐
│ 1. Plan Phase                                                │
│                                                              │
│ Input:  files = {} (or omitted)                             │
│ Detect: mode = implicit                                     │
│ Output: "Project will generate files (determined by Claude)" │
└──────────────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────────────┐
│ 2. Apply Phase - Prompt Construction                        │
│                                                              │
│ Build system prompt with manifest requirement               │
│ Include requirements, features, kits                        │
│ NO file list (implicit mode)                                │
└──────────────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────────────┐
│ 3. Claude Execution                                          │
│                                                              │
│ Claude receives requirements: "Create Flask app"            │
│ Claude creates files based on best practices:               │
│   - app.py                                                  │
│   - config.py                                               │
│   - requirements.txt                                        │
│   - tests/test_app.py                                       │
│ Claude outputs manifest in response                         │
└──────────────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────────────┐
│ 4. Post-Execution - Manifest Parsing                        │
│                                                              │
│ Extract JSON from markdown code block                       │
│ Parse: {"generated_files": ["app.py", "config.py", ...]}   │
│ Validate JSON structure                                     │
│ Validate paths (no absolute, no traversal)                  │
└──────────────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────────────┐
│ 5. File Discovery & Validation                              │
│                                                              │
│ Scan output directory for actual files                      │
│ For each manifest file:                                     │
│   - Verify exists on disk                                   │
│   - Read content                                            │
│   - Compute content_hash                                    │
│   - Compute file_hash                                       │
│   - Record file_modtime                                     │
│ Detect extra files (warning, not error)                     │
└──────────────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────────────┐
│ 6. State Update                                              │
│                                                              │
│ file_generation_mode = "implicit"                           │
│ computed_files = {                                          │
│   "app.py": {                                               │
│     content_hash: "abc123",                                 │
│     file_hash: "abc123",                                    │
│     file_modtime: "2025-11-14T20:00:00Z"                    │
│   },                                                        │
│   "config.py": { ... },                                     │
│   ...                                                       │
│ }                                                           │
└──────────────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────────────┐
│ 7. Next Plan (Drift Detection)                              │
│                                                              │
│ Read() loads computed_files from state                      │
│ For each file:                                              │
│   - Compare mtime (fast check)                              │
│   - If mtime changed, compare hash                          │
│   - If hash mismatch: drift_detected = true                │
│ Plan shows drift if detected                                │
└──────────────────────────────────────────────────────────────┘
                         ↓
┌──────────────────────────────────────────────────────────────┐
│ 8. Next Apply (Drift Restoration)                           │
│                                                              │
│ If drift_detected:                                          │
│   - Add drift warning to instructions                       │
│   - Claude restores files to specification                  │
│   - Clear drift flags                                       │
│ Same cycle as explicit mode drift handling                  │
└──────────────────────────────────────────────────────────────┘
```

## Error Handling

### Error Scenarios & Responses

| Scenario | Detection | Action | User Feedback |
|----------|-----------|--------|---------------|
| **Manifest missing** | No JSON in response | Abort apply, return error | "Claude did not output required file manifest. Enable debug mode to see full response." |
| **Invalid JSON** | JSON parse fails | Abort apply, return error | "Invalid manifest JSON: {parse error details}" |
| **Empty manifest** | `generated_files` array empty | Abort apply, return error | "Manifest contains zero files - at least one file must be declared" |
| **Absolute paths** | Path starts with `/` or contains `:` | Abort apply, return error | "Manifest contains absolute path 'C:/file.txt' - only relative paths allowed" |
| **Path traversal** | Path contains `..` | Abort apply, return error | "Manifest contains path traversal '../../../etc/passwd' - paths must stay within project" |
| **Missing file** | Manifest declares file not on disk | Abort apply, return error | "Manifest declared 'app.py' but file not found in output directory" |
| **Extra files** | File on disk not in manifest | Log warning, continue | "Warning: found 'debug.log' not in manifest (file will not be tracked)" |
| **Mode transition** | Explicit → Implicit or vice versa | Detect in Update(), handle transition | Plan shows mode change and file list changes |
| **Verification failure** | Verification command fails | Retry with context (existing behavior) | Standard verification error messages |

### Error Message Examples

**Manifest missing**:
```
Error: Claude Execution Failed

Claude did not output the required file generation manifest.

In implicit file generation mode (no files defined in configuration),
Claude must declare which files it created by outputting a JSON manifest.

Debug steps:
1. Enable debug mode: debug = true in provider config
2. Check .debug/claude-prompt-*.json for full response
3. Verify Claude received manifest requirement in system prompt
4. Check for errors in Claude's response

If this persists, consider using explicit file definitions instead.
```

**Invalid manifest JSON**:
```
Error: Invalid File Manifest

Failed to parse file generation manifest from Claude's response.

Parse error: invalid character '}' looking for beginning of value

Expected format:
```json
{
  "generated_files": ["file1.py", "file2.py"]
}
```

Found in response:
```json
{
  "generated_files": ["file1.py",}  // trailing comma
}
```

Enable debug mode to see full response.
```

**Missing declared file**:
```
Error: File Discovery Failed

Manifest declared 2 files that don't exist on disk:
- app.py
- config.py

Claude declared these files in the manifest but they were not found
in the output directory after execution.

This usually indicates Claude failed to create the files or
created them in the wrong location.

Check .debug/claude-prompt-*.md for the full conversation.
```

## Testing Strategy

### Unit Tests

**File**: `internal/llm/claude/manifest_parser_test.go`

```go
func TestExtractManifest_Success(t *testing.T)
func TestExtractManifest_MultipleCodeBlocks(t *testing.T)
func TestExtractManifest_NoManifest(t *testing.T)
func TestExtractManifest_InvalidJSON(t *testing.T)
func TestExtractManifest_EmptyFileList(t *testing.T)
func TestValidateManifestPaths_AbsolutePaths(t *testing.T)
func TestValidateManifestPaths_PathTraversal(t *testing.T)
func TestValidateManifestPaths_EmptyPath(t *testing.T)
```

**File**: `internal/files/discovery_test.go`

```go
func TestDiscoverAndValidateFiles_Success(t *testing.T)
func TestDiscoverAndValidateFiles_MissingManifestFile(t *testing.T)
func TestDiscoverAndValidateFiles_ExtraFiles(t *testing.T)
func TestScanDirectory_ExcludesDebug(t *testing.T)
func TestBuildFileModelFromPath_HashAccuracy(t *testing.T)
```

### Integration Tests

**File**: `test/resource_project_implicit_mode_test.go`

```go
func TestProjectImplicitMode_CreateSuccess(t *testing.T) {
    // Create project with no files defined
    // Mock Claude response with valid manifest
    // Assert computed_files populated
    // Assert file_generation_mode = "implicit"
}

func TestProjectImplicitMode_ManifestMissing(t *testing.T) {
    // Mock Claude response without manifest
    // Assert error returned
    // Assert state not updated
}

func TestProjectImplicitMode_InvalidManifest(t *testing.T) {
    // Mock Claude response with malformed JSON
    // Assert parse error returned
}

func TestProjectImplicitMode_MissingFile(t *testing.T) {
    // Mock manifest declares files
    // Simulate missing file on disk
    // Assert validation error
}

func TestProjectImplicitMode_DriftDetection(t *testing.T) {
    // Create project, apply
    // Manually modify file
    // Read() should detect drift
    // Next apply should restore file
}

func TestProjectImplicitMode_SwitchToExplicit(t *testing.T) {
    // Start with implicit mode
    // Change config to explicit files
    // Assert plan shows transition
    // Assert explicit mode takes over
}

func TestProjectExplicitMode_Unchanged(t *testing.T) {
    // Ensure explicit mode still works
    // No regression in existing behavior
}
```

### E2E Tests

**File**: `test/e2e_project_implicit_flask_test.go`

```go
func TestE2EImplicitFlaskProject(t *testing.T) {
    config := `
resource "tofukit_project" "flask_app" {
  name = "test-flask"

  requirements = [{
    name = "Flask Application"
    instructions = [{
      prompt = "Create production-ready Flask application"
    }]
  }]

  # No files defined - implicit mode
}
`
    // Apply
    // Assert standard Flask files created:
    //   - app.py
    //   - config.py
    //   - requirements.txt
    //   - tests/
    // Assert drift detection works
}
```

**File**: `test/e2e_project_implicit_go_cli_test.go`

```go
func TestE2EImplicitGoCliProject(t *testing.T) {
    config := `
resource "tofukit_project" "go_cli" {
  name = "test-cli"

  requirements = [{
    name = "Go CLI Application"
    instructions = [{
      prompt = "Create Go CLI with Cobra and Viper"
    }]
  }]
}
`
    // Apply
    // Assert standard Go structure:
    //   - main.go
    //   - cmd/
    //   - go.mod
    // Assert verifications run
}
```

### Test Coverage Goals

- **Unit tests**: >90% coverage for new code
- **Integration tests**: All error paths covered
- **E2E tests**: At least 2 real-world scenarios (Flask, Go CLI)

## Implementation Plan

### Phase 1: Core Manifest Support (Foundation)

**Goal**: Establish manifest parsing infrastructure

**Tasks**:
1. Create `internal/llm/claude/manifest_parser.go`
   - Implement `ExtractManifest()` function
   - Implement `ValidateManifestPaths()` function
   - Add JSON extraction from markdown code blocks

2. Create `internal/llm/claude/manifest_parser_test.go`
   - Test valid manifest extraction
   - Test invalid JSON handling
   - Test missing manifest detection
   - Test path validation

3. Update `internal/llm/claude/prompt_types.go`
   - Add `buildManifestRequirement()` function
   - Update `buildSystemPrompt()` to include manifest requirement
   - Add `FileGenerationMode` type

4. Add unit tests for prompt building
   - Verify manifest requirement added in implicit mode
   - Verify standard prompt in explicit mode

**Acceptance Criteria**:
- All unit tests pass
- Can parse manifests from realistic Claude responses
- Invalid manifests rejected with clear errors

**Estimated Effort**: 4-6 hours

---

### Phase 2: File Discovery & Validation (Integration)

**Goal**: Implement file discovery and integrate with project resource

**Tasks**:
1. Create `internal/files/discovery.go`
   - Implement `DiscoverAndValidateFiles()` function
   - Implement `scanDirectory()` helper
   - Implement `buildFileModelFromPath()` helper
   - Add hash computation

2. Create `internal/files/discovery_test.go`
   - Test successful discovery
   - Test missing file detection
   - Test extra file warnings
   - Test directory scanning

3. Update `internal/resources/project.go` schema
   - Add `file_generation_mode` computed attribute
   - Add `computed_files` computed map attribute
   - Update struct definitions

4. Implement mode detection
   - Add `determineFileMode()` function
   - Add mode transition detection

5. Update `Update()` method
   - Detect file generation mode
   - Branch to implicit vs explicit logic
   - Call manifest parsing + discovery for implicit mode
   - Store discovered files in state

**Acceptance Criteria**:
- File discovery works end-to-end
- State schema updated correctly
- Mode detection reliable
- Integration tests pass

**Estimated Effort**: 8-10 hours

---

### Phase 3: Drift Detection & Polish (Production Ready)

**Goal**: Complete drift detection and prepare for production use

**Tasks**:
1. Update `Read()` method
   - Support both `files` and `computed_files`
   - Drift detection works for both modes
   - Modification time optimization

2. Update drift restoration logic
   - Ensure implicit mode drift handled same as explicit
   - Test drift warning instructions

3. Error handling improvements
   - Add comprehensive error messages
   - Add debug mode guidance
   - Add helpful context for common failures

4. E2E tests
   - `test/e2e_project_implicit_flask_test.go`
   - `test/e2e_project_implicit_go_cli_test.go`
   - Test full apply cycle with real examples

5. Documentation
   - Update CLAUDE.md with implicit mode section
   - Add examples to docs/
   - Update resource documentation

6. Migration guide
   - Document mode transitions
   - Provide examples of converting explicit → implicit
   - Warn about potential issues

**Acceptance Criteria**:
- Drift detection fully functional for both modes
- E2E tests pass with real Claude (or realistic mocks)
- Documentation complete
- Ready for user testing

**Estimated Effort**: 10-12 hours

---

### Phase 4: Validation & Stabilization (Optional)

**Goal**: Production hardening and edge case coverage

**Tasks**:
1. Performance testing
   - Test with 50+ file projects
   - Optimize directory scanning if needed
   - Profile manifest parsing overhead

2. Edge case testing
   - Large manifests (100+ files)
   - Deep directory structures
   - Unicode filenames
   - Special characters in paths

3. User testing
   - Create sample projects
   - Gather feedback on UX
   - Iterate on error messages

4. Bug fixes
   - Address issues from testing
   - Refine error handling
   - Improve debug output

**Acceptance Criteria**:
- Handles edge cases gracefully
- Performance acceptable for large projects
- User feedback positive

**Estimated Effort**: 6-8 hours

---

### Total Estimated Effort

- **Phase 1**: 4-6 hours
- **Phase 2**: 8-10 hours
- **Phase 3**: 10-12 hours
- **Phase 4**: 6-8 hours (optional)

**Total**: 28-36 hours (excluding Phase 4)

## Migration Considerations

### Existing Projects (Explicit Mode)

**Impact**: None - fully backward compatible

Existing projects with `files = {...}` continue working exactly as before. No changes required.

### New Projects

**Decision point**: Users choose between:
1. **Explicit mode**: Define `files = {...}` for full control
2. **Implicit mode**: Omit `files` to let Claude decide structure

**Recommendation**: Start with implicit for prototyping, move to explicit for production.

### Transitioning Between Modes

**Explicit → Implicit**:

Before:
```hcl
files = {
  "app.py" = { instructions = [...] }
  "config.py" = { instructions = [...] }
}
```

After:
```hcl
# Remove files block entirely
# Claude will recreate with its own structure
```

**Warning**: Files not recreated by Claude will be orphaned (not deleted). Manual cleanup required.

**Implicit → Explicit**:

Before:
```hcl
# No files block
```

After (must list all files you want to keep):
```hcl
files = {
  "app.py" = {
    content = "..." # Copy from generated file
  }
  "config.py" = {
    content = "..."
  }
}
```

**Risk**: If you don't list a file, it won't be managed anymore.

**Best practice**: Avoid mode transitions. Pick one approach and stick with it per project.

### Testing Migration

Before deploying, test with:
1. Fresh implicit mode projects
2. Existing explicit mode projects (verify no regression)
3. Mode transition scenarios (document quirks)

## Future Enhancements

### Possible Extensions

1. **Hybrid mode**: Allow mixing explicit and implicit
   ```hcl
   files = {
     "app.py" = { ... }  # Explicit
     "*" = { implicit = true }  # Let Claude decide others
   }
   ```

2. **File templates**: Implicit mode with template constraints
   ```hcl
   file_template = {
     "*.py" = { style = "black", max_length = 500 }
   }
   ```

3. **Manifest metadata**: Extend manifest with descriptions
   ```json
   {
     "generated_files": [
       {
         "path": "app.py",
         "description": "Main Flask application",
         "dependencies": ["config.py"]
       }
     ]
   }
   ```

4. **Progressive discovery**: Claude incrementally declares files
   - Stream manifest updates during execution
   - Provider tracks files as they're created

5. **File diff in plan**: Show file list changes in plan output
   ```
   Plan: 3 files to add
   + app.py (discovered)
   + config.py (discovered)
   + requirements.txt (discovered)
   ```

These are not commitments, just possibilities for future iterations.

## Open Questions

1. **Should manifest be optional in implicit mode?**
   - Current design: Required (strict)
   - Alternative: Fall back to directory scan if missing (lenient)
   - **Decision**: Start strict, relax if needed

2. **How to handle `.gitignore` and other dot files?**
   - Should these be tracked or ignored?
   - **Decision**: Track all files in manifest, ignore `.debug` directory only

3. **Should we validate file content in addition to existence?**
   - E.g., verify Python files have valid syntax
   - **Decision**: Out of scope for Phase 1, use verifications instead

4. **How verbose should manifest warnings be?**
   - Extra files found - warning level or info?
   - **Decision**: Warning level, but not blocking

5. **Should mode transitions be automatic or require confirmation?**
   - Current: Automatic based on config
   - Alternative: Require explicit flag for transition
   - **Decision**: Automatic for now, add safety checks if problems arise

## References

### Related Issues/Discussions

- None yet (this is the initial design)

### Related Code Sections

- `internal/resources/project.go` - Project resource implementation
- `internal/files/operations.go` - File operation diffing (explicit mode)
- `internal/llm/claude/prompt_types.go` - Prompt construction
- `test/resource_project_inline_file_test.go` - Existing file tests

### External Documentation

- Terraform Plugin Framework: https://developer.hashicorp.com/terraform/plugin/framework
- Claude API Documentation: (internal)

---

## Approval

This design is ready for implementation once approved.

**Next steps**:
1. Review and approve design
2. Create git worktree for implementation
3. Proceed with Phase 1 implementation
