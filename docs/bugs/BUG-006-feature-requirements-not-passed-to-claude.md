# BUG-006: Feature Requirements Not Passed to Claude

**Status:** Fixed
**Priority:** High
**Discovered:** 2025-11-13
**Fixed:** 2025-11-13
**Affects Tests:**
- TestE2EProjectExampleFlaskAPIWeatherAppSuccess
- All tests with features that have requirements

## Summary

When projects reference features that have requirements, those feature requirements are NOT being passed to Claude in the prompt. The `convertFeaturesToRequirements()` function exists and is called, but returns an empty list because it doesn't handle `basetypes.ObjectValue` - only `types.Map`.

## Root Cause

In `/internal/resources/project.go`, the `convertFeaturesToRequirements()` function (lines 3550-3586):

1. **Line 3557**: Extracts underlying value from Dynamic type
2. **Lines 3559-3566**: Tries to cast to `types.Map`
3. **If cast fails**: Logs warning and returns empty list ❌
4. **MISSING**: No fallback to handle `basetypes.ObjectValue` ❌

Compare with `collectFeatureKitIDs()` (lines 3588-3700) which correctly handles BOTH:
- `types.Map` (lines 3611-3657) ✓
- `basetypes.ObjectValue` (lines 3658-3700) ✓

## Evidence

From test execution (`claude-prompt-20251113-132456.json`):

**Expected** (3 requirements):
```json
{
  "specification": {
    "requirements": [
      {"name": "Flask Application Setup"},      // ✓ From project
      {"name": "Project Documentation"},        // ✓ From project
      {"name": "NYC Weather Integration"}       // ❌ MISSING from feature!
    ]
  }
}
```

**Actual** (2 requirements):
```json
{
  "specification": {
    "requirements": [
      {"name": "Flask Application Setup"},
      {"name": "Project Documentation"}
    ]
  }
}
```

The feature requirement "NYC Weather Integration" with all the interpolated integration details (API Base, Documentation URLs, etc.) never reaches Claude.

## Impact

- Feature requirements (containing integration details, implementation guidance) are silently dropped
- Claude doesn't receive integration metadata even though it's defined
- Features with requirements appear to work but their requirements are ignored
- Integration schema redesign benefits (structured docs, environments, examples) never reach Claude
- Users must manually duplicate feature requirements in project requirements as workaround

## Reproduction

```bash
# 1. Create a feature with requirements
# examples/projects/flask-api-nyc-weather/features.tofu
resource "tofukit_feature" "nyc_weather" {
  requirements = [{
    name = "NYC Weather Integration"
    instructions = [{
      prompt = "Implement weather fetching..."
    }]
  }]
}

# 2. Reference feature in project
# examples/projects/flask-api-nyc-weather/project.tofu
resource "tofukit_project" "flask_weather" {
  features = {
    "nyc_weather" = tofukit_feature.nyc_weather
  }
}

# 3. Run test and check Claude prompt
go test -v -run TestE2EProjectExampleFlaskAPIWeatherAppSuccess ./test/

# 4. Examine debug output
cat test-output/*/output/.debug/claude-prompt-*.json | jq '.request.specification.requirements | length'
# Output: 2 (should be 3!)

cat test-output/*/output/.debug/claude-prompt-*.json | jq '.request.specification.requirements[].name'
# Missing "NYC Weather Integration"!
```

## Configuration Example

Feature with requirements (never passed to Claude):
```hcl
# examples/projects/flask-api-nyc-weather/features.tofu
resource "tofukit_feature" "nyc_weather" {
  name = "nyc-weather-data"

  requirements = [
    {
      name = "NYC Weather Integration"
      instructions = [
        {
          prompt = <<-EOF
            Implement NYC weather data fetching using the Open-Meteo Weather API integration.

            INTEGRATION DETAILS:
            - Service: ${tofukit_integration.open_meteo.description}
            - API Base: ${tofukit_integration.open_meteo.environments["production"]}
            - Documentation: ${tofukit_integration.open_meteo.docs["main"]}
            - API Details: ${tofukit_integration.open_meteo.docs["api"]}
          EOF

          constraints = [
            "Use ${tofukit_integration.open_meteo.link} as the data source",
            "NO authentication required (free API)"
          ]
        }
      ]

      verifications = [
        {command = "grep -q 'api.open-meteo.com' app.py"}
      ]
    }
  ]
}

# examples/projects/flask-api-nyc-weather/project.tofu
resource "tofukit_project" "flask_weather" {
  features = {
    "nyc_weather" = tofukit_feature.nyc_weather  # ← Requirements not passed to Claude!
  }
}
```

## Fix Strategy

Update `/internal/resources/project.go` line 3550 - `convertFeaturesToRequirements()` to handle both types like `collectFeatureKitIDs()` does:

