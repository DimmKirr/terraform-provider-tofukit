# Python Click CLI Stack Module
# This module provides a complete Python Click CLI application stack
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
resource "tofukit_language" "python" {
  name        = "python"
  description = "Python high-level programming language runtime"
  version     = "3.12.6"

  requirements = [
    {
      name = "Python Runtime Verification"

      instructions = [
        {
          prompt = "Verify Python 3.12+ is available on the system. If not installed, Python must be installed manually: Ubuntu/Debian: sudo apt install python3 python3-dev, macOS: brew install python@3.12, Windows: Download from python.org"
        }
      ]

      verifications = [
        {
          command = "python3 --version"
          expect  = "Python 3.12"
        }
      ]
    }
  ]
}

# ========================================
# TOOL KITS
# ========================================
resource "tofukit_tool" "pip" {
  name        = "pip"
  description = "Python package installer and dependency manager"
  version     = "24.0"

  depends_on = [tofukit_language.python]

  requirements = [
    {
      name = "Pip Installation"

      instructions = [
        {
          prompt = "Ensure pip is available by running: python3 -m ensurepip --upgrade"
        }
      ]

      verifications = [
        {
          command = "python3 -m pip --version"
          expect  = "pip"
        }
      ]
    },
    {
      name = "Pip Upgrade"

      instructions = [
        {
          prompt = "Upgrade pip to latest version: python3 -m pip install --upgrade pip"
        }
      ]

      verifications = [
        {
          command = "python3 -m pip --version"
          expect  = "24."
        }
      ]
    }
  ]
}

resource "tofukit_tool" "venv" {
  name        = "venv"
  description = "Python virtual environment for dependency isolation"
  version     = "3.12"

  depends_on = [tofukit_language.python]

  requirements = [
    {
      name = "Virtual Environment Creation"

      instructions = [
        {
          prompt      = "Create Python virtual environment in .venv directory: python3 -m venv .venv"
          constraints = [
            "Do NOT create outside current working directory",
            "Do NOT use alternative tools (virtualenv, conda)"
          ]
        }
      ]

      verifications = [
        {
          command = "test -d .venv && echo 'venv exists'"
          expect  = "venv exists"
        }
      ]
    }
  ]
}

# ========================================
# FRAMEWORK KITS
# ========================================
resource "tofukit_framework" "click" {
  name        = "click"
  description = "Composable command line interface toolkit"
  version     = "8.1.7"

  depends_on = [tofukit_tool.pip]

  requirements = [
    {
      name = "CLI Framework Installation"

      instructions = [
        {
          prompt      = "Install Click framework version 8.1.7: pip install click==8.1.7"
          constraints = [
            "Do NOT install any version other than 8.1.7",
            "Do NOT use --user flag (install to project environment only)",
            "Do NOT install from git or source (use official PyPI only)"
          ]
        }
      ]

      verifications = [
        {
          command = "pip show click | grep Version"
          expect  = "8.1.7"
        }
      ]
    }
  ]
}

# ========================================
# APPLICATION STACK
# ========================================
resource "tofukit_stack" "click_app" {
  name        = "python-click-app"
  description = "Complete Python Click CLI application with testing and packaging"

  # Stack references its kits
  kits = [
    tofukit_language.python,
    tofukit_tool.pip,
    tofukit_tool.venv,
    tofukit_framework.click,
  ]

  # Main CLI Application
  files = {
    "src/cli.py" = {
      content = <<-EOF
      #!/usr/bin/env python3
      """
      Simple CLI application using Click framework.
      """
      import click


      @click.group()
      @click.version_option(version='${var.project_version}', prog_name='${var.project_name}')
      def cli() -> None:
          """A simple Python CLI application."""
          pass


      @cli.command()
      def hello() -> None:
          """Output world text."""
          click.echo("world")


      if __name__ == '__main__':
          cli()
      EOF

      verifications = [
        {
          command = "python3 src/cli.py hello"
          expect  = "world"
        },
        {
          command = "python3 src/cli.py --version"
          expect  = var.project_version
        }
      ]
    }

    "requirements.txt" = {
      content = <<-EOF
      click==${tofukit_framework.click.version}
      EOF

      verifications = [
        {
          command = "grep -q 'click==${tofukit_framework.click.version}' requirements.txt"
        },
        {
          command = "test -f requirements.txt"
        }
      ]
    }

    "demo2.txt" = {
      content = <<-EOF
      hello world
      EOF

      verifications = [
        {
          command = "cat demo2.txt"
          expect  = "hello world"
        }
      ]
    }

    "README.md" = {
      instructions = [
        {
          prompt = "Create comprehensive documentation for this Python Click CLI project. Include the following sections: project overview, features, installation instructions, usage examples with command-line examples, configuration options, troubleshooting guide, and contributing guidelines. Make the documentation clear, professional, and easy to follow."
          constraints = [
            "Do NOT include emojis or badges",
            "Use clear section headers (##) for main sections",
            "Include code blocks with proper syntax highlighting",
            "Keep explanations concise but complete",
            "Must include project name '${var.project_name}' in the title",
            "Include description about Python Click CLI application",
            "Include setup instructions for Python 3.12, pip, and Click framework",
            "Include usage examples that demonstrate the CLI commands",
            "Maximum 500 lines to keep it focused and maintainable"
          ]
        }
      ]

      verifications = [
        {
          command = "grep '${var.project_name}' README.md"
          expect  = var.project_name
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
