# ========================================
# DOCUMENTATION MODULES
# ========================================

# Basic README documentation
module "documentation" {
  source = "../../features/tofukit-feature-basic-documentation"

  project_name        = local.project_name
  project_description = "Flask API providing NYC weather data from Open-Meteo"

  features = {
    "Flask Stack"           = "Base Flask application setup: ${module.flask_stack.stack.link}"
    "OpenMeteo Integration" = "Weather data integration: ${tofukit_feature.integration_openmeteo.link}"
    "Weather API"           = "NYC weather endpoints: ${tofukit_feature.api_endpoint_weather.link}"
  }
}

# Architecture diagram
module "architecture_diagram" {
  source = "../../features/tofukit-feature-diagram-drawio"

  file_path = "architecture"
  style     = "generic"
}
