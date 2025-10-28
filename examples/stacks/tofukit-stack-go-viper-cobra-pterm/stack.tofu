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
    "main.go" = {
      instructions = [
        {
          prompt = <<-EOF
          Create main.go as a MINIMAL application entry point following atun's pattern:

          REQUIREMENTS:
          - Package: "package main"
          - Import: ONLY "${var.project_module_path}/cmd"
          - main() function: Contains ONLY "cmd.Execute()"
          - This is a thin entry point that delegates everything to the cmd package
          EOF

          constraints = [
            "NO other imports besides ${var.project_module_path}/cmd",
            "NO initialization code (no logger, no config)",
            "NO comments",
            "NO additional functions or variables",
            "Keep it minimal - just import and execute"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f main.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q '^package main' main.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'import \"${var.project_module_path}/cmd\"' main.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'cmd\\.Execute()' main.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -c 'import' main.go"
          expect  = "1"
        },
        {
          command = "test $(wc -l < main.go) -lt 15 && echo 'minimal'"
          expect  = "minimal"
        },
        {
          command = "go build -o bin/${var.project_name} . 2>&1 | grep -v '^#' || test -f bin/${var.project_name} && echo 'built'"
          expect  = "built"
        }
      ]
    }

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
