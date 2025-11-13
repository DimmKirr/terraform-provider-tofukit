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
          - Simple REST API endpoint

          SETUP:
          - Python ${local.python_version} installation
          - Dependencies: Flask, requests, python-dotenv, flask-cors
          - Environment variables (if any)

          API ENDPOINTS:
          - GET /weather - Returns current NYC weather
          - Example response format

          EXTERNAL INTEGRATIONS:
          - ${tofukit_integration.open_meteo.name}: ${tofukit_integration.open_meteo.description}
          - API docs: ${tofukit_integration.open_meteo.docs_url}
          - Rate limit: ${tofukit_integration.open_meteo.metadata["rate_limit"]}

          USAGE:
          - How to run the application
          - Example curl commands
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

  depends_on = [
    tofukit_integration.open_meteo
  ]
}
