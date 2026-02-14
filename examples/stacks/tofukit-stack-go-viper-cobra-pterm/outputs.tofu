# Stack Module Outputs
#
# Primary output: stack (unified interface for projects)
# Optional outputs: kits, features, files (for named access in prompts/constraints)

# ========================================
# PRIMARY OUTPUT: Unified Stack
# ========================================
output "stack" {
  description = <<-EOT
    The complete Go CLI stack resource containing kits, features, and files.

    Usage in projects:
      stack = module.go_cli.stack

    Note: kits and features are stored as lists in the stack.
    For named access (e.g., in feature prompts), use the separate
    kits/features outputs below.
  EOT
  value       = tofukit_stack.go_cli_app
}

# ========================================
# OPTIONAL OUTPUTS: Named Access
# ========================================

# Use when you need named access to individual kits
# Example: \${module.go_cli.kits.framework.cobra.link}
output "kits" {
  description = <<-EOT
    Named map of kits for accessing individual kit properties.

    Usage examples:
      - module.go_cli.kits.language.link
      - module.go_cli.kits.framework.cobra.link
      - module.go_cli.kits.framework.viper.version

    Common use case: Referencing kits in feature prompts/constraints
  EOT
  value = {
    language = tofukit_language.go
    tool     = tofukit_tool.go_mod
    framework = {
      cobra = tofukit_library.cobra
      viper = tofukit_library.viper
      pterm = tofukit_library.pterm
    }
  }
}

# Use when you need named access to individual features
# Example: \${module.go_cli.features.viper_config.id}
output "features" {
  description = <<-EOT
    Named map of feature resources for accessing individual features.

    Usage examples:
      - module.go_cli.features.viper_config.id
      - module.go_cli.features.version_command

    Note: Full feature resources, not just IDs
  EOT
  value = {
    viper_config    = tofukit_feature.viper_config
    version_command = tofukit_feature.version_command
    pterm_logger    = tofukit_feature.pterm_logger
    taskfile        = tofukit_feature.taskfile
    gitignore       = tofukit_feature.gitignore
    readme          = tofukit_feature.readme
    # Pattern features (design guidance)
    http_client_pattern       = tofukit_feature.http_client_pattern
    error_handling_pattern    = tofukit_feature.error_handling_pattern
    cobra_argument_pattern    = tofukit_feature.cobra_argument_pattern
    iso8601_timestamp_pattern = tofukit_feature.iso8601_timestamp_pattern
    pterm_output_pattern      = tofukit_feature.pterm_output_pattern
  }
}

# Use when you need named access to individual file resources
# Example: \${module.go_cli.files.main_go}
output "files" {
  description = <<-EOT
    Named map of file resources for accessing individual files.

    Usage examples:
      - module.go_cli.files.main_go
      - module.go_cli.files.cobra_root_cmd

    Note: Full file resource objects
  EOT
  value = {
    viper_config_internal = tofukit_file.viper_config_internal
    version_internal      = tofukit_file.version_internal
    version_cmd           = tofukit_file.version_cmd
    pterm_logger_internal = tofukit_file.pterm_logger_internal
    taskfile_yaml         = tofukit_file.taskfile_yaml
    gitignore             = tofukit_file.gitignore
    readme_md             = tofukit_file.readme_md
    cobra_root_cmd        = tofukit_file.cobra_root_cmd
    main_go               = tofukit_file.main_go
  }
}
