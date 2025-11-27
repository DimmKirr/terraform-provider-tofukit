# Configurable Claude Max Turns

**Date**: 2025-11-27
**Status**: Approved
**Author**: Claude (via brainstorming skill)

## Problem

The Claude CLI `--max-turns` flag is hardcoded to 30 in the provider. This creates limitations:

- **Incomplete projects**: Complex projects (20+ files) consistently hit the 30-turn limit before completion
- **Unpredictability**: Users can't know if their project will complete
- **No flexibility**: Can't increase limit for complex projects or decrease for simple ones
- **Real impact**: Purl project analysis showed 30 turns insufficient (90% complete, needed ~35-40 turns)

## Current Implementation

**Location**: `/internal/llm/claude/client.go` lines 249, 422

Hardcoded value:
```go
"--max-turns", "30",
```

No configuration option exists.

## Proposed Solution

Add `claude_max_turns` as a provider-level configuration parameter.

### Design Decisions

**1. Claude-specific parameter** (not abstraction-level)
- OpenAI and Gemini executors are stubs (not implemented)
- "Turns" concept specific to Claude CLI's agentic execution model
- Other LLMs can add their own parameters when implemented
- Name: `claude_max_turns` (explicit, no confusion)

**2. Default value: 100**
- Analysis showed complex projects need ~35-40 turns
- 100 provides 2.5-3x headroom
- Conservative enough to avoid runaway executions
- Users can override per-project

**3. Provider-level configuration**
- Follows existing pattern (`max_retries`, `debug`)
- Single configuration applies to all resources
- Override in provider block, not per-resource

## Architecture

### Data Flow

```
Provider Config (provider.go)
    ↓ claude_max_turns = 100 (default)
    ↓ Parse in Configure()
    ↓ Create Executor with config
Executor (executor.go)
    ↓ SetMaxTurns(100)
    ↓ Store in executor.maxTurns
    ↓ Pass to Client
Client (client.go)
    ↓ client.maxTurns = 100
    ↓ fmt.Sprintf("%d", c.maxTurns)
    ↓ --max-turns 100
Claude CLI Execution
```

### Component Changes

**1. Provider Configuration** (`internal/provider/provider.go`)

Add to `TofukitProviderModel`:
```go
type TofukitProviderModel struct {
    // ... existing fields ...
    ClaudeMaxTurns types.Int64 `tfsdk:"claude_max_turns"`
}
```

Add to schema:
```go
"claude_max_turns": schema.Int64Attribute{
    MarkdownDescription: "Maximum turns for Claude CLI execution (default: 100). Only applies when llm=claude. Higher values allow more complex projects but take longer.",
    Optional:            true,
},
```

Parse in `Configure()`:
```go
claudeMaxTurns := 100 // Default

if !data.ClaudeMaxTurns.IsNull() {
    claudeMaxTurns = int(data.ClaudeMaxTurns.ValueInt64())
    if claudeMaxTurns < 1 {
        claudeMaxTurns = 1 // Minimum 1 turn
    }
}
```

Pass to Executor:
```go
llmExecutor = newClaudeAdapter(claude.NewExecutor(claudeHomeDir, dangerouslySkipPermissions, claudeMaxTurns))
```

**2. Executor** (`internal/llm/claude/executor.go`)

Add field:
```go
type Executor struct {
    client     *Client
    debug      bool
    outputPath string
    model      string
    maxTurns   int  // NEW
}
```

Update `NewExecutor`:
```go
func NewExecutor(claudeHomeDir string, dangerouslySkipPermissions bool, maxTurns int) *Executor {
    return &Executor{
        client:     NewClient(claudeHomeDir, dangerouslySkipPermissions, maxTurns),
        debug:      false,
        outputPath: "",
        maxTurns:   maxTurns,
    }
}
```

**3. Client** (`internal/llm/claude/client.go`)

Add field:
```go
type Client struct {
    claudeHomeDir              string
    dangerouslySkipPermissions bool
    systemPrompt               string
    maxTurns                   int  // NEW
}
```

Update `NewClient`:
```go
func NewClient(claudeHomeDir string, dangerouslySkipPermissions bool, maxTurns int) *Client {
    return &Client{
        claudeHomeDir:              claudeHomeDir,
        dangerouslySkipPermissions: dangerouslySkipPermissions,
        systemPrompt:               "",
        maxTurns:                   maxTurns,
    }
}
```

Replace hardcoded values (lines 249, 422):
```go
// Before
"--max-turns", "30",

// After
"--max-turns", fmt.Sprintf("%d", c.maxTurns),
```

## Usage Example

```hcl
provider "tofukit" {
  llm = "claude"
  claude_max_turns = 100  # Default, can override
  debug = true
}

# For simple projects (save time)
provider "tofukit" {
  alias = "quick"
  llm = "claude"
  claude_max_turns = 30
}

# For complex projects (more headroom)
provider "tofukit" {
  alias = "complex"
  llm = "claude"
  claude_max_turns = 150
}
```

## Implementation Checklist

- [ ] Update `TofukitProviderModel` struct in provider.go
- [ ] Add schema attribute for `claude_max_turns`
- [ ] Parse configuration with default value 100
- [ ] Update `NewExecutor()` signature to accept maxTurns
- [ ] Update `NewClient()` signature to accept maxTurns
- [ ] Replace hardcoded "30" with dynamic value in client.go (2 locations)
- [ ] Update provider instantiation calls
- [ ] Add test for default value (100)
- [ ] Add test for custom value override
- [ ] Update provider documentation

## Testing Strategy

1. **Unit tests**: Verify default value and parsing
2. **E2E test with default**: Run purl project, should complete in <100 turns
3. **E2E test with custom value**: Override to 50, verify it's respected
4. **Debug output validation**: Check that `--max-turns` value matches config

## Backward Compatibility

- **Fully backward compatible**
- Default 100 is higher than previous hardcoded 30
- Existing configurations work without changes
- Projects that completed in 30 turns will still complete
- Projects that failed at 30 turns now have room to succeed

## Future Considerations

- If OpenAI/Gemini implement agentic execution, add separate parameters:
  - `openai_max_iterations`
  - `gemini_max_turns`
- Consider resource-level override (per-project max_turns)
- Add metrics: track actual turn usage vs limit

## Impact Analysis

**Benefits**:
- Complex projects can complete (purl: 30 → ~35-40 turns needed)
- Flexibility for users to tune performance vs completeness
- Predictable behavior (users know headroom available)
- Reduced frustration (fewer incomplete projects)

**Risks**:
- Runaway executions (mitigated by reasonable default 100)
- Longer execution times for complex projects (acceptable trade-off)

**Estimated effort**: 2-3 hours
- Design: ✓ Complete
- Implementation: 1-2 hours (straightforward parameter threading)
- Testing: 30-60 minutes
- Documentation: 30 minutes

## Success Criteria

1. ✅ Provider accepts `claude_max_turns` configuration
2. ✅ Default value is 100
3. ✅ Value passed correctly to Claude CLI
4. ✅ Purl project completes successfully with default 100
5. ✅ Debug output shows correct `--max-turns` value
6. ✅ Backward compatible (no breaking changes)
