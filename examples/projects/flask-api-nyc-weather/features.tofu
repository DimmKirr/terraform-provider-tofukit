# ========================================
# FEATURE: NYC Weather Data
# ========================================
# Product-specific feature for fetching NYC weather using Open-Meteo API

locals {
  # NYC-specific coordinates
  nyc_latitude  = "40.7834"
  nyc_longitude = "-73.9663"
  nyc_timezone  = "America/New_York"
  temp_unit     = "fahrenheit"
}

resource "tofukit_feature" "nyc_weather" {
  name        = "nyc-weather-data"
  description = "NYC weather data fetching capability using Open-Meteo API"

  requirements = [
    {
      name = "NYC Weather Integration"
      instructions = [
        {
          prompt = <<-EOF
          Implement NYC weather data fetching using the Open-Meteo Weather API integration.

          INTEGRATION DETAILS:
          - Service: ${tofukit_integration.open_meteo.description}
          - Base URL: ${tofukit_integration.open_meteo.base_url}
          - Documentation: ${tofukit_integration.open_meteo.docs_url}
          - Authentication: ${tofukit_integration.open_meteo.metadata["auth_required"]}
          - Rate Limit: ${tofukit_integration.open_meteo.metadata["rate_limit"]}

          LOCATION: New York City
          - Latitude: ${local.nyc_latitude}
          - Longitude: ${local.nyc_longitude}
          - Timezone: ${local.nyc_timezone}

          API ENDPOINT:
          ${tofukit_integration.open_meteo.base_url}${tofukit_integration.open_meteo.metadata["endpoint_forecast"]}

          PARAMETERS TO FETCH:
          - Current weather: temperature_2m, relative_humidity_2m, weather_code
          - Temperature unit: ${local.temp_unit}
          - Coordinates: latitude=${local.nyc_latitude}, longitude=${local.nyc_longitude}

          IMPLEMENTATION:
          - Use requests library for HTTP calls
          - Parse JSON response
          - Return structured weather data
          - Handle API errors gracefully
          EOF

          constraints = [
            "Use ${tofukit_integration.open_meteo.link} as the data source",
            "NO authentication required (free API)",
            "Include proper error handling for network failures",
            "Return temperature in ${local.temp_unit}",
            "Include location metadata in response (NYC coordinates)",
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q '${tofukit_integration.open_meteo.base_url}' app.py || grep -q 'api.open-meteo.com' app.py"
        },
        {
          command = "grep -q '${local.nyc_latitude}' app.py"
        }
      ]
    }
  ]

  # Optional: Feature could define files, but in this case project defines them
  # This feature is purely about the capability/requirement

  verifications = [
    {
      command = "test -f app.py"
    }
  ]
}
