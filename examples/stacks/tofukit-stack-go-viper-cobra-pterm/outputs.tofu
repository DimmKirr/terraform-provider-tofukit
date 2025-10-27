output "stack" {
  description = "The complete Go CLI stack resource"
  value       = tofukit_stack.go_cli_app
}

output "kits" {
  description = "List of all kits included in the stack"
  value = [
    tofukit_language.go,
    tofukit_tool.go_mod,
    tofukit_framework.cobra,
    tofukit_framework.viper,
    tofukit_framework.pterm,
  ]
}
