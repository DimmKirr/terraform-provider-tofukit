# Open-Meteo Weather API Integration
# Free weather forecast API with no authentication required

resource "tofukit_integration" "open_meteo" {
  name        = "open-meteo"
  type        = "rest_api"
  description = "Open-Meteo Weather API - Free weather forecast API for NYC"
  base_url    = "https://api.open-meteo.com/v1"
  docs_url    = "https://open-meteo.com/en/docs"
  version     = "v1"

  metadata = {
    auth_required      = "false"
    rate_limit         = "10000/day"
    data_format        = "json"
    latitude           = "40.7834" # NYC coordinates
    longitude          = "-73.9663"
    timezone           = "America/New_York"
    temperature_unit   = "fahrenheit"
    endpoint_forecast  = "/forecast?latitude=40.7834&longitude=-73.9663&current=temperature_2m,relative_humidity_2m,weather_code&hourly=temperature_2m&temperature_unit=fahrenheit"
  }
}
