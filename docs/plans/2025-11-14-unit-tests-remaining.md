# Unit Tests - Remaining Work

**Status**: 🔨 Partially Implemented  
**Priority**: Medium (Code quality improvement)

## What's Already Done ✅

### Integration Tests (test/*.go)
- ✅ 17 unit tests for requirements/files collection
- ✅ Kit verification collection tests
- ✅ Feature/Kit/Stack integration tests
- ✅ Full E2E tests with provider execution

### Internal Unit Tests (internal/)
- ✅ `internal/uri/scanner_test.go` - URI parsing (7435 bytes)
- ✅ `internal/uri/registry_builder_test.go` - Registry building (10032 bytes)
- ✅ `internal/files/hashing_test.go` - SHA256 hashing (1870 bytes)

## What's Missing ❌

### File Operations (internal/files/operations.go)
**Missing:** `internal/files/operations_test.go`

**What needs testing:**
- `ComputeFileOperations(oldFiles, newFiles)` - Detects ADD/MODIFY/REMOVE/RENAME/UNCHANGED
- `EnrichFilesWithInstructions(oldFiles, newFiles)` - Adds action-based instructions
- Rename detection via content/instruction matching
- Edge cases: empty arrays, identical files, etc.

**Example test structure:**
```go
func TestComputeFileOperations_Add(t *testing.T) {
    tests := []struct {
        name     string
        oldFiles []FileModelWithPath
        newFiles []FileModelWithPath
        expected map[string]FileAction
    }{
        {
            name: "new file added",
            oldFiles: []FileModelWithPath{},
            newFiles: []FileModelWithPath{
                {Path: "hello.txt", Content: "hello"},
            },
            expected: map[string]FileAction{
                "hello.txt": ActionAdd,
            },
        },
        // ... more cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ComputeFileOperations(tt.oldFiles, tt.newFiles)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### File Merger (internal/files/merger.go)
**Missing:** `internal/files/merger_test.go`

**What needs testing:**
- Precedence hierarchy: Stack → Kit → Feature → Project
- File overriding (project files override feature files)
- Empty cases (no files from any source)
- Source tracking (`FileSource` enum correctness)

**Example test structure:**
```go
func TestFileMerger_Precedence(t *testing.T) {
    tests := []struct {
        name          string
        stackFiles    []FileModelWithPath
        kitFiles      []FileModelWithPath
        featureFiles  []FileModelWithPath
        projectFiles  []FileModelWithPath
        expectedPath  string
        expectedContent string
        expectedSource FileSource
    }{
        {
            name: "project overrides all",
            stackFiles: []FileModelWithPath{
                {Path: "config.txt", Content: "from stack"},
            },
            featureFiles: []FileModelWithPath{
                {Path: "config.txt", Content: "from feature"},
            },
            projectFiles: []FileModelWithPath{
                {Path: "config.txt", Content: "from project"},
            },
            expectedContent: "from project",
            expectedSource: SourceProject,
        },
        // ... more cases
    }
}
```

## Why These Tests Matter

1. **Regression Prevention**: File operation logic is critical - bugs here affect all projects
2. **Fast Feedback**: Unit tests run in milliseconds vs seconds for E2E tests
3. **Edge Case Coverage**: Easier to test corner cases without full Terraform setup
4. **Documentation**: Tests serve as executable specs for complex logic

## Implementation Guide

### Step 1: Create operations_test.go
```bash
touch internal/files/operations_test.go
```

Add tests for:
- `ComputeFileOperations` - All 5 actions (ADD/MODIFY/REMOVE/RENAME/UNCHANGED)
- `EnrichFilesWithInstructions` - Instruction merging logic
- Edge cases (nil arrays, duplicates, etc.)

### Step 2: Create merger_test.go
```bash
touch internal/files/merger_test.go
```

Add tests for:
- Precedence hierarchy
- File source tracking
- Empty/null handling

### Step 3: Run tests
```bash
go test ./internal/files -v
```

## Priority Assessment

**Medium Priority** because:
- ✅ E2E tests provide good coverage
- ✅ Integration tests cover most scenarios
- ❌ But unit tests would catch regressions faster
- ❌ And make debugging easier

Implement when:
- Adding new features to operations.go or merger.go
- Fixing bugs in file merging/operation detection
- Improving test coverage metrics

---

**Note:** This consolidates `2025-11-14-unit-tests-core-logic.md` with current state.
URI tests are done, only operations and merger need tests.
