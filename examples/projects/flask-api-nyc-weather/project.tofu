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

  # Features: Integration layer + API endpoints + Documentation + Diagram
  # Order matters: Dependencies flow from bottom to top
  # 1. Integration provides weather data fetching capability
  # 2. API endpoints use the integration to expose HTTP endpoints
  # 3. Documentation describes both the integration and endpoints
  # 4. Diagram auto-discovers and visualizes all features
  features = {
    "openmeteo"            = tofukit_feature.integration_openmeteo
    "weather_api"          = tofukit_feature.api_endpoint_weather
    "documentation"        = module.documentation.feature
    "architecture_diagram" = module.architecture_diagram.feature
  }

  # Note: depends_on removed - all dependencies are implicit through attribute references:
  # - stack = module.flask_stack.stack.id
  # - features reference module outputs and feature resources directly
}
