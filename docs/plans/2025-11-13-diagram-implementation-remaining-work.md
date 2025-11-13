# Diagram Feature - Remaining Implementation Work

**Date:** 2025-11-13
**Status:** Implementation Plan
**Author:** Dmitry + Claude
**Prerequisites:** Integration resource implementation (Phase 1) - ✅ COMPLETED

## Overview

This document outlines the remaining work to complete the diagram generation feature with project context introspection. The integration resource (Phase 1) is now complete and tested. This plan covers Phase 2: implementing the introspection mechanism and diagram feature.

## Current Status

### ✅ Completed (Phase 1 - Integration Resource)

1. **Integration Resource Implementation**
   - File: `internal/resources/integration.go`
   - Dedicated IntegrationResource with proper schema
   - Registry methods in `internal/registry/registry.go`
   - BaseComponent updated to handle integration kind
   - URI link format: `tofukit://integration/<name>`

2. **Example Project**
   - Location: `examples/projects/flask-api-nyc-weather/`
   - Demonstrates integration resource usage
   - Shows metadata flowing to project requirements
   - E2E test passing: `test/e2e_project_example_flask_api_weather_app_test.go`

3. **Schema Fields**
   - `name` (required, forces replacement)
   - `type` (required) - Enum: rest_api, graphql_api, grpc_api, webhook, websocket, saas, cloud_service
   - `description` (optional)
   - `base_url` (optional)
   - `docs_url` (optional)
   - `version` (optional)
   - `metadata` (optional) - Map[string]string for key-value data
   - `id` (computed) - Format: `integration.<name>`
   - `link` (computed) - Format: `tofukit://integration/<name>`

## Phase 2 - Remaining Work

### Task 1: Enable Cross-Project Feature References

**Objective:** Allow `tofukit_project.base.features` to be passed to other projects

**Changes Required:**

1. **Update Project Schema** (`internal/resources/project.go`)
   - Location: Line 192-195 (features attribute)
   - Change from `Optional: true` to `Optional: true, Computed: true`
   - This enables features to be both set by users AND read from other projects

   ```go
   // BEFORE
   "features": schema.DynamicAttribute{
       MarkdownDescription: "Map of feature definitions...",
       Optional:            true,
   },

   // AFTER
   "features": schema.DynamicAttribute{
       MarkdownDescription: "Map of feature definitions...",
       Optional:            true,
       Computed:            true,  // ⬅️ ADD THIS
   },
   ```

2. **Testing**
   - Verify `tofukit_project.app.features` can be referenced in another project
   - Test both inline feature definitions and resource references

### Task 2: Implement Project Context Introspection

**Objective:** Build `_project_context` containing all project components for Claude

**Changes Required:**

1. **New Function: `buildProjectContext()`** (`internal/resources/project.go`)
   - Location: Add after `collectAndMergeFiles()` function (around line 1800)
   - Collects all features, integrations, kits from project
   - Returns structured map for Claude

   ```go
   // buildProjectContext constructs the _project_context object for introspection
   func (r *ProjectResource) buildProjectContext(
       ctx context.Context,
       data *ProjectModelFinal,
       reg *registry.Registry,
   ) map[string]interface{} {
       context := make(map[string]interface{})

       // 1. Collect features
       features := []map[string]interface{}{}
       if !data.Features.IsNull() && !data.Features.IsUnknown() {
           underlyingVal := data.Features.UnderlyingValue()
           if featuresMap, ok := underlyingVal.(types.Map); ok {
               // Extract feature metadata
               for name, featureVal := range featuresMap.Elements() {
                   featureMeta := extractFeatureMetadata(ctx, name, featureVal, reg)
                   features = append(features, featureMeta)
               }
           }
       }
       context["features"] = features

       // 2. Collect integrations (from registry)
       integrations := []map[string]interface{}{}
       allIntegrations := reg.GetAllIntegrations()
       for name, integrationData := range allIntegrations {
           integrationMeta := extractIntegrationMetadata(name, integrationData)
           integrations = append(integrations, integrationMeta)
       }
       context["integrations"] = integrations

       // 3. Collect kits
       kits := []map[string]interface{}{}
       if !data.Kits.IsNull() && !data.Kits.IsUnknown() {
           // Extract kit metadata from Dynamic kits field
           kits = extractKitMetadata(ctx, data.Kits)
       }
       context["kits"] = kits

       // 4. Collect files
       files := []string{}
       if !data.Files.IsNull() {
           for path := range data.Files.Elements() {
               files = append(files, path)
           }
       }
       context["files"] = files

       // 5. Project metadata
       context["name"] = data.Name.ValueString()
       context["description"] = data.Description.ValueString()
       context["version"] = data.Version.ValueString()

       return context
   }
   ```

