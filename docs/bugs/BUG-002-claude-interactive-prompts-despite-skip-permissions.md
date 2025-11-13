# BUG-002: Claude Interactive Prompts Despite --dangerously-skip-permissions

**Status:** Fixed
**Priority:** High
**Discovered:** 2025-11-13
**Fixed:** 2025-11-13
**Affects Tests:**
- TestE2EProjectExampleInfra3TierAppSuccess

## Summary

Claude CLI asks interactive questions during execution despite `--dangerously-skip-permissions` flag being enabled, causing test to hang and eventually terminate.

## Root Cause

On retry attempts after verification failures, Claude enters an interactive "design thinking" mode where it asks questions like:

```
Does this match your understanding? Any adjustments needed before I proceed with implementation?

Since the requirements are straightforward and the specification is explicit, I'll proceed with Phase 2: Exploration...
```

Provider expects non-interactive execution but Claude waits for user input that never comes in automated test environment.

## Error Details

**Test Duration:** 552 seconds (hung waiting for input)
**Termination:** `signal: terminated`
**Execution Metadata:** `state="completed"` but output shows questions instead of execution

**Error Log:**
```
Claude Code execution failed: signal: terminated
```

## Evidence

**Debug File:** `test-output/.../output/.debug/claude-execution-metadata-1763032127.json`

Shows:
- Attempt 1: Failed verification (architecture.drawio not created)
- Attempt 2: Claude started asking design questions instead of executing

**Verification Failures:**
```
❌ File 'architecture.drawio': Command 'test -f architecture.drawio && echo 'OK'' failed with error: exit status 1
❌ File 'architecture.drawio': Command 'grep -q '<mxGraphModel' architecture.drawio && echo 'OK'' failed with error: exit status 2
```

## Impact

- Long-running E2E test fails after ~9 minutes
- Wastes CI/CD resources
- Blocks full test suite completion
- Unpredictable behavior on retry attempts

## Possible Causes

1. **Claude CLI Behavior Change:** Recent Claude CLI version may have changed interactive behavior
2. **Retry Logic Issue:** Retry prompts may trigger different Claude behavior than initial prompts
3. **Prompt Structure:** Verification failure messages may be triggering Claude's "design thinking" mode
4. **Missing Flag:** May need additional flag to force non-interactive mode beyond `--dangerously-skip-permissions`

## Reproduction

```bash
go test -v -run TestE2EProjectExampleInfra3TierAppSuccess ./test/
```

## Investigation Steps

1. Check Claude CLI version and flags used
2. Review prompt structure sent on retry attempts
3. Test if verification failure messages trigger interactive mode
4. Check if there's a force-execute or non-interactive flag
5. Review recent Claude CLI release notes for behavior changes

## Workaround

None currently - test must be fixed or skipped.

## Related Files

- `internal/llm/claude/executor.go` - Retry logic
- `internal/llm/claude/client.go` - Claude CLI invocation
- `internal/llm/claude/prompt_types.go` - System prompt
- `test/e2e_project_example_infra_3_tier_app_test.go`

## Resolution

**Fixed in commit:** cb5b544

**Root Cause:**
The system prompt didn't explicitly forbid interactive behavior, and retry prompts were conversational enough to trigger Claude's "design thinking" mode where it asks questions and waits for user input.

**Fix Applied:**

1. **Updated System Prompt** (`internal/llm/claude/prompt_types.go`):
   - Added explicit "CRITICAL EXECUTION MODE" section
   - Explicitly states this is NON-INTERACTIVE automated environment
   - Added rules: "Execute immediately without asking ANY questions"
   - Added: "NEVER ask for clarification, confirmation, or approval"
   - Added: "NEVER enter design thinking or planning mode that requires user input"
   - Added: "Make reasonable assumptions and proceed"

2. **Made Retry Prompts More Directive** (`internal/llm/claude/executor.go`):
   - Changed from conversational to imperative tone
   - Added "This is an automated retry - execute immediately"
   - Added "Do NOT ask questions or request clarification"
   - Added "Do NOT enter design/planning mode"
   - Added "Execute the fix IMMEDIATELY"
   - Emphasized NON-INTERACTIVE environment in both retry locations

**Verification:**
The fix ensures Claude understands it's in an automated environment and must execute immediately without entering interactive modes that cause tests to hang.
