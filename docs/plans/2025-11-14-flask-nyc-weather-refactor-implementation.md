# Flask NYC Weather Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor flask-api-nyc-weather project to use reusable modules (Open-Meteo integration + Flask stack) instead of inline definitions.

**Architecture:** Extract Open-Meteo integration into standalone module following tofukit-tool-opentofu pattern. Replace explicit Flask setup with tofukit-stack-python-flask-app stack module. Project focuses only on NYC weather business logic.

**Tech Stack:** Terraform/OpenTofu, TofuKit provider, Python Flask, Open-Meteo API

---

## Task 1: Create Open-Meteo Integration Module Structure

**Files:**
- Create: `examples/integrations/tofukit-integration-openmeteo/terraform.tofu`
- Create: `examples/integrations/tofukit-integration-openmeteo/integration.tofu`
- Create: `examples/integrations/tofukit-integration-openmeteo/outputs.tofu`

**Step 1: Create terraform.tofu with provider requirements**

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

**Step 2: Create integration.tofu with resource definition**

Copy from `examples/projects/flask-api-nyc-weather/integration.tofu` lines 7-41:

```hcl
resource "tofukit_integration" "open_meteo" {
  name        = "open-meteo"
  type        = "rest_api"
  description = "Open-Meteo Weather API - Free global weather forecast API"
  version     = "v1"

  # Structured documentation references
  docs = {
    main        = "https://open-meteo.com/en/docs"
    api         = "https://open-meteo.com/en/docs#api"
    parameters  = "https://open-meteo.com/en/docs#api_parameters"
    rate_limits = "https://open-meteo.com/en/features#rate-limits"
  }

  # Single production environment (no authentication differentiation)
  environments = {
    production = "https://api.open-meteo.com/v1"
  }

  # Example API usage patterns
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

**Step 3: Create outputs.tofu with module outputs**

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

**Step 4: Verify module structure**

Run: `ls -la examples/integrations/tofukit-integration-openmeteo/`
Expected: Three files (terraform.tofu, integration.tofu, outputs.tofu)

**Step 5: Commit integration module**

```bash
git add examples/integrations/tofukit-integration-openmeteo/
git commit -m "feat: add Open-Meteo integration module

Reusable module for Open-Meteo Weather API integration.
Follows tofukit-tool-opentofu pattern.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Update Flask Project - Add Module References

**Files:**
- Modify: `examples/projects/flask-api-nyc-weather/project.tofu:1-27`

**Step 1: Add module references after locals block**

In `examples/projects/flask-api-nyc-weather/project.tofu`, add after line 26 (after locals):

```hcl
# ========================================
# MODULE DEPENDENCIES
# ========================================
module "openmeteo" {
  source = "../../integrations/tofukit-integration-openmeteo"
}

module "flask_stack" {
  source = "../../stacks/tofukit-stack-python-flask-app"

  # Pass project metadata for health endpoint
  project_name    = local.project_name
  project_version = local.project_version
}
```

**Step 2: Add stack reference to project resource**

In `examples/projects/flask-api-nyc-weather/project.tofu`, add after line 35 (after `model = "haiku"`):

```hcl
  # Use Flask stack for base setup
  stack = module.flask_stack.stack.id
```

**Step 3: Verify syntax**

Run: `cd examples/projects/flask-api-nyc-weather && tofu fmt -check`
Expected: No output (formatting correct)

**Step 4: Commit module references**

```bash
git add examples/projects/flask-api-nyc-weather/project.tofu
git commit -m "feat(flask-weather): add module references

Add Open-Meteo integration and Flask stack modules.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Update Flask Project - Update Feature References

**Files:**
- Modify: `examples/projects/flask-api-nyc-weather/features.tofu:6-84`

**Step 1: Update integration references in feature**

Replace all `tofukit_integration.open_meteo` with `module.openmeteo.integration`:

In `examples/projects/flask-api-nyc-weather/features.tofu`:

Line 27: Change `${tofukit_integration.open_meteo.description}` to `${module.openmeteo.integration.description}`
Line 28: Change `${tofukit_integration.open_meteo.environments["production"]}` to `${module.openmeteo.integration.environments["production"]}`
Line 29: Change `${tofukit_integration.open_meteo.docs["main"]}` to `${module.openmeteo.integration.docs["main"]}`
Line 30: Change `${tofukit_integration.open_meteo.docs["api"]}` to `${module.openmeteo.integration.docs["api"]}`
Line 31: Change `${tofukit_integration.open_meteo.docs["parameters"]}` to `${module.openmeteo.integration.docs["parameters"]}`
Line 39: Change `${tofukit_integration.open_meteo.environments["production"]}/forecast` to `${module.openmeteo.integration.environments["production"]}/forecast`
Line 52: Change `${tofukit_integration.open_meteo.examples["current-weather"].url}` to `${module.openmeteo.integration.examples["current-weather"].url}`
Line 56: Change `"Use ${tofukit_integration.open_meteo.link} as the data source"` to `"Use ${module.openmeteo.integration.link} as the data source"`

**Step 2: Verify references updated**

Run: `grep -n "tofukit_integration.open_meteo" examples/projects/flask-api-nyc-weather/features.tofu`
Expected: No output (all references updated)

**Step 3: Verify new references exist**

Run: `grep -c "module.openmeteo.integration" examples/projects/flask-api-nyc-weather/features.tofu`
Expected: 8 (eight occurrences)

**Step 4: Commit feature updates**

```bash
git add examples/projects/flask-api-nyc-weather/features.tofu
git commit -m "refactor(flask-weather): update feature to use integration module

