# File Deletion Protection Design

**Date:** 2025-10-29
**Status:** Approved for Implementation

## Problem Statement

TofuKit gives Claude CLI full filesystem access to the output directory (`--dangerously-skip-permissions=true`). This creates a risk where Claude could remove files that are not managed by TofuKit (e.g., `.tf`, `.tofu`, `.terraform.lock.hcl`, or manually created files).

**Example Threat Scenario:**
```
Output directory contains:
- test.txt (managed by TofuKit, in state)
- main.tf (unmanaged, manually created)
- .terraform.lock.hcl (unmanaged, Terraform lock file)

If Claude decides to "clean up" the directory, it could remove main.tf and .terraform.lock.hcl
because they're not in its file list.
```

## Key Constraints

1. **Claude has filesystem access** - The provider cannot enforce file protection at the filesystem level
2. **Deletion happens before provider sees it** - Claude executes first, then provider processes results
3. **Unknown files are invisible** - The provider doesn't know about unmanaged files, so it can't filter them
4. **Only defense is the prompt** - Provider can only request Claude's cooperation via system prompt

## Solution: Default-Deny Deletion Policy

Implement a **prompt-based protection policy** that makes Claude's deletion behavior explicit and conservative.

### Core Policy Rules

Add to the system prompt in `internal/llm/claude/prompt_types.go`:

```
CRITICAL FILE OPERATION RULES:

1. File Deletion: NEVER remove any file from the output directory unless you
   receive an explicit instruction with action="remove" and the full file path

2. Directory Deletion: NEVER remove any directory unless it is completely empty
   (no files, no subdirectories)

3. File Discovery: If you discover files in the output directory that are NOT
   in your files array, leave them alone. Do not remove, modify, or clean them up.

4. Scope Limitation: Only operate on files explicitly listed in the files array
   you receive. Ignore all other files in the directory.
```

## Design Rationale

**Why prompt-only approach?**
- Provider-level protection is impossible due to Claude's filesystem access
- File manager's `ProcessFileChanges()` only removes managed files (already safe)
- Unknown files cannot be filtered or validated by the provider
- Prompt policy is the only point of control before Claude acts

**Why default-deny instead of pattern matching?**
- Simpler: "Don't remove anything unless explicitly told" vs "Don't remove *.tf, *.tofu, ..."
- More robust: Protects ALL unmanaged files, not just known patterns
- Future-proof: Works for any file type without updating patterns
- Clearer semantics: Easier for Claude to understand and follow

**What about legitimate deletions?**
- Managed file removals still work: Provider generates explicit `action="remove"` instructions
- Empty directory cleanup still works: Policy allows removing empty directories
- No change to existing functionality - only prevents accidental deletions

## Implementation Plan

### Changes Required

**File:** `internal/llm/claude/prompt_types.go`

**Location:** In the system prompt string (around line 120-180)

**Change:** Add the protection rules to the system prompt before the existing instructions

**Example Integration:**
```go
systemPrompt := `You are Claude Code, managing files for a TofuKit project.

CRITICAL FILE OPERATION RULES:
1. File Deletion: NEVER remove any file from the output directory unless you
   receive an explicit instruction with action="remove" and the full file path

2. Directory Deletion: NEVER remove any directory unless it is completely empty
   (no files, no subdirectories)

3. File Discovery: If you discover files in the output directory that are NOT
   in your files array, leave them alone. Do not remove, modify, or clean them up.

4. Scope Limitation: Only operate on files explicitly listed in the files array
   you receive. Ignore all other files in the directory.

[... existing system prompt continues ...]
`
```

### Testing Strategy

**Test Case 1: Unmanaged File Protection**
- Create output directory with `main.tf` (unmanaged) and `test.txt` (managed)
- Remove `test.txt` from TofuKit config
- Apply changes
- Verify: `test.txt` removed, `main.tf` untouched

**Test Case 2: Infrastructure File Protection**
- Create output directory with `.terraform.lock.hcl`, `project.tofu`
- TofuKit manages `hello.txt`
- Update `hello.txt` content
- Verify: Infrastructure files unchanged

**Test Case 3: Legitimate Deletion Works**
- TofuKit manages `file1.txt` and `file2.txt`
- Remove `file1.txt` from config
- Apply changes
- Verify: `file1.txt` removed (explicit instruction), `file2.txt` unchanged

**Test Case 4: Empty Directory Cleanup**
- TofuKit manages `dir/file.txt`
- Remove `dir/file.txt` from config
- Apply changes
- Verify: File removed, `dir/` removed (empty), other directories untouched

## Risks and Mitigations

**Risk 1: Claude ignores the prompt**
- Mitigation: Make rules prominent, clear, and explicit
- Mitigation: Use strong language ("CRITICAL", "NEVER")
- Fallback: If this happens frequently, we'd need to restrict filesystem access

**Risk 2: Prompt becomes too long**
- Mitigation: Rules are concise (4 bullet points, ~200 words)
- Impact: Negligible token cost increase

**Risk 3: Rules conflict with existing behavior**
- Mitigation: Policy codifies existing behavior (provider already doesn't remove unmanaged files)
- Mitigation: Extensive testing to verify no regressions

## Future Enhancements

If prompt-based protection proves insufficient, consider:

1. **Post-execution validation**: Scan directory after Claude finishes, fail if unexpected files removed
2. **Filesystem restrictions**: Remove `--dangerously-skip-permissions=true`, require explicit permission prompts
3. **Provider-managed whitelist**: Let users specify protected patterns in provider config
4. **Read-only mode**: Add flag to give Claude read-only access for certain operations

## Success Criteria

- [ ] Prompt rules added to `prompt_types.go`
- [ ] All four test cases pass
- [ ] No regression in existing tests
- [ ] Documentation updated (CLAUDE.md)
- [ ] Manual testing confirms infrastructure files are protected
