# Project Context Introspection for Diagram Generation

**Date:** 2025-11-13
**Status:** Design
**Author:** Dmitry + Claude

## Overview

Enable diagram features to automatically introspect their parent project's components (features, integrations, files, kits) without creating circular dependencies. This allows diagrams to be declaratively included as features within projects while dynamically discovering what to visualize.

## Problem Statement

### Current Circular Dependency Issue

When trying to include diagram generation as a feature within a project:

```hcl
# ❌ CIRCULAR DEPENDENCY - Won't work!
module "architecture_diagram" {
  source = "../../stacks/tofukit-feature-diagram-drawio"

  project = {
    services = [...]      # ⬅️ Must manually specify components
    databases = [...]
    integrations = [...]
  }
}

resource "tofukit_project" "app" {
  features = {
    "logging" = tofukit_feature.logging
    "diagram" = module.architecture_diagram.feature  # ⬅️ App needs module
  }
}

# LOOP: app → module → app (module needs app data) → CIRCULAR!
```

**Problems:**
1. **Circular dependency**: Module needs app data, app needs module
2. **Manual duplication**: Components must be manually listed in module call
3. **Drift risk**: Manual list can become outdated as project evolves
4. **Not declarative**: Diagram content separated from actual architecture

### Desired Behavior

```hcl
# ✅ IDEAL - No circular dependency
resource "tofukit_project" "app" {
  features = {
    "logging" = tofukit_feature.logging
    "auth" = tofukit_feature.auth
    "diagram" = tofukit_feature.diagram_auto  # ⬅️ Auto-discovers components!
  }
}
```

**Requirements:**
- Diagram feature included like any other feature
- Automatically discovers all project components
- No manual duplication of component lists
- No circular dependencies in Terraform graph

## Solution: Project Context Introspection

### Core Concept

**Separate Terraform dependency graph from runtime data flow:**

1. **Terraform plan phase**: Feature is just a reference (no data needed)
2. **Provider execution phase**: Provider builds full project context and passes to Claude
3. **Claude execution**: Diagram feature introspects project context at runtime

