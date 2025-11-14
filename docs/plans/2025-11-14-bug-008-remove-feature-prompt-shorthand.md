# BUG-008: Remove Feature Prompt Shorthand Syntax

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix missing `prompt` field in `_project_context.features[]` by removing unsupported shorthand syntax and enforcing nested requirements structure.

**Architecture:** Inline features currently accept a shorthand `prompt = "..."` syntax that is never processed by `parseFeature()`. External `tofukit_feature` resources require nested `requirements` structure. This inconsistency causes the bug. Solution: Remove shorthand support, update test to use nested syntax matching external features.

**Tech Stack:** Go, Terraform Plugin Framework, OpenTofu/Terraform

**Root Cause:** The `parseFeature()` function in `internal/resources/project.go` extracts `requirements`, `files`, `kits`, and `verifications` from inline feature definitions, but ignores a top-level `prompt` attribute. The test uses this unsupported shorthand syntax, expecting it to appear in `_project_context.features[].prompt`, but it never gets extracted.

**Current State:**
- Test: `TestResourceFeatureInlineGeneratePromptSuccess` - FAILS
- External feature resources already require nested `requirements` (no shorthand)
- `buildProjectContext()` correctly extracts prompts from `Requirements[0].Instructions[0].Prompt`

---

## Task 1: Update Test to Use Nested Requirements Syntax

**Files:**
- Modify: `test/resource_feature_test.go:654-750`

**Step 1: Read current test to understand structure**

Run: `cat test/resource_feature_test.go | sed -n '654,750p'`

Expected: See test using shorthand `prompt = "Add hello greeting capability"`

**Step 2: Update test HCL config to use nested requirements**

Replace the features block in the test (around lines 677-686):

```go
// OLD (shorthand - unsupported):
features = {
  "hello" = {
    prompt = "Add hello greeting capability"
    files = {
      "hello.txt" = {
        content = "Hello from inline feature.\n"
      }
    }
  }
}

// NEW (nested - matches external tofukit_feature):
features = {
  "hello" = {
    requirements = [
      {
        name = "Feature Implementation"
        instructions = [
          {
            prompt = "Add hello greeting capability"
          }
        ]
      }
    ]
    files = {
      "hello.txt" = {
        content = "Hello from inline feature.\n"
      }
    }
  }
}
```

**Step 3: Run test to verify it now passes**

Run: `go test -v -run "TestResourceFeatureInlineGeneratePromptSuccess" ./test/ -timeout 30s`

Expected: PASS - prompt should now appear in `_project_context.features[].prompt`

**Step 4: Verify prompt JSON structure**

Check the generated prompt contains the expected structure:

Run:
```bash
TEST_DIR=$(ls -td test-output/*TestResourceFeatureInlineGeneratePromptSuccess* | head -1)
cat "$TEST_DIR/output/.debug/claude-prompt-attempt1-"*.json | jq '.request.specification._project_context.features[0]'
```

Expected output:
```json
{
  "name": "hello",
  "prompt": "Add hello greeting capability",
  "files": ["hello.txt"]
}
```

**Step 5: Commit the fix**

```bash
git add test/resource_feature_test.go
git commit -m "fix(test): use nested requirements syntax for inline features

- Update TestResourceFeatureInlineGeneratePromptSuccess to use requirements[]
- Remove unsupported shorthand prompt syntax
- Makes inline features consistent with external tofukit_feature resources

Fixes: BUG-008"
```

---

## Task 2: Verify Other Feature Tests Still Pass

**Files:**
- Run: `test/resource_feature_test.go` (all feature tests)

**Step 1: Run all feature prompt generation tests**

Run: `go test -v -run ".*Feature.*GeneratePromptSuccess" ./test/ -timeout 2m`

Expected: All tests PASS

Tests that should pass:
- `TestResourceFeatureInlineGeneratePromptSuccess` ✅ (just fixed)
- `TestProjectPrecedenceOverFeatureGeneratePromptSuccess` ✅
- `TestResourceFeatureInlineMergeMultipleFeaturesGeneratePromptSuccess` ✅
- `TestResourceFeatureFilesIndependentGeneratePromptSuccess` ✅
- `TestResourceFeatureCreateGeneratePromptSuccess` ✅
- `TestStackFeaturesFromModuleGeneratePromptSuccess` ✅

**Step 2: Check if any other tests use shorthand syntax**

Run: `grep -n 'prompt = "' test/resource_feature_test.go`

Expected: Should find uses in other tests that might need updating

**Step 3: Update any other tests using shorthand syntax**

If other tests are found, update them to use nested requirements structure (same pattern as Task 1 Step 2).

**Step 4: Re-run all tests to confirm**