```go
func (r *ProjectResourceFinal) convertFeaturesToRequirements(ctx context.Context, data ProjectModelFinal) []schemas.RequirementModel {
	if data.Features.IsNull() || data.Features.IsUnknown() {
		return []schemas.RequirementModel{}
	}

	// Extract the underlying value from Dynamic
	underlyingVal := data.Features.UnderlyingValue()

	var requirements []schemas.RequirementModel

	// Try to cast to types.Map (for map structure)
	if featuresMap, ok := underlyingVal.(types.Map); ok {
		for featureName, featureValue := range featuresMap.Elements() {
			feature, err := r.parseFeature(ctx, featureValue)
			if err != nil {
				continue
			}

			// Feature already has requirements! Just pass them through
			requirements = append(requirements, feature.Requirements...)

			tflog.Debug(ctx, "Converted feature to requirements", map[string]interface{}{
				"feature_name":      featureName,
				"requirement_count": len(feature.Requirements),
			})
		}
	} else if fObj, ok := underlyingVal.(basetypes.ObjectValue); ok {
		// NEW: Handle basetypes.ObjectValue (Dynamic's underlying type for objects)
		tflog.Info(ctx, "Features is basetypes.ObjectValue", map[string]interface{}{
			"attribute_count": len(fObj.Attributes()),
		})

		for featureName, featureValue := range fObj.Attributes() {
			feature, err := r.parseFeature(ctx, featureValue)
			if err != nil {
				tflog.Warn(ctx, "Failed to parse feature from ObjectValue", map[string]interface{}{
					"feature_name": featureName,
					"error":        err.Error(),
				})
				continue
			}

			// Feature already has requirements! Just pass them through
			requirements = append(requirements, feature.Requirements...)

			tflog.Debug(ctx, "Converted feature to requirements from ObjectValue", map[string]interface{}{
				"feature_name":      featureName,
				"requirement_count": len(feature.Requirements),
			})
		}
	} else {
		tflog.Warn(ctx, "Features is neither types.Map nor basetypes.ObjectValue", map[string]interface{}{
			"type": fmt.Sprintf("%T", underlyingVal),
		})
		return []schemas.RequirementModel{}
	}

	return requirements
}
```

## Related Files

- `/internal/resources/project.go:3550` - `convertFeaturesToRequirements()` function (WHERE BUG IS)
- `/internal/resources/project.go:1657` - Where `convertFeaturesToRequirements()` is called
- `/internal/resources/project.go:3588` - `collectFeatureKitIDs()` function (CORRECT PATTERN TO FOLLOW)
- `/examples/projects/flask-api-nyc-weather/features.tofu` - Feature with requirements that get dropped
- `/examples/projects/flask-api-nyc-weather/integration.tofu` - Integration resource with structured docs
- `/test/e2e_project_example_flask_api_nyc_weather_test.go` - Test that exposes bug

## Notes

- The code to ADD feature requirements to outputData exists (lines 1657-1703) ✓
- The problem is convertFeaturesToRequirements returns empty list ✗
- Same pattern as BUG-005 (feature kits) - Dynamic type handling inconsistency
- The fix is copy-paste the basetypes.ObjectValue handling from collectFeatureKitIDs
- This bug prevents the integration schema redesign benefits from reaching Claude

## Resolution

**Fixed in:** Commit 8c1e0ec (integration-schema-redesign worktree)
**Test Status:** ✅ PASSED - TestE2EProjectExampleFlaskAPIWeatherAppSuccess

**Root Cause Confirmed:**
The `convertFeaturesToRequirements()` function only handled `types.Map`, but when features are stored in Terraform state, the Dynamic type's underlying value is `basetypes.ObjectValue`, not `types.Map`. This caused the type cast to fail and the function to return an empty list.

**Fix Applied:**
Updated `/internal/resources/project.go` lines 3550-3659 to handle BOTH types:
1. **Primary path**: `types.Map` (lines 3621-3657)
2. **NEW path**: `basetypes.ObjectValue` (lines 3573-3611)

Both paths now:
- Iterate through features
- Call `parseFeature()` to extract feature data
- Append feature requirements to the requirements list
- Log detailed DEBUG information for troubleshooting

**Verification:**
Before fix:
```json
{
  "specification": {
    "requirements": [
      {"name": "Flask Application Setup"},
      {"name": "Project Documentation"}
    ]
  }
}
```

After fix:
```json
{
  "specification": {
    "requirements": [
      {"name": "Flask Application Setup"},
      {"name": "Project Documentation"},
      {"name": "NYC Weather Integration"}  // ✅ Now present!
    ]
  }
}
```

The "NYC Weather Integration" requirement now includes:
- All integration details (API Base, Documentation URLs)
- Terraform interpolation resolved values
- Full implementation instructions
- Verification commands

**Test Evidence:**
- Test: `TestE2EProjectExampleFlaskAPIWeatherAppSuccess`
- Result: PASSED (69.79s)
- Requirements count: 3 (was 2)
- Integration details confirmed in Claude prompt

## Workaround (No longer needed after fix)

~~Manually duplicate feature requirements in project requirements~~

This workaround is no longer necessary after the fix.
