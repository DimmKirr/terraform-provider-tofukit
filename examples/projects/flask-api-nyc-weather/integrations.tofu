# ========================================
# INTEGRATION: OpenMeteo Weather API
# ========================================
# Technical integration with Open-Meteo Weather API


# This resource describes how are we fetching data from the integration
resource "tofukit_feature" "integration_openmeteo" {
  name        = "openmeteo-integration"
  description = "Integration with Open-Meteo API for weather data retrieval. "

  # Link to implementation file
  files = {
    "app.py" = tofukit_file.app_py
  }

  requirements = [
    {
      name = "OpenMeteo API integration"
      instructions = [
        {
          prompt = <<-EOF
          Integrate with ${module.openmeteo.integration.link} to fetch weather data.

          INTEGRATION DETAILS:
          - Service: ${module.openmeteo.integration.description}
          - API Base URL: ${module.openmeteo.integration.environments["production"]}
          - Documentation: ${module.openmeteo.integration.docs["main"]}
          - API Reference: ${module.openmeteo.integration.docs["api"]}
          - Parameters Guide: ${module.openmeteo.integration.docs["parameters"]}
          - Example Usage: ${module.openmeteo.integration.examples["current-weather"].url}

          API CALL SPECIFICATION:
          - Endpoint: ${module.openmeteo.integration.environments["production"]}/forecast
          - Method: GET
          - Query Parameters Format:
            * latitude=<lat>
            * longitude=<lon>
            * current=temperature_2m,relative_humidity_2m,weather_code
            * temperature_unit=<unit>

          IMPLEMENTATION LOCATION:
          This integration must be implemented in ${tofukit_file.app_py.link}

          Create a reusable function (e.g., fetch_weather_data) that:
          - Accepts latitude, longitude, and temperature unit as parameters
          - Makes HTTP GET request to ${module.openmeteo.integration.link}
          - Returns parsed weather data (temperature, humidity, weather code)
          - Handles network errors and API failures gracefully
          EOF

          constraints = [
            "Use ${module.openmeteo.integration.link} as the data source",
            "NO authentication required (free API)",
            "Include proper error handling for network failures",
            "Use requests library for HTTP calls",
            "Parse JSON response from ${module.openmeteo.integration.link}",
            "Create reusable function for fetching weather data"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'api.open-meteo.com' app.py && echo 'OK'"
          expect  = "OK"
        }
      ]
    }
  ]

  verifications = [{
    command = "test -f app.py"
  }]
}
