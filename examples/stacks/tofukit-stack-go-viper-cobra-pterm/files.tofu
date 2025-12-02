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
      Create Viper configuration abstraction that provides config loading from files and environment variables.

      REQUIRED API (for cobra integration):
      - LoadConfig(configFile string) error - Load config from optional file path or auto-search
        - configFile parameter: if non-empty, use that specific file; if empty, search defaults
        - Returns: error (missing file is NOT an error, only read failures)

      REQUIRED BEHAVIOR:
      - Config struct must include at minimum: LogLevel, LogPlainText fields
      - Environment variable support with project-name-based prefix
      - Search order when configFile empty: ./{project}.yaml, ~/{project}.yaml
      - After loading config, call internal/logger.Initialize() with loaded settings
      - Abstract viper completely - no direct viper exposure

      IMPLEMENTATION HINTS:
      - Use viper.SetConfigFile() if configFile provided, otherwise AddConfigPath()
      - Use viper.AutomaticEnv() for environment variable support
      - Handle viper.ConfigFileNotFoundError gracefully (not an error)
      EOF

      constraints = [
        "LoadConfig signature MUST match: func LoadConfig(configFile string) error",
        "ONLY implement LoadConfig function - do NOT add SaveConfig, GetConfig, or any other functions",
        "Missing config files are NOT errors - use defaults",
        "Never expose viper types in public API",
        "Use os.ExpandEnv for home directory paths - do NOT import path/filepath"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q 'func LoadConfig(.*string).*error' internal/config/config.go"
    },
    {
      command = "grep -q 'SetEnvPrefix\\|AutomaticEnv' internal/config/config.go"
    },
    {
      command = "grep -q 'logger\\.Initialize' internal/config/config.go"
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
      Create structured logging abstraction using Go's log/slog with pterm styling support and quiet mode.

      REQUIRED API (for config integration):
      - Initialize(logLevel string, logPlainText bool) - Setup logger with slog handler
        - logLevel: string like "debug", "info", "warn", "error" - parse to slog.Level
        - logPlainText: if true, disable pterm styling
      - SetQuietMode(bool) - Enable/disable quiet mode
      - Standard log methods: Info, Debug, Warn, Error, Fatal

      REQUIRED BEHAVIOR:
      - Quiet mode: suppress Info/Debug/Warn output, NEVER suppress Error/Fatal
      - Use Go's log/slog for structured logging
      - Support pterm styling integration (can be disabled with logPlainText flag)
      - Fatal() should log and os.Exit(1), never use panic()

      IMPLEMENTATION HINTS:
      - Parse log level: use slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError
      - Use slog.NewTextHandler or slog.NewJSONHandler for the logger handler
      - If logPlainText is true, call pterm.DisableStyling() to disable colors
      - Quiet mode check: skip output for Info/Debug/Warn when enabled
      EOF

      constraints = [
        "Error and Fatal MUST always log (ignore quiet mode)",
        "Fatal must call os.Exit(1), never panic()",
        "All methods must support structured logging (key-value pairs)"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q 'func Initialize.*string.*bool' internal/logger/logger.go"
    },
    {
      command = "grep -q 'func SetQuietMode.*bool' internal/logger/logger.go"
    },
    {
      command = "grep -q 'func Info\\|func Debug\\|func Warn\\|func Error\\|func Fatal' internal/logger/logger.go"
    },
    {
      # Verify slog is used for structured logging
      command = "grep -q 'log/slog' internal/logger/logger.go"
    },
    {
      # Verify pterm package is imported (for styling support)
      command = "grep -q 'github.com/pterm/pterm' internal/logger/logger.go"
    },
    {
      command = "grep -q 'os\\.Exit' internal/logger/logger.go"
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
        "VERSION must be a simple string variable set to '1.0.0', NOT a shell command",
        "Define VERSION as: VERSION: \"1.0.0\" (simple string assignment, not sh: command)",
        "CRITICAL: In vars section, VERSION must be defined BEFORE GO_LDFLAGS (variable expansion happens at definition time)",
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
      - Terraform: Ignore .terraform
      - Tofukit: Ignore .debug
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
      - Imports: fmt, os, ${var.project_module_path}/internal/config, ${var.project_module_path}/internal/logger, github.com/spf13/cobra
      - NOTE: internal/config is provided by viper_config feature
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
      - Add cobra.OnInitialize callback that:
        1. Gets "config" flag value: configFile, _ := rootCmd.PersistentFlags().GetString("config")
        2. Calls config.LoadConfig(configFile) and handles error (log with logger.Error and os.Exit(1))
        3. Gets "quiet" flag value and calls logger.SetQuietMode if true
      - NOTE: Additional commands (version, hello, etc.) are added by features via rootCmd.AddCommand()

      IMPORTANT: This file provides the root command structure that other commands extend.
      Additional subcommands (such as version) are defined in separate files by features.
      Stack-specific commands can also be added by the stack.
      EOF

      constraints = [
        "NO direct cobra.Command initialization - use struct literals",
        "NO ignoring errors from rootCmd.Execute()",
        "Use internal/config package for config loading (provided by viper_config feature)",
        "Use internal/logger package (provided by pterm_logger feature)",
        "MUST use cobra.OnInitialize for config loading - NOT in Execute()",
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
      command = "grep -q 'cobra\\.OnInitialize' cmd/root.go && echo 'found'"
      expect  = "found"
    },
    {
      command = "grep -q 'config\\.LoadConfig' cmd/root.go && echo 'found'"
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
