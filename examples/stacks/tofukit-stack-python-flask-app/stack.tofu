# Python Flask Stack Definition
# The main stack that combines language, tools, framework, and features

# ========================================
# APPLICATION STACK
# ========================================
resource "tofukit_stack" "flask_app" {
  name        = "python-flask-app"
  description = "Complete Python Flask web application stack with health check, logging, and base files"

  # Stack references its core kits
  kits = [
    tofukit_language.python,
    tofukit_tool.uv,
    tofukit_library.flask,
  ]

  # Stack references its features
  features = [
    tofukit_feature.health_check,
    tofukit_feature.base_files,
    tofukit_feature.logging_config,
  ]

  # Stack provides no files directly - features provide all base files
  files = {}
}
