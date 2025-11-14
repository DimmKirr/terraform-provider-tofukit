# Core Unit Tests Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

## ⚠️ CRITICAL WARNING - VERIFY BEFORE IMPLEMENTING

**DO NOT blindly implement tests from this plan!**

This plan makes assumptions about function names, signatures, and types that may not match the actual codebase:

**BEFORE implementing each task:**
1. ✅ Check if the function exists (e.g., `ParseURI`, `MergeFiles`, `ComputeFileOperations`)
2. ✅ Verify the actual function signature and parameters
3. ✅ Check the actual type names (`FileModel`, `FileAction`, etc.)
4. ✅ Understand what the function actually returns
5. ✅ Read the implementation to understand actual behavior

**IF YOU ARE NOT CERTAIN:**
- ❌ **DO NOT create the test** - skip that task entirely
- ❌ **DO NOT guess** function signatures or types
- ❌ **DO NOT assume** the code works as described in this plan

**Only create tests when you are 100% certain about:**
- Function exists and is exported
- You understand its exact signature
- You know what output to expect
- You've verified the types match

---

**Goal:** Add table-driven unit tests for URI validation, file operation detection, and file merge precedence to improve code reliability and catch regressions.

**Architecture:** Follow Go table-driven testing pattern with comprehensive test cases covering happy paths, edge cases, and error conditions. Tests are independent, fast, and focused on business logic without external dependencies.

**Tech Stack:** Go 1.21+, testify/assert, testify/require

---

## Context

The TofuKit provider currently has excellent integration/E2E test coverage but lacks unit tests for core business logic. This plan adds table-driven unit tests for three critical areas:

1. **URI Validation** - Ensures `tofukit://` URIs are parsed correctly
2. **File Operation Detection** - Validates diff computation logic (ADD/MODIFY/REMOVE/RENAME/UNCHANGED)
3. **File Merge Precedence** - Verifies correct precedence hierarchy (Stack → Kit → Feature → Project)

**Related Files:**
- `/internal/uri/scanner.go` - URI parsing implementation
- `/internal/files/operations.go` - File operation detection
- `/internal/files/merger.go` - File merge logic
- `/test/helpers_test.go` - Shared test helpers

**Testing Philosophy:**
- Table-driven tests for variations of same logic
- One test function per logical unit
- Clear test case names describing what's being tested
- Fast execution (no file I/O, no terraform execution)

---

## Task 1: URI Validation Tests

**Goal:** Validate that `tofukit://` URI parsing handles all valid formats and rejects invalid ones.

**Files:**
- Create: `internal/uri/scanner_test.go`
- Reference: `internal/uri/scanner.go` (implementation)

### Step 1: Create test file with package declaration

```bash
touch internal/uri/scanner_test.go
```

