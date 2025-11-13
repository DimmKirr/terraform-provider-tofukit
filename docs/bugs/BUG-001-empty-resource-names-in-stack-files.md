# BUG-001: Empty Resource Names in Stack Files

**Status:** Fixed
**Priority:** High
**Discovered:** 2025-11-13
**Fixed:** 2025-11-13
**Fix Commits:** 740f7c2, fea02a3
**Affects Tests:**
- TestE2EProjectExampleClickCliSuccess
- TestE2EProjectExampleGoViperCliHelloWorldSuccess
- TestStackFeaturesFromModule

## Summary

Stack example files contain resources with empty names, causing HCL syntax errors during test execution.

## Root Cause

Invalid HCL syntax in stack definition files:

1. **File:** `examples/stacks/tofukit-stack-python-click-app/stack.tofu:30`
   ```hcl
   resource "tofukit" "" {}  # ❌ Empty name - invalid
   ```

2. **File:** `examples/stacks/tofukit-stack-go-viper-cobra-pterm/files.tofu:452`
   ```hcl
   resource "tofukit_integration" "" {  # ❌ Empty name - invalid
   ```

## Error Message

```
Error: Invalid resource name

A name must start with a letter or underscore and may contain only letters,
digits, underscores, and dashes.
```

## Impact

- 3 E2E tests fail immediately during Terraform initialization
- Stack examples cannot be used as references
- Documentation examples are broken

## Reproduction

```bash
go test -v -run TestE2EProjectExampleClickCliSuccess ./test/
```

## Fix Strategy

1. Identify the intended resource names from context
2. Replace empty strings with valid resource names
3. Verify tests pass after fix
4. Consider adding linting to prevent this in the future

## Related Files

- `examples/stacks/tofukit-stack-python-click-app/stack.tofu`
- `examples/stacks/tofukit-stack-go-viper-cobra-pterm/files.tofu`
- `test/e2e_project_example_click_cli_test.go`
- `test/e2e_project_example_go_viper_cli_hello_world_test.go`
- `test/e2e_stack_features_from_module_test.go`

## Resolution

**Fixed in commits:** 740f7c2, fea02a3

**Root Cause:**
Two stack files contained invalid placeholder/empty resource declarations:

1. `examples/stacks/tofukit-stack-python-click-app/stack.tofu` line 30:
   ```hcl
   resource "tofukit" "" {}
   ```

2. `examples/stacks/tofukit-stack-go-viper-cobra-pterm/files.tofu` line 452:
   ```hcl
   resource "tofukit_integration" "" {
     name    = ""
     version = ""
   }
   ```

Both were leftover template/placeholder code - likely from abandoned copy-paste or incomplete resource additions during development. Empty resource names violate HCL syntax requirements.

**Fix Applied:**
Removed both invalid empty resource declarations. These lines served no purpose and were not referenced anywhere in the codebase.

**Verification:**
All three affected E2E tests now pass:
- TestE2EProjectExampleClickCliSuccess ✅
- TestE2EProjectExampleGoViperCliHelloWorldSuccess ✅
- TestStackFeaturesFromModule ✅
