# Features for Go CLI Stack
# Each feature represents a distinct capability that can be reused across projects
########################################################################################################################
# User Features
########################################################################################################################

# ========================================
# VERSION COMMAND FEATURE
# ========================================
resource "tofukit_feature" "version_command" {
  name        = "version-command"
  description = "Version command with Git commit tracking via ldflags"

  requirements = [
    {
      name = "Version Command Implementation"
      instructions = [
        {
          prompt = "Implement version command showing project version and Git commit, integrated with Cobra CLI and build-time ldflags injection"

          constraints = [
            "Version and GitCommit variables must be injectable via ldflags",
            "Use runtime/debug.ReadBuildInfo() as fallback",
            "Version command must use pterm for formatted output"
          ]
        }
      ]
    }
  ]

  files = {
    "internal/version/version.go" = tofukit_file.version_internal
    "cmd/version.go"               = tofukit_file.version_cmd
  }

  # Note: Taskfile integration mentioned - ldflags will be set via:
  # GO_LDFLAGS: "-X {module}/internal/version.GitCommit={{.GIT_COMMIT_HASH}}"

  kits = [tofukit_library.cobra, tofukit_library.pterm]

  verifications = []
}

########################################################################################################################
# Dev Features
########################################################################################################################

resource "tofukit_feature" "viper_config" {
  name        = "viper-config"
  description = "Viper-based configuration management with YAML support"

  requirements = [
    {
      name = "Configuration Management"
      instructions = [
        {
          prompt = "Implement configuration management using Viper with environment variable support and YAML config files"

          constraints = [
            "Configuration file must be optional, not required",
            "Use environment prefix matching project name",
            "Support user home directory and current directory for config search",
          ]
        }
      ]
    }
  ]

  files = {
    "internal/config/config.go" = tofukit_file.viper_config_internal
  }

  kits = [tofukit_library.viper]

  # Note: depends_on removed - dependency is implicit through file reference above (line 70)
}

resource "tofukit_feature" "pterm_logger" {
  name        = "pterm-logger"
  description = "Structured logging with pterm and slog integration"

  requirements = [
    {
      name = "Logger Implementation"
      instructions = [
        {
          prompt = "Implement logging abstraction using pterm slog handler with quiet mode support and customizable styles"

          constraints = [
            "Quiet mode must suppress Info/Debug/Warn but NOT Error/Fatal",
            "Use pterm slog handler for structured logging",
            "Provide Unicode symbols for log levels"
          ]
        }
      ]
    }
  ]

  files = {
    "internal/logger/logger.go" = tofukit_file.pterm_logger_internal
  }


  kits = [tofukit_library.pterm]
}

resource "tofukit_feature" "taskfile" {
  name        = "taskfile"
  description = "Task automation with Taskfile supporting version injection"

  requirements = [
    {
      name = "Taskfile Configuration"
      instructions = [
        {
          prompt = "Create Taskfile.yaml for build automation with Git commit injection via ldflags"

          constraints = [
            "Version must be injected via ldflags during build",
            "Build output to bin/ directory",
            "Support CGO_ENABLED=0 for static builds"
          ]
        }
      ]
    }
  ]

  files = {
    "Taskfile.yaml" = tofukit_file.taskfile_yaml
  }

  kits = [tofukit_tool.gotask]
}


resource "tofukit_feature" "gitignore" {
  name        = "gitignore"
  description = "Standard Go project .gitignore"

  requirements = [
    {
      name = "Git Ignore Configuration"
      instructions = [
        {
          prompt = "Create .gitignore following Go community standards with IDE and OS file patterns"

          constraints = [
            "Cover binaries, test outputs, coverage files, IDE files, OS files",
            "Include project-specific binary name",
            "Include project-specific config files"
          ]
        }
      ]
    }
  ]

  files = {
    ".gitignore" = tofukit_file.gitignore
  }

  kits = []
}

resource "tofukit_feature" "readme" {
  name        = "readme"
  description = "Comprehensive README.md documentation"

  requirements = [
    {
      name = "README Documentation"
      instructions = [
        {
          prompt = "Create professional README.md with project overview, installation, usage, and development sections"

          constraints = [
            "NO emojis or badges",
            "Use proper markdown headers (##)",
            "Include code blocks with syntax highlighting",
            "Maximum 500 lines for maintainability"
          ]
        }
      ]
    }
  ]

  files = {
    "README.md" = tofukit_file.readme_md
  }

  kits = []
}