**Key insight**: The diagram feature doesn't need project data at *Terraform plan time*, only at *Claude execution time*.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ Terraform Plan Phase (Dependency Graph)                         │
│                                                                  │
│  feature.logging ──┐                                            │
│  feature.auth ─────┼──▶ project.app                             │
│  feature.diagram ──┘      (collects features)                   │
│                                                                  │
│  ✅ No circular dependency at Terraform level                   │
└─────────────────────────────────────────────────────────────────┘
                            │
                            │ Provider executes project
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ Provider Execution Phase (Runtime Data Flow)                    │
│                                                                  │
│  1. Collect all features ────────┐                              │
│  2. Collect all integrations ────┤                              │
│  3. Collect all kits ────────────┼──▶ Build _project_context   │
│  4. Collect all files ───────────┘                              │
│                                                                  │
│  5. Pass context to Claude in outputData["_project_context"]    │
└─────────────────────────────────────────────────────────────────┘
                            │
                            │ Claude receives prompt
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│ Claude Execution (Feature Introspection)                        │
│                                                                  │
│  Diagram feature prompt:                                         │
│  "Inspect project_context field for all components"             │
│                                                                  │
│  Claude sees:                                                    │
│  {                                                               │
│    "project_context": {                                          │
│      "features": [                                               │
│        {"name": "logging", "prompt": "...", "files": [...]}     │
│        {"name": "auth", "prompt": "...", "files": [...]}        │
│      ],                                                          │
│      "integrations": [...],                                      │
│      "kits": [...]                                               │
│    }                                                             │
│  }                                                               │
│                                                                  │
│  ✅ Diagram feature generates visualization from introspected    │
│     context without needing Terraform-time data                 │
└─────────────────────────────────────────────────────────────────┘
```

## Implementation Design

### Component 1: Project Context Collection

**File:** `internal/resources/project.go`

**Location:** In `Update()` method, after collecting all features/kits/integrations, before building `outputData`

**New function:** `buildProjectContext()`

```go
// buildProjectContext creates metadata about the project for introspection by features
func (r *ProjectResource) buildProjectContext(
    ctx context.Context,
    data *ProjectResourceModel,
    collectedFeatures []FeatureModel,
    collectedKits []map[string]interface{},
    collectedIntegrations []string,
) map[string]interface{} {
    projectContext := map[string]interface{}{
        "project_info": map[string]interface{}{
            "name":        data.Name.ValueString(),
            "description": data.Description.ValueString(),
            "version":     data.Version.ValueString(),
        },
    }

    // Add features metadata
    if len(collectedFeatures) > 0 {
        featuresMetadata := []map[string]interface{}{}
        for _, feature := range collectedFeatures {
            featureMeta := map[string]interface{}{
                "name":        feature.Name,
                "prompt":      feature.Prompt,
                "description": feature.Description,
            }

            // Add feature files if present
            if len(feature.Files) > 0 {
                files := []string{}
                for path := range feature.Files {
                    files = append(files, path)
                }
                featureMeta["files"] = files
            }

            // Add feature kits if present
            if len(feature.Kits) > 0 {
                featureMeta["kits"] = feature.Kits
            }

            featuresMetadata = append(featuresMetadata, featureMeta)
        }
        projectContext["features"] = featuresMetadata
    }

    // Add integrations metadata
    if len(collectedIntegrations) > 0 {
        integrationsMetadata := []map[string]interface{}{}
        for _, integrationID := range collectedIntegrations {
            // Look up integration from registry
            if integration, found := registry.GetIntegration(integrationID); found {
                integrationsMetadata = append(integrationsMetadata, map[string]interface{}{
                    "name":        integration.Name.ValueString(),
                    "type":        integration.Type.ValueString(),
                    "description": integration.Description.ValueString(),
                    "base_url":    integration.BaseURL.ValueString(),
                })
            }
        }
        if len(integrationsMetadata) > 0 {
            projectContext["integrations"] = integrationsMetadata
        }
    }

    // Add kits metadata
    if len(collectedKits) > 0 {
        projectContext["kits"] = collectedKits
    }

    // Add requirements metadata
    if !data.Requirements.IsNull() && !data.Requirements.IsUnknown() {
        var requirements []RequirementModel
        diags := data.Requirements.ElementsAs(ctx, &requirements, false)
        if !diags.HasError() && len(requirements) > 0 {
            requirementsMetadata := []map[string]interface{}{}
            for _, req := range requirements {
                requirementsMetadata = append(requirementsMetadata, map[string]interface{}{
                    "name": req.Name.ValueString(),
                })
            }
            projectContext["requirements"] = requirementsMetadata
        }
    }

    return projectContext
}
```

**Integration point in `Update()`:**

```go
// Line ~630 in Update(), after collecting features/kits/integrations:

// Build project context for feature introspection
projectContext := r.buildProjectContext(ctx, data, collectedFeatures, collectedKits, collectedIntegrationIDs)

// Build output data with enriched files
outputData := r.buildOutputDataWithFiles(ctx, data, enrichedFiles)

// Add project context to output data
if projectContext != nil && len(projectContext) > 0 {
    outputData["_project_context"] = projectContext
}

// Add resource registry to output data if present
if resourceRegistry != nil && len(resourceRegistry) > 0 {
    outputData["_resource_registry"] = resourceRegistry
}
```

**Similar changes needed in:**
- `Create()` method (~line 290)
- Retry logic in `Update()` (~line 1325)

### Component 2: Prompt Enhancement

**File:** `internal/llm/claude/prompt_types.go`

**Location:** In `BuildProjectPrompt()` function, after resource registry extraction

**Change:** Add project context extraction and system prompt enhancement

```go
// Line ~207, after resource registry extraction:

// Extract project context if present
var projectContext map[string]interface{}
if ctx, ok := projectSpec["_project_context"].(map[string]interface{}); ok {
    projectContext = ctx

    // Enhance system prompt with project context instructions
    systemPrompt += "\n\n## Project Context Introspection\n\n" +
        "The project_context field in the specification contains complete metadata about this project:\n" +
        "- features: All features defined in this project with their prompts, files, and capabilities\n" +
        "- integrations: All external API/service integrations referenced by this project\n" +
        "- kits: All language/framework/tool kits configured for this project\n" +
        "- requirements: All high-level requirements for this project\n\n" +
        "Features can introspect this context to automatically discover project components without requiring explicit configuration. " +
        "For example, diagram generation features can visualize the entire architecture by reading the project_context field."
}

