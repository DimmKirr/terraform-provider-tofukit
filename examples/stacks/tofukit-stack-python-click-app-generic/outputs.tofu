output "stack" {
  description = "The complete Python Click CLI stack resource"
  value       = tofukit_stack.click_app
}

output "kits" {
  description = "Individual kit resources for advanced customization"
  value = {
    python = tofukit_language.python
    pip    = tofukit_tool.pip
    venv   = tofukit_tool.venv
    click  = tofukit_framework.click
  }
}
