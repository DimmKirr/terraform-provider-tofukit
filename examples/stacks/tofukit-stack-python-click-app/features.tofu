# Features for Python Click Stack
# Each feature represents a distinct capability

# ========================================
# VERSION COMMAND FEATURE
# ========================================
resource "tofukit_feature" "version_command" {
  name        = "version-command"
  description = "Version command showing project version information"

  requirements = [
    {
      name = "Version Command Implementation"
      instructions = [
        {
          prompt = "Implement a Click CLI with built-in version display using --version flag and version subcommand"

          constraints = [
            "Use Click's built-in @click.version_option decorator",
            "Version must show project version ${var.project_version}",
            "Program name must be ${var.project_name}",
            "Provide both --version flag and 'version' subcommand"
          ]
        }
      ]
    }
  ]

  files = {
    "src/__init__.py" = {
      instructions = [
        {
          prompt = "Create empty __init__.py to make src a Python package"
        }
      ]

      verifications = [
        {
          command = "test -f src/__init__.py && echo 'found'"
          expect  = "found"
        }
      ]
    }

    "src/cli.py" = {
      instructions = [
        {
          prompt = <<-EOF
          Create Click CLI application with version support:

          STRUCTURE:
          - Shebang: #!/usr/bin/env python3
          - Docstring: """${var.project_name} - A Python CLI application."""
          - Import: import click

          MAIN GROUP:
          - @click.group() decorator
          - @click.version_option(version='${var.project_version}', prog_name='${var.project_name}')
          - Function: def cli() -> None with docstring "Main CLI group for ${var.project_name}."

          VERSION COMMAND (explicit subcommand):
          - @cli.command() decorator
          - Function: def version() -> None with docstring "Show version information."
          - Implementation: click.echo(f"${var.project_name} version ${var.project_version}")

          ENTRY POINT:
          - if __name__ == '__main__': cli()

          Note: The @click.version_option decorator provides --version flag automatically,
          but the explicit 'version' subcommand provides an alternative access method.
          EOF

          constraints = [
            "NO print() - use click.echo() only",
            "Include both --version flag and 'version' subcommand",
            "Version output must include project name and version",
            "Include type hints (-> None)",
            "Follow PEP 8 style guide"
          ]
        }
      ]

      verifications = [
        {
          command = "python3 src/cli.py --version 2>&1 | grep -o '${var.project_version}'"
          expect  = var.project_version
        },
        {
          command = "python3 src/cli.py version | grep -o '${var.project_version}'"
          expect  = var.project_version
        },
        {
          command = "python3 src/cli.py version | grep -o '${var.project_name}'"
          expect  = var.project_name
        },
        {
          command = "grep -q '@click.group()' src/cli.py && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q '@click.version_option' src/cli.py && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'def version' src/cli.py && echo 'found'"
          expect  = "found"
        }
      ]
    }
  }

  kits = [tofukit_library.click]
}

# ========================================
# DOCUMENTATION FEATURE
# ========================================
resource "tofukit_feature" "readme" {
  name        = "readme"
  description = "Comprehensive README.md documentation"

  requirements = [
    {
      name = "README Documentation"
      instructions = [
        {
          prompt = "Create professional README.md for Python Click CLI application"

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
    "README.md" = {
      instructions = [
        {
          prompt = <<-EOF
          Create README with sections:
          - Title with project name: ${var.project_name}
          - Overview: Python Click CLI application
          - Features: List key capabilities (version command)
          - Installation: Python ${var.python_version}, uv package manager, Click ${var.click_version}
          - Usage: Command examples showing 'version' subcommand and --version flag
          - Development: uv commands for adding dependencies, running tests
          - Contributing: Guidelines for contributors
          EOF

          constraints = [
            "Keep concise but complete",
            "Include project name '${var.project_name}' in title",
            "Show practical command-line examples",
            "Mention uv for package management"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f README.md && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q '${var.project_name}' README.md && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q '^# ' README.md && echo 'has title'"
          expect  = "has title"
        }
      ]
    }
  }

  kits = []
}
