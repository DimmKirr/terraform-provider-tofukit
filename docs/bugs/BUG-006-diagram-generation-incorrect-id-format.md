# BUG-006: Diagram Generation Uses Wrong ID Format

**Status:** Open
**Priority:** Medium
**Discovered:** 2025-11-13
**Affects Tests:**
- TestE2EProjectExampleInfra3TierAppSuccess

## Summary

When generating draw.io diagrams, Claude creates IDs using snake_case (e.g., `id="lb"`) instead of the expected format that test assertions check for (e.g., `"load_balancer"`). Additionally, the PNG export file `architecture.drawio.png` is not being generated.

## How to Reproduce

**Test Command:**
```bash
go test -v -run "^TestE2EProjectExampleInfra3TierAppSuccess$" ./test/ -timeout 10m
```

**Expected Behavior:**
1. Generated XML diagram should contain specific IDs that match the architecture components:
   - `"load_balancer"` for load balancer component
   - `"web_server"` for web server components
   - `"database"` for database component

2. PNG file `architecture.drawio.png` should be generated from the XML diagram

**Actual Behavior:**
1. XML diagram uses human-readable labels but abbreviated IDs:
   - Uses `id="lb"` instead of full identifier
   - Uses `id="ws1"`, `id="ws2"`, `id="ws3"` for web servers
   - Uses `id="db"` for database
   - Test assertions fail looking for full names in the XML

2. PNG file is not generated:
   ```
   Error: unable to find file ".../output/architecture.drawio.png"
   ```

**Test Output:**
```
Error: "<?xml version=\"1.0\"...Load Balancer..." does not contain "load_balancer"
Error: ...does not contain "web_server"
Error: ...does not contain "database"
Error: unable to find file ".../architecture.drawio.png"
```

## Files Involved

**Test File:**
- `/Users/dmitry/dev/dimmkirr/terraform-provider-tofukit/test/e2e_project_example_infra_3_tier_app_test.go:126-149`
  - Lines 126-128: XML content assertions checking for component IDs
  - Line 143: PNG file existence check
  - Line 149: PNG content validation

**Provider Code:**
- Likely in diagram generation logic (Claude prompt or post-processing)
- May involve XML parsing/generation code
- PNG export functionality

## Top 3 Theories for Fix

### Theory 1: Test Assertions Are Too Strict
**Likelihood:** High
**Reasoning:** The XML uses short IDs (`lb`, `ws1`, `db`) which are valid but don't match full names. Tests expect full identifiers like `"load_balancer"`.

**Potential Fix:**
- Update test assertions to check for the actual ID format used (`id="lb"`, `id="ws1"`, `id="db"`)
- OR update test to check for the human-readable values instead (`value="Load Balancer"`)
- Tests should match the actual output format, not assume a specific ID scheme

### Theory 2: Claude Prompt Needs to Specify ID Format
**Likelihood:** Medium
**Reasoning:** Claude is generating valid diagrams but using abbreviated IDs. The prompt may not specify the required ID format.

**Potential Fix:**
- Update the system prompt or requirement instructions to specify: "Use full component names as IDs (e.g., `id=\"load_balancer\"` not `id=\"lb\"`)"
- Add examples showing the expected ID format
- Add verification step that checks ID format matches requirements

### Theory 3: PNG Export Not Implemented
**Likelihood:** High
**Reasoning:** The draw.io XML is generated correctly, but there's no code to convert it to PNG format.

**Potential Fix:**
- Implement PNG export using draw.io CLI tools or libraries
- Add a post-processing step after XML generation that:
  1. Checks if draw.io CLI is available
  2. Runs: `drawio --export --format png --output architecture.drawio.png architecture.drawio.xml`
  3. Handles case where draw.io is not installed (skip PNG generation with warning)
- Alternative: Use headless Chrome/Puppeteer to render the XML as PNG
- Alternative: Make PNG generation optional and update test to check XML only

## Related Code Locations

**Diagram Generation:**
- Likely in project execution flow when requirements include diagram generation
- May be in Claude prompt templates
- XML parsing/validation logic

**PNG Export:**
- May need new implementation (doesn't exist yet)
- Would require external tool (draw.io CLI) or library integration

## Additional Context

- The XML diagram itself is valid and contains all required components
- The issue is with ID naming convention and missing PNG export
- This is a test expectation mismatch combined with missing feature (PNG export)
