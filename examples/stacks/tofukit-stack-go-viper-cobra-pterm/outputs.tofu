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

output "features" {
  description = "Feature IDs for registry lookup (passing objects causes Unknown during Create)"
  value = {
    viper_config   = tofukit_feature.viper_config.id
    version_command = tofukit_feature.version_command.id
    pterm_logger   = tofukit_feature.pterm_logger.id
    taskfile       = tofukit_feature.taskfile.id
    gitignore      = tofukit_feature.gitignore.id
    readme         = tofukit_feature.readme.id
  }
}