Add:
```go
package uri

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

### Step 2: Write failing test for valid URIs

Add to `internal/uri/scanner_test.go`:

```go
func TestParseURI_ValidURIs(t *testing.T) {
	tests := []struct {
		name         string
		uri          string
		expectedType string
		expectedName string
		expectedSub  string
	}{
		{
			name:         "simple feature",
			uri:          "tofukit://feature/logging",
			expectedType: "feature",
			expectedName: "logging",
			expectedSub:  "",
		},
		{
			name:         "kit with subtype",
			uri:          "tofukit://kit/language/python",
			expectedType: "kit",
			expectedName: "python",
			expectedSub:  "language",
		},
		{
			name:         "file reference",
			uri:          "tofukit://file/gitignore",
			expectedType: "file",
			expectedName: "gitignore",
			expectedSub:  "",
		},
		{
			name:         "stack reference",
			uri:          "tofukit://stack/python-app",
			expectedType: "stack",
			expectedName: "python-app",
			expectedSub:  "",
		},
		{
			name:         "name with hyphens",
			uri:          "tofukit://feature/structured-logging",
			expectedType: "feature",
			expectedName: "structured-logging",
			expectedSub:  "",
		},
		{
			name:         "name with underscores",
			uri:          "tofukit://feature/api_client",
			expectedType: "feature",
			expectedName: "api_client",
			expectedSub:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseURI(tt.uri)
			require.NoError(t, err, "Valid URI should parse without error")
			assert.Equal(t, tt.expectedType, parsed.Type)
			assert.Equal(t, tt.expectedName, parsed.Name)
			if tt.expectedSub != "" {
				assert.Equal(t, tt.expectedSub, parsed.Subtype)
			}
		})
	}
}
```

### Step 3: Run test to verify it fails

```bash
cd internal/uri
go test -v -run TestParseURI_ValidURIs
```

**Expected:** FAIL - "ParseURI undefined" or test failures if function exists but has bugs

### Step 4: Check if ParseURI exists

```bash
grep -n "func ParseURI" scanner.go
```

**If function doesn't exist:** You'll need to implement it based on the existing URI scanning logic in `scanner.go`. Look for existing parsing code and extract it into a testable function.

**If function exists:** Fix any bugs to make tests pass.

### Step 5: Run test to verify it passes

```bash
go test -v -run TestParseURI_ValidURIs
```

**Expected:** PASS - All 6 test cases pass

### Step 6: Write failing test for invalid URIs

Add to `internal/uri/scanner_test.go`:

```go
func TestParseURI_InvalidURIs(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing scheme",
			uri:     "feature/logging",
			wantErr: true,
			errMsg:  "missing tofukit://",
		},
		{
			name:    "wrong scheme",
			uri:     "https://feature/logging",
			wantErr: true,
			errMsg:  "must start with tofukit://",
		},
		{
			name:    "empty resource type",
			uri:     "tofukit:///logging",
			wantErr: true,
			errMsg:  "empty resource type",
		},
		{
			name:    "empty resource name",
			uri:     "tofukit://feature/",
			wantErr: true,
			errMsg:  "empty resource name",
		},
		{
			name:    "invalid characters in name",
			uri:     "tofukit://feature/my feature",
			wantErr: true,
			errMsg:  "invalid characters",
		},
		{
			name:    "too many path segments",
			uri:     "tofukit://kit/language/python/extra",
			wantErr: true,
			errMsg:  "invalid URI format",
		},
		{
			name:    "unknown resource type",
			uri:     "tofukit://invalid/test",
			wantErr: true,
			errMsg:  "unknown resource type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseURI(tt.uri)
			if tt.wantErr {
				require.Error(t, err, "Invalid URI should return error")
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, parsed)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, parsed)
			}
		})
	}
}
```

### Step 7: Run test to verify it fails (or passes with good validation)

```bash
go test -v -run TestParseURI_InvalidURIs
```

**Expected:** May PASS if validation already exists, or FAIL if validation is missing

### Step 8: Implement/fix validation as needed

Review error messages and add validation logic to `ParseURI` to handle all invalid cases.

### Step 9: Run all URI tests

```bash
go test -v -run TestParseURI
```

**Expected:** PASS - All test cases pass

### Step 10: Commit URI validation tests

```bash
git add internal/uri/scanner_test.go internal/uri/scanner.go
git commit -m "test(uri): add table-driven tests for URI parsing validation

- Add TestParseURI_ValidURIs covering 6 valid URI formats
- Add TestParseURI_InvalidURIs covering 7 error cases
- Validates scheme, resource type, name format, path segments
- Ensures proper error messages for invalid URIs

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: File Operation Detection Tests

**Goal:** Test that `ComputeFileOperations` correctly detects ADD, MODIFY, REMOVE, RENAME, and UNCHANGED operations.

**Files:**
- Create: `internal/files/operations_test.go`
- Reference: `internal/files/operations.go` (implementation)

### Step 1: Create test file with package and imports

```bash
touch internal/files/operations_test.go
```

Add:
```go
package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

### Step 2: Define test FileModel struct

Add helper type to `internal/files/operations_test.go`:

```go
// TestFileModel is a simplified file model for testing
type TestFileModel struct {
	Path    string
	Content string
}
```

### Step 3: Write failing test for ADD operation

Add to `internal/files/operations_test.go`:

```go
func TestComputeFileOperations_Add(t *testing.T) {
	tests := []struct {
		name     string
		oldFiles []TestFileModel
		newFiles []TestFileModel
		expected map[string]FileAction
	}{
		{
			name:     "add single new file",
			oldFiles: []TestFileModel{},
			newFiles: []TestFileModel{
				{Path: "hello.txt", Content: "hello world"},
			},
			expected: map[string]FileAction{
				"hello.txt": FileActionAdd,
			},
		},
		{
			name: "add multiple new files",
			oldFiles: []TestFileModel{
				{Path: "existing.txt", Content: "existing"},
			},
			newFiles: []TestFileModel{
				{Path: "existing.txt", Content: "existing"},
				{Path: "new1.txt", Content: "new file 1"},
				{Path: "new2.txt", Content: "new file 2"},
			},
			expected: map[string]FileAction{
				"existing.txt": FileActionUnchanged,
				"new1.txt":     FileActionAdd,
				"new2.txt":     FileActionAdd,
			},
		},
		{
			name:     "add nested file",
			oldFiles: []TestFileModel{},
			newFiles: []TestFileModel{
				{Path: "src/main.go", Content: "package main"},
			},
			expected: map[string]FileAction{
				"src/main.go": FileActionAdd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert TestFileModel to actual FileModel type
			oldFiles := convertToFileModels(tt.oldFiles)
			newFiles := convertToFileModels(tt.newFiles)

			result := ComputeFileOperations(oldFiles, newFiles)

			for path, expectedAction := range tt.expected {
				assert.Equal(t, expectedAction, result[path].Action,
					"File %s should have action %s", path, expectedAction)
			}
		})
	}
}

