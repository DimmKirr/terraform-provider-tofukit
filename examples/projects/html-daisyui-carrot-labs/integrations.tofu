resource "tofukit_feature" "integration_google_font_api" {
  name = "integration_google_font_api"
  description = "Google Fonts API integration for loading custom web fonts"
  requirements = [
    {
      name = "Google Fonts API Integration"
      instructions = [
        {
          prompt = <<-PROMPT
            **Integration Type:** CDN Service
            **Service Provider:** Google
            **API Version:** v2 (css2)
            **API Endpoint:** https://fonts.googleapis.com/css2
            **Official Documentation:** https://developers.google.com/fonts/docs/getting_started
            
            **Authentication:**
            - Method: No Authentication
            - Required Credentials: None
            - How to Obtain: Public API, no credentials needed
            
            **Required Parameters:**
            - family: Font family names with weights (e.g., {FONT_FAMILY}:wght@{WEIGHTS})
            - display: Font display strategy (recommended: 'swap')
            
            **Optional Parameters:**
            - text: Subset of characters to include (for optimization)
            - subset: Language subset (e.g., latin, latin-ext, cyrillic)
            
            **Dynamic Font Loading:**
            This integration is referenced by `ui_font_*` resources. The actual font link is constructed dynamically by combining all fonts that reference this integration.
            
            URL Pattern (placeholder example):
            ```
            https://fonts.googleapis.com/css2?family={FONT_FAMILY_1}:wght@{WEIGHTS}&family={FONT_FAMILY_2}:wght@{WEIGHTS}&display=swap
            ```
            
            **Response Format:**
            - Content-Type: text/css
            - Structure: CSS @font-face rules with font-file URLs hosted on fonts.gstatic.com
            
            **Implementation Pattern:**
            ```html
            <!-- Preconnect to Google Fonts domains for performance -->
            <link rel="preconnect" href="https://fonts.googleapis.com">
            <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
            
            <!-- Load fonts dynamically based on ui_font_* resources -->
            <link href="{CONSTRUCTED_URL}" rel="stylesheet">
            ```
            
            **CSS Usage Pattern:**
            ```css
            body {
              font-family: '{FONT_NAME}', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            }
            
            h1, h2, h3 {
              font-family: '{DISPLAY_FONT_NAME}', Georgia, serif;
            }
            ```
            
            **Usage Context:**
            Provides web font hosting for `ui_font_*` resources. Each font resource specifies its family, weight, and references this integration. The implementation layer combines all referenced fonts into a single optimized request.
            
            **Rate Limits:**
            No rate limits - public CDN service with no usage quotas
            
            **Error Handling:**
            Fonts should have fallback fonts in CSS (e.g., font-family: '{FONT_NAME}', sans-serif). If Google Fonts CDN is unavailable, browser will gracefully fall back to system fonts.
            
            **Fallback Strategy:**
            Always specify fallback font stack:
            - For display/heading fonts: '{FONT_NAME}', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif
            - For body/UI fonts: '{FONT_NAME}', system-ui, -apple-system, sans-serif
            - Match fallback fonts to web font metrics (x-height, weight) to minimize layout shift
            
            **Performance Optimization:**
            - Use preconnect hints to establish early connections
            - Use font-display: swap to prevent FOIT (Flash of Invisible Text)
            - Only load font weights actually used (defined in ui_font_* resources)
            - Consider using variable fonts for better performance if multiple weights are needed
            - Use &subset parameter to load only required character sets
          PROMPT
          constraints = [
            "Do not block page rendering - use font-display: swap to prevent FOIT (Flash of Invisible Text)",
            "Must include preconnect hints (<link rel=\"preconnect\">) for both fonts.googleapis.com and fonts.gstatic.com to optimize performance",
            "Must define fallback fonts in CSS for graceful degradation when CDN is unavailable",
            "Do not hardcode font URLs - construct dynamically from ui_font_* resources that reference this integration",
            "Never self-host Google Fonts without proper licensing review - use CDN unless there's a specific requirement",
            "Avoid FOUT (Flash of Unstyled Text) by ensuring fallback fonts match x-height and weight of web fonts",
            "Do not load unnecessary font weights - only include weights actually used in the design to minimize payload",
            "Must use css2 API (not css) for better performance and font-display support",
            "Do not use @import in CSS - use <link> tags in HTML for better performance",
            "Must load fonts in <head> before CSS that uses them to prevent render-blocking",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "integration_open_meteo_api" {
  name = "integration_open-meteo-api"
  description = "Open-Meteo API integration for weather forecast data"
  requirements = [
    {
      name = "Open-Meteo API Integration"
      instructions = [
        {
          prompt = <<-PROMPT
            **Integration Type:** REST API
            **Service Provider:** Open-Meteo
            **API Version:** v1
            **API Endpoint:** https://api.open-meteo.com/v1/forecast
            **Official Documentation:** https://open-meteo.com/en/docs
            
            **Authentication:**
            - Method: No Authentication
            - Required Credentials: None
            - How to Obtain: Public API, no credentials needed (free tier available)
            
            **Required Parameters:**
            - latitude: Latitude coordinate (decimal degrees, e.g., 52.52)
            - longitude: Longitude coordinate (decimal degrees, e.g., 13.41)
            - current: Current weather variables (comma-separated, e.g., temperature_2m,relative_humidity_2m,wind_speed_10m)
            - hourly: Hourly forecast variables (comma-separated, e.g., temperature_2m,precipitation)
            - daily: Daily forecast variables (comma-separated, e.g., temperature_2m_max,temperature_2m_min)
            
            **Optional Parameters:**
            - timezone: IANA timezone identifier (e.g., "America/New_York", default: GMT)
            - temperature_unit: celsius (default) | fahrenheit
            - wind_speed_unit: kmh (default) | ms | mph | kn
            - precipitation_unit: mm (default) | inch
            - forecast_days: Number of forecast days (1-16, default: 7)
            
            **Response Format:**
            - Content-Type: application/json
            - Structure: JSON object with metadata (latitude, longitude, timezone, elevation) and weather data arrays
            
            **Example Response:**
            ```json
            {
              "latitude": 52.52,
              "longitude": 13.41,
              "timezone": "GMT",
              "current": {
                "time": "2024-01-15T12:00",
                "temperature_2m": 15.3,
                "relative_humidity_2m": 65,
                "wind_speed_10m": 12.5
              },
              "hourly": {
                "time": ["2024-01-15T00:00", "2024-01-15T01:00", ...],
                "temperature_2m": [12.1, 12.3, ...]
              },
              "daily": {
                "time": ["2024-01-15", "2024-01-16", ...],
                "temperature_2m_max": [18.5, 19.2, ...],
                "temperature_2m_min": [10.1, 11.3, ...]
              }
            }
            ```
            
            **Implementation Pattern:**
            ```javascript
            // Example: Fetch current weather for a location
            async function fetchWeather(latitude, longitude) {
              try {
                const params = new URLSearchParams({
                  latitude: latitude.toString(),
                  longitude: longitude.toString(),
                  current: 'temperature_2m,relative_humidity_2m,wind_speed_10m,precipitation,weather_code',
                  hourly: 'temperature_2m,precipitation_probability',
                  daily: 'temperature_2m_max,temperature_2m_min,precipitation_sum',
                  timezone: 'auto',
                  forecast_days: '7'
                });
                
                const response = await fetch(`https://api.open-meteo.com/v1/forecast?$${params}`);
                
                if (!response.ok) {
                  throw new Error(`HTTP error! status: $${response.status}`);
                }
                
                const data = await response.json();
                return data;
              } catch (error) {
                console.error('Failed to fetch weather data:', error);
                // Return fallback data or show error message to user
                return null;
              }
            }
            
            // Example: Display current temperature
            fetchWeather(52.52, 13.41).then(data => {
              if (data && data.current) {
                console.log(`Current temperature: $${data.current.temperature_2m}°C`);
              }
            });
            ```
            
            **Usage Context:**
            This integration is used to fetch weather forecast data for displaying current weather conditions, hourly forecasts, and daily forecasts. The API provides meteorological data based on geographic coordinates (latitude/longitude).
            
            **Rate Limits:**
            Free tier:
            - 10,000 API calls per day
            - No API key required for free tier
            - Commercial usage requires subscription for higher limits
            
            **Error Handling:**
            - Network timeouts: Implement retry logic with exponential backoff (max 3 retries)
            - HTTP 4xx errors: Validate latitude/longitude parameters are within valid ranges
            - HTTP 5xx errors: Cache last successful response and show stale data with warning
            - Invalid coordinates: Display user-friendly error message
            - No internet connection: Show cached data or "Unable to fetch weather" message
            
            **Fallback Strategy:**
            - Implement client-side caching (localStorage or sessionStorage) to store last successful API response
            - Cache expiration: 30-60 minutes for current weather data
            - If API is unavailable, display cached data with timestamp showing when it was last updated
            - Provide manual refresh button for users to retry failed requests
            - Consider showing a "Weather data may be outdated" banner when using cached data
          PROMPT
          constraints = [
            "Do not make requests without valid latitude/longitude coordinates - validate ranges (lat: -90 to 90, lon: -180 to 180)",
            "Must implement client-side caching to avoid hitting rate limits - cache responses for at least 30 minutes",
            "Must implement exponential backoff for failed requests - do not retry immediately on errors",
            "Do not ignore timezone parameter - always specify timezone for accurate time display (use 'auto' or IANA timezone)",
            "Must handle network failures gracefully - show cached data or user-friendly error messages",
            "Do not request unnecessary variables - only request weather parameters actually displayed to users to minimize payload size",
            "Must respect the 10,000 requests/day free tier limit - implement request throttling for high-traffic applications",
            "Do not hardcode coordinates - allow users to search/select locations or use geolocation API",
            "Must display data units to users (°C/°F, km/h, mm) - the API returns data in specified units",
            "Do not assume API will always be available - implement timeout handling (recommended: 10-15 second timeout)",
          ]
        },
      ]
    },
  ]
}