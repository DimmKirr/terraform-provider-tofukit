# Per-File Drift Detection Design

**Date:** 2025-11-01
**Status:** Approved
**Author:** Claude Code (with @dmitry)

## Executive Summary

Implement granular per-file drift detection for TofuKit provider, following the `hashicorp/local_file` pattern. This enhancement fixes a critical bug where manual file edits outside Terraform are not detected during `tofu plan`, and adds the ability to identify exactly which files drifted and provide targeted restoration instructions to Claude.

## Problem Statement

### Current Behavior (Broken)

```
1. User manually edits .gitignore on disk
2. Terraform config unchanged
3. tofu plan → calls Read()
4. Read() does NOT check for file-level drift ❌
5. Plan shows "No changes" (incorrect!)
6. User's manual edits persist (violates declarative model)
```

**Root Cause:** Drift detection exists only in `Update()` method (line 882-894 of project.go), which is only called when config changes. The `Read()` method does not perform drift detection.

### Desired Behavior (Fixed)

```
1. User manually edits .gitignore on disk
2. Terraform config unchanged
3. tofu plan → calls Read()
4. Read() detects drift for .gitignore specifically ✓
5. Plan shows: "drift_detected: false → true, drifted_files: [.gitignore]"
6. tofu apply → Update() tells Claude to restore .gitignore
7. Claude receives: "⚠️ DRIFT: .gitignore was modified, restore to spec"
```

## Design Goals

1. **Fix drift detection bug**: Move drift detection to `Read()` method
2. **Granular tracking**: Know exactly which files drifted (not just aggregate hash)
3. **Performance**: Optimize for 100+ file projects using modification time checks
4. **Convention**: Follow `hashicorp/local_file` pattern exactly
5. **Backward compatible**: No breaking changes to user-facing config
6. **Works for all file types**: Both inline files and referenced `tofukit_file` resources

## Architectural Pattern

### Inspiration: hashicorp/local_file

