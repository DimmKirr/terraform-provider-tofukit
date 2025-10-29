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
    viper_config    = tofukit_feature.viper_config.id
    version_command = tofukit_feature.version_command.id
    pterm_logger    = tofukit_feature.pterm_logger.id
    taskfile        = tofukit_feature.taskfile.id
    gitignore       = tofukit_feature.gitignore.id
    readme          = tofukit_feature.readme.id
  }
}

output "files" {
  description = "File resources that need to be created in the project scope"
  value = {
    viper_config_internal  = tofukit_file.viper_config_internal
    version_internal       = tofukit_file.version_internal
    version_cmd            = tofukit_file.version_cmd
    pterm_logger_internal  = tofukit_file.pterm_logger_internal
    taskfile_yaml          = tofukit_file.taskfile_yaml
    gitignore              = tofukit_file.gitignore
    readme_md              = tofukit_file.readme_md
    cobra_root_cmd         = tofukit_file.cobra_root_cmd
    main_go                = tofukit_file.main_go
  }
}
