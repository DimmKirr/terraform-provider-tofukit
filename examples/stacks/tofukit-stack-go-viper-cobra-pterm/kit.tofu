# Go CLI Stack Module (Cobra + Viper + pterm)
# This module provides a complete Go CLI application stack
# with all necessary kits and file scaffolding

terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# ========================================
# LANGUAGE KITS
# ========================================
resource "tofukit_language" "go" {
  name        = "go"
  description = "Go programming language"
  version     = var.go_version

  requirements = [
    {
      name = "Go Runtime Verification"

      instructions = [
        {
          prompt = "Verify Go ${var.go_version}+ is available on the system. If not installed, Go must be installed manually: Ubuntu/Debian: sudo apt install golang-go, macOS: brew install go, Windows: Download from golang.org"
        }
      ]

      verifications = [
        {
          command = "go version"
          expect  = "go${var.go_version}"
        }
      ]
    },
    {
      name = "Idiomatic Go Code Standards"

      instructions = [
        {
          prompt = <<-EOF
          ALL Go code MUST follow idiomatic, community-preferred, conventional Go patterns:

          - Use standard library patterns and idioms
          - Follow Effective Go guidelines (https://go.dev/doc/effective_go)
          - Use conventional project structure (cmd/, internal/, pkg/ when appropriate)
          - Prefer composition over inheritance
          - Use interfaces appropriately (accept interfaces, return structs)
          - Follow standard naming conventions (mixedCase for unexported, PascalCase for exported)
          - Use standard error handling patterns (return errors, don't panic)
          - Prefer simple, readable code over clever abstractions
          - Use context.Context for cancellation and request-scoped values
          - Follow Go proverbs: "Clear is better than clever", "Don't communicate by sharing memory, share memory by communicating"
          EOF

          constraints = [
            "NO non-standard patterns that deviate from community conventions",
            "NO unnecessary abstractions or over-engineering",
            "NO panic() in production code (except for unrecoverable programmer errors)",
            "NO ignoring errors silently",
            "NO deep inheritance hierarchies"
          ]
        }
      ]

      verifications = []
    }
  ]
}

# ========================================
# TOOL KITS
# ========================================
resource "tofukit_tool" "go_mod" {
  name        = "go-mod"
  description = "Go modules dependency management"
  version     = var.go_version

  depends_on = [tofukit_language.go]

  requirements = [
    {
      name = "Go Modules Initialization"

      instructions = [
        {
          prompt = "Initialize Go modules for the project: go mod init ${var.project_module_path}"
        }
      ]

      verifications = [
        {
          command = "test -f go.mod && echo 'go.mod exists'"
          expect  = "go.mod exists"
        }
      ]
    }
  ]
}

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

# ========================================
# APPLICATION STACK
# ========================================
resource "tofukit_stack" "go_cli_app" {
  name        = "go-cli-app"
  description = "Complete Go CLI application with Cobra, Viper, and pterm"

  # Stack references its kits
  kits = [
    tofukit_language.go,
    tofukit_tool.go_mod,
    tofukit_framework.cobra,
    tofukit_framework.viper,
    tofukit_framework.pterm,
  ]

  # Main entry point
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

    "internal/config/config.go" = {
      instructions = [
        {
          prompt = <<-EOF
          Create internal/config/config.go as a Viper configuration abstraction layer:

          PACKAGE & IMPORTS:
          - Package: config
          - Required imports: os, path/filepath, ${var.project_module_path}/internal/logger, github.com/spf13/viper

          TYPE DEFINITIONS:
          - Config struct with fields: LogLevel (string), LogPlainText (bool), ConfigFile (string)
          - Package-level variable: AppConfig *Config

          LOADCONFIG() FUNCTION:
          - Set environment prefix: "${var.project_name}"
          - Enable automatic environment variable reading
          - Set config file name: ".${var.project_name}"
          - Set config file type: "yaml"
          - Add config search paths: current directory and user home directory
          - Set defaults: LOG_LEVEL="info", LOG_PLAIN_TEXT=false
          - Read config file (ignore error if not found, log with logger.Debug)
          - Unmarshal into AppConfig struct
          - Re-initialize logger with config settings from AppConfig
          - Return error or nil

          SAVECONFIG() FUNCTION:
          - Get current working directory
          - Build config path: current_dir/.${var.project_name}.yaml
          - Write config using viper.WriteConfigAs
          - Return error or nil
          EOF

          constraints = [
            "DO NOT expose viper directly to other packages - this is an abstraction layer",
            "DO NOT hard-code file paths - use os.UserHomeDir() and os.Getwd()",
            "DO NOT fail if config file doesn't exist - it's optional",
            "DO NOT use complex error handling - just return errors up the stack"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q '^package config' internal/config/config.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'type Config struct' internal/config/config.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func LoadConfig()' internal/config/config.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func SaveConfig()' internal/config/config.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'github.com/spf13/viper' internal/config/config.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q '${var.project_module_path}/internal/logger' internal/config/config.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'SetEnvPrefix' internal/config/config.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'SetConfigName.*\\.${var.project_name}' internal/config/config.go && echo 'found'"
          expect  = "found"
        }
      ]
    }

    "internal/version/version.go" = {
      instructions = [
        {
          prompt = <<-EOF
          Create internal/version/version.go for version management with Git commit tracking:

          PACKAGE & IMPORTS:
          - Package: version
          - Imports: fmt, runtime/debug

          VARIABLES:
          - GitCommit (string) - will be set via ldflags during build, default empty
          - Version (string) - set to "${var.project_version}"

          GETVERSION() FUNCTION:
          - Try to read version from debug.ReadBuildInfo()
          - If available and non-empty, return b.Main.Version
          - Otherwise return the Version variable
          - Return type: string

          FULLVERSIONSTRING() FUNCTION:
          - Start with Version variable
          - If GitCommit is not empty, append it: "version (gitcommit)"
          - Use fmt.Sprintf for formatting
          - Return type: string
          EOF

          constraints = [
            "NO hard-coded version strings besides ${var.project_version}",
            "NO complex version parsing logic",
            "NO external dependencies besides stdlib",
            "Keep it simple and maintainable"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q '^package version' internal/version/version.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'var.*GitCommit.*string' internal/version/version.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'Version.*=.*\"${var.project_version}\"' internal/version/version.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func GetVersion()' internal/version/version.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func FullVersionString()' internal/version/version.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'debug\\.ReadBuildInfo' internal/version/version.go && echo 'found'"
          expect  = "found"
        }
      ]
    }

    "internal/logger/logger.go" = {
      instructions = [
        {
          prompt = <<-EOF
          Create internal/logger/logger.go as a pterm+slog logging wrapper:

          PACKAGE & IMPORTS:
          - Package: logger
          - Imports: log/slog, os, strings, github.com/pterm/pterm

          PACKAGE VARIABLES:
          - defaultLogger (*slog.Logger) - package-level logger instance
          - quietMode (bool) - when true, suppress Info/Debug/Warn logs

          SETQUIETMODE(BOOL) FUNCTION:
          - Sets the quietMode variable
          - Used to suppress non-error logs

          INITIALIZE(LOGLEVEL STRING, LOGPLAINTEXT BOOL) FUNCTION:
          - Parse logLevel string (case-insensitive) to pterm.LogLevel: "debug"→LogLevelDebug, "info"→LogLevelInfo, "warning"→LogLevelWarn, "error"→LogLevelError
          - Default to LogLevelInfo if unknown
          - Create pterm slog handler: pterm.NewSlogHandler(&pterm.DefaultLogger)
          - Set pterm.DefaultLogger.Level to the parsed level
          - If logPlainText is false, call customizePtermStyle()
          - Initialize defaultLogger with the handler

          LOGGING METHODS:
          - Info(msg string, keysAndValues ...interface{}) - log if !quietMode
          - Debug(msg string, keysAndValues ...interface{}) - log if !quietMode
          - Warn(msg string, keysAndValues ...interface{}) - log if !quietMode
          - Error(msg string, keysAndValues ...interface{}) - always log
          - Fatal(msg string, keysAndValues ...interface{}) - log error and os.Exit(1)

          CUSTOMIZEPTERMSTYLE() FUNCTION:
          - Customize pterm prefixes with Unicode symbols and colors:
            - Info: ℹ (cyan, bold)
            - Warning: ⚠ (yellow, bold)
            - Success: ✔ (light green, bold)
            - Error: ⨯ (red, bold)
            - Debug: ⚙ (magenta)
          - Use pterm.Prefix struct with Text and Style fields
          - Use pterm.NewStyle with appropriate colors
          EOF

          constraints = [
            "NO direct log/slog usage outside Initialize - use wrapper methods",
            "NO panic() - use Fatal() for unrecoverable errors",
            "NO complex logging logic - keep it simple",
            "quietMode must NOT suppress Error or Fatal logs"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q '^package logger' internal/logger/logger.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func Initialize' internal/logger/logger.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func Info' internal/logger/logger.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func Debug' internal/logger/logger.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func Warn' internal/logger/logger.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func Error' internal/logger/logger.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func Fatal' internal/logger/logger.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'func SetQuietMode' internal/logger/logger.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'pterm\\.NewSlogHandler' internal/logger/logger.go && echo 'found'"
          expect  = "found"
        }
      ]
    }

    "Taskfile.yaml" = {
      instructions = [
        {
          prompt = <<-EOF
          Create Taskfile.yaml for task automation with version injection via ldflags:

          VERSION: '3'

          VARIABLES:
          - GIT_COMMIT_HASH: Shell command "git rev-parse --short HEAD 2>/dev/null || echo 'unknown'"
          - GIT_IMPORT_URL: "${var.project_module_path}/internal/version"
          - GIT_IS_DIRTY: Shell command "test -n \"`git status --porcelain 2>/dev/null`\" && echo '+CHANGES' || true"
          - GO_LDFLAGS: "-s -w -X {{.GIT_IMPORT_URL}}.GitCommit={{.GIT_COMMIT_HASH}}{{.GIT_IS_DIRTY}} -X {{.GIT_IMPORT_URL}}.Version=${var.project_version}"
          - CGO_ENABLED: 0

          TASKS:
          - default: depends on build
          - build:
            - Description: "Build the ${var.project_name} binary"
            - Commands: mkdir -p bin, CGO_ENABLED={{.CGO_ENABLED}} go build -ldflags "{{.GO_LDFLAGS}}" -o ./bin/${var.project_name}, chmod +x ./bin/${var.project_name}
          - test:
            - Description: "Run tests"
            - Command: go test -v ./...
          - clean:
            - Description: "Clean build artifacts"
            - Commands: rm -rf bin/, go clean
          - install:
            - Description: "Install the binary to GOPATH/bin"
            - Depends: build
            - Command: go install -ldflags "{{.GO_LDFLAGS}}"
          EOF

          constraints = [
            "NO hard-coded version strings - use ldflags injection",
            "Build output MUST go to bin/ directory",
            "NO complex shell scripts - keep tasks simple",
            "Use Go toolchain commands (go build, go test, go clean, go install)"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f Taskfile.yaml && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q \"version: '3'\" Taskfile.yaml && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'build:' Taskfile.yaml && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'GO_LDFLAGS' Taskfile.yaml && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'bin/${var.project_name}' Taskfile.yaml && echo 'found'"
          expect  = "found"
        }
      ]
    }

    ".gitignore" = {
      instructions = [
        {
          prompt = <<-EOF
          Create .gitignore for a Go project following community standards:

          IGNORE PATTERNS (grouped with comments):

          Binaries:
          - bin/ (build output directory)
          - ${var.project_name} (project binary)
          - *.exe, *.dll, *.so, *.dylib (platform binaries)
          - *.test (test binaries from go test -c)

          Go coverage:
          - *.out
          - coverage.txt

          Go workspace:
          - go.work (Go 1.18+ workspace file)

          Dependencies:
          - vendor/ (vendored dependencies)

          IDE files:
          - .idea/ (IntelliJ/GoLand)
          - .vscode/ (VS Code)
          - *.swp, *.swo, *~ (vim swap files)

          OS files:
          - .DS_Store (macOS)
          - Thumbs.db (Windows)

          Config files (may contain sensitive data):
          - .${var.project_name}.yaml
          - .${var.project_name}.yml
          EOF

          constraints = [
            "Follow standard Go .gitignore patterns",
            "NO application-specific files beyond project name",
            "Include comments for clarity",
            "Cover common IDEs and OS"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f .gitignore && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'bin/' .gitignore && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q '\\.${var.project_name}\\.yaml' .gitignore && echo 'found'"
          expect  = "found"
        }
      ]
    }

    "cmd/root.go" = {
      instructions = [
        {
          prompt = <<-EOF
          Create cmd/root.go as the Cobra CLI root command implementation:

          PACKAGE & IMPORTS:
          - Package: cmd
          - Imports: fmt, os, ${var.project_module_path}/internal/logger, ${var.project_module_path}/internal/version, github.com/pterm/pterm, github.com/spf13/cobra

          ROOT COMMAND (var rootCmd):
          - Use: "${var.project_name}"
          - Short: "A CLI application built with Cobra, Viper, and pterm"
          - Version: version.FullVersionString()
          - Long: Multi-line description mentioning Cobra, Viper, pterm

          EXECUTE() FUNCTION:
          - Call rootCmd.Execute() and check for error
          - If error, log with logger.Error and os.Exit(1)

          INIT() FUNCTION:
          - Define PersistentFlags: "config" (empty default), "log-level" (default "info"), "quiet" (bool, default false)
          - Add subcommands: rootCmd.AddCommand(helloCmd, versionCmd)

          HELLO COMMAND (var helloCmd):
          - Use: "hello [name]"
          - Short: "Say hello to someone"
          - Long: "Greet someone by name, or the world if no name is provided"
          - Args: cobra.MaximumNArgs(1)
          - Run: Extract name from args (default "World"), log with logger.Info: "Hello, <name>!"

          VERSION COMMAND (var versionCmd):
          - Use: "version"
          - Short: "Print version information"
          - Run: Use pterm.DefaultBox.WithTitle("Version Information").WithTitleTopCenter().Println with version.FullVersionString()
          EOF

          constraints = [
            "NO direct cobra.Command initialization - use struct literals",
            "NO ignoring errors from rootCmd.Execute()",
            "Use internal/logger and internal/version packages",
            "PersistentFlags for flags that apply to all subcommands",
            "Keep subcommands simple - hello and version only"
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
        },
        {
          command = "grep -q 'var versionCmd' cmd/root.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "./bin/${var.project_name} hello 2>&1 | grep -o 'Hello, World!'"
          expect  = "Hello, World!"
        },
        {
          command = "./bin/${var.project_name} --version 2>&1 | grep -o '${var.project_version}'"
          expect  = "${var.project_version}"
        }
      ]
    }

    # NOTE: go.mod is NOT declared as a file - it's generated automatically by:
    # 1. tofukit_tool.go_mod requirement: "go mod init ${var.project_module_path}"
    # 2. Framework kit requirements: "go get ..." commands that add dependencies
    # 3. "go mod tidy" commands that clean up dependencies
    # Declaring it as a file would conflict with "go mod init"

    "README.md" = {
      instructions = [
        {
          prompt = "Create comprehensive documentation for this Go CLI project. Include the following sections: project overview, features, installation instructions, usage examples with command-line examples, configuration options, development setup, and contributing guidelines. Make the documentation clear, professional, and easy to follow."
          constraints = [
            "Do NOT include emojis or badges",
            "Use clear section headers (##) for main sections",
            "Include code blocks with proper syntax highlighting using ```go and ```bash",
            "Keep explanations concise but complete",
            "Must include project name '${var.project_name}' in the title",
            "Include description about Go CLI application using Cobra, Viper, and pterm",
            "Include setup instructions for Go ${var.go_version}",
            "Include usage examples that demonstrate the CLI commands",
            "Include build instructions: go build -o ${var.project_name} .",
            "Maximum 500 lines to keep it focused and maintainable"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q '${var.project_name}' README.md && echo 'found'"
          expect  = "found"
        },
        {
          command = "test -f README.md && echo 'README exists'"
          expect  = "README exists"
        },
        {
          command = "grep -q '^# ' README.md && echo 'Has title'"
          expect  = "Has title"
        }
      ]
    }
  }
}