// Helper to convert test models to actual FileModel
func convertToFileModels(testFiles []TestFileModel) []FileModel {
	models := make([]FileModel, len(testFiles))
	for i, tf := range testFiles {
		models[i] = FileModel{
			Path:    tf.Path,
			Content: tf.Content,
		}
	}
	return models
}
```

### Step 4: Run test to verify it fails

```bash
cd internal/files
go test -v -run TestComputeFileOperations_Add
```

**Expected:** FAIL - Type errors or logic errors

### Step 5: Check FileModel and FileAction types

```bash
grep -n "type FileModel" *.go
grep -n "type FileAction" *.go
```

Adjust test code to match actual types in the codebase.

### Step 6: Run test to verify it passes

```bash
go test -v -run TestComputeFileOperations_Add
```

**Expected:** PASS

### Step 7: Write tests for MODIFY operation

Add to `internal/files/operations_test.go`:

```go
func TestComputeFileOperations_Modify(t *testing.T) {
	tests := []struct {
		name     string
		oldFiles []TestFileModel
		newFiles []TestFileModel
		expected map[string]FileAction
	}{
		{
			name: "modify content only",
			oldFiles: []TestFileModel{
				{Path: "config.txt", Content: "old config"},
			},
			newFiles: []TestFileModel{
				{Path: "config.txt", Content: "new config"},
			},
			expected: map[string]FileAction{
				"config.txt": FileActionModify,
			},
		},
		{
			name: "modify multiple files",
			oldFiles: []TestFileModel{
				{Path: "file1.txt", Content: "content1"},
				{Path: "file2.txt", Content: "content2"},
			},
			newFiles: []TestFileModel{
				{Path: "file1.txt", Content: "updated1"},
				{Path: "file2.txt", Content: "updated2"},
			},
			expected: map[string]FileAction{
				"file1.txt": FileActionModify,
				"file2.txt": FileActionModify,
			},
		},
		{
			name: "modify one, keep one unchanged",
			oldFiles: []TestFileModel{
				{Path: "changed.txt", Content: "old"},
				{Path: "unchanged.txt", Content: "same"},
			},
			newFiles: []TestFileModel{
				{Path: "changed.txt", Content: "new"},
				{Path: "unchanged.txt", Content: "same"},
			},
			expected: map[string]FileAction{
				"changed.txt":   FileActionModify,
				"unchanged.txt": FileActionUnchanged,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldFiles := convertToFileModels(tt.oldFiles)
			newFiles := convertToFileModels(tt.newFiles)

			result := ComputeFileOperations(oldFiles, newFiles)

			for path, expectedAction := range tt.expected {
				assert.Equal(t, expectedAction, result[path].Action,
					"File %s should have action %s", path, expectedAction)
			}
		})
	}
}
```

### Step 8: Run MODIFY tests

```bash
go test -v -run TestComputeFileOperations_Modify
```

**Expected:** PASS

### Step 9: Write tests for REMOVE operation

Add to `internal/files/operations_test.go`:

```go
func TestComputeFileOperations_Remove(t *testing.T) {
	tests := []struct {
		name     string
		oldFiles []TestFileModel
		newFiles []TestFileModel
		expected map[string]FileAction
	}{
		{
			name: "remove single file",
			oldFiles: []TestFileModel{
				{Path: "delete.txt", Content: "will be removed"},
			},
			newFiles: []TestFileModel{},
			expected: map[string]FileAction{
				"delete.txt": FileActionRemove,
			},
		},
		{
			name: "remove multiple files",
			oldFiles: []TestFileModel{
				{Path: "keep.txt", Content: "keep this"},
				{Path: "delete1.txt", Content: "remove this"},
				{Path: "delete2.txt", Content: "remove this too"},
			},
			newFiles: []TestFileModel{
				{Path: "keep.txt", Content: "keep this"},
			},
			expected: map[string]FileAction{
				"keep.txt":    FileActionUnchanged,
				"delete1.txt": FileActionRemove,
				"delete2.txt": FileActionRemove,
			},
		},
		{
			name: "remove nested file",
			oldFiles: []TestFileModel{
				{Path: "src/old/deprecated.go", Content: "old code"},
			},
			newFiles: []TestFileModel{},
			expected: map[string]FileAction{
				"src/old/deprecated.go": FileActionRemove,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldFiles := convertToFileModels(tt.oldFiles)
			newFiles := convertToFileModels(tt.newFiles)

			result := ComputeFileOperations(oldFiles, newFiles)

			for path, expectedAction := range tt.expected {
				assert.Equal(t, expectedAction, result[path].Action,
					"File %s should have action %s", path, expectedAction)
			}
		})
	}
}
```

### Step 10: Run REMOVE tests

```bash
go test -v -run TestComputeFileOperations_Remove
```

**Expected:** PASS

### Step 11: Write tests for RENAME operation

Add to `internal/files/operations_test.go`:

```go
func TestComputeFileOperations_Rename(t *testing.T) {
	tests := []struct {
		name     string
		oldFiles []TestFileModel
		newFiles []TestFileModel
		expected map[string]FileAction
	}{
		{
			name: "rename file with same content",
			oldFiles: []TestFileModel{
				{Path: "old-name.txt", Content: "same content"},
			},
			newFiles: []TestFileModel{
				{Path: "new-name.txt", Content: "same content"},
			},
			expected: map[string]FileAction{
				"new-name.txt": FileActionRename,
			},
		},
		{
			name: "rename detects content match",
			oldFiles: []TestFileModel{
				{Path: "config.yaml", Content: "port: 8080\nhost: localhost"},
			},
			newFiles: []TestFileModel{
				{Path: "config.yml", Content: "port: 8080\nhost: localhost"},
			},
			expected: map[string]FileAction{
				"config.yml": FileActionRename,
			},
		},
		{
			name: "rename in nested path",
			oldFiles: []TestFileModel{
				{Path: "src/utils/helpers.go", Content: "package utils"},
			},
			newFiles: []TestFileModel{
				{Path: "src/utils/util.go", Content: "package utils"},
			},
			expected: map[string]FileAction{
				"src/utils/util.go": FileActionRename,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldFiles := convertToFileModels(tt.oldFiles)
			newFiles := convertToFileModels(tt.newFiles)

			result := ComputeFileOperations(oldFiles, newFiles)

			for path, expectedAction := range tt.expected {
				assert.Equal(t, expectedAction, result[path].Action,
					"File %s should have action %s", path, expectedAction)
			}
		})
	}
}
```

### Step 12: Run RENAME tests

```bash
go test -v -run TestComputeFileOperations_Rename
```

**Expected:** PASS if rename detection works correctly, FAIL if logic has bugs

### Step 13: Write tests for UNCHANGED operation

Add to `internal/files/operations_test.go`:

```go
func TestComputeFileOperations_Unchanged(t *testing.T) {
	tests := []struct {
		name     string
		oldFiles []TestFileModel
		newFiles []TestFileModel
		expected map[string]FileAction
	}{
		{
			name: "single unchanged file",
			oldFiles: []TestFileModel{
				{Path: "static.txt", Content: "never changes"},
			},
			newFiles: []TestFileModel{
				{Path: "static.txt", Content: "never changes"},
			},
			expected: map[string]FileAction{
				"static.txt": FileActionUnchanged,
			},
		},
		{
			name: "multiple unchanged files",
			oldFiles: []TestFileModel{
				{Path: "file1.txt", Content: "content1"},
				{Path: "file2.txt", Content: "content2"},
				{Path: "file3.txt", Content: "content3"},
			},
			newFiles: []TestFileModel{
				{Path: "file1.txt", Content: "content1"},
				{Path: "file2.txt", Content: "content2"},
				{Path: "file3.txt", Content: "content3"},
			},
			expected: map[string]FileAction{
				"file1.txt": FileActionUnchanged,
				"file2.txt": FileActionUnchanged,
				"file3.txt": FileActionUnchanged,
			},
		},
		{
			name: "unchanged with exact whitespace match",
			oldFiles: []TestFileModel{
				{Path: "code.py", Content: "def hello():\n    print('hello')\n"},
			},
			newFiles: []TestFileModel{
				{Path: "code.py", Content: "def hello():\n    print('hello')\n"},
			},
			expected: map[string]FileAction{
				"code.py": FileActionUnchanged,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldFiles := convertToFileModels(tt.oldFiles)
			newFiles := convertToFileModels(tt.newFiles)

			result := ComputeFileOperations(oldFiles, newFiles)

			for path, expectedAction := range tt.expected {
				assert.Equal(t, expectedAction, result[path].Action,
					"File %s should have action %s", path, expectedAction)
			}
		})
	}
}
```

### Step 14: Run UNCHANGED tests

```bash
go test -v -run TestComputeFileOperations_Unchanged
```

**Expected:** PASS

### Step 15: Run all file operation tests

```bash
go test -v ./internal/files/
```

**Expected:** PASS - All tests pass

### Step 16: Commit file operation tests

```bash
git add internal/files/operations_test.go
git commit -m "test(files): add table-driven tests for file operation detection

