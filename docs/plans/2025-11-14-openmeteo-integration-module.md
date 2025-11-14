# Flask NYC Weather Project Refactoring

**Date:** 2025-11-14
**Status:** Approved
**Author:** Claude Code

## Overview

Refactor the `flask-api-nyc-weather` project to use reusable modules instead of inline definitions:

1. **Open-Meteo Integration Module**: Extract integration definition into `examples/integrations/tofukit-integration-openmeteo/`
2. **Flask Stack Module**: Replace explicit Flask setup with `tofukit-stack-python-flask-app` stack reference

This transforms the project from a monolithic configuration into a composition of reusable modules.

## Background

### Current State

The `flask-api-nyc-weather` project currently has two issues:

1. **Inline Integration**: The `tofukit_integration.open_meteo` resource is defined inline, creating duplication risk
2. **Explicit Flask Setup**: The project includes basic Flask boilerplate (health endpoint, logging, .gitignore) that duplicates what `tofukit-stack-python-flask-app` already provides

### Desired State

- **Integration Module**: `tofukit_integration.open_meteo` extracted to reusable module following `tofukit-tool-opentofu` pattern
- **Stack Usage**: Project references `tofukit-stack-python-flask-app` for all Flask boilerplate
- **Project Focus**: Project contains only NYC-weather-specific business logic (the `/weather` endpoint)

## Design

### Part 1: Open-Meteo Integration Module

#### Module Structure

```
examples/integrations/tofukit-integration-openmeteo/
├── terraform.tofu      # Provider requirements (tofukit >= 0.1.0)
├── integration.tofu    # tofukit_integration resource definition
└── outputs.tofu        # Expose integration resource and attributes
```

This follows the exact pattern established by `examples/tools/tofukit-tool-opentofu/`.

### Module Files

#### terraform.tofu

Standard provider configuration declaring the tofukit provider requirement:

```hcl
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = ">= 0.1.0"
    }
  }
}
```

**Note:** No `provider` block - consumers provide their own provider configuration.

#### integration.tofu

The main resource definition, moved from the flask project with no content changes:

```hcl
resource "tofukit_integration" "open_meteo" {
  name        = "open-meteo"
  type        = "rest_api"
  description = "Open-Meteo Weather API - Free global weather forecast API"
  version     = "v1"

  docs = {
    main        = "https://open-meteo.com/en/docs"
    api         = "https://open-meteo.com/en/docs#api"
    parameters  = "https://open-meteo.com/en/docs#api_parameters"
    rate_limits = "https://open-meteo.com/en/features#rate-limits"
  }

  environments = {
    production = "https://api.open-meteo.com/v1"
  }

  examples = {
    "current-weather" = {
      url         = "https://open-meteo.com/en/docs#current_weather"
      description = "Get current weather conditions for any coordinates"
    }
    "forecast" = {
      url         = "https://open-meteo.com/en/docs#forecast"
      description = "Get weather forecast up to 16 days ahead"
    }
    "historical" = {
      url         = "https://open-meteo.com/en/docs#archive"
      description = "Access historical weather data archives"
    }
  }
}
```

#### outputs.tofu

Expose the integration resource and key attributes for consumers:

```hcl
output "integration" {
  description = "The complete Open-Meteo integration resource"
  value       = tofukit_integration.open_meteo
}

output "link" {
  description = "URI link to the Open-Meteo integration (tofukit://integration/open-meteo)"
  value       = tofukit_integration.open_meteo.link
}

output "id" {
  description = "Integration resource ID"
  value       = tofukit_integration.open_meteo.id
}

output "name" {
  description = "Integration name"
  value       = tofukit_integration.open_meteo.name
}
```

The `integration` output provides full access to all resource attributes (`.description`, `.environments`, `.docs`, etc.).

### Consumer Updates (flask-api-nyc-weather)

#### 1. Add Module Reference (project.tofu)

Add module source at the top, before resources:

```hcl
module "openmeteo" {
  source = "../../integrations/tofukit-integration-openmeteo"
}
```

#### 2. Update Feature References (features.tofu)

Replace all `tofukit_integration.open_meteo.*` references with `module.openmeteo.integration.*`:

**Before:**
```hcl
- Service: ${tofukit_integration.open_meteo.description}
- API Base: ${tofukit_integration.open_meteo.environments["production"]}
- Documentation: ${tofukit_integration.open_meteo.docs["main"]}
```

**After:**
```hcl
- Service: ${module.openmeteo.integration.description}
- API Base: ${module.openmeteo.integration.environments["production"]}
- Documentation: ${module.openmeteo.integration.docs["main"]}
```

Also update the constraint:
```hcl
"Use ${module.openmeteo.integration.link} as the data source",
```

#### 3. Update Dependency (project.tofu)

**Before:**
```hcl
depends_on = [
  tofukit_integration.open_meteo,
  tofukit_feature.nyc_weather
]
```

**After:**
```hcl
depends_on = [
  module.openmeteo,
  tofukit_feature.nyc_weather
]
```

#### 4. Remove Inline Definition

Delete `examples/projects/flask-api-nyc-weather/integration.tofu` - the module replaces it.

### Part 2: Flask Stack Module Usage

#### What the Stack Provides

The `tofukit-stack-python-flask-app` stack already provides:

- **Kits**: Python language, uv tool, Flask framework
- **Health Check Feature**: `/health` endpoint with status/service/version
- **Base Files Feature**: `.env.example`, `.gitignore`
- **Logging Feature**: Structured logging configuration

