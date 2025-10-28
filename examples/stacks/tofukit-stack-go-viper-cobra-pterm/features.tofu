# Features for Go CLI Stack
# Each feature represents a distinct capability that can be reused across projects

# ========================================
# CONFIGURATION FEATURE
# ========================================
resource "tofukit_feature" "viper_config" {
  name        = "viper-config"
  description = "Viper-based configuration management with YAML support"

  prompt = "Implement configuration management using Viper with environment variable support and YAML config files"

  constraints = [
    "Configuration file must be optional, not required",
    "Use environment prefix matching project name",
    "Support user home directory and current directory for config search",
  ]

  files = {
    "internal/config/config.go" = {
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
  }

  kits = [tofukit_framework.viper]
}

# ========================================
# VERSION COMMAND FEATURE
# ========================================
resource "tofukit_feature" "version_command" {
  name        = "version-command"
  description = "Version command with Git commit tracking via ldflags"

  prompt = "Implement version command showing project version and Git commit, integrated with Cobra CLI and build-time ldflags injection"

  constraints = [
    "Version and GitCommit variables must be injectable via ldflags",
    "Use runtime/debug.ReadBuildInfo() as fallback",
    "Version command must use pterm for formatted output"
  ]

  files = {
    "internal/version/version.go" = {
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

    "cmd/version.go" = {
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
  }

  # Note: Taskfile integration mentioned - ldflags will be set via:
  # GO_LDFLAGS: "-X {module}/internal/version.GitCommit={{.GIT_COMMIT_HASH}}"

  kits = [tofukit_framework.cobra, tofukit_framework.pterm]

  verifications = []
}

# ========================================
# LOGGER FEATURE
# ========================================
resource "tofukit_feature" "pterm_logger" {
  name        = "pterm-logger"
  description = "Structured logging with pterm and slog integration"

  prompt = "Implement logging abstraction using pterm slog handler with quiet mode support and customizable styles"

  constraints = [
    "Quiet mode must suppress Info/Debug/Warn but NOT Error/Fatal",
    "Use pterm slog handler for structured logging",
    "Provide Unicode symbols for log levels"
  ]

  files = {
    "internal/logger/logger.go" = {
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
  }

  kits = [tofukit_framework.pterm]
}

# ========================================
# DEVOPS FEATURE
# ========================================
resource "tofukit_feature" "taskfile" {
  name        = "taskfile"
  description = "Task automation with Taskfile supporting version injection"

  prompt = "Create Taskfile.yaml for build automation with Git commit injection via ldflags"

  constraints = [
    "Version must be injected via ldflags during build",
    "Build output to bin/ directory",
    "Support CGO_ENABLED=0 for static builds"
  ]

  files = {
    "Taskfile.yaml" = {
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
  }

  kits = [tofukit_tool.gotask]
}

# ========================================
# GIT FEATURE
# ========================================
resource "tofukit_feature" "gitignore" {
  name        = "gitignore"
  description = "Standard Go project .gitignore"

  prompt = "Create .gitignore following Go community standards with IDE and OS file patterns"

  constraints = [
    "Cover binaries, test outputs, coverage files, IDE files, OS files",
    "Include project-specific binary name",
    "Include project-specific config files"
  ]

  files = {
    ".gitignore" = {
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
  }

  kits = []
}

# ========================================
# DOCUMENTATION FEATURE
# ========================================
resource "tofukit_feature" "readme" {
  name        = "readme"
  description = "Comprehensive README.md documentation"

  prompt = "Create professional README.md with project overview, installation, usage, and development sections"

  constraints = [
    "NO emojis or badges",
    "Use proper markdown headers (##)",
    "Include code blocks with syntax highlighting",
    "Maximum 500 lines for maintainability"
  ]

  files = {
    "README.md" = {
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
  }

  kits = []
}