- Add TestComputeFileOperations_Add (3 test cases)
- Add TestComputeFileOperations_Modify (3 test cases)
- Add TestComputeFileOperations_Remove (3 test cases)
- Add TestComputeFileOperations_Rename (3 test cases)
- Add TestComputeFileOperations_Unchanged (3 test cases)
- Total: 15 test cases covering all file operations
- Validates core diff computation logic

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: File Merge Precedence Tests

**Goal:** Verify that file merge follows correct precedence hierarchy: Stack → Kit → Feature → Project.

**Files:**
- Create: `internal/files/merger_test.go`
- Reference: `internal/files/merger.go` (implementation)
- Reference: `CLAUDE.md` (precedence documentation)

### Step 1: Create test file with package and imports

```bash
touch internal/files/merger_test.go
```

Add:
```go
package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)
```

### Step 2: Write test for project overrides all

Add to `internal/files/merger_test.go`:

```go
func TestMergeFiles_ProjectOverridesAll(t *testing.T) {
	tests := []struct {
		name            string
		stackFiles      map[string]TestFileModel
		kitFiles        map[string]TestFileModel
		featureFiles    map[string]TestFileModel
		projectFiles    map[string]TestFileModel
		expectedContent string
		filePath        string
	}{
		{
			name: "project overrides feature, kit, and stack",
			stackFiles: map[string]TestFileModel{
				"config.txt": {Path: "config.txt", Content: "from stack"},
			},
			kitFiles: map[string]TestFileModel{
				"config.txt": {Path: "config.txt", Content: "from kit"},
			},
			featureFiles: map[string]TestFileModel{
				"config.txt": {Path: "config.txt", Content: "from feature"},
			},
			projectFiles: map[string]TestFileModel{
				"config.txt": {Path: "config.txt", Content: "from project"},
			},
			filePath:        "config.txt",
			expectedContent: "from project",
		},
		{
			name: "project wins for multiple files",
			stackFiles: map[string]TestFileModel{
				"file1.txt": {Path: "file1.txt", Content: "stack 1"},
				"file2.txt": {Path: "file2.txt", Content: "stack 2"},
			},
			kitFiles: map[string]TestFileModel{
				"file1.txt": {Path: "file1.txt", Content: "kit 1"},
				"file2.txt": {Path: "file2.txt", Content: "kit 2"},
			},
			featureFiles: map[string]TestFileModel{
				"file1.txt": {Path: "file1.txt", Content: "feature 1"},
			},
			projectFiles: map[string]TestFileModel{
				"file1.txt": {Path: "file1.txt", Content: "project 1"},
				"file2.txt": {Path: "file2.txt", Content: "project 2"},
			},
			filePath:        "file1.txt",
			expectedContent: "project 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to actual FileModel maps
			stackFiles := convertMapToFileModels(tt.stackFiles)
			kitFiles := convertMapToFileModels(tt.kitFiles)
			featureFiles := convertMapToFileModels(tt.featureFiles)
			projectFiles := convertMapToFileModels(tt.projectFiles)

			merged := MergeFiles(stackFiles, kitFiles, featureFiles, projectFiles)

			require.Contains(t, merged, tt.filePath, "Merged result should contain file")
			assert.Equal(t, tt.expectedContent, merged[tt.filePath].Content,
				"Project file should override all other sources")
		})
	}
}

// Helper to convert map of test models to map of FileModels
func convertMapToFileModels(testFiles map[string]TestFileModel) map[string]FileModel {
	models := make(map[string]FileModel)
	for path, tf := range testFiles {
		models[path] = FileModel{
			Path:    tf.Path,
			Content: tf.Content,
		}
	}
	return models
}
```

