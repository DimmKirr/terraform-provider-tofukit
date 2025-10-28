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
  # Files now use reusable file resources defined in files.tofu
  # This demonstrates the file resource pattern for better modularity and reusability
  files = {
    "src/cli.py"       = tofukit_file.python_click_cli
    "requirements.txt" = tofukit_file.python_requirements
    "demo2.txt"        = tofukit_file.demo_file
    "README.md"        = tofukit_file.readme_docs
  }
}
