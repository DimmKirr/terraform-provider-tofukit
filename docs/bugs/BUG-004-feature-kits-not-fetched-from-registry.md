# BUG-004: Feature Kit Data Not Fetched from Registry

**Status:** Fixed
**Priority:** High
**Discovered:** 2025-11-13
**Fixed:** 2025-11-13
**Affects Tests:**
- TestE2EProjectExampleInfra3TierAppSuccess (PNG export verification)

## Summary

When features reference kit resources (e.g., `kits = [tofukit_tool.drawio]`), the kit IDs are collected but the full kit data (requirements, verifications) is never fetched from the registry and passed to Claude. This results in Claude receiving an empty `kits: {}` object, preventing kit requirements from being executed.

## Root Cause

In `/internal/resources/project.go`, the `buildOutputDataWithFiles()` function:

1. **Lines 1909-2027**: Processes `data.Kits` (project's direct kits from state) ✓
2. **Lines 2032-2041**: Merges stack kits into kits map ✓
3. **MISSING**: No code to fetch and merge feature kits ✗

The `collectFeatureKitIDs()` function (line 3475) correctly extracts kit IDs from features:
```go
featureKitIDs := r.collectFeatureKitIDs(ctx, data)  // Returns ["tool.drawio"]
```

However, these IDs are never used to:
1. Look up each kit in the registry via `registry.GetComponent(kitID)`
2. Extract the full kit data (name, description, requirements, verifications)
3. Add it to the `kits` map that gets passed to Claude

## Evidence

From test execution debug logs (`claude-prompt-20251113-110848.json`):

```json
{
  "specification": {
    "kits": {}  // ❌ Empty despite feature having kits = [tofukit_tool.drawio]
  }
}
```

Expected:
```json
{
  "specification": {
    "kits": {
      "tool.drawio": {
        "id": "tool.drawio",
        "name": "drawio",
        "description": "Draw.io desktop CLI for diagram generation and export",
        "version": "latest",
        "requirements": [
          {
            "name": "Draw.io CLI Installation",
            "instructions": [
              {
                "prompt": "Verify draw.io CLI is installed. If not installed, provide installation instructions...",
                "constraints": [...]
              }
            ],
            "verifications": [
              {"command": "drawio --version && echo 'OK'", "expect": "OK"}
            ]
          }
        ]
      }
    }
  }
}
```

## Impact

- Kit requirements (installation instructions, verifications) are not passed to Claude
- Claude cannot execute kit setup tasks
- File verifications that depend on kits fail (e.g., PNG export requiring drawio CLI)
- Feature kit dependencies are effectively ignored

## Reproduction

```bash
# Run the test that exercises feature kit dependencies
go test -v -run TestE2EProjectExampleInfra3TierAppSuccess ./test/

# Examine debug output
cat test-output/*/output/.debug/claude-prompt-*.json | jq '.specification.kits'
# Output: {} (empty - BUG!)
```

## Configuration Example

Feature with kit dependency:
```hcl
# examples/stacks/tofukit-feature-diagram-drawio/feature.tofu
resource "tofukit_feature" "architecture_diagram" {
  name = "architecture-diagram"

  kits = [tofukit_tool.drawio]  # ← Kit referenced but data not passed to Claude

  files = {
    "architecture.drawio" = {
      verifications = [
        {
          # This verification depends on drawio CLI being installed
          command = "drawio -x -f png -o architecture.drawio.png architecture.drawio && echo 'OK'"
          expect  = "OK"
        }
      ]
    }
  }
}

# examples/projects/infra-3-tier-app/project.tofu
resource "tofukit_project" "three_tier_app" {
  features = {
    "architecture_diagram" = tofukit_feature.architecture_diagram
  }
}
```

## Fix Strategy

Add code in `/internal/resources/project.go` after line 2027 (after processing `data.Kits`, before merging stack kits):

```go
// Collect feature kit IDs
featureKitIDs := r.collectFeatureKitIDs(ctx, data)

// Fetch and add feature kits from registry
if reg != nil && len(featureKitIDs) > 0 {
    for _, kitID := range featureKitIDs {
        if compData, exists := reg.GetComponent(kitID); exists {
            // Convert component data to kit format
            // Add to kits map similar to how stack kits are processed
            if componentModel, ok := compData.(ComponentResourceModel); ok {
                kitData := convertComponentToKitData(ctx, componentModel)
                kits[kitID] = kitData

                tflog.Info(ctx, "Added feature kit from registry", map[string]interface{}{
                    "kit_id": kitID,
                })
            }
        } else {
            tflog.Warn(ctx, "Feature kit not found in registry", map[string]interface{}{
                "kit_id": kitID,
            })
        }
    }
}
```

Helper function needed:
```go
func convertComponentToKitData(ctx context.Context, component ComponentResourceModel) map[string]interface{} {
    // Similar to lines 1957-2017 for processing data.Kits
    // Extract: id, name, description, version, requirements (with instructions/verifications)
}
```

## Related Files

- `/internal/resources/project.go:1594` - `buildOutputDataWithFiles()` function (where fix goes)
- `/internal/resources/project.go:3475` - `collectFeatureKitIDs()` function (already works)
- `/internal/resources/base_component.go:56` - Component registration (tools → `registry.SetComponent()`)
- `/internal/registry/registry.go:36` - `GetComponent()` API
- `/examples/stacks/tofukit-feature-diagram-drawio/feature.tofu` - Feature with kit dependency
- `/examples/projects/infra-3-tier-app/project.tofu` - Project using feature
- `/test/e2e_project_example_infra_3_tier_app_test.go` - Test that exposes bug

## Notes

- Kit IDs are collected correctly - the issue is purely that the full data is never fetched
- Tools (and other components) DO register themselves: `registry.SetComponent(id, data)`
- Stack kits work correctly (lines 2032-2041), providing a pattern to follow
- The system prompt tells Claude to "Install and configure all kits" but receives empty kits object

## Resolution

**Fixed in commit:** a0539d1

**Root Cause:**
The bug had two parts:

1. **Part 1:** `collectFeatureKitIDs()` couldn't access features because it expected `types.Map` but the Dynamic type's underlying value was `basetypes.ObjectValue`
   - After the Features field was changed to Dynamic, `data.Features.UnderlyingValue()` returned `basetypes.ObjectValue` instead of `types.Map`
   - The function only handled `types.Map`, causing the cast to fail and return an empty list

2. **Part 2:** Even with kit IDs collected, they were never fetched from the registry and added to the kits map sent to Claude
   - Kit IDs were collected but never used to look up kit data via `registry.GetComponent(kitID)`
   - The kits map remained empty

**Fix Applied:**

1. **Lines 3524-3632:** Updated `collectFeatureKitIDs()` to handle both `types.Map` AND `basetypes.ObjectValue`:
   - Added conditional logic to check for both types
   - Extract attributes from ObjectValue using `.Attributes()`
   - Process features the same way regardless of underlying type

2. **Lines 2032-2144:** Added feature kit fetching in `buildOutputDataWithFiles()`:
   - Call `collectFeatureKitIDs()` to get kit IDs
   - For each kit ID, fetch component from registry via `registry.GetComponent(kitID)`
   - Convert `ComponentResourceModel` to kit data format (id, name, description, version, requirements with instructions/verifications)
   - Add kit data to the kits map that gets passed to Claude

**Verification:**
- Debug logs confirm kit is collected, fetched from registry, and added to kits map
- Claude prompt JSON shows full kit data in `.request.specification.kits["tool.drawio"]` with all requirements, instructions, and verifications
- Kit requirements are now properly passed to Claude for execution
