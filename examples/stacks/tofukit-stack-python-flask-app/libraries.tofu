# Python Flask Stack Framework
# Framework definition for the Python Flask stack

# ========================================
# FRAMEWORK KITS
# ========================================
resource "tofukit_library" "flask" {
  name        = "flask"
  description = "Lightweight WSGI web application framework"
  version     = var.flask_version

  # Note: depends_on removed - dependency is implicit through .link reference in instructions (line 20)

  requirements = [
    {
      name = "Flask Framework Installation"

      instructions = [
        {
          prompt      = "Install Flask framework version ${var.flask_version} and Flask-CORS ${var.flask_cors_version} using ${tofukit_tool.uv.link}: uv add flask==${var.flask_version} flask-cors==${var.flask_cors_version}"
          constraints = [
            "Do NOT install any version other than ${var.flask_version} for Flask",
            "Must use ${tofukit_tool.uv.link} for installation",
            "Install from PyPI only",
            "Include Flask-CORS for CORS support"
          ]
        }
      ]

      verifications = [
        {
          command = "uv pip show flask | grep Version"
          expect  = "${var.flask_version}"
        },
        {
          command = "uv pip show flask-cors | grep Version"
          expect  = "${var.flask_cors_version}"
        }
      ]
    },
    {
      name = "Idiomatic Flask Usage"

      instructions = [
        {
          prompt = <<-EOF
          Use Flask idiomatically following best practices:

          APPLICATION STRUCTURE:
          - Create Flask app with app = Flask(__name__)
          - Use @app.route() decorators for endpoints
          - Use jsonify() for JSON responses
          - Implement proper error handling

          ENTRY POINT (CRITICAL):
          - app.py MUST be the sole entry point - include if __name__ == '__main__': app.run() block
          - DO NOT create main.py or any other entry point file
          - Run directly with: python app.py or flask run
          - This is the conventional Flask pattern for single-file applications

          CONFIGURATION:
          - Load config from environment variables
          - Use python-dotenv for local development
          - Set FLASK_ENV and PORT

          CORS:
          - Initialize CORS with CORS(app) for cross-origin support
          - Configure allowed origins if needed

          LOGGING:
          - Use app.logger for logging
          - Configure logging level
          - Log requests and errors

          PRODUCTION READY:
          - Include health check endpoint
          - Proper error responses with status codes
          - Structured JSON responses
          EOF

          constraints = [
            "NO print() statements - use app.logger",
            "Return JSON responses with jsonify()",
            "Include proper HTTP status codes",
            "Handle exceptions gracefully",
            "Use environment variables for configuration",
            "app.py is the ONLY entry point - DO NOT create main.py",
            "Include if __name__ == '__main__': app.run() in app.py"
          ]
        }
      ]

      verifications = []
    }
  ]
}
