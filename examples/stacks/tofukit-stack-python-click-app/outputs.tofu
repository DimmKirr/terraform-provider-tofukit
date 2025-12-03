# Stack Module Outputs
#
# Primary output: stack (unified interface for projects)
# Optional outputs: kits, features (for named access in prompts/constraints)

# ========================================
# PRIMARY OUTPUT: Unified Stack
# ========================================
output "stack" {
  description = <<-EOT
    The complete Python Click CLI stack resource containing kits, features, and files.

    Usage in projects:
      stack = module.click_app.stack

    Note: kits and features are stored as lists in the stack.
    For named access (e.g., in feature prompts), use the separate
    kits/features outputs below.
  EOT
  value       = tofukit_stack.click_app
}

# ========================================
# OPTIONAL OUTPUTS: Named Access
# ========================================

# Use when you need named access to individual kits
# Example: ${module.click_app.kits.framework.click.link}
output "kits" {
  description = <<-EOT
    Named map of kits for accessing individual kit properties.

    Usage examples:
      - module.click_app.kits.language.link
      - module.click_app.kits.framework.click.link
      - module.click_app.kits.tool.uv.version

    Common use case: Referencing kits in feature prompts/constraints
  EOT
  value = {
    language = tofukit_language.python
    tool = {
      uv = tofukit_tool.uv
    }
    framework = {
      click = tofukit_library.click
    }
  }
}

# Use when you need named access to individual features
# Example: ${module.click_app.features.version_command.id}
output "features" {
  description = <<-EOT
    Named map of feature resources for accessing individual features.

    Usage examples:
      - module.click_app.features.version_command.id
      - module.click_app.features.readme

    Note: Full feature resources, not just IDs
  EOT
  value = {
    version_command = tofukit_feature.version_command
    readme          = tofukit_feature.readme
  }
}
