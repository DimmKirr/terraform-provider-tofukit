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

  # Use NYC weather feature (contains integration-specific logic)
  features = {
    "nyc_weather" = tofukit_feature.nyc_weather
  }

  # Additional Flask-specific requirements
  requirements = [
    {
      name = "Flask Application Setup"
      instructions = [
        {
          prompt = <<-EOF
          Create a Flask REST API application that provides NYC weather data.

          ARCHITECTURE:
          - Flask web server on port 5000
          - Single endpoint: GET /weather
          - Returns JSON with temperature, humidity, weather code
          - Uses the NYC Weather feature for data fetching

          IMPLEMENTATION:
          - Use requests library for HTTP calls
          - Handle API errors gracefully with proper HTTP status codes
          - Add basic logging for requests and errors
          - Include CORS support for frontend integration
          - NO database persistence - direct API calls only

          The NYC weather capability is provided by the nyc_weather feature.
          Reference its requirements for integration details.
          EOF

          constraints = [
            "Use Flask 3.0+",
            "Return JSON responses",
            "Include proper error handling",
            "Temperature in Fahrenheit",
            "Add /health endpoint for monitoring"
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
          - Integration with Open-Meteo API
          - Simple REST API endpoint

          SETUP:
          - Python ${local.python_version} installation
          - Dependencies: Flask, requests, python-dotenv, flask-cors
          - Environment variables (if any)

          API ENDPOINTS:
          - GET /weather - Returns current NYC weather
          - GET /health - Health check endpoint
          - Example response format

          EXTERNAL INTEGRATIONS:
          - Open-Meteo Weather API (free, no auth required)
          - API documentation link
          - Rate limits

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
          prompt = "Create Flask application with /weather and /health endpoints as specified in requirements"
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
          prompt = "Create requirements.txt with Flask, requests, python-dotenv, flask-cors with specific versions"
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
    tofukit_integration.open_meteo,
    tofukit_feature.nyc_weather
  ]
}
