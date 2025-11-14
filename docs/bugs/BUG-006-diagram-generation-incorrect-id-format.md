# BUG-006: Diagram Generation Missing Feature Names

**Status:** Open - In Progress
**Priority:** High
**Discovered:** 2025-11-13
**Updated:** 2025-11-14
**Affects Tests:**
- TestE2EProjectExampleInfra3TierAppSuccess

## Summary

When generating draw.io diagrams, Claude creates generic labels ("Load Balancer", "Web Server", "Database") derived from the project description instead of using actual feature names from the Terraform configuration ("load_balancer", "web_server", "database"). This occurs because `project_context` sent to Claude is missing the `features` field entirely.

**Root Cause:** `data.Features` is Null or Unknown when `buildProjectContext()` is called during Create/Update, causing the features processing block to be skipped entirely.

## How to Reproduce

**Test Command:**
```bash
go test -v -run "^TestE2EProjectExampleInfra3TierAppSuccess$" ./test/ -timeout 10m
```

**Expected Behavior:**
1. `project_context` sent to Claude should contain:
   ```json
   {
     "features": [
       {"name": "load_balancer", "prompt": "Configure load balancer infrastructure"},
       {"name": "web_server", "prompt": "Setup web server tier"},
       {"name": "database", "prompt": "Configure database tier"},
       {"name": "architecture_diagram", "prompt": "Generate architecture diagram..."}
     ],
     "kits": [{"id": "tool.drawio"}],
     "project_info": {...}
   }
   ```

2. Claude should generate diagram with IDs/labels matching feature names

**Actual Behavior:**
1. `project_context` sent to Claude only contains:
   ```json
   {
     "kits": [{"id": "tool.drawio"}],
     "project_info": {
       "description": "3-tier infrastructure diagram example with load balancer, web server, and database",
       ...
     }
     // ❌ NO "features" field
   }
   ```

2. Claude generates diagram using generic names from project description:
   - "Load Balancer" instead of "load_balancer"
   - "Web Server 1/2/3" instead of "web_server"
   - "Primary Database" instead of "database"

**Test Output:**
```
Error: "...Load Balancer..." does not contain "load_balancer"
Error: "...Web Server..." does not contain "web_server"
Error: "...Database..." does not contain "database"
```

## Root Cause Investigation

### Key Finding: Features Are Null/Unknown in buildProjectContext()

**Evidence:**
1. `project_context` HAS `kits` but MISSING `features`
2. Kits are added unconditionally (line 3052)
3. Features are only added if `!data.Features.IsNull() && !data.Features.IsUnknown()` (line 2882)
4. Therefore: `data.Features.IsNull()` or `.IsUnknown()` must be TRUE

### Mystery: Features Are Populated in Other Functions

`collectFeatureKitIDs()` debug logs show:
```
collectFeatureKitIDs: Features is basetypes.ObjectValue with 4 attributes
collectFeatureKitIDs: Processing feature 'load_balancer' from ObjectValue
collectFeatureKitIDs: Processing feature 'web_server' from ObjectValue
collectFeatureKitIDs: Processing feature 'architecture_diagram' from ObjectValue
collectFeatureKitIDs: Processing feature 'database' from ObjectValue
```

**This proves features ARE populated as `basetypes.ObjectValue` when `collectFeatureKitIDs()` is called.**

### Theory: Execution Timing Issue

**Hypothesis:** `buildProjectContext()` and `collectFeatureKitIDs()` are called at different times or with different `data` values:
- `collectFeatureKitIDs()`: Called from `buildOutputData()` → features are populated
- `buildProjectContext()`: Called separately → features are Null/Unknown

**Supporting Evidence:**
- Both functions are called from different code paths
- `buildOutputData()` is called from Read(), Create(), Update()
- `buildProjectContext()` is only called from Create() and Update()
- Debug logs show `collectFeatureKitIDs()` executes, but NO logs from `buildProjectContext()`

## Files Involved

**Test File:**
- `/test/e2e_project_example_infra_3_tier_app_test.go:126-128`
  - Lines checking for feature names in XML

**Provider Code:**
- `/internal/resources/project.go`
  - Line 643-650: Create() calls `buildProjectContext()`
  - Line 1386-1399: Update() calls `buildProjectContext()`
  - Line 2853-3073: `buildProjectContext()` implementation
  - Line 2882: Conditional check that skips features if Null/Unknown
  - Line 3028-3053: Kits added unconditionally (why diagram test has kits)

**Terraform Config:**
- `/examples/projects/infra-3-tier-app/project.tofu:136-212`
  - Defines 4 features: load_balancer, web_server, database, architecture_diagram

## Investigation Progress

### Attempted Fix (Incomplete)

Added `basetypes.ObjectValue` handling to `buildProjectContext()` following pattern from commit 8c1e0ec (lines 2959-3021).

**Status:** Fix implemented but NOT verified because `buildProjectContext()` appears to not be executing or receiving Null features.

### Debugging Blockers

1. **No Create()/Update() logs appear** despite adding extensive debug logging
2. Cannot confirm if `buildProjectContext()` is actually being called
3. Cannot confirm the actual state of `data.Features` when called
4. File-based debug logging works for Read() but not Create()/Update()

### Comparison with Working Code

`collectFeatureKitIDs()` successfully processes `basetypes.ObjectValue` features. The same pattern was added to `buildProjectContext()` but cannot verify it's working.

## Next Steps

1. ✅ Add explicit Null/Unknown logging to `buildProjectContext()`
2. ⏸️ Investigate why Create()/Update() debug logs don't appear
3. ⏸️ Determine execution timing difference between `buildProjectContext()` and `collectFeatureKitIDs()`
4. ⏸️ Find why `data.Features` is Null in one context but populated in another

## Related Issues

**See Also:** BUG-008 - Features missing from project_context in buildProjectContext()

## Additional Context

- System prompt tells Claude that `project_context` will contain features
- Requirement instructs Claude to "introspect project_context"
- But actual `project_context` data has no features field
- Claude only has project description: "3-tier infrastructure diagram example with load balancer, web server, and database"
- Claude generates reasonable names from description text since no feature metadata available
