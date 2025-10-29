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
    "cmd/root.go" = tofukit_file.cobra_root_cmd
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
