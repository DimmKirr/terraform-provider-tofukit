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

# ========================================
# MAIN PROJECT RESOURCE
# ========================================
resource "tofukit_project" "flask_weather" {
  name        = local.project_name
  description = "Flask API providing NYC weather data from Open-Meteo"
  version     = local.project_version

  model = "haiku"

  # Use Flask stack for base setup
  stack = module.flask_stack.stack.id

  # Use NYC weather feature (contains integration-specific logic)
  features = {
    "nyc_weather" = tofukit_feature.nyc_weather
  }

  # Additional Flask-specific requirements
  requirements = [
    {
      name = "NYC Weather Endpoint"
      instructions = [
        {
          prompt = <<-EOF
          Implement the NYC weather endpoint for the Flask application.

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

  # Files to generate
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

  depends_on = [
    module.openmeteo,
    module.flask_stack,
    tofukit_feature.nyc_weather
  ]
}
