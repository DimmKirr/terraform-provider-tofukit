# Diagram Generation and Integration Resource Design

**Date:** 2025-11-13
**Status:** Design/Brainstorm (Updated with Introspection Approach)
**Author:** Dmitry + Claude

> **⚠️ IMPORTANT UPDATE:** The diagram generation approach has been updated to use **project context introspection** instead of manual component specification. This eliminates circular dependencies and enables fully automatic diagram generation.
>
> **See:** [`2025-11-13-project-context-introspection-for-diagrams.md`](./2025-11-13-project-context-introspection-for-diagrams.md) for the introspection mechanism implementation details.
>
> This document has been updated to reflect the introspection-based design.

## Overview

Add draw.io diagram generation capability through a reusable feature and introduce a new `tofukit_integration` resource type for representing external API/service dependencies. This enables:
- **Automated architecture diagram generation via project introspection** (no manual component specification)
- Declarative representation of external service integrations (APIs, SaaS, etc.)
- Visual documentation that stays synchronized with infrastructure code
- Zero-maintenance diagrams that auto-discover project components

## Motivation

**Current problem:**
- No way to represent external API/service dependencies in TofuKit (Open-Meteo, Stripe, AWS services, etc.)
- Architecture diagrams must be manually created and maintained separately from code
- Documentation drift when architecture changes but diagrams aren't updated

**Goal:**
- Create `tofukit_integration` resource for representing external service dependencies
- Build reusable diagram generation feature that accepts structured component data
- Enable projects to automatically generate draw.io diagrams from their architecture

## Architecture Components

### 1. New Resource: `tofukit_integration`

Registry-based resource (like `tofukit_feature`) that represents external API/service integrations.

**Key characteristics:**
- No physical files created (registry-only resource)
- Stores API metadata (name, type, description, documentation URL, base URL)
- Provides `.link` attribute for URI references (`tofukit://integration/name`)
- Can be referenced by projects for documentation and diagram generation

**Use cases:**
- External APIs (Open-Meteo, Stripe, Google Maps)
- SaaS services (Auth0, SendGrid, Datadog)
- Cloud services (AWS S3, GCP Pub/Sub)
- Third-party integrations (Slack, GitHub, Jira)

### 2. Diagram Generation Feature

Standalone feature that generates draw.io XML diagrams via **project context introspection**.

**Location:** `examples/stacks/tofukit-feature-diagram-drawio/`

**Inputs:**
- `file_path` - Output path for diagram file (default: "architecture.drawio")
- **No manual component specification** - automatically discovers components from parent project via `_project_context`

**Outputs:**
- Draw.io XML file
- PNG export (via drawio CLI verification)

**Dependencies:**
- drawio CLI tool (for PNG export and validation)
- Project context introspection mechanism (see: `docs/plans/2025-11-13-project-context-introspection-for-diagrams.md`)

### 3. Example Project: Flask NYC Weather API

Demonstrates full usage of integration resource and diagram generation.

**Location:** `examples/projects/flask-api-nyc-weather/`

**Architecture:**
- Flask API service (web server)
- Open-Meteo API integration (external weather data)
- Nginx load balancer
- PostgreSQL database (optional for caching)

**Demonstrates:**
- Defining external API integration with `tofukit_integration`
- Passing structured architecture data to diagram feature
- Generating automated architecture documentation

## Detailed Design

### Component 1: `tofukit_integration` Resource

#### Schema Definition

**File:** `internal/resources/integration.go`

```go
type IntegrationResourceModel struct {
    ID          types.String   `tfsdk:"id"`
    Name        types.String   `tfsdk:"name"`
    Link        types.String   `tfsdk:"link"`
    Type        types.String   `tfsdk:"type"`
    Description types.String   `tfsdk:"description"`
    BaseURL     types.String   `tfsdk:"base_url"`
    DocsURL     types.String   `tfsdk:"docs_url"`
    Version     types.String   `tfsdk:"version"`
    Metadata    types.Map      `tfsdk:"metadata"` // Optional key-value pairs
}
```

**Attributes:**
- `id` (computed) - Format: `integration.<name>`
- `name` (required) - Unique identifier for the integration
- `link` (computed) - URI format: `tofukit://integration/<name>`
- `type` (required) - Integration category: `rest_api`, `graphql_api`, `grpc_api`, `webhook`, `websocket`, `saas`, `cloud_service`
- `description` (optional) - Human-readable description
- `base_url` (optional) - API base endpoint URL
- `docs_url` (optional) - Link to API documentation
- `version` (optional) - API version (e.g., "v1", "2.0")
- `metadata` (optional) - Additional key-value metadata (auth type, rate limits, etc.)