### Step 3: Run test to verify it fails

```bash
cd internal/files
go test -v -run TestMergeFiles_ProjectOverridesAll
```

**Expected:** FAIL - Function signature mismatch or logic errors

### Step 4: Check MergeFiles function signature

```bash
grep -A5 "func MergeFiles" merger.go
```

Adjust test to match actual function signature. The function might be named differently or have different parameters.

### Step 5: Fix test and run again

```bash
go test -v -run TestMergeFiles_ProjectOverridesAll
```

**Expected:** PASS

### Step 6: Write test for feature overrides kit and stack

Add to `internal/files/merger_test.go`:

```go
func TestMergeFiles_FeatureOverridesKitAndStack(t *testing.T) {
	tests := []struct {
		name            string
		stackFiles      map[string]TestFileModel
		kitFiles        map[string]TestFileModel
		featureFiles    map[string]TestFileModel
		projectFiles    map[string]TestFileModel
		expectedContent string
		filePath        string
	}{
		{
			name: "feature overrides kit and stack when no project override",
			stackFiles: map[string]TestFileModel{
				"config.txt": {Path: "config.txt", Content: "from stack"},
			},
			kitFiles: map[string]TestFileModel{
				"config.txt": {Path: "config.txt", Content: "from kit"},
			},
			featureFiles: map[string]TestFileModel{
				"config.txt": {Path: "config.txt", Content: "from feature"},
			},
			projectFiles:    map[string]TestFileModel{},
			filePath:        "config.txt",
			expectedContent: "from feature",
		},
		{
			name: "feature wins over multiple layers",
			stackFiles: map[string]TestFileModel{
				"setup.sh": {Path: "setup.sh", Content: "stack setup"},
			},
			kitFiles: map[string]TestFileModel{
				"setup.sh": {Path: "setup.sh", Content: "kit setup"},
			},
			featureFiles: map[string]TestFileModel{
				"setup.sh": {Path: "setup.sh", Content: "feature setup"},
			},
			projectFiles:    map[string]TestFileModel{},
			filePath:        "setup.sh",
			expectedContent: "feature setup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stackFiles := convertMapToFileModels(tt.stackFiles)
			kitFiles := convertMapToFileModels(tt.kitFiles)
			featureFiles := convertMapToFileModels(tt.featureFiles)
			projectFiles := convertMapToFileModels(tt.projectFiles)

			merged := MergeFiles(stackFiles, kitFiles, featureFiles, projectFiles)

			require.Contains(t, merged, tt.filePath)
			assert.Equal(t, tt.expectedContent, merged[tt.filePath].Content,
				"Feature should override kit and stack")
		})
	}
}
```

