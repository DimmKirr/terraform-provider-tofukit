# Prompt Generation Test Strategy

**Date:** 2025-11-14
**Status:** Approved
**Goal:** Add fast prompt validation tests alongside e2e tests to improve CI speed and developer experience

## Problem Statement

Current test suite has two pain points:
1. **Slow feedback** - E2E tests take 10-30 seconds each due to Claude execution
2. **Limited prompt validation** - Can't verify prompt structure without running full Claude execution

Recent BUG-008 revealed that prompt generation bugs (missing `prompt` field in features) can go undetected because we only validate final file output, not the prompt sent to Claude.

## Solution: Dual Test Strategy

Maintain **two complementary test types** for comprehensive coverage:

### 1. E2E Tests (Existing)
- Run full `tofu apply` with Claude execution
- Verify actual file output, drift detection, verification logic
- Prove end-to-end functionality works
- Runtime: 10-30 seconds per test

### 2. Prompt Generation Tests (New)
- Run `tofu apply` with `dry_run=true` (skips Claude execution)
- Verify prompt JSON structure, instructions, metadata
- Catch prompt generation bugs immediately
- Runtime: <1 second per test

**Both types are valuable:**
- E2E tests prove behavior works
- Prompt tests provide fast feedback on prompt structure

## Architecture

### Naming Convention: Suffix-Based Pattern

For every e2e test, create a prompt generation variant:

```
Pattern: Test<Scenario>Success → Test<Scenario>GeneratePromptSuccess

Example:
  TestResourceProjectInlineFileCreateSuccess              (e2e, 10s)
  TestResourceProjectInlineFileCreateGeneratePromptSuccess (prompt, <1s)
```

**Benefits:**
- Clear 1:1 mapping between test types
- Easy to find corresponding tests
- Consistent pattern across codebase
- Intuitive for new contributors

### File Organization

**No new files created** - Add prompt tests to existing test files:

```
resource_project_inline_file_test.go
├─ E2E Tests (13 existing)
│  ├─ TestResourceProjectInlineFileCreateSuccess
│  ├─ TestResourceProjectInlineFileRenameSuccess
│  └─ ...
├─ Prompt Generation Tests (16 new)
│  ├─ TestResourceProjectInlineFileCreateGeneratePromptSuccess
│  ├─ TestResourceProjectInlineFileRenameGeneratePromptSuccess
│  └─ ...

resource_feature_test.go
├─ E2E Tests (6 existing)
├─ Prompt Generation Tests (4 existing + 2 new)

resource_file_test.go
├─ E2E Tests (4 existing)
├─ Prompt Generation Tests (2 new)
```

**Example tests stay unchanged:**
- `e2e_project_example_*.go` files remain e2e only
- They validate that complete examples work end-to-end
- Not in scope for prompt generation duplication

### Provider Configuration

All prompt generation tests use identical provider config:

```hcl
provider "tofukit" {
  output_path = "output"
  debug       = true   # Writes prompt JSON to .debug/
  dry_run     = true   # Skips LLM execution
}
```

**Key difference from e2e tests:** Only the `dry_run = true` flag changes.

### Test Setup Pattern

Use standardized helper functions (already implemented in `helpers_test.go`):

```go
func TestResourceProjectInlineFileCreateGeneratePromptSuccess(t *testing.T) {
    testDir := createTestDirectory(t, "TestResourceProjectInlineFileCreateGeneratePromptSuccess")

    // Write terraform config (same as e2e test)
    config := `...` // Identical to e2e test config
    configPath := filepath.Join(testDir, "project.tofu")
    err := os.WriteFile(configPath, []byte(config), 0644)
    require.NoError(t, err)

    // Setup Terraform and run apply (with TF_LOG=INFO)
    iacTool := setupTerraform(t, testDir)
    runTerraformApply(t, iacTool, testDir)

    // Verify prompt JSON (focused verification)
    // ... category-specific assertions
}
```

**Helpers provide:**
- Auto-detect tofu/terraform
- Enable `TF_LOG=INFO` (shows prompt path in logs)
- Consistent error handling
- Clean test output

## Verification Strategy

### Focused Verification Principle

Each test category verifies **specific prompt aspects** rather than entire prompt structure:

| Test Category | What to Verify | JSON Path | Expected Value |
|--------------|----------------|-----------|----------------|
| **File Operations** | Operation instructions | `.request.specification.files[path].instructions[0].prompt` | "Create new file...", "Rename file...", etc. |
| **Feature System** | Feature metadata completeness | `.request.specification._project_context.features[n]` | `{name, prompt, files, kits}` all present |
| **Drift Detection** | Drift warnings | `.request.specification.files[path].instructions` | Contains "⚠️ DRIFT DETECTED" |
| **Verification** | Verification commands | `.request.specification.requirements[n].verifications` | Command structure present |
| **File Resources** | File specifications | `.request.specification.files[path]` | Content or instructions present |