#### Project Changes Required

**Remove from project.tofu:**

1. **Flask Application Setup** requirement - The stack handles Flask boilerplate
   - Remove "Flask Application Setup" requirement entirely
   - Remove instructions about Flask server, CORS, logging setup
   - Remove `/health` endpoint instructions (stack provides this)

2. **Files to remove:**
   - `.env.example` - Stack's base_files feature provides this
   - `.gitignore` - Stack's base_files feature provides this
   - `requirements.txt` - Stack uses uv (pyproject.toml)

**Keep in project.tofu:**

1. **app.py** - But simplify instructions:
   - Remove Flask initialization details (stack handles it)
   - Focus only on `/weather` endpoint implementation
   - Reference NYC weather feature for data fetching logic

2. **README.md** - Update to reflect stack usage
   - Mention the stack provides base Flask setup
   - Document only weather-specific functionality

**Add to project.tofu:**

Reference the stack module:

```hcl
module "flask_stack" {
  source = "../../stacks/tofukit-stack-python-flask-app"

  # Pass project metadata for health endpoint
  project_name    = local.project_name
  project_version = local.project_version
}

resource "tofukit_project" "flask_weather" {
  # ... existing config ...

  # Use the stack
  stack = module.flask_stack.stack.id

  # Simplified requirements - only weather-specific logic
  requirements = [
    {
      name = "NYC Weather Endpoint"
      instructions = [
        {
          prompt = <<-EOF
          Add /weather endpoint to the Flask application.

          ENDPOINT: GET /weather
          RETURNS: Current NYC weather data (temperature, humidity, weather code)

          IMPLEMENTATION:
          - Use the NYC Weather feature for data fetching
          - Return JSON response
          - Handle API errors with proper HTTP status codes
          - The Flask app, health endpoint, and logging are already configured by the stack
          EOF

          constraints = [
            "Only implement /weather endpoint logic",
            "Health endpoint is provided by stack",
            "Logging is configured by stack",
            "Use app.logger for logging (configured by stack)"
          ]
        }
      ]
    },
    {
      name = "Project Documentation"
      instructions = [
        {
          prompt = <<-EOF
          Create README.md documenting the NYC weather API.

          NOTE: The Flask stack provides base setup (health check, logging, .gitignore).
          Document only weather-specific functionality.

          INCLUDE:
          - Project overview (Flask API for NYC weather)
          - Stack: tofukit-stack-python-flask-app (provides base Flask setup)
          - Integration: Open-Meteo API
          - API Endpoints: /weather (returns NYC weather), /health (provided by stack)
          - Example responses
          EOF
        }
      ]
    }
  ]

  files = {
    "app.py" = {
      instructions = [
        {
          prompt = "Implement /weather endpoint as specified. Flask app setup is handled by the stack."
        }
      ]

      verifications = [
        {
          command = "python3 -m py_compile app.py && echo 'OK'"
          expect  = "OK"
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
  }
}
```

## Implementation Steps

### 1. Create Open-Meteo Integration Module

- Create directory: `examples/integrations/tofukit-integration-openmeteo/`
- Create `terraform.tofu`, `integration.tofu`, `outputs.tofu`
- Move integration definition from project to module

### 2. Update Flask Project - Integration Module

- Add `module "openmeteo"` reference in `project.tofu`
- Update all `tofukit_integration.open_meteo.*` references in `features.tofu` to `module.openmeteo.integration.*`
- Update `depends_on` from `tofukit_integration.open_meteo` to `module.openmeteo`
- Delete `integration.tofu` file

### 3. Update Flask Project - Stack Module

- Add `module "flask_stack"` reference in `project.tofu`
- Add `stack = module.flask_stack.stack.id` to project resource
- Remove "Flask Application Setup" requirement
- Simplify "NYC Weather Endpoint" requirement (remove Flask boilerplate instructions)
- Update README requirement to mention stack
- Remove `.env.example` file definition
- Remove `.gitignore` file definition
- Remove `requirements.txt` file definition
- Simplify `app.py` instructions to focus only on `/weather` endpoint

### 4. Verification

- Run `tofu init` in flask project directory
- Run `tofu plan` - should show changes for stack addition
- Run `tofu apply` to apply stack-based configuration
- Verify generated files include stack-provided files
- Test that `/health` endpoint works (from stack)
- Test that `/weather` endpoint works (from project)

## Benefits

### Integration Module
- **Reusability**: Any TofuKit project can use `module "openmeteo"`
- **Consistency**: Follows established tofukit module pattern (matches `tofukit-tool-opentofu`)
- **Maintainability**: Single source of truth for Open-Meteo integration metadata
- **Discoverability**: Integration modules organized in dedicated directory

### Stack Usage
- **Separation of Concerns**: Project focuses on business logic, stack handles infrastructure
- **Code Reduction**: ~70 lines removed from project (Flask boilerplate eliminated)
- **Consistency**: All Flask projects can use same base stack
- **DRY Principle**: Health checks, logging, base files defined once in stack
- **Easier Testing**: Project tests focus on weather logic, not Flask setup

## Future Enhancements

- Add README.md to integration module with usage examples
- Create additional integration modules for other common APIs (OpenWeather, WeatherAPI, etc.)
- Extract NYC coordinates to variables for location-agnostic weather integration
- Add integration tests for the complete stack-based project