### Step 7: Run feature precedence tests

```bash
go test -v -run TestMergeFiles_FeatureOverridesKitAndStack
```

**Expected:** PASS

### Step 8: Write test for kit overrides stack

Add to `internal/files/merger_test.go`:

```go
func TestMergeFiles_KitOverridesStack(t *testing.T) {
	tests := []struct {
		name            string
		stackFiles      map[string]TestFileModel
		kitFiles        map[string]TestFileModel
		featureFiles    map[string]TestFileModel
		projectFiles    map[string]TestFileModel
		expectedContent string
		filePath        string
	}{
		{
			name: "kit overrides stack when no feature or project",
			stackFiles: map[string]TestFileModel{
				"Makefile": {Path: "Makefile", Content: "stack makefile"},
			},
			kitFiles: map[string]TestFileModel{
				"Makefile": {Path: "Makefile", Content: "kit makefile"},
			},
			featureFiles:    map[string]TestFileModel{},
			projectFiles:    map[string]TestFileModel{},
			filePath:        "Makefile",
			expectedContent: "kit makefile",
		},
		{
			name: "kit provides language-specific override",
			stackFiles: map[string]TestFileModel{
				".gitignore": {Path: ".gitignore", Content: "*.log"},
			},
			kitFiles: map[string]TestFileModel{
				".gitignore": {Path: ".gitignore", Content: "*.log\n__pycache__/\n*.pyc"},
			},
			featureFiles:    map[string]TestFileModel{},
			projectFiles:    map[string]TestFileModel{},
			filePath:        ".gitignore",
			expectedContent: "*.log\n__pycache__/\n*.pyc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stackFiles := convertMapToFileModels(tt.stackFiles)
			kitFiles := convertMapToFileModels(tt.kitFiles)
			featureFiles := convertMapToFileModels(tt.featureFiles)
			projectFiles := convertMapToFileModels(tt.projectFiles)

			merged := MergeFiles(stackFiles, kitFiles, featureFiles, projectFiles)

			require.Contains(t, merged, tt.filePath)
			assert.Equal(t, tt.expectedContent, merged[tt.filePath].Content,
				"Kit should override stack")
		})
	}
}
```

### Step 9: Run kit precedence tests

```bash
go test -v -run TestMergeFiles_KitOverridesStack
```

**Expected:** PASS

### Step 10: Write test for stack is lowest precedence

Add to `internal/files/merger_test.go`:

