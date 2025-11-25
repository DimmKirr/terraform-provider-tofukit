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

  # Use NYC weather feature (contains integration-specific logic + files)
  features = {
    "nyc_weather" = tofukit_feature.nyc_weather
  }

  depends_on = [
    module.openmeteo,
    module.flask_stack,
    tofukit_feature.nyc_weather
  ]
}
