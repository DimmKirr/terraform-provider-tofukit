# BUG-008: Features Are Null/Unknown in buildProjectContext()

**Status:** Open
**Priority:** High
**Discovered:** 2025-11-14
**Affects:** All projects using features
**Related:** BUG-006 (diagram generation uses wrong IDs)

## Summary

**UPDATE 2025-11-14:** Fast tests with `dry_run` reveal features ARE included in `_project_context` but with **incomplete metadata** - the `prompt` field is missing.

The `buildProjectContext()` function includes features in the output but does not extract all feature metadata fields. Specifically:
- ✅ Feature `name` is included
- ✅ Feature `files` array is included
- ❌ Feature `prompt` is **MISSING**
- ❌ Feature `constraints` likely **MISSING**
- ❌ Feature `kits` likely **MISSING**

**Impact:** Claude cannot introspect complete feature metadata, leading to:
- Diagram generation using generic names instead of feature prompts
- Lost feature context that should guide LLM generation
- Incomplete feature introspection system

## How to Reproduce

Run the dedicated test that catches this bug:

```bash
go test -v -run "TestResourceFeatureInlineGeneratePromptSuccess" ./test/ -timeout 30s
```

**Expected:** Test should pass with feature `prompt` field present in `_project_context`

**Actual:** Test fails with this error:
```
Error:      	Not equal:
            	expected: string("Add hello greeting capability")
            	actual  : <nil>(<nil>)
Test:       	TestResourceFeatureInlineGeneratePromptSuccess
Messages:   	Feature prompt should match
```

**What the test checks:** After running `tofu apply` with `dry_run=true`, the test reads the generated prompt JSON and verifies:
```go
// Line 742 in resource_feature_test.go
assert.Equal(t, "Add hello greeting capability", feature0["prompt"], "Feature prompt should match")
```

**Actual JSON structure found:**
```json
{
  "_project_context": {
    "features": [
      {
        "files": ["hello.txt"],
        "name": "hello"
        // ❌ MISSING: "prompt" field
      }
    ]
  }
}
```

**Expected JSON structure:**
```json
{
  "_project_context": {
    "features": [
      {
        "name": "hello",
        "prompt": "Add hello greeting capability",
        "files": ["hello.txt"]
      }
    ]
  }
}
```

**Test runtime:** <0.2 seconds (uses `dry_run=true` to skip LLM execution)

**Test location:** `/Users/dmitry/dev/dimmkirr/terraform-provider-tofukit/test/resource_feature_test.go:654`

## Root Cause

**Immediate Cause:** Line 2882 in `buildProjectContext()`:
```go
if !data.Features.IsNull() && !data.Features.IsUnknown() {
    // Process features
}
```

When `data.Features.IsNull()` or `.IsUnknown()` returns TRUE, features processing is skipped entirely.

**Mystery:** `collectFeatureKitIDs()` successfully processes the same features as `basetypes.ObjectValue with 4 attributes`, proving features ARE populated in the data model.

**Theory:** `buildProjectContext()` and `collectFeatureKitIDs()` are called at different times or receive different `data` values, where features are Null in one context but populated in another.

## Evidence

### 1. project_context HAS kits but MISSING features

**Claude prompt JSON shows:**
```json
{
  "kits": [{"id": "tool.drawio"}],  // ✓ Present (added unconditionally at line 3052)
  "project_info": {...}              // ✓ Present
  // ❌ NO "features" field
}
```

### 2. collectFeatureKitIDs() Sees Features Successfully

**Debug logs from `/tmp/tofukit-debug.log`:**
```
=== collectFeatureKitIDs START ===
collectFeatureKitIDs: Features is basetypes.ObjectValue with 4 attributes
collectFeatureKitIDs: Processing feature 'load_balancer' from ObjectValue
collectFeatureKitIDs: Processing feature 'web_server' from ObjectValue
collectFeatureKitIDs: Processing feature 'architecture_diagram' from ObjectValue
collectFeatureKitIDs: Processing feature 'database' from ObjectValue
=== collectFeatureKitIDs END: Total 1 kits collected ===
```

### 3. buildProjectContext() Logs Never Appear

