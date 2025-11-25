# ========================================
# FILE RESOURCES - Implementation Details
# ========================================

resource "tofukit_file" "app_py" {
  name = "app.py"

  instructions = [{
    prompt = <<-EOF
    Implement Flask application logic following ${module.flask_stack.stack.link} conventions.

    NOTE: This is a single-file application. All Flask routes, logic, and integrations should be implemented in this file.

    ENTRY POINT: This file (app.py) IS the application entry point. Include:
    if __name__ == '__main__':
        app.run(host='0.0.0.0', port=PORT, debug=os.getenv('FLASK_ENV') == 'development')
    EOF

    constraints = [
      "Follow ${module.flask_stack.stack.link} patterns and conventions",
      "Implement all application logic in this single file",
      "app.py is the ONLY entry point - DO NOT create main.py or any other entry point files",
      "Include if __name__ == '__main__': app.run() block at the end of this file"
    ]
  }]

  verifications = [{
    command = "grep -q 'from flask import Flask' app.py && echo 'OK'"
    expect  = "OK"
  }, {
    command = "python3 -m py_compile app.py && echo 'OK'"
    expect  = "OK"
  }]
}
