# ========================================
# FEATURE: Weather API Endpoints
# ========================================
# Product feature providing weather API endpoints

locals {
  # NYC-specific coordinates
  nyc_latitude  = "40.7834"
  nyc_longitude = "-73.9663"
  nyc_timezone  = "America/New_York"
  temp_unit     = "fahrenheit"
}

resource "tofukit_feature" "api_endpoint_weather" {
  name        = "api-endpoint-weather"
  description = "Weather API endpoints for NYC weather data"

  # Link to implementation file
  files = {
    "app.py" = tofukit_file.app_py
  }

  # Note: depends_on removed - dependency is implicit through .link references in instructions
  # (line 43, 49, 70 use tofukit_feature.integration_openmeteo.link)

  requirements = [
    # Requirement 1: Weather Endpoint
    {
      name = "/weather"
      instructions = [
        {
          prompt = <<-EOF
          Implement a weather API endpoint that provides current NYC weather data.

          BUSINESS REQUIREMENTS:
          - Endpoint: GET /weather
          - Returns: Current NYC weather (temperature, humidity, weather code)
          - Location: New York City (${local.nyc_latitude}°N, ${local.nyc_longitude}°W)
          - Temperature Unit: ${local.temp_unit}
          - Response Format: JSON with weather data and location metadata
          - Data Source: Use ${tofukit_feature.integration_openmeteo.link} to fetch weather data

          IMPLEMENTATION LOCATION:
          This endpoint must be implemented in ${tofukit_file.app_py.link}

          IMPLEMENTATION APPROACH:
          - Call the weather fetching function from ${tofukit_feature.integration_openmeteo.link}
          - Pass NYC coordinates: latitude=${local.nyc_latitude}, longitude=${local.nyc_longitude}
          - Use temperature unit: ${local.temp_unit}
          - Return JSON response with location metadata and current weather data

          EXPECTED RESPONSE STRUCTURE:
          {
            "location": {
              "latitude": ${local.nyc_latitude},
              "longitude": ${local.nyc_longitude},
              "timezone": "${local.nyc_timezone}"
            },
            "current": {
              "temperature": <value in ${local.temp_unit}>,
              "humidity": <value in %>,
              "weather_code": <code>
            }
          }
          EOF

          constraints = [
            "Use ${tofukit_feature.integration_openmeteo.link} for weather data fetching",
            "Return JSON response",
            "Include location metadata (NYC coordinates)",
            "Return temperature in ${local.temp_unit}",
            "Handle errors with proper HTTP status codes"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q '/weather' app.py && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "grep -q '${local.nyc_latitude}' app.py && echo 'OK'"
          expect  = "OK"
        }
      ]
    },

  ]

  verifications = [{
    command = "test -f app.py"
  }]
}