Despite adding extensive debug logging to `buildProjectContext()` and Create()/Update() methods, **ZERO logs appear** from these functions in `/tmp/tofukit-debug.log`.

**Expected logs (never seen):**
- `=== Create() ENTRY ===`
- `=== buildProjectContext START ===`
- `buildProjectContext: Features.IsNull()=... IsUnknown()=...`

**This suggests either:**
- `buildProjectContext()` is not being called
- Logging is broken for Create()/Update() methods
- Features are Null when called

## Code Locations

### buildProjectContext() - Where Features Should Be Added

**File:** `/internal/resources/project.go`

**Lines 2882-3021:** Features processing (CONDITIONAL)
```go
if !data.Features.IsNull() && !data.Features.IsUnknown() {
    underlyingVal := data.Features.UnderlyingValue()

    // Try types.Map
    if featuresMap, ok := underlyingVal.(types.Map); ok {
        // Process features from Map
    } else {
        // Try basetypes.ObjectValue (ADDED IN FIX ATTEMPT)
        if fObj, objOk := underlyingVal.(basetypes.ObjectValue); objOk {
            // Process features from ObjectValue
        }
    }
}
```

**Lines 3028-3053:** Kits processing (UNCONDITIONAL)
```go
// Collect kits metadata
projectKitIDs := extractIDsFromDynamicList(ctx, data.Kits)
featureKitIDs := r.collectFeatureKitIDs(ctx, data)  // This works!
...
if len(uniqueKitIDs) > 0 {
    projectContext["kits"] = kitsMetadata  // Always added
}
```

### Callers of buildProjectContext()

**Create()** - Line 643-650
```go
projectContext := r.buildProjectContext(ctx, data)
if projectContext != nil && len(projectContext) > 0 {
    outputData["_project_context"] = projectContext
}
```

**Update()** - Line 1386-1399
```go
projectContext := r.buildProjectContext(ctx, data)
if projectContext != nil && len(projectContext) > 0 {
    outputData["_project_context"] = projectContext
}
```

### Comparison: collectFeatureKitIDs() - Works Correctly

**Line 3819-3920:** Called from `buildOutputData()`, successfully processes `basetypes.ObjectValue`
```go
func (r *ProjectResourceFinal) collectFeatureKitIDs(...) []string {
    underlyingVal := data.Features.UnderlyingValue()

    if fObj, objOk := underlyingVal.(basetypes.ObjectValue); objOk {
        // This branch executes successfully!
        for featureName, featureValue := range fObj.Attributes() {
            // Processes all 4 features
        }
    }
}
```

## Call Graph Analysis

```
Create() / Update()
│
├── buildOutputDataWithFiles()
│   └── buildOutputData()
│       └── collectFeatureKitIDs() → ✓ Features work here (basetypes.ObjectValue)
│
└── buildProjectContext() → ❌ Features are Null/Unknown here
```

**Question:** Why do features work in `collectFeatureKitIDs()` but not in `buildProjectContext()` when both receive the same `data` parameter?

## Attempted Fix

### What Was Done

Added `basetypes.ObjectValue` handling to `buildProjectContext()` following the exact pattern from:
1. Commit 8c1e0ec (`convertFeaturesToRequirements()`)
2. `collectFeatureKitIDs()` implementation

**Lines 2959-3021:** Handles both `types.Map` AND `basetypes.ObjectValue`

### Why It Didn't Work

Cannot verify the fix works because:
1. `buildProjectContext()` logs never appear
2. Cannot confirm if the function is being called
3. Cannot confirm the actual state of `data.Features` when called
4. The conditional check at line 2882 might be preventing execution entirely

## Debugging Blockers

1. **No Create()/Update() Execution Logs**
   - Added debug logging to Create() entry/exit
   - Added debug logging before/after `buildProjectContext()` call
   - Added debug logging inside `buildProjectContext()`
   - **None of these logs appear in `/tmp/tofukit-debug.log`**

2. **File Logging Works for Read() but Not Create()/Update()**
   - `buildOutputData()` logs appear (called from Read())
   - `collectFeatureKitIDs()` logs appear (called from buildOutputData())
   - Create()/Update() logs never appear
   - Suggests logging issue or execution path difference