// Update PromptRequest struct initialization to include project context:
prompt := &ProjectPrompt{
    SystemPrompt: systemPrompt,
    Request: PromptRequest{
        Type:             "project_implementation",
        ProjectInfo:      projectInfo,
        Specification:    projectSpec,
        ResourceRegistry: resourceRegistry,
        ProjectContext:   projectContext,  // ⬅️ NEW FIELD
        Instructions:     instructions,
        // ... rest of fields
    },
}
```

**Update `PromptRequest` struct:**

```go
// PromptRequest contains the actual project implementation request
type PromptRequest struct {
    Type             string                   `json:"type"`
    ProjectInfo      ProjectInfo              `json:"project_info"`
    Specification    map[string]interface{}   `json:"specification"`
    ResourceRegistry map[string]interface{}   `json:"resource_registry,omitempty"`
    ProjectContext   map[string]interface{}   `json:"project_context,omitempty"`  // ⬅️ NEW
    Instructions     []string                 `json:"instructions"`
    FileDetails      *FileInstructions        `json:"file_details,omitempty"`
    FileOperations   []map[string]interface{} `json:"file_operations,omitempty"`
    Guidelines       []string                 `json:"guidelines"`
    Deliverables     []string                 `json:"deliverables"`
}
```

### Component 3: Diagram Feature Implementation

**File:** `examples/stacks/tofukit-feature-diagram-drawio/feature.tofu`

**Change:** Remove `var.project` requirement, use introspection instead

**New approach:**

```hcl
resource "tofukit_feature" "diagram_drawio" {
  name        = "diagram-drawio"
  description = "Auto-generate draw.io architecture diagrams from project introspection"

  requirements = [
    {
      name = "Draw.io Diagram Generation via Project Introspection"

      instructions = [
        {
          prompt = <<-EOF
          Generate a professional architecture diagram in draw.io XML format by introspecting the project context.

          INTROSPECTION INSTRUCTIONS:
          1. Read the project_context field from the specification JSON
          2. Extract all components:
             - features: Application capabilities and their files
             - integrations: External APIs and services
             - kits: Languages, frameworks, and tools
             - requirements: High-level architectural requirements
          3. Analyze relationships between components (which features use which integrations/kits)

          DIAGRAM STRUCTURE:
          1. Create valid draw.io XML with <mxGraphModel> root element
          2. Organize components in logical tiers (top to bottom):
             - External Integrations (cloud shapes)
             - Application Features (rounded rectangles)
             - Infrastructure/Kits (cylinders or rectangles)
          3. Add arrows showing dependencies and data flow:
             - Feature → Integration: API calls
             - Feature → Kit: Uses framework/tool
             - Feature → Feature: Internal dependencies
          4. Label each component with:
             - Name (bold)
             - Type/description (smaller text)
          5. Use professional color scheme:
             - Features: Blue tones
             - Integrations: Orange/cloud colors
             - Kits: Gray/infrastructure colors

          OUTPUT FILE: ${var.file_path}.drawio

          The diagram should comprehensively visualize the entire project architecture
          without requiring manual component specification.
          EOF

          constraints = [
            "MUST read project_context field from specification",
            "MUST include ALL features found in project_context.features",
            "MUST include ALL integrations found in project_context.integrations",
            "MUST include ALL kits found in project_context.kits",
            "Output must be valid draw.io XML format",
            "File must be saved to exact path: ${var.file_path}.drawio",
            "Use standard draw.io shape library (no custom shapes)",
            "Arrange components logically with clear visual hierarchy",
            "Show dependencies with arrows (shortest path, above all blocks)",
            "NO manual text descriptions - use draw.io native labels",
            "Diagram must be viewable in draw.io desktop application"
          ]
        }
      ]
    }
  ]

  files = {
    "${var.file_path}.drawio" = {
      instructions = [
        {
          prompt = "Generate draw.io XML diagram via project introspection as specified in requirements"
        }
      ]

      verifications = [
        {
          command = "test -f ${var.file_path}.drawio && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "grep -q '<mxGraphModel' ${var.file_path}.drawio && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "drawio -x -f png --scale 2.5 -o ${var.file_path}.drawio.png ${var.file_path}.drawio && echo 'OK'"
          expect  = "OK"
        }
      ]
    }
  }

  kits = [tofukit_tool.drawio]
}
```

**Update `variables.tofu`:**

```hcl
# Remove var.project entirely - no longer needed!