Replace direct integration references with module references.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Update Flask Project - Simplify Requirements

**Files:**
- Modify: `examples/projects/flask-api-nyc-weather/project.tofu:44-124`

**Step 1: Replace Flask Application Setup requirement**

Delete lines 44-89 ("Flask Application Setup" requirement).

Replace with simplified "NYC Weather Endpoint" requirement:

```hcl
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

          NOTE: The tofukit-stack-python-flask-app provides:
          - Flask application initialization
          - /health endpoint
          - Structured logging configuration
          - Base files (.env.example, .gitignore)
          EOF

          constraints = [
            "Only implement /weather endpoint logic",
            "Health endpoint is provided by stack",
            "Logging is configured by stack",
            "Use app.logger for logging (configured by stack)"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'from flask import Flask' app.py && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "grep -q 'api.open-meteo.com' app.py && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "grep -q '/weather' app.py && echo 'OK'"
          expect  = "OK"
        }
      ]
    },
```

**Step 2: Update Project Documentation requirement**

Replace lines 91-124 with updated README requirement:

```hcl
    {
      name = "Project Documentation"
      instructions = [
        {
          prompt = <<-EOF
          Create README.md documenting the NYC weather API.

          NOTE: The tofukit-stack-python-flask-app provides base Flask setup.
          Document only weather-specific functionality.

          INCLUDE:
          - Project overview (Flask API for NYC weather)
          - Stack: tofukit-stack-python-flask-app (provides Flask base setup, health check, logging)
          - Integration: Open-Meteo API (via tofukit-integration-openmeteo module)
          - API Endpoints:
            - GET /weather - Returns current NYC weather
            - GET /health - Health check (provided by stack)
          - Example responses for both endpoints
          - Setup instructions (Python ${local.python_version}, uv tool)
          EOF
        }
      ]
    }
  ]
```

**Step 3: Verify requirements structure**

Run: `tofu fmt examples/projects/flask-api-nyc-weather/project.tofu`
Expected: File formatted successfully

**Step 4: Commit simplified requirements**

```bash
git add examples/projects/flask-api-nyc-weather/project.tofu
git commit -m "refactor(flask-weather): simplify requirements using stack

Remove Flask boilerplate from requirements.
Stack provides: Flask init, /health, logging, base files.
Project focuses only on /weather endpoint.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Update Flask Project - Simplify Files

**Files:**
- Modify: `examples/projects/flask-api-nyc-weather/project.tofu:126-198`

**Step 1: Simplify app.py instructions**

Replace lines 128-142 with simplified app.py definition:

```hcl
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
```

**Step 2: Remove stack-provided files**

Delete lines 144-166 (requirements.txt, .env.example, .gitignore) - these are provided by the stack.

Keep only README.md:

```hcl
    "README.md" = {
      instructions = [
        {
          prompt = "Create README as specified in requirements"
        }
      ]
    }
  }
```

**Step 3: Update depends_on**

Replace lines 200-203:

```hcl
  depends_on = [
    module.openmeteo,
    module.flask_stack,
    tofukit_feature.nyc_weather
  ]
```

**Step 4: Verify file structure**

Run: `tofu validate examples/projects/flask-api-nyc-weather/`
Expected: Success! The configuration is valid.

**Step 5: Commit simplified files**

```bash
git add examples/projects/flask-api-nyc-weather/project.tofu
git commit -m "refactor(flask-weather): remove stack-provided files

Stack provides: .env.example, .gitignore, requirements.txt.
Project defines only: app.py, README.md.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: Remove Inline Integration Definition

**Files:**
- Delete: `examples/projects/flask-api-nyc-weather/integration.tofu`

**Step 1: Remove inline integration file**

Run: `git rm examples/projects/flask-api-nyc-weather/integration.tofu`
Expected: File removed from git

**Step 2: Verify no references remain**

Run: `grep -r "integration.tofu" examples/projects/flask-api-nyc-weather/`
Expected: No output (no references)

**Step 3: Commit deletion**