2. **Helper Functions** (same file)
   ```go
   func extractFeatureMetadata(ctx context.Context, name string, featureVal attr.Value, reg *registry.Registry) map[string]interface{}
   func extractIntegrationMetadata(name string, integrationData interface{}) map[string]interface{}
   func extractKitMetadata(ctx context.Context, kits types.Dynamic) []map[string]interface{}
   ```

3. **Integration Point: Update `Create()` and `Update()`**
   - Location: Lines 638-676 (where Claude execution happens)
   - Build project context before Claude execution
   - Pass to `outputData["_project_context"]`

   ```go
   // In Create() and Update(), before calling executeClaude():

   // Build project context for introspection
   projectContext := r.buildProjectContext(ctx, &data, reg)
   outputData["_project_context"] = projectContext
   ```

4. **System Prompt Enhancement** (`internal/llm/claude/prompt_types.go`)
   - Detect `_project_context` in outputData
   - Add instructions to system prompt about using project context
   - Example: "The `project_context` field contains all features, integrations, and kits defined in this project. Use this for generating diagrams or documentation."

### Task 3: Create Diagram Feature Stack

**Objective:** Build reusable diagram generation feature

**Location:** `examples/stacks/tofukit-feature-diagram-drawio/`

**Structure:**
```
examples/stacks/tofukit-feature-diagram-drawio/
├── feature.tofu              # Main feature definition
├── README.md                 # Usage documentation
└── templates/
    └── drawio-template.xml   # (Optional) Base draw.io template
```

**File: `feature.tofu`**
```hcl
resource "tofukit_feature" "diagram_drawio" {
  name        = "diagram-drawio"
  description = "Automatically generates draw.io architecture diagrams via project introspection"

  requirements = [
    {
      name = "Diagram Generation"
      instructions = [
        {
          prompt = <<-EOF
          Generate a draw.io XML diagram visualizing the project architecture.

          DIAGRAM STRUCTURE:
          - Use professional draw.io XML format
          - Create layers for: application, integrations, infrastructure
          - Use standard shapes: rectangles (services), cylinders (databases), clouds (external APIs)

          DATA SOURCE - PROJECT CONTEXT:
          Use the `project_context` field to discover:
          - Features: ${jsonencode(project_context.features)}
          - Integrations: ${jsonencode(project_context.integrations)}
          - Kits/Stack: ${jsonencode(project_context.kits)}
          - Files: ${jsonencode(project_context.files)}

          INTEGRATION RENDERING:
          For each integration in project_context.integrations:
          - Type "rest_api": Use cloud/API icon
          - Type "database": Use cylinder icon
          - Type "saas": Use cloud icon with service name
          - Include base_url if available
          - Show connections to services that use them

          FEATURE RENDERING:
          For each feature in project_context.features:
          - Render as component/module boxes
          - Group related features
          - Show dependencies if available

          LAYOUT:
          - Top: External integrations (APIs, SaaS)
          - Middle: Application services/features
          - Bottom: Infrastructure (databases, caching)
          - Clear directional flow (top-down or left-right)

          FILE FORMAT:
          - Valid draw.io XML
          - Embedded styles for readability
          - Layers for organization
          EOF

          constraints = [
            "Use ONLY data from project_context field - do NOT invent components",
            "Generate valid draw.io XML that can be opened in diagrams.net",
            "Include metadata comments in XML for maintainability",
            "Keep layout clean and organized (avoid overlapping)",
            "Use consistent color scheme (e.g., blue=services, orange=integrations, green=databases)"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q '<mxGraphModel' architecture.drawio && echo 'OK'"
          expect  = "OK"
        }
      ]
    }
  ]

  files = {
    "architecture.drawio" = {
      instructions = [
        {
          prompt = "Generate the draw.io diagram as specified in requirements"
        }
      ]
    }

    "diagram-generation.log" = {
      instructions = [
        {
          prompt = <<-EOF
          Create a log file documenting what was included in the diagram:

          DISCOVERED COMPONENTS:
          - List all features from project_context
          - List all integrations from project_context
          - List all kits from project_context

          DIAGRAM CONTENTS:
          - What was rendered
          - Layout decisions made
          - Any components skipped (with reason)
          EOF
        }
      ]
    }
  }

  verifications = [
    {
      command = "test -f architecture.drawio"
    },
    {
      command = "grep -q 'mxGraphModel' architecture.drawio"
    }
  ]
}

output "feature" {
  value = tofukit_feature.diagram_drawio
}
```