variable "file_path" {
  description = "Output path for diagram file (without .drawio extension)"
  type        = string
  default     = "architecture"

  validation {
    condition     = !endswith(var.file_path, ".drawio")
    error_message = "file_path should not include .drawio extension (it will be added automatically)"
  }
}
```

### Component 4: Updated Example Usage

**File:** `examples/projects/infra-3-tier-app/project.tofu`

**New approach - No module, direct feature reference:**

```hcl
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  debug                 = true
  claude_home_directory = "~/.claude"
}

# Define diagram feature (can be standalone or inline)
resource "tofukit_feature" "architecture_diagram" {
  name        = "architecture-diagram"
  description = "Auto-generated architecture diagram"

  requirements = [
    {
      name = "Generate Architecture Diagram"

      instructions = [
        {
          prompt = <<-EOF
          Generate architecture diagram by introspecting project_context.

          Show all features, integrations, and infrastructure components
          discovered in the project context.
          EOF
        }
      ]
    }
  ]

  files = {
    "docs/architecture.drawio" = {
      instructions = [
        {
          prompt = "Generate draw.io diagram from project_context"
        }
      ]

      verifications = [
        {
          command = "test -f docs/architecture.drawio && echo 'OK'"
          expect  = "OK"
        }
      ]
    }
  }
}

