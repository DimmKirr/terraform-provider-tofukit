# Go CLI Stack Definition
# The main stack that combines language, tools, frameworks, and provides the minimal entry point

# ========================================
# APPLICATION STACK
# ========================================
resource "tofukit_stack" "go_cli_app" {
  name        = "go-cli-app"
  description = "Complete Go CLI application with Cobra, Viper, and pterm. Stack provides main.go entry point. Cobra framework provides cmd/root.go. Features provide additional capabilities (config, version, logger, etc.)"

  # Stack references its core kits
  kits = [
    tofukit_language.go,
    tofukit_tool.go_mod,
    tofukit_framework.cobra,
    tofukit_framework.viper,
    tofukit_framework.pterm,
  ]

  # Stack provides only the minimal entry point (main.go)
  # Cobra framework provides cmd/root.go with extensible command structure
  # Features in features.tofu provide additional capabilities
  files = {
    "main.go" = tofukit_file.main_go

    # Note: cmd/root.go is provided by the Cobra framework (tofukit_framework.cobra)
    # Note: Config, Version, Logger, Taskfile, .gitignore are provided via features
    # See features.tofu for: viper_config, version_command, pterm_logger, taskfile, gitignore features

    # NOTE: go.mod is NOT declared as a file - it's generated automatically by:
    # 1. tofukit_tool.go_mod requirement: "go mod init ${var.project_module_path}"
    # 2. Framework kit requirements: "go get ..." commands that add dependencies
    # 3. "go mod tidy" commands that clean up dependencies
    # Declaring it as a file would conflict with "go mod init"

    # Note: README.md is now provided via the readme feature
    # See features.tofu for the readme feature definition
  }
}