Analysis of [terraform-provider-local source code](https://github.com/hashicorp/terraform-provider-local/blob/main/internal/provider/resource_local_file.go) reveals:

**Read() Method:**
```go
// 1. Calculate hash (temporary, in-memory)
outputContent, _ := os.ReadFile(outputPath)
currentHash := sha1.Sum(outputContent)

// 2. Compare with stored hash
if currentHash != state.ID.ValueString() {
    resp.State.RemoveResource(ctx)  // Mark for recreation
}

// 3. Never updates stored hash (read-only for state)
```

**Create() Method:**
```go
// Compute and STORE hashes
checksums := genFileChecksums(content)
plan.ID = types.StringValue(checksums.sha1Hex)
plan.ContentSha256 = types.StringValue(checksums.sha256Hex)

// Persist to state
resp.State.Set(ctx, &plan)
```

**Key Insight:** Read() is "read-only" for STATE but actively calculates hashes for COMPARISON.

### Adaptation for TofuKit

TofuKit differs from `local_file`:
- **Multiple files per resource** vs one file per resource
- **Can't use RemoveResource()** (would delete entire project)
- **Need LLM context** about which files drifted

**Solution:** Store per-file hashes, detect drift in Read(), set computed trigger fields.

## Schema Changes

### FileModel (internal/schemas/common.go)

```go
type FileModel struct {
    // Existing fields (user-specified)
    Content       types.String         `tfsdk:"content"`
    Instructions  []InstructionModel   `tfsdk:"instructions"`
    Verifications []VerificationModel  `tfsdk:"verifications"`

    // NEW: Computed drift detection fields
    ContentHash   types.String         `tfsdk:"content_hash"`
    FileHash      types.String         `tfsdk:"file_hash"`
    FileModTime   types.String         `tfsdk:"file_modtime"`
}
```

**Field Descriptions:**

- **content_hash**: SHA256 of file specification
  - Static files: Hash of `content` string
  - Generated files: Hash of `instructions` JSON
  - Used to detect: "User changed what they WANT"

- **file_hash**: SHA256 of actual file on disk
  - Computed after Claude execution
  - Used to detect: "File diverged from expected state"

- **file_modtime**: File modification timestamp (RFC3339)
  - Optimization: Skip hash computation if mtime unchanged
  - Performance benefit for 100+ file projects

### ProjectModel (internal/resources/project.go)

```go
type ProjectModelFinal struct {
    // ... existing fields ...

    // NEW: Computed drift detection (triggers Update)
    DriftDetected types.Bool   `tfsdk:"drift_detected"`
    DriftedFiles  types.List   `tfsdk:"drifted_files"`  // List<String>
}
```

**Field Descriptions:**

- **drift_detected**: Boolean flag set by Read()
  - Changes from `false → true` trigger Update() execution
  - Terraform sees this as a state change requiring reconciliation

- **drifted_files**: List of file paths that drifted
  - Example: `[".gitignore", "cmd/hello.go"]`
  - Used in Update() to provide context to Claude

## Implementation Design

### 1. Hash Computation Functions

**New file:** `internal/files/hashing.go`

```go
// ComputeContentHash generates hash of file specification
func ComputeContentHash(file schemas.FileModel) string {
    var hashInput []byte

    if !file.Content.IsNull() && file.Content.ValueString() != "" {
        hashInput = []byte(file.Content.ValueString())
    } else if len(file.Instructions) > 0 {
        jsonBytes, _ := json.Marshal(file.Instructions)
        hashInput = jsonBytes
    } else {
        return ""
    }

    hash := sha256.Sum256(hashInput)
    return hex.EncodeToString(hash[:])
}

// ComputeFileHash generates SHA256 hash of actual file
func ComputeFileHash(content []byte) string {
    hash := sha256.Sum256(content)
    return hex.EncodeToString(hash[:])
}
```

**Why SHA256?**
- Industry standard for content integrity
- Collision-resistant
- Better than MD5/SHA1 (deprecated for security)
- Fast enough for 100+ files

### 2. Read() Method - Drift Detection

**Location:** `internal/resources/project.go`

**Flow:**
```
Read() called during "tofu plan"
    ↓
For each file in state.Files:
    ↓
    Check if file exists on disk
    ├─ No → Mark as drifted (deleted)
    └─ Yes → Continue
         ↓
         Check modification time
         ├─ Unchanged → Skip (optimization)
         └─ Changed → Continue
              ↓
              Read file content
              ↓
              Calculate hash (temporary)
              ↓
              Compare with state.FileHash
              ├─ Match → No drift
              └─ Mismatch → Mark as drifted
    ↓
If any files drifted:
    state.DriftDetected = true
    state.DriftedFiles = [...paths...]
    ↓
resp.State.Set(ctx, &state)
```

**Performance Optimization:**
- Check `mtime` before hashing (fast filesystem stat)
- Only hash files where `mtime` changed
- Avoids 90%+ of hash computations on unchanged files

**Key Code:**
```go
// Optimization: Check modification time first
currentModTime := fileInfo.ModTime().Format(time.RFC3339)
if currentModTime == fileModel.FileModTime.ValueString() {
    continue // File untouched, skip hashing
}

// Only hash if mtime changed
actualContent, _ := os.ReadFile(fullPath)
currentHash := files.ComputeFileHash(actualContent)

if currentHash != fileModel.FileHash.ValueString() {
    driftedFiles = append(driftedFiles, path)
}
```

### 3. Create() Method - Store Hashes

**Location:** `internal/resources/project.go`

**Flow:**
```
Create() called during "tofu apply"
    ↓
Execute Claude Code
    ↓
Success?
    ├─ No → Return error
    └─ Yes → Continue
         ↓
         For each file in mergedFiles:
             ↓
             Read actual file from disk
             ↓
             Compute content_hash (from spec)
             Compute file_hash (from disk)
             Store file_modtime
             ↓
             Update state.Files[path]
         ↓
         Set drift_detected = false
         Set drifted_files = []
         ↓
         resp.State.Set(ctx, &data)
```

**Helper Function:**
```go
func (r *ProjectResourceFinal) computeAndStoreFileHashes(
    ctx context.Context,
    data *ProjectModelFinal,
    mergedFiles []schemas.FileModelWithPath,
) error {
    // For each file: read, hash, store
    // Returns error if file read fails
}
```

### 4. Update() Method - Handle Drift

**Location:** `internal/resources/project.go`

**Enhancements:**

1. **Detect drift trigger:**
   ```go
   isDriftTriggered := state.DriftDetected.ValueBool()
   ```

2. **Add drift instructions:**
   ```go
   if isDriftTriggered {
       var driftedPaths []string
       state.DriftedFiles.ElementsAs(ctx, &driftedPaths, false)
       enrichedFiles = r.addDriftInstructions(enrichedFiles, driftedPaths)
   }
   ```

3. **After successful execution:**
   ```go
   // Recompute all hashes
   r.computeAndStoreFileHashes(ctx, &data, mergedFiles)

   // Clear drift flags
   data.DriftDetected = types.BoolValue(false)
   data.DriftedFiles = types.ListNull(types.StringType)
   ```

**Drift Instructions to Claude:**
```go
driftInstruction := schemas.InstructionModel{
    Prompt: types.StringValue(
        "⚠️ DRIFT DETECTED: File 'X' was modified outside Terraform. " +
        "Restore it to match the specification below.",
    ),
    Constraints: []types.String{
        types.StringValue("Restore file to match Terraform specification"),
        types.StringValue("Ignore any manual edits made outside Terraform"),
    },
}
```

## User Experience

### Plan Output

**Before (Broken):**
```bash
$ tofu plan

No changes. Your infrastructure matches the configuration.
```

**After (Fixed):**
```bash
$ tofu plan

Terraform will perform the following actions:

  # tofukit_project.app will be updated in-place
  ~ resource "tofukit_project" "app" {
        id               = "project.my-app"
        name             = "my-app"
      ~ drift_detected   = false -> true
      ~ drifted_files    = [] -> [
          + ".gitignore",
        ]
        # (10 unchanged attributes hidden)
    }

Plan: 0 to add, 1 to change, 0 to destroy.
```

### Apply Output

```bash
$ tofu apply

tofukit_project.app: Modifying... [id=project.my-app]
tofukit_project.app: Modifications complete after 2s

Apply complete! Resources: 0 added, 1 changed, 0 destroyed.
```

**Logs will show:**
```
[INFO] Drift detected in project files
  project_id: project.my-app
  drifted_count: 1
  drifted_files: [".gitignore"]

[INFO] Update triggered by drift detection
  drifted_files: [".gitignore"]
```

## Edge Cases

### 1. LLM Non-Determinism

**Problem:** Claude generates slightly different output each time.

**Detection:**
```go
// First apply: Claude creates README.md
file_hash = "abc123..."

// Manual edit: User doesn't touch README.md
// But content_hash unchanged, file_hash unchanged → No drift

// Second apply: Config change triggers re-execution
// Claude generates slightly different README.md
file_hash = "def456..." (different from "abc123...")

// Next plan:
// content_hash still unchanged (spec didn't change)
// file_hash changed → Looks like drift!
```

**Solution:** Only flag as drift if file_hash changed AND file is NOT in current operation's modified files list.

### 2. Files Created by Claude (Not in Spec)

**Example:** Claude creates `__pycache__/`, `.pytest_cache/`

**Current behavior:** Tracked by `output_hash` (aggregate)

**With per-file hashing:** These files have no entry in `state.Files`, so they're ignored by drift detection.

**Solution:** Keep hybrid approach:
- `file_hash`: Track specified files
- `output_hash`: Track ALL files (including untracked)

### 3. Binary Files

**Problem:** Hashing large binaries is expensive

**Solution (future optimization):**
```go
fileInfo, _ := os.Stat(fullPath)
if fileInfo.Size() > 10*1024*1024 { // 10MB threshold
    // Track size instead of hash
    fileModel.FileHash = fmt.Sprintf("size:%d", fileInfo.Size())
}
```

### 4. Line Ending Normalization

**Problem:** Windows CRLF vs Unix LF causes hash mismatch

**Solution (future enhancement):**
```go
// Normalize line endings before hashing
normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
hash := sha256.Sum256([]byte(normalized))
```

### 5. Deleted Files

**Behavior:**
```go
if os.IsNotExist(err) {
    driftedFiles = append(driftedFiles, path)
}

// Claude instruction:
"⚠️ File 'X' was deleted outside Terraform. Recreate it."
```

## Testing Strategy

### Unit Tests

**Location:** `internal/files/hashing_test.go`

Test cases:
- ComputeContentHash for static files
- ComputeContentHash for instruction-based files
- ComputeFileHash with various content

### Integration Tests

**Location:** `test/project_drift_test.go`

Test scenarios:
1. **Drift detection on static files**
   - Create project with `.gitignore`
   - Manually edit `.gitignore`
   - Run plan → verify drift detected
   - Run apply → verify file restored

2. **Drift detection on generated files**
   - Create project with instruction-based file
   - Manually edit generated file
   - Run plan → verify drift detected
   - Run apply → verify file restored

3. **No false positives**
   - Create project
   - Run plan (no changes) → verify no drift

4. **Multiple files drifted**
   - Create project with 5 files
   - Edit 3 files manually
   - Verify drift lists all 3 correctly

5. **Deleted file recovery**
   - Create project
   - Delete file manually
   - Run plan → verify drift detected
   - Run apply → verify file recreated

### Performance Tests

**Location:** `test/project_performance_test.go`

Test scenarios:
- 100 files, none modified → verify mtime optimization works
- 100 files, all modified → verify hash computation performance

## Implementation Phases

### Phase 1: Foundation (Non-Breaking)
- Add schema fields (content_hash, file_hash, file_modtime, drift_detected, drifted_files)
- Add hash computation functions (hashing.go)
- Unit tests for hash functions

**Verification:** Schema changes don't break existing projects

### Phase 2: Hash Storage
- Implement computeAndStoreFileHashes() helper
- Call from Create() after Claude execution
- Call from Update() after Claude execution

**Verification:** State contains computed hashes after apply

### Phase 3: Drift Detection
- Implement drift detection in Read()
- Set drift_detected and drifted_files computed fields
- Integration tests for drift scenarios

**Verification:** Plan shows drift when files manually edited

### Phase 4: Drift Restoration
- Implement addDriftInstructions() helper
- Call from Update() when drift detected
- Add drift warnings to enriched files sent to Claude

**Verification:** Apply restores drifted files to spec

### Phase 5: Optimizations
- Add mtime check before hashing
- Performance tests
- Edge case handling

**Verification:** 100+ file projects perform well

## Backward Compatibility

**State Migration:** None required
- New fields are computed, not user-specified
- Empty/null values acceptable for first run
- Hashes computed on next apply

**Config Changes:** None required
- Users don't specify drift fields
- Existing configs work unchanged

**Behavior Changes:**
- Before: Manual file edits silently persist ❌
- After: Manual file edits detected and restored ✓

This is a **bug fix**, not a breaking change.

## Success Criteria

1. ✅ Manual file edits detected during `tofu plan`
2. ✅ Plan output shows which files drifted
3. ✅ Apply restores drifted files to specification
4. ✅ Works for both inline and referenced files
5. ✅ Performance acceptable for 100+ file projects
6. ✅ No false positives (unchanged files don't show drift)
7. ✅ All integration tests pass

## References

- [hashicorp/local_file source](https://github.com/hashicorp/terraform-provider-local/blob/main/internal/provider/resource_local_file.go)
- [Terraform Plugin Framework - Computed Attributes](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#computed)
- [TofuKit CLAUDE.md - File Operations](../CLAUDE.md#file-operations)

## Appendix: Current vs Future Flow

### Current Flow (Broken)

```
User edits .gitignore
    ↓
tofu plan
    ↓
Read() called
    ├─ No drift detection ❌
    └─ Plan shows "No changes"
    ↓
User confused why edits persist
```

### Future Flow (Fixed)

```
User edits .gitignore
    ↓
tofu plan
    ↓
Read() called
    ├─ Detects .gitignore drift ✓
    └─ Sets drift_detected = true
    ↓
Plan shows:
  ~ drift_detected: false → true
  ~ drifted_files: [] → [".gitignore"]
    ↓
tofu apply
    ↓
Update() called
    ├─ Sees drift trigger
    ├─ Adds drift instruction to .gitignore
    └─ Executes Claude with context
    ↓
Claude restores .gitignore to spec
    ↓
Hashes recomputed, drift cleared
```