### Common Assertion Pattern

All tests follow this structure:

```go
// 1. Locate and read prompt JSON
debugFiles, err := filepath.Glob(filepath.Join(testDir, "output", ".debug", "claude-prompt-attempt1-*.json"))
require.NoError(t, err)
require.NotEmpty(t, debugFiles, "Should have prompt JSON file")

jsonData, err := os.ReadFile(debugFiles[0])
require.NoError(t, err, "Failed to read prompt JSON")

// 2. Parse JSON
var promptJSON map[string]interface{}
err = json.Unmarshal(jsonData, &promptJSON)
require.NoError(t, err, "Failed to parse prompt JSON")

// 3. Navigate to relevant section
request := promptJSON["request"].(map[string]interface{})
specification := request["specification"].(map[string]interface{})

// 4. Category-specific focused verification
// Example for file operations:
files := specification["files"].(map[string]interface{})
helloFile := files["hello.txt"].(map[string]interface{})
instructions := helloFile["instructions"].([]interface{})
instruction0 := instructions[0].(map[string]interface{})

assert.Contains(t, instruction0["prompt"], "Create new file 'hello.txt'",
    "Should have create instruction for new file")
```

### Verification Examples

**File Operation Test:**
```go
// Verify rename instruction
files := specification["files"].(map[string]interface{})
renamedFile := files["hello2.txt"].(map[string]interface{})
instruction := renamedFile["instructions"].([]interface{})[0].(map[string]interface{})

assert.Contains(t, instruction["prompt"], "Rename file from 'hello.txt' to 'hello2.txt'",
    "Should have explicit rename instruction")
```

**Feature Test:**
```go
// Verify feature metadata
projectContext := specification["_project_context"].(map[string]interface{})
features := projectContext["features"].([]interface{})

require.NotEmpty(t, features, "Should have features in project context")

feature0 := features[0].(map[string]interface{})
assert.Equal(t, "hello", feature0["name"])
assert.NotNil(t, feature0["prompt"], "Feature should have prompt field")
assert.NotEmpty(t, feature0["files"], "Feature should have files array")
```

**Drift Detection Test:**
```go
// Verify drift warning instruction
files := specification["files"].(map[string]interface{})
driftedFile := files["config.txt"].(map[string]interface{})
instructions := driftedFile["instructions"].([]interface{})

hasDriftWarning := false
for _, inst := range instructions {
    instMap := inst.(map[string]interface{})
    if strings.Contains(instMap["prompt"].(string), "DRIFT DETECTED") {
        hasDriftWarning = true
        break
    }
}
assert.True(t, hasDriftWarning, "Should have drift warning instruction")
```

## Implementation Plan

### Phase 1: Infrastructure ✅ COMPLETE

- ✅ `dry_run` provider flag added
- ✅ Terraform helper functions in `helpers_test.go`
- ✅ 4 example prompt tests in `resource_feature_test.go`

### Phase 2: File Operations Tests (16 tests)

**Target:** `resource_project_inline_file_test.go`

**Tests to add:**
1. `TestResourceProjectInlineFileCreateGeneratePromptSuccess`
   - Verify: "Create new file" instruction present

2. `TestResourceProjectInlineFileAddMiddleFileGeneratePromptSuccess`
   - Verify: Multiple files with correct instructions

3. `TestResourceProjectInlineFileRemovalGeneratePromptSuccess`
   - Verify: "Delete file" instruction present

4. `TestResourceProjectInlineFileRenameGeneratePromptSuccess`
   - Verify: "Rename file from X to Y" instruction

5. `TestResourceProjectInlineFileNestedCreateGeneratePromptSuccess`
   - Verify: Parent directory creation in instructions

6. `TestResourceProjectInlineFileNestedAddGeneratePromptSuccess`
   - Verify: Nested path in file specifications

7. `TestResourceProjectInlineFileNestedDeeperNestingGeneratePromptSuccess`
   - Verify: Deep nested paths handled correctly

8. `TestResourceProjectInlineFileNestedRemovalGeneratePromptSuccess`
   - Verify: Empty directory cleanup in instructions

9. `TestResourceProjectInlineFileVerificationGeneratePromptSuccess`
   - Verify: Verification commands in requirements

10. `TestResourceProjectInlineFileVerificationFailureGeneratePromptSuccess`
    - Verify: Verification commands structured correctly

11. `TestResourceProjectInlineFileVerificationRetryGeneratePromptSuccess`
    - Verify: Retry logic doesn't affect prompt structure

12. `TestResourceProjectInlineFileInstructionRenameGeneratePromptSuccess`
    - Verify: Generated files maintain instruction structure

13. `TestResourceProjectInlineFileOrderingConsistencyGeneratePromptSuccess`
    - Verify: File ordering in prompt matches specification

14. `TestResourceProjectDriftDetection_StaticFileGeneratePromptSuccess`
    - Verify: Drift warning in instructions