**File: `README.md`**
```markdown
# TofuKit Diagram Generation Feature (Draw.io)

Automatically generates draw.io architecture diagrams via **project context introspection**.

## Features

- **Zero-configuration**: Auto-discovers all project components
- **Draw.io format**: Opens in diagrams.net, VS Code, desktop app
- **Multi-layer**: Separates integrations, services, infrastructure
- **Customizable**: Modify generated diagram in draw.io editor

## Usage

### Option 1: Inline Feature

```hcl
resource "tofukit_project" "app" {
  name = "my-app"

  features = {
    "logging" = tofukit_feature.logging
    "auth"    = tofukit_feature.auth
    "diagram" = tofukit_feature.diagram_drawio  # Auto-discovers components!
  }
}
```

### Option 2: Via Module

```hcl
module "diagram" {
  source = "../../stacks/tofukit-feature-diagram-drawio"
}

resource "tofukit_project" "app" {
  features = {
    "diagram" = module.diagram.feature
  }
}
```

## Output Files

- `architecture.drawio` - Draw.io XML diagram
- `diagram-generation.log` - Generation metadata and component list

## How It Works

1. **Terraform Plan**: Feature is referenced like any other feature (no circular dependency)
2. **Provider Execution**: Provider builds `_project_context` with all features, integrations, kits
3. **Claude Execution**: Diagram feature reads `project_context` and generates diagram
4. **Result**: Draw.io file ready to open and customize

## Requirements

- TofuKit provider 0.1.0+
- Integration resource support
- Project context introspection enabled

## Customization

After generation, open `architecture.drawio` in:
- **diagrams.net** (web): https://app.diagrams.net
- **VS Code**: Install Draw.io Integration extension
- **Desktop**: Download Draw.io app

Customize layout, colors, labels while preserving component structure.

## Example

See: `examples/projects/infra-3-tier-app/` for complete usage example.
```

### Task 4: URI Metadata Extraction for Integrations

**Objective:** Enable URI references to integrations in prompts

**File:** `internal/uri/metadata_integration.go` (NEW)

```go
package uri

import (
    "github.com/tofukit/opentofu-provider-tofukit/internal/resources"
)

// ExtractIntegrationMetadata extracts metadata from an integration resource
func ExtractIntegrationMetadata(data interface{}) map[string]interface{} {
    integrationModel, ok := data.(*resources.IntegrationResourceModel)
    if !ok {
        return nil
    }

    metadata := map[string]interface{}{
        "type":        "integration",
        "name":        integrationModel.Name.ValueString(),
        "link":        integrationModel.Link.ValueString(),
        "description": integrationModel.Description.ValueString(),
    }

    // Optional fields
    if !integrationModel.Type.IsNull() {
        metadata["integration_type"] = integrationModel.Type.ValueString()
    }
    if !integrationModel.BaseURL.IsNull() {
        metadata["base_url"] = integrationModel.BaseURL.ValueString()
    }
    if !integrationModel.DocsURL.IsNull() {
        metadata["docs_url"] = integrationModel.DocsURL.ValueString()
    }
    if !integrationModel.Version.IsNull() {
        metadata["version"] = integrationModel.Version.ValueString()
    }

    return metadata
}
```

**Update:** `internal/uri/registry_builder.go`
```go
// Add to ResolveURI() switch statement:
case "integration":
    data, exists := reg.GetIntegration(uri.Name)
    if !exists {
        return nil, fmt.Errorf("integration not found: %s", uri.Name)
    }
    return ExtractIntegrationMetadata(data), nil
```