# Main project with features
resource "tofukit_project" "three_tier_app" {
  name        = "infra-3-tier-app"
  description = "3-tier infrastructure with auto-generated diagram"
  version     = "1.0.0"

  model = "haiku"

  # ✅ Diagram feature just like any other feature!
  features = {
    "load_balancer" = {
      prompt = "Configure load balancer infrastructure"
      # ... feature definition
    }

    "web_server" = {
      prompt = "Setup web server tier"
      # ... feature definition
    }

    "database" = {
      prompt = "Configure database tier"
      # ... feature definition
    }

    "architecture_diagram" = tofukit_feature.architecture_diagram  # ⬅️ Auto-discovers all above features!
  }
}
```

## Data Flow Example

### Input Project Configuration

```hcl
resource "tofukit_project" "app" {
  features = {
    "logging" = tofukit_feature.logging
    "auth" = tofukit_feature.auth
    "diagram" = tofukit_feature.diagram_auto
  }
}
```

### Provider Builds Context

```json
{
  "_project_context": {
    "project_info": {
      "name": "app",
      "description": "Application with logging and auth",
      "version": "1.0.0"
    },
    "features": [
      {
        "name": "logging",
        "prompt": "Add structured logging capability",
        "description": "JSON-structured logging with rotation",
        "files": ["logger.py", "logger_config.json"],
        "kits": ["python"]
      },
      {
        "name": "auth",
        "prompt": "Implement JWT authentication",
        "description": "Token-based authentication system",
        "files": ["auth.py", "middleware/auth_middleware.py"],
        "kits": ["python", "jwt"]
      },
      {
        "name": "diagram",
        "prompt": "Generate architecture diagram via project introspection",
        "description": "Auto-generated draw.io diagram",
        "files": ["architecture.drawio"]
      }
    ],
    "kits": ["python", "jwt"]
  }
}
```

### Claude Receives Prompt

```json
{
  "system_prompt": "...\n\n## Project Context Introspection\n\nThe project_context field contains complete metadata...",
  "request": {
    "type": "project_implementation",
    "project_info": { "name": "app", ... },
    "specification": { ... },
    "project_context": {
      "features": [
        {"name": "logging", "files": ["logger.py"], ...},
        {"name": "auth", "files": ["auth.py"], ...}
      ]
    },
    "instructions": [...]
  }
}
```

### Diagram Feature Introspects Context

When executing the diagram feature's instructions, Claude:
1. Reads `project_context.features` array
2. Discovers "logging" and "auth" features (excluding itself)
3. Generates draw.io XML with:
   - Box for "logging" feature
   - Box for "auth" feature
   - Arrows showing relationships
4. Writes to `architecture.drawio`

## Implementation Steps

### Phase 1: Core Introspection Mechanism (1-2 days)

> **⚠️ SCHEMA REQUIREMENT:** For features to be referenceable from other projects (e.g., `tofukit_project.base.features`), the `features` attribute must be **both Optional AND Computed**.
>
> **Current state** (`internal/resources/project.go:192-195`):
> ```go
> "features": schema.DynamicAttribute{
>     MarkdownDescription: "Features to implement...",
>     Optional:            true,   // ✅ Already set
>     // ❌ MISSING: Computed: true
> },
> ```
>
> **Required change:** Add `Computed: true` to enable cross-project feature references:
> ```go
> "features": schema.DynamicAttribute{
>     MarkdownDescription: "Features to implement...",
>     Optional:            true,   // Users can SET features
>     Computed:            true,   // Terraform can READ features from other projects
> },
> ```

**1.1 Add Project Context Collection**
- [ ] Create `buildProjectContext()` function in `internal/resources/project.go`
- [ ] Extract feature metadata (name, prompt, files, kits)
- [ ] Extract integration metadata (via registry lookup)
- [ ] Extract kit metadata
- [ ] Extract requirements metadata

**1.2 Integrate into Project Lifecycle**
- [ ] Verify `features` attribute is both Optional AND Computed (add if missing)
- [ ] Call `buildProjectContext()` in `Create()` method
- [ ] Call `buildProjectContext()` in `Update()` method
- [ ] Call `buildProjectContext()` in retry logic
- [ ] Add `_project_context` to `outputData` map

**1.3 Testing**
- [ ] Unit test for `buildProjectContext()` function
- [ ] Verify context is populated correctly with multiple features
- [ ] Verify context includes integrations when referenced
- [ ] Verify context passed to Claude execution

**Test file:** `test/project_context_test.go`

### Phase 2: Prompt Enhancement (0.5 day)

**2.1 Update Prompt Types**
- [ ] Add `ProjectContext` field to `PromptRequest` struct
- [ ] Extract `_project_context` from `projectSpec` in `BuildProjectPrompt()`
- [ ] Add project context instructions to system prompt
- [ ] Update JSON serialization to include project context

**2.2 Testing**
- [ ] Verify project context appears in prompt JSON
- [ ] Verify system prompt includes introspection instructions
- [ ] Test with empty project context (should be omitted)

### Phase 3: Diagram Feature Refactor (1 day)

**3.1 Remove Module Variables**
- [ ] Remove `var.project` from `variables.tofu`
- [ ] Keep only `var.file_path` variable
- [ ] Update feature prompt to use introspection
- [ ] Add explicit constraints to read from `project_context`

**3.2 Update Documentation**
- [ ] Update `README.md` in diagram stack directory
- [ ] Document introspection mechanism
- [ ] Provide examples of project context structure

### Phase 4: Example Projects (1 day)

**4.1 Update Existing Examples**
- [ ] Refactor `infra-3-tier-app` to use direct feature reference
- [ ] Remove module call, use inline or resource feature
- [ ] Verify diagram generates correctly

**Expected structure for `infra-3-tier-app` after refactor:**

```hcl
# examples/projects/infra-3-tier-app/project.tofu

resource "tofukit_feature" "diagram_auto" {
  name        = "architecture-diagram"
  description = "Auto-generated 3-tier architecture diagram"

  requirements = [{
    instructions = [{
      prompt = "Generate architecture diagram by introspecting project_context"
    }]
  }]

  files = {
    "docs/architecture.drawio" = {
      instructions = [{
        prompt = "Generate draw.io diagram from project_context"
      }]
    }
  }
}

