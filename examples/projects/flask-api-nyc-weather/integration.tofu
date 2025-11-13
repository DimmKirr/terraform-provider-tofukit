# ========================================
# INTEGRATION: Open-Meteo Weather API
# ========================================
# Generic Open-Meteo API integration (not location-specific)
# Can be reused for any location worldwide

resource "tofukit_integration" "open_meteo" {
  name        = "open-meteo"
  type        = "rest_api"
  description = "Open-Meteo Weather API - Free global weather forecast API"
  base_url    = "https://api.open-meteo.com/v1"
  docs_url    = "https://open-meteo.com/en/docs"
  version     = "v1"

  metadata = {
    # API characteristics (integration-specific)
    auth_required    = "false"
    rate_limit       = "10000/day"
    data_format      = "json"

    # Available endpoints
    endpoint_forecast = "/forecast"
    endpoint_archive  = "/archive"

    # Supported parameters
    supported_params = "latitude,longitude,current,hourly,daily,temperature_unit,timezone"

    # Available current weather variables
    current_vars = "temperature_2m,relative_humidity_2m,apparent_temperature,precipitation,weather_code,cloud_cover,wind_speed_10m,wind_direction_10m"

    # Temperature units supported
    temp_units = "celsius,fahrenheit"
  }
}