#### Terraform Configuration Example

```hcl
resource "tofukit_integration" "open_meteo" {
  name        = "open-meteo"
  type        = "rest_api"
  description = "Open-Meteo Weather API - Free weather forecast API"
  base_url    = "https://api.open-meteo.com/v1"
  docs_url    = "https://open-meteo.com/en/docs"
  version     = "v1"

  metadata = {
    auth_required = "false"
    rate_limit    = "10000/day"
    data_format   = "json"
    latitude      = "40.7834"  # NYC coordinates
    longitude     = "-73.9663"
  }
}
```

#### Resource Behavior

**Create Operation:**
1. Validate required fields (name, type)
2. Compute `id` = `integration.<name>`
3. Compute `link` = `tofukit://integration/<name>`
4. Register in global registry: `registry.SetIntegration(name, model)`
5. Initialize hash fields to null (no files to track)

**Read Operation:**
- No-op (registry-only resource, no external state to refresh)

**Update Operation:**
1. Update registry entry with new metadata
2. Recompute any dependent resources if needed

**Delete Operation:**
1. Remove from global registry: `registry.RemoveIntegration(name)`

**Validation:**
- `name` must be unique across all integrations
- `type` must be one of the allowed enum values
- `base_url` must be valid URL if provided
- `docs_url` must be valid URL if provided

#### Registry Integration

**File:** `internal/registry/registry.go`

Add integration-specific methods to global registry:

```go
// Integration registry methods
func SetIntegration(name string, integration *resources.IntegrationResourceModel) error
func GetIntegration(name string) (*resources.IntegrationResourceModel, bool)
func RemoveIntegration(name string) error
func ListIntegrations() map[string]*resources.IntegrationResourceModel
```

#### URI Support

**File:** `internal/uri/metadata_integration.go`

Add metadata extractor for integration URIs:

```go
func ExtractIntegrationMetadata(uri *ParsedURI) (*ResourceMetadata, error) {
    integration, found := registry.GetIntegration(uri.Name)
    if !found {
        return nil, fmt.Errorf("integration not found: %s", uri.Name)
    }

    return &ResourceMetadata{
        Type:        "integration",
        Name:        integration.Name.ValueString(),
        Subtype:     integration.Type.ValueString(),
        Description: integration.Description.ValueString(),
        Link:        integration.Link.ValueString(),
        Metadata: map[string]interface{}{
            "base_url": integration.BaseURL.ValueString(),
            "docs_url": integration.DocsURL.ValueString(),
            "version":  integration.Version.ValueString(),
        },
    }, nil
}
```

### Component 2: Diagram Generation Feature

#### Directory Structure

```
examples/stacks/tofukit-feature-diagram-drawio/
├── feature.tofu      # Main feature definition
├── variables.tofu    # Input variables (file_path, project)
├── tool.tofu         # drawio CLI tool definition
├── outputs.tofu      # Feature resource export
└── .gitignore        # Ignore generated outputs
```

#### File: `variables.tofu`

```hcl
# Diagram Feature Variables
# Note: No 'project' variable needed - components are auto-discovered via introspection!

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

#### File: `tool.tofu`

```hcl
# Draw.io CLI Tool
# Used for PNG export and diagram validation

