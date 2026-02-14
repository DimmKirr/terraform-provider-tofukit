# Features for Python Flask Stack
# Each feature represents a distinct capability

# ========================================
# HEALTH CHECK FEATURE
# ========================================
resource "tofukit_feature" "health_check" {
  name        = "health-check"
  description = "Health check endpoint for monitoring and load balancers"

  requirements = [
    {
      name = "Health Check Endpoint"
      instructions = [
        {
          prompt = <<-EOF
          Implement a /health endpoint that returns service health status.

          ENDPOINT: GET /health

          RESPONSE:
          {
            "status": "healthy",
            "service": "${var.project_name}",
            "version": "${var.project_version}"
          }

          IMPLEMENTATION:
          - Always returns 200 OK (unless service is down)
          - JSON response with health status
          - Include service name and version
          EOF

          constraints = [
            "Use @app.route('/health', methods=['GET'])",
            "Return jsonify() response",
            "HTTP 200 status code",
            "Simple implementation - no external dependency checks"
          ]
        }
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q '/health' app.py"
    }
  ]
}

# ========================================
# BASE FILES FEATURE
# ========================================
resource "tofukit_feature" "base_files" {
  name        = "base-files"
  description = "Essential Flask application files (.env.example, .gitignore)"

  requirements = [
    {
      name = "Base Files Setup"
      instructions = [
        {
          prompt = "Ensure essential base files are present: .env.example, .gitignore. Dependencies are managed via uv (pyproject.toml)."
        }
      ]
    }
  ]

  files = {
    ".env.example" = {
      instructions = [
        {
          prompt = <<-EOF
          Create .env.example file with Flask configuration:

          PORT=5000
          FLASK_ENV=development
          FLASK_APP=app.py
          LOG_LEVEL=INFO
          EOF
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
      env/
      .venv/

      # Flask
      instance/
      .env

      # IDE
      .vscode/
      .idea/
      *.swp
      *.swn
      *.swo

      # OS
      .DS_Store
      Thumbs.db
      *.log

      # TofuKit
      .terraform/
      *.tfstate
      *.tfstate.backup

      # uv
      .python-version
      EOF
    }
  }

  verifications = [
    {
      command = "test -f .env.example && test -f .gitignore"
    }
  ]
}

# ========================================
# LOGGING FEATURE
# ========================================
resource "tofukit_feature" "logging_config" {
  name        = "logging-config"
  description = "Structured logging configuration for Flask"

  requirements = [
    {
      name = "Logging Configuration"
      instructions = [
        {
          prompt = <<-EOF
          Configure Flask logging:

          SETUP:
          - Import logging module
          - Configure basicConfig with format: '%(asctime)s - %(levelname)s - %(message)s'
          - Set level from environment variable LOG_LEVEL (default: INFO)
          - Use app.logger for all logging

          USAGE:
          - Log incoming requests: app.logger.info(f'Request to {endpoint}')
          - Log errors: app.logger.error(f'Error: {error}')
          - NO print() statements in production code
          EOF

          constraints = [
            "Use logging module, not print()",
            "Configure at app initialization",
            "Read LOG_LEVEL from environment",
            "Include timestamp in log format"
          ]
        }
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q 'logging' app.py"
    }
  ]
}
