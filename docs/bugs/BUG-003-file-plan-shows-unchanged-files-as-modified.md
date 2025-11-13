# BUG-003: Terraform Plan Shows Unchanged Files as Modified on File Removal

**Status:** Open
**Priority:** Medium
**Discovered:** 2025-11-13
**Affects Tests:**
- TestResourceProjectInlineFileRemovalSuccess

## Summary

When a file is removed from a `tofukit_project` resource, Terraform's plan output incorrectly shows ALL remaining files as being "modified" with changed `content_hash`, `file_hash`, and `file_modtime` values, even though these files are completely unchanged.

The actual file removal works correctly - files are properly deleted - but the plan display is misleading and confusing for users.

## Root Cause

This appears to be related to how Terraform computes changes for computed attributes in map schemas. When file removal is detected:

1. The provider correctly identifies which files should be removed
2. File operations are computed correctly (`removed=[hello2.txt]`, `unchanged=[hello.txt, hello3.txt]`)
3. However, Terraform's plan output marks unchanged files with `~` (modify) instead of leaving them hidden or showing them as unchanged

The debug logs show:
```
ModifyPlan: File operations detected:
  removed=[hello2.txt]
  unchanged=[hello.txt, hello3.txt]
  modified=[]
  added=[]
```

But the plan output shows:
```hcl
~ "hello.txt" = {
    ~ content_hash = "..." -> (known after apply)
    ~ file_hash    = "..." -> (known after apply)
    ~ file_modtime = "..." -> (known after apply)
}
~ "hello3.txt" = {
    ~ content_hash = "..." -> (known after apply)
    ~ file_hash    = "..." -> (known after apply)
    ~ file_modtime = "..." -> (known after apply)
}
```

## Expected Behavior

When removing `hello2.txt` from a project with three files:

**Plan should show:**
```hcl
~ resource "tofukit_project" "hello_world" {
    ~ files = {
        - "hello2.txt" = { ... } -> null
        # (2 unchanged elements hidden)
    }
}
```

**Actual plan shows:**
```hcl
~ resource "tofukit_project" "hello_world" {
    ~ files = {
        ~ "hello.txt"  = { ... } # INCORRECTLY shown as modified
        - "hello2.txt" = { ... } -> null
        ~ "hello3.txt" = { ... } # INCORRECTLY shown as modified
    }
}
```

## Impact

- **User Experience**: Confusing plan output makes users think files will be modified when they won't
- **Test Failure**: `TestResourceProjectInlineFileRemovalSuccess` fails on plan verification
- **Trust**: Users may lose confidence in the provider's behavior
- **Severity**: Medium - functionality works correctly, but UX is poor

## Reproduction

**Test Command:**
```bash
cd /Users/dmitry/dev/dimmkirr/terraform-provider-tofukit
go test -v -run TestResourceProjectInlineFileRemovalSuccess ./test/
```

**Manual Reproduction:**

1. Create a project with three files:
```hcl
resource "tofukit_project" "test" {
  name = "test"
  files = {
    "hello.txt"  = { content = "hello world\n" }
    "hello2.txt" = { content = "hello world2\n" }
    "hello3.txt" = { content = "hello world3\n" }
  }
}
```

2. Apply: `tofu apply`

3. Remove the middle file:
```hcl
resource "tofukit_project" "test" {
  name = "test"
  files = {
    "hello.txt"  = { content = "hello world\n" }
    # hello2.txt removed
    "hello3.txt" = { content = "hello world3\n" }
  }
}
```

4. Run `tofu plan`

5. Observe that `hello.txt` and `hello3.txt` are shown as `~` (modified) when they should be hidden as unchanged

## Test Details

**Test Location:** `test/resource_project_inline_file_test.go:382`

**Test Flow:**
1. Creates project with 3 files (hello.txt, hello2.txt, hello3.txt)
2. Applies successfully
3. Removes hello2.txt from config (middle file)
4. Runs `tofu plan`
5. **FAILS HERE**: Checks that hello.txt and hello3.txt are NOT shown as modified
6. Applies the removal
7. **PASSES**: Verifies files are correct (hello2.txt deleted, others unchanged)

**Failure Output:**
```
resource_project_inline_file_test.go:600: ❌ Files section shows hello.txt being modified/removed - should be unchanged
```

## Debug Information

**Provider Logs Show Correct Detection:**
```
ModifyPlan: File operations detected:
  removed=[hello2.txt]
  unchanged=[hello.txt, hello3.txt]
  modified=[]
  added=[]
```

**But Plan Output Shows:**
```
~ "hello.txt" = {
    ~ content_hash = "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447" -> (known after apply)
    ~ file_hash    = "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447" -> (known after apply)
    ~ file_modtime = "2025-11-13T13:01:35Z" -> (known after apply)
}
```

Notice that hash values are IDENTICAL before and after, yet Terraform marks them as changed.

## Possible Causes

1. **Computed Attribute Behavior**: Terraform Plugin Framework may mark all computed attributes as "unknown" during update operations, even when they won't actually change

2. **PlanModifier Issue**: The provider's `ModifyPlan` implementation may not be properly preserving computed values for unchanged files

3. **Map Schema Limitation**: Terraform may not support partial updates in map schemas without marking other elements as changed

4. **Hash Computation Timing**: The message "Hash computation deferred to apply phase" suggests hashes aren't available at plan time, causing Terraform to assume they'll change

## Investigation Areas

1. **Review `ModifyPlan()` in `/internal/resources/project.go`**:
   - Check if computed attributes for unchanged files can be preserved
   - See if we can set `RequiresReplace` or other modifiers to prevent "(known after apply)"

2. **Check Terraform Plugin Framework docs**:
   - How to handle computed attributes for unchanged elements in map schemas
   - PlanModifier best practices for map elements

3. **Examine hash computation**:
   - Can hashes be computed at plan time instead of apply time?
   - Would this prevent "(known after apply)" for unchanged files?

4. **Compare with other providers**:
   - How do providers like `hashicorp/local_file` handle file removal?
   - Do they have similar issues with computed attributes?

## Workaround

None currently. The actual functionality works (files are removed correctly), but the plan output is misleading.

## Related Files

- `/internal/resources/project.go:360` - `ModifyPlan()` method
- `/internal/resources/project.go:407` - File change analysis
- `/internal/resources/project.go:457` - File operations logging
- `/internal/files/operations.go` - File operation computation (works correctly)
- `/test/resource_project_inline_file_test.go:382` - Failing test

## Notes

- The file removal DOES work correctly in actual execution
- This is purely a plan display/UX issue
- Debug logs show the provider correctly identifies which files are unchanged
- The test expects Terraform to hide unchanged files or at least not mark them as modified
- This may be a fundamental limitation of how Terraform handles computed attributes in collections