### Task 5: Update Example Project (infra-3-tier-app)

**Objective:** Demonstrate full diagram generation with introspection

**Location:** `examples/projects/infra-3-tier-app/`

**Expected Structure:**
```
examples/projects/infra-3-tier-app/
├── main.tofu                # Terraform/provider config
├── integrations.tofu        # External API integrations
├── infrastructure.tofu      # Database, cache, load balancer features
├── application.tofu         # API service, workers features
├── project.tofu             # Main project with diagram feature
└── README.md                # Documentation
```

**Key Elements:**
1. Define multiple integrations (payment API, auth service, monitoring)
2. Include multiple features (API service, worker, cache)
3. Include diagram feature (auto-discovers all components)
4. Demonstrate cross-project feature references

**Example `project.tofu`:**
```hcl
module "diagram" {
  source = "../../stacks/tofukit-feature-diagram-drawio"
}

resource "tofukit_project" "infra" {
  name        = "3-tier-infrastructure"
  description = "Three-tier application infrastructure"
  version     = "1.0.0"

  features = {
    "api_service"  = tofukit_feature.api_service
    "worker"       = tofukit_feature.worker
    "cache"        = tofukit_feature.redis_cache
    "load_balancer" = tofukit_feature.nginx_lb
    "diagram"      = module.diagram.feature  # ⬅️ Auto-discovers all above!
  }
}
```

### Task 6: Testing & Validation

**E2E Test:** `test/e2e_project_example_infra_3_tier_app_test.go`

```go
func TestE2EProjectExampleInfra3TierAppSuccess(t *testing.T) {
    // 1. Run tofu apply
    // 2. Verify architecture.drawio exists
    // 3. Verify diagram contains mxGraphModel XML
    // 4. Verify diagram-generation.log mentions features and integrations
    // 5. Verify draw.io file can be parsed as XML
    // 6. Verify diagram contains integration names from project
}
```

**Validation Steps:**
1. Generated draw.io file is valid XML
2. File can be opened in diagrams.net without errors
3. Diagram contains discovered components
4. Log file lists what was included
5. No manual component specification needed

## Implementation Order

### Recommended Sequence:

1. **Task 1** - Schema update (5 minutes)
   - Enable cross-project feature references
   - Quick win, unlocks other features

2. **Task 4** - URI metadata (30 minutes)
   - Integration URI support
   - Required for diagram feature

3. **Task 2** - Project context introspection (2-3 hours)
   - Core mechanism
   - Most complex but critical

4. **Task 3** - Diagram feature stack (1-2 hours)
   - Build reusable feature
   - Depends on Task 2

5. **Task 5** - Example project (1 hour)
   - Demonstrate full usage
   - Integration test

6. **Task 6** - Testing (1 hour)
   - E2E validation
   - Final verification

**Total Estimated Time:** 6-9 hours

## Success Criteria

- [ ] `tofukit_project.base.features` can be referenced in other projects
- [ ] Provider builds `_project_context` during execution
- [ ] Diagram feature auto-discovers components without manual specification
- [ ] Generated draw.io files open correctly in diagrams.net
- [ ] Example project demonstrates full workflow
- [ ] E2E test validates end-to-end functionality
- [ ] No circular dependencies in Terraform graph
- [ ] Documentation is complete and accurate

## Future Enhancements (Out of Scope)

- PNG export via drawio CLI verification
- Multiple diagram layouts (infrastructure, data flow, sequence)
- Custom styling configuration
- Interactive diagram updates on infrastructure changes
- Integration with CI/CD for diagram diffs

## References

- **Design Document:** `2025-11-13-diagram-generation-integration-resource-design.md`
- **Introspection Spec:** `2025-11-13-project-context-introspection-for-diagrams.md`
- **Integration Resource:** `internal/resources/integration.go`
- **Example Project:** `examples/projects/flask-api-nyc-weather/`

## Notes

- Integration resource (Phase 1) is complete and tested ✅
- This plan covers only Phase 2 (introspection + diagram feature)
- All design decisions follow existing TofuKit patterns
- Implementation should maintain backward compatibility
