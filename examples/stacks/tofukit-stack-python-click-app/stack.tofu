# Python Click Stack Definition
# The main stack that combines language, tools, framework
# Features provide the application structure

# ========================================
# APPLICATION STACK
# ========================================
resource "tofukit_stack" "click_app" {
  name        = "python-click-app"
  description = "Complete Python Click CLI application. Stack provides kits and features."

  # Stack references its core kits
  kits = [
    tofukit_language.python,
    tofukit_tool.uv,
    tofukit_framework.click,
  ]

  # Stack references its features
  features = [
    tofukit_feature.version_command,
    tofukit_feature.readme,
  ]

  # Stack provides no files directly - features provide all application files
  files = {}
}


resource "tofukit" "" {}
