resource "tofukit_feature" "weather_info_viewing" {
  name = "weather_info_viewing"
  description = "User views real-time Miami weather conditions to understand growing environment quality"
  requirements = [
    {
      name = "Weather Information Viewing Journey"
      instructions = [
        {
          prompt = <<-PROMPT
            **User Story:**
            As a user I am willing to view real-time weather information for Miami to understand why the carrots are so fresh. I can tolerate a loading delay while weather data is fetched from the API. When the weather loads I am going to see current temperature, weather icon, humidity, wind speed, feels-like temperature, and atmospheric pressure.
            
            **User Intent:**
            The user wants validation that the "Weather is Always Good in the Burrow" marketing claim is true. They're curious about the actual growing conditions. This builds trust through transparency - showing real data instead of generic marketing claims. Weather information serves as social proof that the environment is ideal for fresh carrots.
            
            **Marketing Pattern (Claim → Evidence → Trust):**
            This feature validates marketing claims with real data:
            1. **Claim** - "Perfect conditions for growing the crunchiest, sweetest carrots all year round!"
            2. **Evidence** - Live weather data from Miami (temperature, humidity, conditions)
            3. **Trust** - User sees actual sunny, warm conditions and believes the freshness claim
            
            Real-time data is more credible than static testimonials. Users trust verifiable information.
            
            **User Actions (Step by Step):**
            1. User scrolls to weather section after reading hero content
            2. User sees section headline: "Weather is Always Good in the Burrow"
            3. User reads marketing copy about Miami's perfect climate
            4. User's attention shifts to weather widget on right side
            5. Weather widget displays loading state (if API is slow)
            6. Weather data loads from Open-Meteo API (Miami coordinates: 25.7743, -80.1937)
            7. User sees large temperature display (e.g., "26°")
            8. User sees weather icon matching conditions (sun, cloud, rain, etc.)
            9. User sees weather description (e.g., "Clear Sky")
            10. User scans detail cards for additional metrics:
                - Humidity percentage (e.g., "54%")
                - Wind speed (e.g., "12 km/h")
                - Feels like temperature (e.g., "26°")
                - Atmospheric pressure (e.g., "1018 hPa")
            11. User validates that conditions match marketing claims
            12. User continues scrolling with increased trust in product quality
            
            **User Inputs:**
            - **None** - This is a passive data consumption feature
            - **Scroll/View** - User navigates to section to view weather
            - **No Interaction Required** - Data loads automatically on page load
            
            **User Expectations:**
            - Weather data appears current (not stale or obviously cached)
            - Temperature matches current Miami conditions (user may verify externally)
            - Weather icon matches description (sun icon for "Clear Sky")
            - Metrics are displayed in familiar units (°C or °F, km/h, %)
            - Loading happens quickly (<3 seconds)
            - If API fails, user sees friendly fallback message (not blank widget)
            - Weather updates periodically (15-minute refresh interval)
            
            **User Success Criteria:**
            - Weather widget displays current Miami weather accurately
            - All 6 metrics are visible and readable (temp, icon, description, humidity, wind, pressure)
            - Weather conditions validate marketing claims (sunny, warm climate)
            - User feels reassured about carrot growing conditions
            - No API errors or broken widget display
            
            **User Failure Scenarios:**
            - API timeout (>10 seconds) → User sees loading state indefinitely, assumes feature is broken
            - Network error → Weather widget shows "Weather unavailable" with warning icon
            - Incorrect coordinates → Shows wrong city's weather, user notices discrepancy
            - Stale cached data → User sees yesterday's weather, loses trust in real-time claim
            - Missing weather icon → User sees blank space, confusing visual hierarchy
            - Units not displayed → User doesn't know if temperature is Celsius or Fahrenheit
            
            **Feature Dependencies (Tofukit Links):**
            - **Fetches from:** $${var.integrations.integration_open_meteo_api.link} for weather forecast data
            - **Uses icons from:** $${var.libraries.font_awesome.link} for weather condition icons
            - **Uses jQuery:** $${var.libraries.jquery.link} for AJAX requests and DOM manipulation
            - **Styled with:** CSS classes defined in assets/css/style.css for weather widget layout
            - **Updates on:** Page load and every 15 minutes (900000ms interval)
          PROMPT
          constraints = [
            "Do not hardcode weather data - user will notice outdated information immediately",
            "Do not show loading state longer than 10 seconds - user assumes feature is broken",
            "Do not hide error states - user needs to know if weather is unavailable",
            "Do not fetch weather data on every scroll - wastes API quota and slows performance",
            "Do not show weather for wrong location - breaks trust in marketing claims",
            "Do not omit units (°C, km/h, %) - creates confusion about measurement system",
            "Do not use weather codes without mapping to descriptions - user doesn't understand numeric codes",
            "Do not block page rendering on weather API - if API is slow, rest of page should still load",
          ]
        },
      ]
    },
  ]
}