Run: `go test -v -run ".*Feature.*GeneratePromptSuccess" ./test/ -timeout 2m`

Expected: All PASS

**Step 5: Commit if changes were made**

```bash
git add test/resource_feature_test.go
git commit -m "fix(test): update remaining feature tests to use nested requirements

- Ensure all feature tests use consistent syntax
- Remove shorthand prompt usage across test suite"
```

---

## Task 3: Update Bug Documentation

**Files:**
- Modify: `docs/bugs/BUG-008-features-null-in-buildProjectContext.md`

**Step 1: Update bug status to RESOLVED**

Change line 3:
```markdown
**Status:** Open
```

To:
```markdown
**Status:** RESOLVED (2025-11-14)
```

**Step 2: Add resolution section at the end**

Append to end of file:

```markdown
## Resolution (2025-11-14)

**Fix:** Removed unsupported shorthand `prompt = "..."` syntax from inline features.

**Root Cause:** The `parseFeature()` function never extracted a top-level `prompt` attribute. It only processes `requirements`, `files`, `kits`, and `verifications`. The test was using an undocumented shorthand syntax that was never implemented.

**Solution:** Updated test to use nested `requirements` structure matching external `tofukit_feature` resources:

```hcl
# BEFORE (broken - shorthand not supported):
features = {
  "hello" = {
    prompt = "Add hello greeting"
    files = { ... }
  }
}

# AFTER (fixed - nested requirements):
features = {
  "hello" = {
    requirements = [
      {
        name = "Feature Implementation"
        instructions = [{ prompt = "Add hello greeting" }]
      }
    ]
    files = { ... }
  }
}
```

**Verification:**
- Test now passes: `go test -run TestResourceFeatureInlineGeneratePromptSuccess`
- Prompt appears in `_project_context.features[].prompt`
- Inline features now consistent with external feature resources

**Commits:**
- `fix(test): use nested requirements syntax for inline features`
```

**Step 3: Commit documentation update**

```bash
git add docs/bugs/BUG-008-features-null-in-buildProjectContext.md
git commit -m "docs: mark BUG-008 as resolved with fix details"
```

---

## Task 4: Run Full Test Suite and Create PR

**Step 1: Run all prompt generation tests**

Run: `go test -v -run "GeneratePromptSuccess" ./test/ -timeout 5m`

Expected: All 23 tests PASS (including the now-fixed feature test)

**Step 2: Run full test suite to ensure no regressions**

Run: `task test` or `go test -v ./test/ -timeout 30m`

Expected: All tests PASS (may take 10-30 minutes)

**Step 3: Build provider to verify compilation**

Run: `task build`

Expected: Build succeeds with no errors

**Step 4: Review all changes**

Run: `git log --oneline origin/feature/wip..HEAD`

Expected: Should see 2-3 commits for the fix

**Step 5: Submit for human review**

Report to user:

```
✅ BUG-008 Fix Complete

Changes:
- Updated TestResourceFeatureInlineGeneratePromptSuccess to use nested requirements
- All 23 prompt generation tests passing
- Full test suite passing
- Documentation updated

Ready for review. Once approved, will merge to feature/wip branch.
```

---

## Verification Checklist

Before marking complete, verify:

- [ ] `TestResourceFeatureInlineGeneratePromptSuccess` passes
- [ ] All 23 prompt generation tests pass
- [ ] Full test suite passes
- [ ] `_project_context.features[].prompt` field present in generated JSON
- [ ] Bug documentation updated with resolution
- [ ] Commits follow conventional commit format
- [ ] No shorthand `prompt` syntax remains in tests

---

## Notes for Engineer

**Key Files:**
- `test/resource_feature_test.go` - Feature tests (main file to modify)
- `internal/resources/project.go:3123-3479` - `parseFeature()` function (NO CHANGES NEEDED)
- `internal/resources/project.go:2892-3053` - `buildProjectContext()` function (NO CHANGES NEEDED)
- `docs/bugs/BUG-008-features-null-in-buildProjectContext.md` - Bug documentation

**Why This Works:**
- `parseFeature()` already correctly extracts `requirements`
- `buildProjectContext()` already correctly extracts prompts from `Requirements[0].Instructions[0].Prompt`
- The bug was using unsupported shorthand syntax in the test
- Fixing the test to use supported syntax makes everything work

**Testing Strategy:**
- Use `dry_run=true` for fast feedback (<0.2s per test)
- Verify prompt JSON structure with `jq`
- Keep test output directories (don't set CLEANUP_TEST_OUTPUT=true)

**Worktree Info:**
- Location: `/Users/dmitry/dev/dimmkirr/terraform-provider-tofukit/.worktrees/fix-bug-008`
- Branch: `fix/bug-008-feature-prompt-field`
- Original branch: `feature/wip`