```go
func TestMergeFiles_StackLowestPrecedence(t *testing.T) {
	tests := []struct {
		name            string
		stackFiles      map[string]TestFileModel
		kitFiles        map[string]TestFileModel
		featureFiles    map[string]TestFileModel
		projectFiles    map[string]TestFileModel
		expectedContent string
		filePath        string
	}{
		{
			name: "stack used when no overrides exist",
			stackFiles: map[string]TestFileModel{
				"README.md": {Path: "README.md", Content: "# Stack README"},
			},
			kitFiles:        map[string]TestFileModel{},
			featureFiles:    map[string]TestFileModel{},
			projectFiles:    map[string]TestFileModel{},
			filePath:        "README.md",
			expectedContent: "# Stack README",
		},
		{
			name: "stack provides base files",
			stackFiles: map[string]TestFileModel{
				"base1.txt": {Path: "base1.txt", Content: "base content 1"},
				"base2.txt": {Path: "base2.txt", Content: "base content 2"},
			},
			kitFiles:     map[string]TestFileModel{},
			featureFiles: map[string]TestFileModel{},
			projectFiles: map[string]TestFileModel{},
			filePath:     "base1.txt",
			expectedContent: "base content 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stackFiles := convertMapToFileModels(tt.stackFiles)
			kitFiles := convertMapToFileModels(tt.kitFiles)
			featureFiles := convertMapToFileModels(tt.featureFiles)
			projectFiles := convertMapToFileModels(tt.projectFiles)

			merged := MergeFiles(stackFiles, kitFiles, featureFiles, projectFiles)

			require.Contains(t, merged, tt.filePath)
			assert.Equal(t, tt.expectedContent, merged[tt.filePath].Content,
				"Stack should be used when no overrides")
		})
	}
}
```

### Step 11: Run stack precedence tests

```bash
go test -v -run TestMergeFiles_StackLowestPrecedence
```

**Expected:** PASS

### Step 12: Write test for files from different sources merge correctly

Add to `internal/files/merger_test.go`:

```go
func TestMergeFiles_DifferentFilesDontConflict(t *testing.T) {
	tests := []struct {
		name          string
		stackFiles    map[string]TestFileModel
		kitFiles      map[string]TestFileModel
		featureFiles  map[string]TestFileModel
		projectFiles  map[string]TestFileModel
		expectedFiles []string
	}{
		{
			name: "all sources contribute unique files",
			stackFiles: map[string]TestFileModel{
				"stack.txt": {Path: "stack.txt", Content: "from stack"},
			},
			kitFiles: map[string]TestFileModel{
				"kit.txt": {Path: "kit.txt", Content: "from kit"},
			},
			featureFiles: map[string]TestFileModel{
				"feature.txt": {Path: "feature.txt", Content: "from feature"},
			},
			projectFiles: map[string]TestFileModel{
				"project.txt": {Path: "project.txt", Content: "from project"},
			},
			expectedFiles: []string{"stack.txt", "kit.txt", "feature.txt", "project.txt"},
		},
		{
			name: "merge creates superset of all unique files",
			stackFiles: map[string]TestFileModel{
				"README.md":  {Path: "README.md", Content: "readme"},
				"Makefile":   {Path: "Makefile", Content: "makefile"},
			},
			kitFiles: map[string]TestFileModel{
				".gitignore": {Path: ".gitignore", Content: "*.pyc"},
				"setup.py":   {Path: "setup.py", Content: "setup"},
			},
			featureFiles: map[string]TestFileModel{
				"logging.py": {Path: "logging.py", Content: "logging"},
			},
			projectFiles: map[string]TestFileModel{
				"main.py": {Path: "main.py", Content: "main"},
			},
			expectedFiles: []string{"README.md", "Makefile", ".gitignore", "setup.py", "logging.py", "main.py"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stackFiles := convertMapToFileModels(tt.stackFiles)
			kitFiles := convertMapToFileModels(tt.kitFiles)
			featureFiles := convertMapToFileModels(tt.featureFiles)
			projectFiles := convertMapToFileModels(tt.projectFiles)

			merged := MergeFiles(stackFiles, kitFiles, featureFiles, projectFiles)

			assert.Len(t, merged, len(tt.expectedFiles),
				"Merged result should contain all unique files")

			for _, expectedFile := range tt.expectedFiles {
				assert.Contains(t, merged, expectedFile,
					"Should contain file from source: %s", expectedFile)
			}
		})
	}
}
```

### Step 13: Run file merging tests

```bash
go test -v -run TestMergeFiles_DifferentFilesDontConflict
```

**Expected:** PASS

### Step 14: Run all merge tests

```bash
go test -v ./internal/files/
```

**Expected:** PASS - All tests pass

### Step 15: Commit file merge precedence tests