3. **Cannot Confirm Execution**
   - Test runs successfully and creates files
   - Claude prompt JSON exists with missing features
   - But cannot trace execution through Create()/Update()

## Next Steps to Resolve

### Immediate Actions

1. ✅ Add explicit Null/Unknown warning logs to `buildProjectContext()`
2. ✅ **DONE: Added fast tests with dry_run** - Can now reproduce in <1 second
3. 🔍 **Current Finding:** Features ARE included in `_project_context` but **missing `prompt` field**
4. ⏸️ Fix feature metadata extraction in `buildProjectContext()` to include `prompt` field
5. ⏸️ Verify fix with fast tests: `go test -v -run "GeneratePromptSuccess" ./test/`

### Investigation Paths

**Path A: Execution Timing**
- Determine when `buildProjectContext()` is called vs `collectFeatureKitIDs()`
- Check if features are populated at different lifecycle stages
- Verify data flow from Plan → Create → Update

**Path B: Type System Issue**
- Investigate how `types.Dynamic` behaves in different contexts
- Check if features are Unknown during plan but known during apply
- Verify if there's a race condition or ordering dependency

**Path C: Logging Infrastructure**
- Fix debug logging for Create()/Update() methods
- Use Terraform's native `tflog` instead of file logging
- Add structured logging to trace execution flow

## Workarounds

**None available.** The feature introspection system is broken until this is fixed.

## Testing

### Fast Reproduction (< 1 second with dry_run)

**NEW: Fast test added with dry_run support!**

```bash
# Fast test - skips LLM execution, just validates prompt generation
go test -v -run "TestResourceFeatureInlineGeneratePromptSuccess" ./test/ -timeout 30s

# All three fast tests (single feature, precedence, multiple features)
go test -v -run "GeneratePromptSuccess" ./test/ -timeout 1m
```

**These tests use `dry_run = true` to skip Claude execution and complete in <1 second each.**

**Test Results:**
- ✅ `TestProjectPrecedenceOverFeatureGeneratePromptSuccess` - PASSES (features present but incomplete)
- ✅ `TestResourceFeatureMergeMultipleFeaturesGeneratePromptSuccess` - PASSES (features present but incomplete)
- ❌ `TestResourceFeatureInlineGeneratePromptSuccess` - **FAILS** - Catches missing `prompt` field in feature

**Actual Output from Failing Test:**
```json
{
  "_project_context": {
    "features": [
      {
        "files": ["hello.txt"],
        "name": "hello"
        // ❌ MISSING: "prompt" field
      }
    ]
  }
}
```

**Expected:** `features[0].prompt == "Add hello greeting capability"`
**Actual:** `features[0].prompt == nil`

### Slow Reproduction (10+ seconds with full LLM execution)

```bash
# Diagram test (uses features) - runs actual Claude execution
go test -v -run "^TestE2EProjectExampleInfra3TierAppSuccess$" ./test/ -timeout 10m

# Check Claude prompt
jq '.request.specification._project_context' test-output/.../claude-prompt-attempt1-*.json
```

### Verify Fix

Once resolved, `project_context` should contain:
```json
{
  "features": [
    {"name": "hello", "prompt": "Add hello greeting capability", "files": ["hello.txt"]},
    {"name": "load_balancer", "prompt": "...", "files": [...], "kits": [...]},
    {"name": "web_server", "prompt": "...", "files": [...], "kits": [...]},
    {"name": "database", "prompt": "...", "files": [...], "kits": [...]},
    {"name": "architecture_diagram", "prompt": "...", "kits": ["tool.drawio"]}
  ],
  "kits": [{...}],
  "project_info": {...}
}
```

## Related Issues

- **BUG-006:** Diagram generation incorrect ID format (caused by this bug)
- **BUG-003:** Fixed - Unchanged files showing as modified (similar type handling issue)

## Impact Assessment

**Severity:** High

**Affected Functionality:**
- ✅ Kits still work (added unconditionally)
- ✅ Requirements still work
- ❌ Features not available in `project_context`
- ❌ Feature introspection broken
- ❌ Diagram generation cannot use feature names
- ❌ Any LLM prompt relying on feature metadata fails

**Workaround:** None - system design assumes features are available in `project_context`
