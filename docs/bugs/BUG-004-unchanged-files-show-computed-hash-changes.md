# BUG-003: Unchanged Files Show Computed Hash Changes in Plan

**Status:** Open
**Priority:** Medium
**Discovered:** 2025-11-13
**Affects Tests:**
- TestResourceProjectInlineFileRemovalSuccess

## Summary

When removing a file from a project, other unchanged files incorrectly show their computed hash fields changing to `(known after apply)` in the Terraform plan, even though the files shouldn't change.

## Root Cause

When the `files` map is modified (e.g., removing `hello2.txt`), Terraform recomputes all computed attributes for all files in the map, including files that should remain unchanged.

## Expected Behavior

When removing `hello2.txt`:
```hcl
files = {
  "hello.txt" = { content = "hello world\n" }
  # hello2.txt removed
  "hello3.txt" = { content = "hello world3\n" }
}
```

**Expected Plan:**
- `hello.txt`: No changes
- `hello2.txt`: Removed (shown with `-`)
- `hello3.txt`: No changes

## Actual Behavior

**Actual Plan:**
```hcl
~ "hello.txt" = {
    ~ content_hash = "a948904f..." -> (known after apply)
    ~ file_hash    = "a948904f..." -> (known after apply)
    ~ file_modtime = "2025-11-13T10:40:51Z" -> (known after apply)
      # (1 unchanged attribute hidden)
  }
- "hello2.txt" = { ... } -> null
~ "hello3.txt" = {
    ~ content_hash = "199367ec..." -> (known after apply)
    ~ file_hash    = "199367ec..." -> (known after apply)
    ~ file_modtime = "2025-11-13T10:40:51Z" -> (known after apply)
      # (1 unchanged attribute hidden)
  }
```

## Impact

- Test assertion fails: "Files section shows hello.txt being modified/removed - should be unchanged"
- Plan output is misleading - suggests files will change when they won't
- Actual file operations work correctly (files verified to exist with correct content)
- Only affects plan representation, not execution

## Test Failure Location

`test/resource_project_inline_file_test.go:600`
```go
assert.NotContains(t, planOutput, `~ "hello.txt"`,
  "Files section shows hello.txt being modified/removed - should be unchanged")
```

## Possible Causes

1. **Computed Field Logic:** `computeAndStoreFileHashes()` may be triggering recomputation for all files
2. **Map Modification:** Terraform may mark entire map as changed when any element changes
3. **Schema Definition:** Computed fields in nested map may not preserve values correctly
4. **State Management:** Provider may not be preserving computed values for unchanged files

## Investigation Steps

1. Check how `computeAndStoreFileHashes()` determines which files to update
2. Review schema definition for `files` map - are computed fields marked correctly?
3. Test if setting `UseStateForUnknown: true` helps preserve values
4. Check if we need to explicitly preserve computed values for unchanged files in Update()

## Workaround

Files work correctly - only the plan representation is wrong. Could skip plan verification in test, but that reduces test value.

## Related Files

- `internal/resources/project.go` - Update() logic, computeAndStoreFileHashes()
- `internal/schemas/common.go` - FileModel schema definition
- `test/resource_project_inline_file_test.go:600` - Failing assertion
