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
          - Service: ${module.openmeteo.integration.description}
          - API Base: ${module.openmeteo.integration.environments["production"]}
          - Documentation: ${module.openmeteo.integration.docs["main"]}
          - API Details: ${module.openmeteo.integration.docs["api"]}
          - Parameters: ${module.openmeteo.integration.docs["parameters"]}

          LOCATION: New York City
          - Latitude: ${local.nyc_latitude}
          - Longitude: ${local.nyc_longitude}
          - Timezone: ${local.nyc_timezone}

          API ENDPOINT:
          ${module.openmeteo.integration.environments["production"]}/forecast

          PARAMETERS TO FETCH:
          - Current weather: temperature_2m, relative_humidity_2m, weather_code
          - Temperature unit: ${local.temp_unit}
          - Coordinates: latitude=${local.nyc_latitude}, longitude=${local.nyc_longitude}

          IMPLEMENTATION:
          - Use requests library for HTTP calls
          - Parse JSON response
          - Return structured weather data
          - Handle API errors gracefully

          See ${module.openmeteo.integration.examples["current-weather"].url} for example usage
          EOF

          constraints = [
            "Use ${module.openmeteo.integration.link} as the data source",
            "NO authentication required (free API)",
            "Include proper error handling for network failures",
            "Return temperature in ${local.temp_unit}",
            "Include location metadata in response (NYC coordinates)",
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'api.open-meteo.com' app.py"
        },
        {
          command = "grep -q '${local.nyc_latitude}' app.py"
        }
      ]
    }
  ]

  # Link to implementation files
  files = {
    "app.py"     = tofukit_file.app_py
    "README.md" = tofukit_file.readme_md
  }

  verifications = [{
    command = "test -f app.py"
  }]
}
