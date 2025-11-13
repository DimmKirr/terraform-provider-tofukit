# ========================================
# INTEGRATION: Open-Meteo Weather API
# ========================================
# Generic Open-Meteo API integration (not location-specific)
# Can be reused for any location worldwide

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
