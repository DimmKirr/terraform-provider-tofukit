# ========================================
# FILE RESOURCES - Implementation Details
# ========================================

resource "tofukit_file" "app_py" {
  name = "app.py"

  instructions = [{
    prompt = <<-EOF
    Implement the NYC weather endpoint for the ${module.flask_stack.stack.link} application.

    ENDPOINT: GET /weather
    RETURNS: Current NYC weather data (temperature, humidity, weather code)

    IMPLEMENTATION:
    - Use the NYC Weather feature for data fetching
    - Return JSON response
    - Handle API errors with proper HTTP status codes
    - The Flask app, health endpoint, and logging are already configured by the stack

    NOTE: The tofukit-stack-python-flask-app provides:
    - Flask application initialization
    - /health endpoint
    - Structured logging configuration
    - Base files (.env.example, .gitignore)
    EOF

    constraints = [
      "Only implement /weather endpoint logic",
      "Health endpoint is provided by ${module.flask_stack.stack.link}",
      "Logging is configured by stack",
      "Use app.logger for logging (configured by stack)"
    ]
  }]

  verifications = [{
    command = "grep -q 'from flask import Flask' app.py && echo 'OK'"
    expect  = "OK"
  }, {
    command = "grep -q 'api.open-meteo.com' app.py && echo 'OK'"
    expect  = "OK"
  }, {
    command = "grep -q '/weather' app.py && echo 'OK'"
    expect  = "OK"
  }, {
    command = "python3 -m py_compile app.py && echo 'OK'"
    expect  = "OK"
  }]
}

resource "tofukit_file" "readme_md" {
  name = "README.md"

  instructions = [{
    prompt = <<-EOF
    Create README.md documenting the NYC weather API.

    NOTE: The ${module.flask_stack.stack.link} provides base Flask setup.
    Document only weather-specific functionality.

    INCLUDE:
    - Project overview (Flask API for NYC weather)
    - Stack: ${module.flask_stack.stack.link} (provides Flask base setup, health check, logging)
    - Integration: ${module.openmeteo.integration.link} (via tofukit-integration-openmeteo module)
    - API Endpoints:
      - GET /weather - Returns current NYC weather
      - GET /health - Health check (provided by stack)
    - Example responses for both endpoints
    - Setup instructions (Python ${local.python_version}, uv tool)
    EOF
  }]
}
