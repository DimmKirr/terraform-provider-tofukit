# Go CLI Stack Frameworks
# Framework definitions for the Go CLI stack (Cobra, Viper, pterm)

# ========================================
# FRAMEWORK KITS
# ========================================
resource "tofukit_framework" "cobra" {
  name        = "cobra"
  description = "CLI framework for creating powerful command-line applications"
  version     = var.cobra_version

  depends_on = [tofukit_tool.go_mod]

  requirements = [
    {
      name = "Cobra Installation"

      instructions = [
        {
          prompt      = "Install Cobra framework version ${var.cobra_version}: go get github.com/spf13/cobra@v${var.cobra_version}"
          constraints = [
            "Do NOT install any version other than ${var.cobra_version}",
            "Use go get, not go install",
            "Run go mod tidy after installation"
          ]
        }
      ]

      verifications = [
        {
          command = "go list -m github.com/spf13/cobra | grep ${var.cobra_version}"
          expect  = "v${var.cobra_version}"
        }
      ]
    },
    {
      name = "Idiomatic Cobra Usage"

      instructions = [
        {
          prompt = "Use Cobra in idiomatic way: standard command patterns, PersistentFlags for global flags, RunE for error handling, cobra.Command struct initialization"
          constraints = [
            "NO custom flag parsing - use cobra.Command flags",
            "NO ignoring RunE errors",
            "Use PreRun/PostRun hooks appropriately"
          ]
        }
      ]

      verifications = []
    }
  ]

  # Cobra framework provides the root command structure
  # Additional commands are added by features and stack-specific implementations
  files = {
    "cmd/root.go" = {
      instructions = [
        {
          prompt = <<-EOF
          Create cmd/root.go as the Cobra CLI root command with extensible command structure:

          PACKAGE & IMPORTS:
          - Package: cmd
          - Imports: fmt, os, ${var.project_module_path}/internal/logger, github.com/spf13/cobra
          - NOTE: internal/logger is provided by pterm_logger feature
          - NOTE: Additional imports for version and other commands will be added by features

          ROOT COMMAND (var rootCmd):
          - Use: "${var.project_name}"
          - Short: "A CLI application built with Cobra, Viper, and pterm"
          - Long: Multi-line description mentioning this is an extensible CLI framework

          EXECUTE() FUNCTION:
          - Call rootCmd.Execute() and check for error
          - If error, log with logger.Error and os.Exit(1)

          INIT() FUNCTION:
          - Define PersistentFlags: "config" (empty default), "log-level" (default "info"), "quiet" (bool, default false)
          - Add basic subcommands: rootCmd.AddCommand(helloCmd)
          - NOTE: Additional commands (version, etc.) are added by features via rootCmd.AddCommand()

          HELLO COMMAND (var helloCmd):
          - Use: "hello [name]"
          - Short: "Say hello to someone"
          - Long: "Greet someone by name, or the world if no name is provided"
          - Args: cobra.MaximumNArgs(1)
          - Run: Extract name from args (default "World"), log with logger.Info: "Hello, <name>!"

          IMPORTANT: This file provides the root command structure that other commands extend.
          Additional subcommands (such as version) are defined in separate files by features.
          Stack-specific commands can also be added by the stack.
          EOF

          constraints = [
            "NO direct cobra.Command initialization - use struct literals",
            "NO ignoring errors from rootCmd.Execute()",
            "Use internal/logger package (provided by pterm_logger feature)",
            "PersistentFlags for flags that apply to all subcommands",
            "Keep base commands minimal - features and stack add more commands"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q '^package cmd' cmd/root.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'var rootCmd.*cobra\\.Command' cmd/root.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func Execute()' cmd/root.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func init()' cmd/root.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'var helloCmd' cmd/root.go && echo 'found'"
          expect  = "found"
        }
      ]
    }
  }
}

resource "tofukit_framework" "viper" {
  name        = "viper"
  description = "Configuration management library for Go applications"
  version     = var.viper_version

  depends_on = [tofukit_tool.go_mod]

  requirements = [
    {
      name = "Viper Installation"

      instructions = [
        {
          prompt      = "Install Viper configuration library version ${var.viper_version}: go get github.com/spf13/viper@v${var.viper_version}"
          constraints = [
            "Do NOT install any version other than ${var.viper_version}",
            "Use go get, not go install",
            "Run go mod tidy after installation"
          ]
        }
      ]

      verifications = [
        {
          command = "go list -m github.com/spf13/viper | grep ${var.viper_version}"
          expect  = "v${var.viper_version}"
        }
      ]
    },
    {
      name = "Idiomatic Viper Usage"

      instructions = [
        {
          prompt = "Use Viper idiomatically: SetEnvPrefix, AutomaticEnv, config file search paths, SetDefault for defaults, proper unmarshaling into structs"
          constraints = [
            "NO direct environment variable parsing - use viper",
            "NO hard-coded config paths",
            "Config files should be optional, not required"
          ]
        }
      ]

      verifications = []
    }
  ]
}

resource "tofukit_framework" "pterm" {
  name        = "pterm"
  description = "Modern terminal UI library for beautiful CLI output"
  version     = var.pterm_version

  depends_on = [tofukit_tool.go_mod]

  requirements = [
    {
      name = "pterm Installation"

      instructions = [
        {
          prompt      = "Install pterm UI library version ${var.pterm_version}: go get github.com/pterm/pterm@v${var.pterm_version}"
          constraints = [
            "Do NOT install any version other than ${var.pterm_version}",
            "Use go get, not go install",
            "Run go mod tidy after installation"
          ]
        }
      ]

      verifications = [
        {
          command = "go list -m github.com/pterm/pterm | grep ${var.pterm_version}"
          expect  = "v${var.pterm_version}"
        }
      ]
    },
    {
      name = "Idiomatic pterm Usage"

      instructions = [
        {
          prompt = "Use pterm idiomatically: DefaultBox, DefaultSpinner, proper color styling, respect --plain flag for non-TTY output"
          constraints = [
            "NO hardcoded ANSI codes - use pterm",
            "Support plain text mode for scripts/pipes",
            "Don't force colors when output is not a TTY"
          ]
        }
      ]

      verifications = []
    }
  ]
}
