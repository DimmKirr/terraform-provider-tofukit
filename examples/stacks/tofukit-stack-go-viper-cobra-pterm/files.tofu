# Centralized File Resources for Go CLI Stack
# Each file is a real tofukit_file resource that gets registered and can be referenced

# ========================================
# VIPER CONFIG FILES
# ========================================
resource "tofukit_file" "viper_config_internal" {
  name        = "internal/config/config.go"
  description = "Viper configuration abstraction layer"

  instructions = [
    {
      prompt = <<-EOF
      Create Viper configuration abstraction with:
      - Config struct: LogLevel (string), LogPlainText (bool), ConfigFile (string)
      - LoadConfig(): Set env prefix to project name, enable AutomaticEnv, search for config in current dir and home dir, unmarshal to struct
      - SaveConfig(): Write config to .{project_name}.yaml in current directory
      - Integrate with internal/logger for re-initialization after config load
      EOF

      constraints = [
        "DO NOT expose viper directly - abstraction layer only",
        "Config file absence should NOT cause errors"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q 'func LoadConfig()' internal/config/config.go"
    },
    {
      command = "grep -q 'func SaveConfig()' internal/config/config.go"
    },
    {
      command = "grep -q 'SetEnvPrefix' internal/config/config.go"
    }
  ]
}

# ========================================
# VERSION FILES
# ========================================
resource "tofukit_file" "version_internal" {
  name        = "internal/version/version.go"
  description = "Version package with build-time ldflags injection"

  instructions = [
    {
      prompt = <<-EOF
      Create version package following atun pattern:
      - Variables: GitCommit (string, empty default), Version (string, set via var)
      - GetVersion(): Try debug.ReadBuildInfo(), return Main.Version if available, else return Version variable
      - FullVersionString(): Return "version (commit)" format, append GitCommit only if non-empty
      - Use fmt.Sprintf for formatting
      Reference: ${file("/Users/dmitry/dev/dimmkirr/terraform-provider-tofukit/.context/atun/internal/version/version.go")}
      EOF

      constraints = [
        "NO external dependencies beyond stdlib",
        "Keep logic simple - just version formatting"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q 'var.*GitCommit.*string' internal/version/version.go"
    },
    {
      command = "grep -q 'func GetVersion()' internal/version/version.go"
    },
    {
      command = "grep -q 'func FullVersionString()' internal/version/version.go"
    },
    {
      command = "grep -q 'debug\\.ReadBuildInfo' internal/version/version.go"
    }
  ]
}

resource "tofukit_file" "version_cmd" {
  name        = "cmd/version.go"
  description = "Cobra version subcommand with pterm output"

  instructions = [
    {
      prompt = <<-EOF
      Create Cobra version subcommand:
      - Cobra command with Use: "version", Short description
      - Run function that displays version using pterm.DefaultBox with title "Version Information"
      - Call version.FullVersionString() for content
      - This command will be registered in cmd/root.go via rootCmd.AddCommand()
      EOF

      constraints = [
        "Must integrate with Cobra framework",
        "Use pterm for visual output"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q 'var versionCmd' cmd/version.go"
    },
    {
      command = "grep -q 'pterm\\.DefaultBox' cmd/version.go"
    },
    {
      command = "grep -q 'version\\.FullVersionString' cmd/version.go"
    }
  ]
}

# ========================================
# LOGGER FILES
# ========================================
resource "tofukit_file" "pterm_logger_internal" {
  name        = "internal/logger/logger.go"
  description = "Structured logging with pterm and slog integration"

  instructions = [
    {
      prompt = <<-EOF
      Create logger package with pterm+slog integration:
      - Package variables: defaultLogger (*slog.Logger), quietMode (bool)
      - Initialize(logLevel string, logPlainText bool): Parse log level, create pterm handler, set styles
      - SetQuietMode(bool): Control Info/Debug/Warn suppression
      - Methods: Info, Debug, Warn, Error, Fatal
      - customizePtermStyle(): Set Unicode symbols (ℹ ⚠ ✔ ⨯ ⚙) with colors
      EOF

      constraints = [
        "Error and Fatal must ALWAYS log regardless of quiet mode",
        "NO panic() - use Fatal() instead"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q 'func Initialize' internal/logger/logger.go"
    },
    {
      command = "grep -q 'func SetQuietMode' internal/logger/logger.go"
    },
    {
      command = "grep -q 'func Info\\|func Debug\\|func Warn\\|func Error\\|func Fatal' internal/logger/logger.go"
    },
    {
      command = "grep -q 'pterm\\.NewSlogHandler' internal/logger/logger.go"
    }
  ]
}

# ========================================
# DEVOPS FILES
# ========================================
resource "tofukit_file" "taskfile_yaml" {
  name        = "Taskfile.yaml"
  description = "Task automation with version injection via ldflags"

  instructions = [
    {
      prompt = <<-EOF
      Create Taskfile v3 with:
      - Variables: GIT_COMMIT_HASH (git rev-parse --short HEAD), GIT_IS_DIRTY (git status check), GO_LDFLAGS with -X flags
      - Tasks: default (depends: build), build (with ldflags), test, clean, install
      - Build command: CGO_ENABLED=0 go build -ldflags with GitCommit and Version injection
      - Target module path: {project_module}/internal/version
      EOF

      constraints = [
        "NO hard-coded versions",
        "Build output MUST go to bin/ directory",
        "Use Go toolchain commands only"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q \"version: '3'\" Taskfile.yaml"
    },
    {
      command = "grep -q 'GO_LDFLAGS' Taskfile.yaml"
    },
    {
      command = "grep -q 'build:' Taskfile.yaml"
    },
    {
      command = "task --list 2>&1 | grep -q 'build' && echo 'task can parse Taskfile'"
      expect  = "task can parse Taskfile"
    }
  ]
}

# ========================================
# GIT FILES
# ========================================
resource "tofukit_file" "gitignore" {
  name        = ".gitignore"
  description = "Standard Go project .gitignore"

  instructions = [
    {
      prompt = <<-EOF
      Create .gitignore with grouped patterns:
      - Binaries: bin/, *.exe, *.dll, *.so, *.dylib, *.test, project binary name
      - Coverage: *.out, coverage.txt
      - Workspace: go.work
      - Dependencies: vendor/
      - IDE: .idea/, .vscode/, vim swap files
      - OS: .DS_Store, Thumbs.db
      - Config: project-specific .yaml/.yml files (may contain secrets)
      EOF

      constraints = [
        "Follow standard Go .gitignore patterns",
        "Include comments for clarity"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q 'bin/' .gitignore"
    }
  ]
}

# ========================================
# DOCUMENTATION FILES
# ========================================
resource "tofukit_file" "readme_md" {
  name        = "README.md"
  description = "Comprehensive project documentation"

  instructions = [
    {
      prompt = <<-EOF
      Create README with sections:
      - Title with project name
      - Overview: Go CLI application using Cobra, Viper, pterm
      - Features: List key capabilities
      - Installation: Go version requirements, build instructions
      - Usage: Command examples with outputs
      - Configuration: Config file format and environment variables
      - Development: Build, test, task commands
      - Contributing: Guidelines for contributors
      EOF

      constraints = [
        "Keep concise but complete",
        "Include project name in title",
        "Show practical command-line examples"
      ]
    }
  ]

  verifications = [
    {
      command = "test -f README.md && echo 'found'"
      expect  = "found"
    },
    {
      command = "grep -q '^# ' README.md"
    }
  ]
}

# ========================================
# FRAMEWORK FILES
# ========================================
resource "tofukit_file" "cobra_root_cmd" {
  name        = "cmd/root.go"
  description = "Cobra CLI root command with extensible structure"

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

# ========================================
# STACK FILES
# ========================================
resource "tofukit_file" "main_go" {
  name        = "main.go"
  description = "Minimal application entry point"

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