```bash
git add internal/files/merger_test.go
git commit -m "test(files): add table-driven tests for file merge precedence

- Add TestMergeFiles_ProjectOverridesAll (2 test cases)
- Add TestMergeFiles_FeatureOverridesKitAndStack (2 test cases)
- Add TestMergeFiles_KitOverridesStack (2 test cases)
- Add TestMergeFiles_StackLowestPrecedence (2 test cases)
- Add TestMergeFiles_DifferentFilesDontConflict (2 test cases)
- Total: 10 test cases verifying precedence hierarchy
- Validates critical merge correctness guarantees

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Documentation and CI Integration

**Goal:** Document the new tests and ensure they run in CI.

**Files:**
- Modify: `README.md` or `CLAUDE.md` (testing section)
- Check: `.github/workflows/` or CI configuration

### Step 1: Update testing documentation

Add to `CLAUDE.md` in "Testing Philosophy" section:

```markdown
## Unit Tests

The provider includes table-driven unit tests for core business logic:

**URI Validation** (`internal/uri/scanner_test.go`)
- Validates `tofukit://` URI parsing
- Tests valid formats: feature, stack, file, kit references
- Tests error cases: invalid scheme, empty names, unknown types

**File Operations** (`internal/files/operations_test.go`)
- Tests diff computation logic (ADD, MODIFY, REMOVE, RENAME, UNCHANGED)
- Validates rename detection via content matching
- Ensures correct action assignment for file changes

**File Merge Precedence** (`internal/files/merger_test.go`)
- Tests precedence hierarchy: Stack → Kit → Feature → Project
- Validates project always wins conflicts
- Ensures unique files from all sources are preserved

**Running Unit Tests:**
```bash
# All tests
go test ./...

# Specific package
go test ./internal/files/ -v

# Specific test
go test ./internal/uri/ -v -run TestParseURI_ValidURIs

# With coverage
go test ./internal/... -cover
```
```

### Step 2: Verify tests run in package mode

```bash
cd /Users/dmitry/dev/dimmkirr/terraform-provider-tofukit
go test ./internal/... -v
```

**Expected:** All new unit tests pass

### Step 3: Check test coverage

```bash
go test ./internal/... -cover
```

**Expected:** Coverage report showing percentage for each package

### Step 4: Add coverage report generation

```bash
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

Open `coverage.html` in browser to see visual coverage report.

### Step 5: Check if Taskfile has test command

```bash
grep -A5 "test:" Taskfile.yml
```

If test task exists, verify it includes unit tests:

```bash
task test
```

### Step 6: Commit documentation updates

```bash
git add CLAUDE.md
git commit -m "docs: add unit testing documentation and coverage guidance

- Document new unit tests for URI, file operations, and merge precedence
- Add commands for running unit tests and generating coverage reports
- Explain table-driven testing approach
- Link to test files in internal/ packages

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Verification Checklist

After completing all tasks, verify:

- [ ] All unit tests pass: `go test ./internal/... -v`
- [ ] No compilation errors: `go build ./...`
- [ ] Coverage is reasonable: `go test ./internal/... -cover` (aim for >80% on tested packages)
- [ ] Tests are fast: `time go test ./internal/...` (should be <5 seconds)
- [ ] Integration tests still pass: `task test`
- [ ] Documentation is updated
- [ ] All changes committed with descriptive messages

---

## Success Criteria

**Unit Tests Added:**
- ✅ URI validation: 13+ test cases (6 valid, 7 invalid)
- ✅ File operations: 15 test cases (3 per operation type)
- ✅ File merge precedence: 10 test cases (covering all precedence rules)

**Quality Metrics:**
- ✅ All tests follow table-driven pattern
- ✅ Clear test case names describing what's tested
- ✅ Fast execution (no external dependencies)
- ✅ Good coverage of edge cases and error conditions

**Documentation:**
- ✅ Testing section updated in CLAUDE.md
- ✅ Examples of running unit tests provided
- ✅ Coverage measurement documented

---

## Notes for Implementation

**Type Compatibility:**
The test code uses `TestFileModel` as a simplified structure. You'll need to:
1. Check actual types in the codebase (`FileModel`, `FileAction`, etc.)
2. Adjust helper functions to match
3. May need to import additional types or packages

**Function Signatures:**
Functions like `MergeFiles`, `ComputeFileOperations`, `ParseURI` may have different signatures than assumed. Check the actual implementation and adjust test code accordingly.

**Testing Best Practices:**
- Keep tests independent (no shared state)
- Use descriptive test case names
- Test happy path first, then edge cases, then errors
- Use `require` for setup assertions, `assert` for test assertions

**Performance:**
Unit tests should be fast (<100ms each). If tests are slow:
- Check for accidental file I/O
- Look for expensive operations in test setup
- Consider using mock objects if needed

**Coverage Goals:**
- URI scanner: aim for 90%+ coverage
- File operations: aim for 85%+ coverage
- File merger: aim for 80%+ coverage