15. `TestResourceProjectDriftDetection_DeletedFileGeneratePromptSuccess`
    - Verify: Restore instruction for deleted files

16. `TestResourceProjectDriftDetection_MultipleFilesGeneratePromptSuccess`
    - Verify: Multiple drift warnings handled

### Phase 3: File Resource Tests (2 tests)

**Target:** `resource_file_test.go`

**Tests to add:**
1. `TestResourceFile_CreateGeneratePromptSuccess`
   - Verify: File resource content in files specification

2. `TestResourceFile_DriftDetectionGeneratePromptSuccess`
   - Verify: File resource drift warnings

### Phase 4: Additional Feature Tests (2 tests)

**Target:** `resource_feature_test.go`

**Tests to add:**
1. `TestResourceFeatureCreateGeneratePromptSuccess`
   - Verify: Standalone feature resource metadata

2. `TestStackFeaturesFromModuleGeneratePromptSuccess`
   - Verify: Stack module features in project context

### Summary

**Total new tests:** 20
**Existing prompt tests:** 4
**Final prompt test count:** 24

**Distribution:**
- File operations: 16 tests
- Feature system: 6 tests (4 existing + 2 new)
- File resources: 2 tests

## Test Execution

### Local Development

```bash
# Run only prompt tests (fast feedback)
go test -v -run "GeneratePromptSuccess" ./test/ -timeout 1m

# Run specific test category
go test -v -run "InlineFile.*GeneratePromptSuccess" ./test/

# Run both e2e and prompt tests
go test -v ./test/ -timeout 30m
```

### CI Pipeline (Recommended)

```yaml
name: Tests

on: [push, pull_request]

jobs:
  prompt-tests:
    name: Prompt Generation Tests (Fast)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run Prompt Tests
        run: go test -v -run "GeneratePromptSuccess" ./test/ -timeout 2m

  e2e-tests:
    name: E2E Tests (Full)
    runs-on: ubuntu-latest
    # Only run on main branch or PR approval
    if: github.ref == 'refs/heads/main' || github.event.review.state == 'approved'
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run E2E Tests
        run: go test -v ./test/ -timeout 30m
```

**Benefits:**
- Prompt tests run on every push (fast feedback)
- E2E tests run only when merging (thorough validation)
- Developers get <1 minute feedback on PRs

## Maintenance Guidelines

### Adding New Tests

When adding a new feature or behavior:

1. **Write e2e test first** (TDD approach)
   - Proves the behavior works end-to-end
   - Validates actual file output

2. **Copy to prompt generation variant**
   - Same test setup
   - Change: `dry_run = false` → `dry_run = true`
   - Change: File output verification → Prompt JSON verification

3. **Add focused assertion**
   - Verify the specific prompt aspect for this behavior
   - Keep verification simple and targeted

### Updating Existing Tests

When behavior changes:

1. **Update e2e test** to match new behavior
2. **Update prompt test** verification to match new prompt structure
3. **Both should fail/pass together** - if they diverge, investigate

### Test Organization

Keep tests organized in each file:

```go
// ========================
// E2E Tests (Full Execution)
// ========================

func TestResourceProjectInlineFileCreateSuccess(t *testing.T) { ... }
func TestResourceProjectInlineFileAddMiddleFileSuccess(t *testing.T) { ... }
func TestResourceProjectInlineFileRenameSuccess(t *testing.T) { ... }

// ========================
// Prompt Generation Tests (Fast)
// ========================

func TestResourceProjectInlineFileCreateGeneratePromptSuccess(t *testing.T) { ... }
func TestResourceProjectInlineFileAddMiddleFileGeneratePromptSuccess(t *testing.T) { ... }
func TestResourceProjectInlineFileRenameGeneratePromptSuccess(t *testing.T) { ... }
```

## Success Metrics

### Development Experience
- ✅ Prompt bugs caught in <1 second (vs 10-30 seconds)
- ✅ Fast iteration on prompt generation logic
- ✅ Pre-commit validation possible

### CI Pipeline
- ✅ PR feedback in <2 minutes (prompt tests only)
- ✅ Thorough validation on merge (e2e tests)
- ✅ Reduced CI costs (fewer Claude API calls)

### Code Quality
- ✅ Prompt structure regressions caught immediately
- ✅ Better test coverage (structure + behavior)
- ✅ Clear 1:1 mapping between test types

## Related Work

### Completed Infrastructure
- `dry_run` provider flag implementation
- Terraform helper functions with TF_LOG=INFO
- Example prompt tests demonstrating pattern

### Future Enhancements (Out of Scope)
- Automated prompt test generation from e2e tests
- Prompt JSON schema validation
- Diff-based prompt assertions (comparing old vs new prompts)

## References

- **BUG-008:** Features missing prompt field in `_project_context`
- **helpers_test.go:** Standardized Terraform test helpers
- **resource_feature_test.go:** Example prompt generation tests