resource "tofukit_project" "three_tier_app" {
  name        = "infra-3-tier-app"
  description = "3-tier infrastructure with auto-generated diagram"
  version     = "1.0.0"
  model       = "haiku"

  # Define infrastructure features
  features = {
    "load_balancer" = {
      prompt = "Configure load balancer infrastructure"
      # ... inline feature definition
    }

    "web_server" = {
      prompt = "Setup web server tier"
      # ... inline feature definition
    }

    "database" = {
      prompt = "Configure database tier"
      # ... inline feature definition
    }

    # Diagram feature - auto-discovers all features above!
    "diagram" = tofukit_feature.diagram_auto
  }
}

# Result: diagram.drawio will show:
# - load_balancer feature (box)
# - web_server feature (box)
# - database feature (box)
# - Arrows showing dependencies
```

**4.2 Create New Example**
- [ ] Create comprehensive example with features + integrations
- [ ] Demonstrate automatic discovery of components
- [ ] Show diagram including all discovered elements

**Example location:** `examples/projects/flask-api-nyc-weather/`

### Phase 5: Documentation (0.5 day)

**5.1 Update CLAUDE.md**
- [ ] Document project context introspection mechanism
- [ ] Add section on context-aware features
- [ ] Explain how diagram generation works
- [ ] Add troubleshooting section

**5.2 Create User Guide**
- [ ] Document best practices for introspectable features
- [ ] Explain when to use introspection vs explicit configuration
- [ ] Provide examples of project context structure

### Phase 6: Testing and Validation (1 day)

**6.1 Integration Tests**
- [ ] E2E test: Project with features + diagram generation
- [ ] Test with multiple features (5+)
- [ ] Test with integrations referenced
- [ ] Test with empty project (no features)

**6.2 Manual Validation**
- [ ] Generate diagram for complex project
- [ ] Verify all components appear
- [ ] Verify relationships shown correctly
- [ ] Verify PNG export works

**Test files:**
- `test/e2e_diagram_introspection_test.go`
- `test/integration_diagram_test.go`

## Flexible Feature Specification

Introspection works with **any method** of specifying features:

### Method 1: Inline Feature Definitions
```hcl
resource "tofukit_project" "app" {
  features = {
    "logging" = {
      prompt = "Add structured logging"
      # ... inline definition
    }
    "diagram" = tofukit_feature.diagram_auto
  }
}
# ✅ Introspection discovers: logging, diagram
```

### Method 2: Feature Resource References
```hcl
resource "tofukit_project" "app" {
  features = {
    "logging" = tofukit_feature.logging
    "auth" = tofukit_feature.auth
    "diagram" = tofukit_feature.diagram_auto
  }
}
# ✅ Introspection discovers: logging, auth, diagram
```

### Method 3: Reference Another Project's Features
```hcl
resource "tofukit_project" "base" {
  features = {
    "logging" = tofukit_feature.logging
    "auth" = tofukit_feature.auth
  }
}

resource "tofukit_project" "extended" {
  # Pass all features from base project
  features = tofukit_project.base.features  # ⬅️ If features is computed attribute

  # OR add diagram to base features
  features = merge(
    tofukit_project.base.features,
    {
      "diagram" = tofukit_feature.diagram_auto
    }
  )
}
# ✅ Introspection discovers: all features from base + diagram
```

### Method 4: List of Feature Resources
```hcl
locals {
  common_features = [
    tofukit_feature.logging,
    tofukit_feature.auth,
    tofukit_feature.monitoring
  ]
}

