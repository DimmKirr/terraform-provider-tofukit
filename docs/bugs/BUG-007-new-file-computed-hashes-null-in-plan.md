# BUG-007: New File Computed Hashes Are Null in Plan But Populated After Apply

**Status:** Open
**Priority:** High
**Discovered:** 2025-11-13
**Affects Tests:**
- TestResourceProjectInlineFileAddMiddleFileSuccess
- TestResourceProjectInlineFileRenameSuccess
- TestResourceProjectInlineFileNestedAddSuccess
- TestResourceProjectInlineFileNestedDeeperNestingSuccess

## Summary

When adding new files to a project, the computed hash attributes (`content_hash`, `file_hash`, `file_modtime`) are null/unknown during the plan phase but become populated during apply. This violates Terraform's consistency requirement that "planned values must match applied values".

**Root Cause:** The `UseStateForUnknown()` PlanModifier added in BUG-003 fix (commit d58736a) only preserves values from state. For NEW files with no prior state, the modifier doesn't help - values remain unknown in plan but are computed during apply.

## How to Reproduce

**Test Commands:**
```bash
# Test 1: Add middle file
go test -v -run "^TestResourceProjectInlineFileAddMiddleFileSuccess$" ./test/ -timeout 10m

# Test 2: Rename file
go test -v -run "^TestResourceProjectInlineFileRenameSuccess$" ./test/ -timeout 10m

# Test 3: Add nested file
go test -v -run "^TestResourceProjectInlineFileNestedAddSuccess$" ./test/ -timeout 10m

# Test 4: Add deeply nested file
go test -v -run "^TestResourceProjectInlineFileNestedDeeperNestingSuccess$" ./test/ -timeout 10m
```

**Expected Behavior:**
When adding a new file (e.g., `b.txt`), Terraform should either:
1. Show computed hashes as `(known after apply)` in plan AND in final state (consistent)
2. OR compute the hashes during plan phase so they match apply phase

**Actual Behavior:**
```
Error: Provider produced inconsistent result after apply

When applying changes to tofukit_project.hello_world, provider
"provider[\"registry.terraform.io/dimmkirr/tofukit\"]" produced an unexpected
new value: .files["b.txt"].content_hash: was null, but now
cty.StringVal("e795ce5846c40aa45c31d78e1330e7369435517ce7eeb95de540e94fad4b354b").
```

The same error occurs for:
- `.files["b.txt"].file_hash`
- `.files["b.txt"].file_modtime`

## Files Involved

**Test Files:**
- `/Users/dmitry/dev/dimmkirr/terraform-provider-tofukit/test/resource_project_inline_file_test.go`
  - Line 397: TestResourceProjectInlineFileAddMiddleFileSuccess failure
  - Line 780: TestResourceProjectInlineFileRenameSuccess failure
  - Line 1006: TestResourceProjectInlineFileNestedAddSuccess failure
  - Line 1149: TestResourceProjectInlineFileNestedDeeperNestingSuccess failure

**Provider Code:**
- `/Users/dmitry/dev/dimmkirr/terraform-provider-tofukit/internal/schemas/common.go:242-265`
  - Schema definition with `UseStateForUnknown()` PlanModifiers
  - Current modifiers don't handle new files (no state to preserve)

- `/Users/dmitry/dev/dimmkirr/terraform-provider-tofukit/internal/resources/project.go`
  - `ModifyPlan()` method (lines 360-484) - Could compute hashes here
  - `Update()` method - Where hashes are computed during apply
  - `computeAndStoreFileHashes()` - Hash computation logic

## Top 3 Theories for Fix

### Theory 1: Compute Hashes During ModifyPlan Phase (Recommended)
**Likelihood:** High
**Reasoning:** For files with static `content`, we can compute hashes during plan phase since content is known. This makes plan match apply.

**Potential Fix:**
```go
// In ModifyPlan() method, after identifying added files:
for _, path := range added {
    configFile := configFilesByPath[path]

    // If file has static content (not instructions), compute hashes now
    if !configFile.Content.IsNull() && !configFile.Content.IsUnknown() {
        content := configFile.Content.ValueString()

        // Compute hashes
        contentHash := sha256.Sum256([]byte(content))
        contentHashStr := hex.EncodeToString(contentHash[:])

        // Update plan with computed values
        planFile := planFilesByPath[path]
        planFile.ContentHash = types.StringValue(contentHashStr)
        planFile.FileHash = types.StringValue(contentHashStr) // Same for static content
        planFile.FileModtime = types.StringValue("") // Will be set after file write

        planFilesByPath[path] = planFile
    }
}
// Then update resp.Plan with modified files
```

**Pros:**
- Makes plan consistent with apply
- Works for static content files
- No schema changes needed

**Cons:**
- Requires careful handling of file_modtime (can't know before write)
- May need separate handling for files with instructions (dynamic content)

### Theory 2: Use Different PlanModifier for New Files
**Likelihood:** Medium
**Reasoning:** Instead of `UseStateForUnknown()`, use a modifier that computes values during plan or marks them as correctly unknown.

**Potential Fix:**
```go
// In common.go schema definition:
"content_hash": schema.StringAttribute{
    Optional:  true,
    Computed:  true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(), // For existing files
        customplanmodifier.ComputeIfNew(),       // NEW: For new files
    },
},
```

Create custom PlanModifier that:
1. Checks if state value exists (use it - handled by UseStateForUnknown)
2. If no state (new file), compute hash from config content
3. If content is unknown (instructions-based), return unknown (will match apply)

**Pros:**
- Clean separation of concerns
- Reusable for other scenarios
- Follows Terraform framework patterns

**Cons:**
- More complex implementation
- Need to handle all edge cases in modifier
- May need access to content from config

### Theory 3: Mark Hashes As Truly Unknown for New Files
**Likelihood:** Low
**Reasoning:** Accept that hashes are unknown until apply, and ensure they stay unknown in plan state.

**Potential Fix:**
Modify ModifyPlan to explicitly set new file hashes to Unknown:
```go
for _, path := range added {
    planFile := planFilesByPath[path]
    planFile.ContentHash = types.StringUnknown()
    planFile.FileHash = types.StringUnknown()
    planFile.FileModtime = types.StringUnknown()
    planFilesByPath[path] = planFile
}
```

**Pros:**
- Simple change
- Consistent with "known after apply" semantics

**Cons:**
- Doesn't solve the problem - Terraform will still see inconsistency when they become known after apply
- Plan will show `(known after apply)` which might confuse users about unchanged files
- Doesn't improve user experience

## Recommendation

**Theory 1 (Compute during ModifyPlan)** is the best approach because:
1. We have all the information needed (static content is in config)
2. Makes plan output more accurate and useful
3. Aligns with how Terraform expects computed values to work
4. Can be combined with Theory 2 for a robust solution

## Additional Context

- This bug surfaced after fixing BUG-003, which added `UseStateForUnknown()` modifiers
- The modifier works perfectly for UNCHANGED files (preserves from state)
- But for NEW files, there's no state to preserve from
- All 4 failing tests follow the same pattern: adding a new file to an existing project

## Related Issues

- **BUG-003** (Fixed): Unchanged files showing as modified - fixed with `UseStateForUnknown()`
- This bug is the flip side: NEW files need special handling since they have no state