```bash
git commit -m "refactor(flask-weather): remove inline integration

Integration moved to reusable module at:
examples/integrations/tofukit-integration-openmeteo/

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 7: Verify Integration Module Works

**Files:**
- Test: `examples/integrations/tofukit-integration-openmeteo/`

**Step 1: Initialize integration module**

Run: `cd examples/integrations/tofukit-integration-openmeteo && tofu init`
Expected: OpenTofu initialized successfully

**Step 2: Validate integration module**

Run: `tofu validate`
Expected: Success! The configuration is valid.

**Step 3: Plan integration module**

Run: `tofu plan`
Expected: Plan shows creation of tofukit_integration.open_meteo

**Step 4: Return to main directory**

Run: `cd ../../..`
Expected: Back in repository root

**Step 5: Commit verification notes**

No code changes - verification successful.

---

## Task 8: Verify Flask Project Works

**Files:**
- Test: `examples/projects/flask-api-nyc-weather/`

**Step 1: Initialize flask project**

Run: `cd examples/projects/flask-api-nyc-weather && tofu init`
Expected: OpenTofu initialized successfully, modules downloaded

**Step 2: Validate flask project**

Run: `tofu validate`
Expected: Success! The configuration is valid.

**Step 3: Plan flask project**

Run: `tofu plan`
Expected: Plan shows:
- Module resources (integration, stack components)
- Project resource with simplified configuration
- No errors about missing references

**Step 4: Check module outputs accessible**

Run: `tofu console` then enter: `module.openmeteo.integration.name`
Expected: "open-meteo"

Run: `module.flask_stack.stack.name`
Expected: "python-flask-app"

Exit console: `exit`

**Step 5: Return to repository root**

Run: `cd ../../..`
Expected: Back in repository root

**Step 6: Commit verification success**

No code changes - verification successful.

---

## Task 9: Update Flask Project README (Optional)

**Files:**
- Modify: `examples/projects/flask-api-nyc-weather/README.md` (if exists)

**Step 1: Check if README exists**

Run: `test -f examples/projects/flask-api-nyc-weather/README.md && echo "EXISTS" || echo "SKIP"`

If "SKIP": Skip to Task 10.
If "EXISTS": Continue.

**Step 2: Add module section to README**

Add section after project overview:

```markdown
## Architecture

This project demonstrates TofuKit's module composition:

- **Integration Module**: `tofukit-integration-openmeteo` - Reusable Open-Meteo API integration
- **Stack Module**: `tofukit-stack-python-flask-app` - Base Flask application setup
- **Project**: NYC-specific weather endpoint implementation

The project focuses only on business logic while leveraging reusable modules for infrastructure.
```

**Step 3: Commit README update**

```bash
git add examples/projects/flask-api-nyc-weather/README.md
git commit -m "docs(flask-weather): document module architecture

Add module composition explanation to README.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 10: Run E2E Test (if exists)

**Files:**
- Test: `test/e2e_project_example_flask_api_weather_app_test.go` (if exists)

**Step 1: Check if E2E test exists**

Run: `test -f test/e2e_project_example_flask_api_weather_app_test.go && echo "EXISTS" || echo "SKIP"`

If "SKIP": Skip to Task 11.
If "EXISTS": Continue.

**Step 2: Run E2E test**

Run: `task install && go test ./test -v -run TestE2EProjectExampleFlaskAPIWeatherAppSuccess -timeout=10m`
Expected: Test passes

**Step 3: Verify generated files**

Check test output directory for:
- app.py (contains /weather endpoint)
- README.md (mentions stack and integration)
- .env.example (from stack)
- .gitignore (from stack)
- pyproject.toml (from stack's uv tool)

**Step 4: Commit test verification**

No code changes - test passed.

---

## Task 11: Final Verification and Cleanup

**Files:**
- All modified files in repository

**Step 1: Run full test suite**

Run: `task test`
Expected: All tests pass

**Step 2: Verify git status clean**

Run: `git status`
Expected: Working tree clean (all changes committed)

**Step 3: Review commit history**

Run: `git log --oneline -10`
Expected: See commits from this refactoring

**Step 4: Push branch (if ready)**

Run: `git push origin refactor/flask-nyc-weather-modules`
Expected: Branch pushed successfully

**Step 5: Create summary**

Document completed:
- ✅ Open-Meteo integration module created
- ✅ Flask project updated to use modules
- ✅ Inline definitions removed
- ✅ All tests passing
- ✅ Code reduced by ~70 lines (Flask boilerplate eliminated)

---

## Success Criteria

- [ ] Integration module exists at `examples/integrations/tofukit-integration-openmeteo/`
- [ ] Integration module validates successfully
- [ ] Flask project references both modules
- [ ] Flask project focuses only on weather logic
- [ ] No inline integration definition remains
- [ ] `tofu validate` passes for both module and project
- [ ] All existing tests still pass
- [ ] Commits follow conventional commit format
- [ ] Working tree is clean

## Notes

- Integration module follows exact pattern of `tofukit-tool-opentofu`
- Stack provides: Flask init, /health endpoint, logging, base files
- Project provides: /weather endpoint, NYC-specific logic, README
- Total code reduction: ~70 lines from project.tofu
- Improved reusability: Integration can be used by other projects