resource "tofukit_project" "app" {
  features = { for f in local.common_features : f.name => f }
}
# ✅ Introspection discovers: all features in the list
```

**Key Point:** Project context introspection happens **after** feature collection, so it works regardless of how features are specified!

## Benefits

### 1. No Circular Dependencies
- Diagram feature is just another feature in the project
- No Terraform-level dependency on project data
- Data flow happens at runtime, not plan time

### 2. Fully Automatic
- No manual listing of components
- Diagram always reflects actual project state
- Zero maintenance overhead

### 3. Declarative
- Diagram is part of project definition
- Single source of truth for architecture
- Versioned alongside code

### 4. Extensible Pattern
- Any feature can use project context introspection
- Documentation generators, test generators, etc.
- Enables "meta-features" that operate on project structure

### 5. Flexible
- Works with any feature specification method
- Can reference other projects' features
- Can be combined with explicit configuration

## Example: Advanced Diagram with Integrations

```hcl
# Define external integration
resource "tofukit_integration" "stripe" {
  name = "stripe"
  type = "rest_api"
  description = "Stripe Payment API"
}

# Main project
resource "tofukit_project" "ecommerce" {
  features = {
    "payment" = {
      prompt = "Implement payment processing using ${tofukit_integration.stripe.link}"
      # ... feature definition
    }

    "inventory" = {
      prompt = "Manage product inventory"
      # ... feature definition
    }

    "diagram" = tofukit_feature.diagram_auto
  }
}
```

**Generated diagram will show:**
- "payment" feature box
- "inventory" feature box
- "stripe" integration box (external, different color)
- Arrow from "payment" → "stripe" (dependency detected from URI reference)

## Edge Cases and Considerations

### 1. Self-Referencing
**Issue**: Diagram feature appears in its own project context

**Solution**: Diagram feature should filter itself out when visualizing:
```
prompt = "Show all features in project_context EXCEPT the diagram feature itself"
```

### 2. Large Projects
**Issue**: Projects with 50+ features may create cluttered diagrams

**Solution**:
- Support grouping/clustering in diagram layout
- Allow filtering via diagram feature configuration
- Future: hierarchical diagrams with zoom levels

### 3. Missing Context
**Issue**: Empty project context (no features)

**Solution**:
- Provider should always populate basic project info
- Diagram feature should handle empty components gracefully
- Generate minimal diagram with just project name

### 4. Performance
**Issue**: Building context for large projects may be slow

**Solution**:
- Context building is fast (just metadata extraction)
- No additional registry lookups beyond existing code
- Context is small compared to full specification

## Future Enhancements

### 1. Context Filtering
Allow features to request specific context subsets:

```hcl
resource "tofukit_feature" "diagram_services_only" {
  introspect = {
    include = ["features", "integrations"]
    exclude = ["kits", "requirements"]
  }
}
```

### 2. Cross-Project Context
Allow diagrams to visualize multiple projects:

```hcl
resource "tofukit_feature" "system_diagram" {
  introspect_projects = [
    "api-project",
    "web-project",
    "database-project"
  ]
}
```

### 3. Custom Context Enrichment
Allow projects to add custom metadata:

```hcl
resource "tofukit_project" "app" {
  context_metadata = {
    deployment_env = "production"
    region = "us-east-1"
    owner_team = "platform"
  }
}
```

## Success Criteria

- [ ] ✅ Project context is collected and passed to Claude
- [ ] ✅ Diagram feature can introspect project context
- [ ] ✅ No circular dependencies in Terraform graph
- [ ] ✅ Diagram generation works without module variables
- [ ] ✅ Diagram includes all features from project
- [ ] ✅ Diagram includes integrations when referenced
- [ ] ✅ All tests pass (unit + integration + e2e)
- [ ] ✅ Documentation updated with introspection pattern
- [ ] ✅ Example projects demonstrate automatic discovery

## Timeline Estimate

**Total: 5-6 days**

- Phase 1 (Core Mechanism): 1-2 days
- Phase 2 (Prompt Enhancement): 0.5 day
- Phase 3 (Diagram Refactor): 1 day
- Phase 4 (Examples): 1 day
- Phase 5 (Documentation): 0.5 day
- Phase 6 (Testing): 1 day

## Conclusion

Project context introspection solves the circular dependency problem by separating Terraform's dependency graph (plan time) from data flow (runtime). Features can automatically discover their parent project's components without requiring explicit configuration, enabling truly declarative diagram generation and opening the door for other "meta-features" that operate on project structure.

The implementation follows existing patterns (`_resource_registry`) and integrates cleanly into the current architecture without breaking changes.