resource "tofukit_tool" "drawio" {
  name        = "drawio"
  description = "Draw.io desktop CLI for diagram generation and export"
  version     = "latest"

  requirements = [
    {
      name = "Draw.io CLI Installation"

      instructions = [
        {
          prompt = <<-EOF
          Verify draw.io CLI is installed. If not installed, provide installation instructions:

          Installation options:

          Linux:
          1. Download from https://github.com/jgraph/drawio-desktop/releases
          2. Install package:
             - Debian/Ubuntu: sudo dpkg -i drawio-amd64-<version>.deb
             - RPM-based: sudo rpm -i drawio-<version>.rpm
          3. Verify: drawio --version

          macOS:
          1. Homebrew: brew install --cask drawio
          2. Or download from https://github.com/jgraph/drawio-desktop/releases
          3. Verify: drawio --version

          Windows:
          1. Download installer from https://github.com/jgraph/drawio-desktop/releases
          2. Run drawio-<version>.exe
          3. Add to PATH if using CLI

          Alternative (Cross-platform):
          Use Docker: docker run -it --rm -v $(pwd):/data jgraph/drawio-desktop --help
          EOF

          constraints = [
            "drawio binary must be available in system PATH",
            "CLI must support -x (export), -f (format), and -o (output) flags",
            "Prefer native installation over Docker for better performance"
          ]
        }
      ]

      verifications = [
        {
          command = "drawio --version && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "which drawio && echo 'drawio found in PATH'"
          expect  = "drawio found in PATH"
        }
      ]
    }
  ]
}
```

#### File: `feature.tofu`

```hcl
# Draw.io Diagram Generation Feature
# Auto-generates architecture diagrams via project context introspection

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
             - features: Application capabilities defined in this project
             - integrations: External APIs and services (from tofukit_integration resources)
             - kits: Languages, frameworks, and tools configured
             - requirements: High-level architectural requirements
          3. Analyze relationships between components:
             - Which features reference which integrations (via tofukit:// URIs)
             - Which features depend on which kits
             - Feature interdependencies

          DIAGRAM STRUCTURE:
          1. Create valid draw.io XML with <mxGraphModel> root element
          2. Organize components in logical tiers (top to bottom):
             - External Integrations (cloud shapes with orange/external colors)
             - Application Features (rounded rectangles with blue tones)
             - Infrastructure/Kits (cylinders or rectangles with gray/infrastructure colors)
          3. Add arrows showing dependencies and data flow:
             - Feature → Integration: API calls (solid line)
             - Feature → Kit: Uses framework/tool (dashed line)
             - Feature → Feature: Internal dependencies (dotted line)
          4. Label each component with:
             - Name (bold, primary text)
             - Type or description (smaller, secondary text)
          5. Use professional color scheme:
             - Features: Blue tones (#4A90E2, #357ABD)
             - Integrations: Orange/external (#F5A623, #E67E22)
             - Kits: Gray/infrastructure (#7F8C8D, #95A5A6)
          6. Ensure proper spacing and alignment for readability

          OUTPUT FILE: ${var.file_path}.drawio

          IMPORTANT: Exclude the diagram feature itself from the visualization (don't show self-reference).
          The diagram should comprehensively visualize the entire project architecture
          without requiring manual component specification.
          EOF

          constraints = [
            "MUST read project_context field from specification",
            "MUST include ALL features found in project_context.features (except diagram feature itself)",
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

#### File: `outputs.tofu`

```hcl
output "feature" {
  description = "The draw.io diagram generation feature resource"
  value       = tofukit_feature.diagram_drawio
}
```

#### File: `.gitignore`

```
# Ignore generated diagram outputs
*.drawio
*.png
*.pdf
*.svg

# Allow the feature definition files
!*.tofu
```

### Component 3: Flask NYC Weather API Example Project

#### Directory Structure

```
examples/projects/flask-api-nyc-weather/
├── project.tofu      # Main project configuration
├── integration.tofu  # Open-Meteo integration definition
├── diagram.tofu      # Diagram generation module reference
├── variables.tofu    # Project variables
├── outputs.tofu      # Project outputs
└── .gitignore        # Ignore generated outputs
```

#### File: `integration.tofu`

```hcl
# Open-Meteo Weather API Integration
# Free weather forecast API with no authentication required

resource "tofukit_integration" "open_meteo" {
  name        = "open-meteo"
  type        = "rest_api"
  description = "Open-Meteo Weather API - Free weather forecast API for NYC"
  base_url    = "https://api.open-meteo.com/v1"
  docs_url    = "https://open-meteo.com/en/docs"
  version     = "v1"

  metadata = {
    auth_required   = "false"
    rate_limit      = "10000/day"
    data_format     = "json"
    latitude        = "40.7834"   # NYC coordinates
    longitude       = "-73.9663"
    timezone        = "America/New_York"
    temperature_unit = "fahrenheit"

    # Example endpoint
    endpoint_forecast = "/forecast?latitude=40.7834&longitude=-73.9663&current=temperature_2m,relative_humidity_2m,weather_code&hourly=temperature_2m&temperature_unit=fahrenheit"
  }
}
```

#### File: `diagram.tofu`

```hcl
# Architecture Diagram Feature Definition
# Auto-generates draw.io diagram via project introspection

resource "tofukit_feature" "architecture_diagram" {
  name        = "architecture-diagram"
  description = "Auto-generated architecture diagram from project context"

  requirements = [
    {
      name = "Generate Architecture Diagram"

      instructions = [
        {
          prompt = <<-EOF
          Generate architecture diagram by introspecting project_context.

          The diagram should show:
          - All features defined in this project
          - All integrations (external APIs like Open-Meteo)
          - All infrastructure components
          - Dependencies and data flow between components

          Use project context introspection to automatically discover all components.
          EOF
        }
      ]
    }
  ]

  files = {
    "docs/architecture.drawio" = {
      instructions = [
        {
          prompt = "Generate draw.io diagram from project_context as specified"
        }
      ]

      verifications = [
        {
          command = "test -f docs/architecture.drawio && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "grep -q '<mxGraphModel' docs/architecture.drawio && echo 'OK'"
          expect  = "OK"
        }
      ]
    }
  }
}
```

#### File: `project.tofu`

```hcl
# ========================================
# TERRAFORM & PROVIDER CONFIGURATION
# ========================================
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

# ========================================
# LOCALS
# ========================================
locals {
  project_name    = "flask-nyc-weather"
  project_version = "1.0.0"
  python_version  = "3.12"
}

# ========================================
# MAIN PROJECT RESOURCE
# ========================================
resource "tofukit_project" "flask_weather" {
  name        = local.project_name
  description = "Flask API providing NYC weather data from Open-Meteo"
  version     = local.project_version

  model = "haiku"

  # Project requirements describe what to build
  requirements = [
    {
      name = "Flask API Setup"
      instructions = [
        {
          prompt = <<-EOF
          Create a Flask REST API application that fetches NYC weather data from Open-Meteo API.

          ARCHITECTURE:
          - Flask web server on port 5000
          - Single endpoint: GET /weather
          - Fetches current weather from: ${tofukit_integration.open_meteo.base_url}/forecast
          - NYC coordinates: latitude=${tofukit_integration.open_meteo.metadata["latitude"]}, longitude=${tofukit_integration.open_meteo.metadata["longitude"]}
          - Returns JSON with temperature, humidity, weather code

          INTEGRATION DETAILS:
          ${tofukit_integration.open_meteo.description}
          API Documentation: ${tofukit_integration.open_meteo.docs_url}
          No authentication required (rate limit: ${tofukit_integration.open_meteo.metadata["rate_limit"]})
          EOF

          constraints = [
            "Use requests library for HTTP calls to Open-Meteo API",
            "Handle API errors gracefully with proper HTTP status codes",
            "Include temperature in Fahrenheit",
            "Add basic logging for requests and errors",
            "NO database persistence in MVP - direct API calls only",
            "Include CORS support for frontend integration"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'from flask import Flask' app.py && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "grep -q '${tofukit_integration.open_meteo.base_url}' app.py && echo 'OK'"
          expect  = "OK"
        }
      ]
    },

    {
      name = "Project Documentation"
      instructions = [
        {
          prompt = <<-EOF
          Create comprehensive README.md with:

          OVERVIEW:
          - Flask API for NYC weather data
          - Integration with ${tofukit_integration.open_meteo.name}
          - Architecture diagram: docs/architecture.drawio.png

          SETUP:
          - Python ${local.python_version} installation
          - Dependencies: Flask, requests, python-dotenv
          - Environment variables (if any)

          API ENDPOINTS:
          - GET /weather - Returns current NYC weather
          - Example response format

          EXTERNAL INTEGRATIONS:
          - ${tofukit_integration.open_meteo.name}: ${tofukit_integration.open_meteo.description}
          - API docs: ${tofukit_integration.open_meteo.docs_url}
          - Rate limit: ${tofukit_integration.open_meteo.metadata["rate_limit"]}

          ARCHITECTURE:
          - Reference docs/architecture.drawio.png for visual diagram
          EOF
        }
      ]
    }
  ]

  # Files to generate
  files = {
    "app.py" = {
      instructions = [
        {
          prompt = "Create Flask application with /weather endpoint as specified in requirements"
        }
      ]

      verifications = [
        {
          command = "python3 -m py_compile app.py && echo 'OK'"
          expect  = "OK"
        }
      ]
    }

    "requirements.txt" = {
      instructions = [
        {
          prompt = "Create requirements.txt with Flask, requests, python-dotenv, flask-cors"
        }
      ]
    }

    "README.md" = {
      instructions = [
        {
          prompt = "Create README as specified in requirements"
        }
      ]
    }

    ".env.example" = {
      instructions = [
        {
          prompt = "Create example environment file with PORT=5000, FLASK_ENV=development"
        }
      ]
    }

    ".gitignore" = {
      content = <<-EOF
      # Python
      __pycache__/
      *.py[cod]
      *$py.class
      *.so
      .Python
      venv/
      ENV/

      # Flask
      instance/
      .env

      # IDE
      .vscode/
      .idea/
      *.swp

      # OS
      .DS_Store
      Thumbs.db

      # TofuKit
      .terraform/
      *.tfstate
      *.tfstate.backup
      EOF
    }
  }

  # Reference diagram generation feature (auto-discovers components via introspection)
  features = {
    "architecture_diagram" = tofukit_feature.architecture_diagram
  }

  depends_on = [
    tofukit_integration.open_meteo
  ]
}
```

#### File: `variables.tofu`

```hcl
# No variables needed for this example
# All configuration is hardcoded for simplicity
```

#### File: `outputs.tofu`

```hcl
output "project_id" {
  description = "TofuKit project identifier"
  value       = tofukit_project.flask_weather.id
}

output "integration_link" {
  description = "URI link to Open-Meteo integration"
  value       = tofukit_integration.open_meteo.link
}

output "diagram_path" {
  description = "Path to generated architecture diagram"
  value       = "docs/architecture.drawio"
}

output "diagram_png" {
  description = "Path to PNG export of diagram"
  value       = "docs/architecture.drawio.png"
}
```

## Implementation Outline

### Phase 1: Integration Resource Foundation (3-4 days)

> **⚠️ IMPLEMENTATION NOTE:** A generic integration resource stub exists at `internal/resources/components.go:148-152` (`NewIntegrationResource()`), but it uses the generic `ComponentResource` schema which is **insufficient** for integrations. The current implementation has:
> - ❌ **Wrong link format**: `tofukit://kit/integration/<name>` (should be `tofukit://integration/<name>`)
> - ❌ **Missing fields**: `type` (enum), `base_url`, `docs_url`, `metadata` (map)
> - ❌ **Unnecessary fields**: `requirements`, `files` (integrations are metadata-only)
> - ❌ **Wrong validation**: `version` is required (should be optional)
>
> **Solution:** Create dedicated `IntegrationResource` with proper schema, replacing the generic component approach.

**1.1 Schema Implementation**
- Create `internal/resources/integration.go` with dedicated `IntegrationResource` class
- Define `IntegrationResourceModel` struct with integration-specific fields
- Implement schema with all attributes (type enum, base_url, docs_url, metadata map)
- Add validation logic (type enum values, URL formats)
- Fix link generation to use `tofukit://integration/<name>` (not `tofukit://kit/integration/<name>`)

**Files:**
- `internal/resources/integration.go` (NEW - dedicated resource class)
- Update `internal/provider/provider.go` to replace generic integration with dedicated one
- Remove or deprecate `NewIntegrationResource()` in `components.go` (or keep for backward compatibility)

**1.2 CRUD Operations**
- Implement `Create()` - validate, compute ID/link, register
- Implement `Read()` - no-op for registry resource
- Implement `Update()` - update registry entry
- Implement `Delete()` - remove from registry
- Implement `ModifyPlan()` - compute ID during plan phase

**1.3 Registry Integration**
- Add integration methods to `internal/registry/registry.go`
- Thread-safe access with mutex
- Methods: `SetIntegration`, `GetIntegration`, `RemoveIntegration`, `ListIntegrations`

**1.4 Testing**
- Unit tests for integration resource CRUD
- Registry integration tests
- Validation tests (required fields, URL formats)

**Test files:**
- `test/resource_integration_test.go`

### Phase 2: URI Support for Integrations (1-2 days)

**2.1 URI Metadata Extraction**
- Create `internal/uri/metadata_integration.go`
- Implement `ExtractIntegrationMetadata()` function
- Register integration URI pattern in scanner

**2.2 URI Scanner Updates**
- Update `internal/uri/scanner.go` to recognize `tofukit://integration/*` pattern
- Add integration to URI type enum

**2.3 Testing**
- URI parsing tests for integration links
- Metadata extraction tests
- Registry lookup tests

**Test files:**
- `test/uri_integration_test.go`

### Phase 3: Diagram Feature Implementation (2-3 days)

**3.1 Tool Definition**
- Create `examples/stacks/tofukit-feature-diagram-drawio/tool.tofu`
- Define drawio CLI installation requirements
- Add verifications for tool availability

**3.2 Feature Definition**
- Create `feature.tofu` with diagram generation logic
- Define file output with verifications
- Create variables for file_path and project structure

**3.3 Module Structure**
- Create `variables.tofu` with validation
- Create `outputs.tofu` exporting feature resource
- Add `.gitignore` for generated files

**3.4 Testing**
- Manual test: Generate simple diagram with 2-3 components
- Verify PNG export works
- Test with complex project (10+ components)

### Phase 4: Flask Weather Example (2-3 days)

**4.1 Integration Definition**
- Create `examples/projects/flask-api-nyc-weather/integration.tofu`
- Define Open-Meteo integration with full metadata
- Document API endpoints and parameters

**4.2 Project Configuration**
- Create `project.tofu` with Flask API requirements
- Reference integration resource in instructions
- Add file definitions for app.py, requirements.txt, README

**4.3 Diagram Module Reference**
- Create `diagram.tofu` module call
- Pass structured component data
- Include integration details in architecture

**4.4 Testing**
- Run `tofu plan` and `tofu apply`
- Verify Flask app generation
- Verify diagram generation with integration
- Test PNG export
- Manual verification of generated code

### Phase 5: Documentation (1 day)

**5.1 Provider Documentation**
- Update `CLAUDE.md` with integration resource section
- Document diagram feature usage pattern
- Add examples of passing structured data to features

**5.2 Resource Documentation**
- Create `docs/resources/tofukit_integration.md`
- Schema reference
- Usage examples
- Best practices

**5.3 Example Documentation**
- README in `examples/stacks/tofukit-feature-diagram-drawio/`
- README in `examples/projects/flask-api-nyc-weather/`
- Usage instructions and architecture explanations

### Phase 6: Integration Testing (1-2 days)

**6.1 End-to-End Tests**
- Create integration test for full workflow
- Test integration resource lifecycle
- Test diagram generation with integration reference
- Verify PNG export

**6.2 Edge Cases**
- Empty project structure (no components)
- Large project (100+ components)
- Invalid integration references
- Missing drawio CLI

**Test files:**
- `test/e2e_diagram_generation_test.go`
- `test/integration_workflow_test.go`

## Data Flow Diagram (Introspection-Based)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Terraform Plan Phase (Dependency Graph)                                     │
│                                                                              │
│  ┌──────────────────────┐                                                   │
│  │ integration.tofu     │                                                   │
│  │                      │                                                   │
│  │ tofukit_integration  │──┐                                                │
│  │   .open_meteo        │  │                                                │
│  └──────────────────────┘  │                                                │
│                             │ referenced in                                 │
│  ┌──────────────────────┐  │                                                │
│  │ diagram.tofu         │  │                                                │
│  │                      │  │                                                │
│  │ tofukit_feature      │  │                                                │
│  │   .diagram           │  │                                                │
│  └──────────────────────┘  │                                                │
│            │                │                                                │
│            ├────────────────┘                                                │
│            │                                                                 │
│            ▼                                                                 │
│  ┌──────────────────────┐                                                   │
│  │ project.tofu         │                                                   │
│  │                      │                                                   │
│  │ tofukit_project      │  ✅ No circular dependency!                       │
│  │   features:          │                                                   │
│  │   - diagram          │     Diagram feature is just another feature       │
│  │   - (others)         │                                                   │
│  └──────────────────────┘                                                   │
└─────────────────────────────────────────────────────────────────────────────┘
                            │
                            │ Provider executes project
                            ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Provider Execution Phase (Context Building)                                 │
│                                                                              │
│  1. Collect all features ────────────┐                                      │
│  2. Collect all integrations ────────┤                                      │
│  3. Collect all kits ────────────────┼──▶ buildProjectContext()            │
│  4. Collect all requirements ────────┘                                      │
│                                                                              │
│  outputData["_project_context"] = {                                         │
│    "features": [                                                            │
│      {"name": "diagram", "prompt": "...", "files": [...]},                  │
│      {"name": "api", "prompt": "...", "files": [...]},                      │
│      ...                                                                    │
│    ],                                                                       │
│    "integrations": [                                                        │
│      {"name": "open-meteo", "type": "rest_api", "base_url": "..."}         │
│    ],                                                                       │
│    "kits": [...]                                                            │
│  }                                                                          │
│                                                                              │
│  5. Pass to Claude in structured prompt                                     │
└─────────────────────────────────────────────────────────────────────────────┘
                            │
                            │ Claude receives context
                            ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Claude Execution (Feature Introspection)                                    │
│                                                                              │
│  Diagram feature reads project_context:                                     │
│                                                                              │
│  {                                                                           │
│    "project_context": {                                                     │
│      "features": [                                                          │
│        {"name": "api", "prompt": "Flask API", "files": ["app.py"]},        │
│        ...                                                                  │
│      ],                                                                     │
│      "integrations": [                                                      │
│        {"name": "open-meteo", "type": "rest_api"}                           │
│      ]                                                                      │
│    }                                                                        │
│  }                                                                          │
│                                                                              │
│  ✅ Diagram generates draw.io XML showing:                                  │
│     - All discovered features (boxes)                                       │
│     - All discovered integrations (cloud shapes)                            │
│     - Dependencies and relationships (arrows)                               │
│                                                                              │
│  Output: architecture.drawio + architecture.drawio.png                      │
└─────────────────────────────────────────────────────────────────────────────┘
                            │
                            │ Result
                            ▼
                 ┌──────────────────────┐
                 │ Visual Documentation │
                 │                      │
                 │ ┌──────────────────┐ │
                 │ │  Open-Meteo API  │ │
                 │ │  (integration)   │ │
                 │ └─────────┬────────┘ │
                 │           │          │
                 │ ┌─────────▼────────┐ │
                 │ │  Flask API       │ │
                 │ │  (feature)       │ │
                 │ └──────────────────┘ │
                 │                      │
                 │ Auto-generated from  │
                 │ project context!     │
                 └──────────────────────┘
```

## Success Criteria

1. ✅ `tofukit_integration` resource can be created, updated, deleted
2. ✅ Integration metadata is stored in global registry
3. ✅ Integration `.link` attribute generates valid URI
4. ✅ **Project context introspection mechanism implemented** (see separate design doc)
5. ✅ **Diagram feature introspects project_context automatically** (no manual component specification)
6. ✅ Feature generates valid draw.io XML from introspected data
7. ✅ PNG export verification works with drawio CLI
8. ✅ Flask weather project successfully references integration
9. ✅ Generated README includes integration details
10. ✅ **Architecture diagram includes all auto-discovered components** (features, integrations, kits)
11. ✅ **No circular dependencies in Terraform graph**
12. ✅ All tests pass (unit, integration, e2e)

## Examples and Usage Patterns

### Example 1: Simple Integration Definition

```hcl
resource "tofukit_integration" "stripe" {
  name        = "stripe"
  type        = "rest_api"
  description = "Stripe Payment Processing API"
  base_url    = "https://api.stripe.com/v1"
  docs_url    = "https://stripe.com/docs/api"
  version     = "v1"

  metadata = {
    auth_type     = "bearer_token"
    rate_limit    = "100/second"
    webhook_support = "true"
  }
}
```

### Example 2: Referencing Integration in Project

```hcl
resource "tofukit_project" "ecommerce" {
  requirements = [{
    name = "Payment Processing"
    instructions = [{
      prompt = <<-EOF
      Implement payment processing using ${tofukit_integration.stripe.name}.

      API Documentation: ${tofukit_integration.stripe.docs_url}
      Base URL: ${tofukit_integration.stripe.base_url}
      Authentication: ${tofukit_integration.stripe.metadata["auth_type"]}
      EOF
    }]
  }]
}
```

### Example 3: Auto-Generated Architecture Diagram via Introspection

```hcl
# Define integrations
resource "tofukit_integration" "stripe" {
  name = "stripe"
  type = "rest_api"
  # ...
}

resource "tofukit_integration" "sendgrid" {
  name = "sendgrid"
  type = "saas"
  # ...
}

# Main project with features
resource "tofukit_project" "ecommerce" {
  name = "ecommerce-platform"

  features = {
    "auth" = {
      prompt = "JWT authentication service"
      # ... feature definition
    }

    "payment" = {
      prompt = "Payment processing using ${tofukit_integration.stripe.link}"
      # ... feature definition
    }

    "notification" = {
      prompt = "Email notifications using ${tofukit_integration.sendgrid.link}"
      # ... feature definition
    }

    # Diagram feature - auto-discovers all features and integrations above!
    "architecture_diagram" = tofukit_feature.diagram_auto
  }
}

# Result: Diagram automatically shows:
# - auth feature (box)
# - payment feature (box) → stripe integration (cloud)
# - notification feature (box) → sendgrid integration (cloud)
# - All discovered without manual specification!
```

## Future Enhancements

### Phase 2: Integration Health Checks

Add health check capabilities to integration resources:

```hcl
resource "tofukit_integration" "api" {
  health_check = {
    endpoint = "/health"
    interval = "5m"
    timeout  = "10s"
  }
}
```

### Phase 3: Integration Mocking

Generate mock servers for integrations during development:

```hcl
resource "tofukit_integration" "external_api" {
  mock = {
    enabled      = true
    port         = 8080
    fixtures_dir = "./fixtures"
  }
}
```

### Phase 4: Diagram Customization

Add styling and layout options:

```hcl
module "diagram" {
  source = "..."

  style = {
    theme      = "modern"      # modern, classic, minimal
    layout     = "hierarchical" # hierarchical, organic, circular
    color_scheme = "blue"       # blue, green, monochrome
  }
}
```

### Phase 5: Multiple Diagram Types

Support different diagram formats:

```hcl
module "diagrams" {
  source = "..."

  outputs = {
    drawio = "docs/architecture.drawio"
    mermaid = "docs/architecture.mmd"
    plantuml = "docs/architecture.puml"
  }
}
```

### Phase 6: Auto-Discovery

Automatically discover components from project configuration:

```hcl
resource "tofukit_project" "app" {
  features = {
    "diagram" = {
      auto_discover = true  # Scans project for components
    }
  }
}
```

## Open Questions

1. **Integration versioning:** Should integrations support multiple versions simultaneously?
2. **Diagram layouts:** Should we provide multiple layout algorithms (hierarchical, organic, circular)?
3. **Component icons:** Should we support custom icons for component types?
4. **Diagram updates:** How should we handle diagram updates when architecture changes? (regenerate vs manual edit)
5. **Integration categories:** Do we need a taxonomy of integration types beyond the basic enum?
6. **Validation strictness:** Should drawio CLI be required or optional (graceful degradation)?
7. **Metadata schema:** Should integration metadata have a defined schema or remain flexible?

## Non-Goals (For Initial Implementation)

- **Not implementing:** Integration runtime health checks or monitoring
- **Not implementing:** Automatic API client code generation from integration specs
- **Not implementing:** Integration testing framework or mock server generation
- **Not implementing:** Diagram version control or change tracking
- **Not implementing:** Interactive diagram editor within TofuKit
- **Not implementing:** Multiple diagram output formats (Mermaid, PlantUML) - only draw.io
- **Not implementing:** Terraform provider validation for conflicting integrations
- **Not implementing:** Integration credential management or secrets handling

## Security Considerations

1. **API Credentials:** Integration resources do NOT store credentials or API keys
2. **Documentation Only:** Integrations are metadata resources for documentation purposes
3. **URL Validation:** base_url and docs_url are validated as proper URLs
4. **No Execution:** Integration resources don't make actual API calls (that's the project's job)
5. **Metadata Safety:** Metadata map should not contain sensitive information

## Migration Path

N/A - This is a new feature with no existing resources to migrate.

## Testing Strategy

### Unit Tests
- Integration resource CRUD operations
- URI parsing and metadata extraction
- Registry operations (add, get, remove)
- Validation logic (required fields, URL formats)

### Integration Tests
- End-to-end workflow: create integration → reference in project → generate diagram
- Module passing of structured data
- PNG export with drawio CLI

### Manual Testing
- Create multiple integrations and verify registry state
- Generate diagrams with varying complexity (2-50 components)
- Test with missing drawio CLI (should fail gracefully)
- Verify draw.io desktop can open generated diagrams

## Timeline Estimate

Total: **10-15 days**

- Phase 1 (Integration Resource): 3-4 days
- Phase 2 (URI Support): 1-2 days
- Phase 3 (Diagram Feature): 2-3 days
- Phase 4 (Flask Example): 2-3 days
- Phase 5 (Documentation): 1 day
- Phase 6 (Testing): 1-2 days

## Conclusion

This design introduces two major capabilities:
1. **tofukit_integration** - A new resource type for representing external service dependencies
2. **Diagram generation** - Automated architecture diagram creation from structured component data

The integration resource fills a gap in TofuKit's ability to document external dependencies, while the diagram feature provides a practical demonstration of how structured data can be passed to features (similar to stack references).

The Flask NYC Weather API example showcases both features working together, demonstrating real-world usage patterns that other projects can follow.
