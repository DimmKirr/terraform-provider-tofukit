# ========================================
# LANGUAGE KIT - Python Runtime
# ========================================
resource "tofukit_language" "python" {
  name        = "python"
  description = "Python high-level programming language runtime"
  version     = "3.12.6"

  requirement {
    name     = "Python Runtime Installation"
    priority = 100
    instructions = [
      "Install Python 3.12+ via system package manager",
      "Ubuntu/Debian: sudo apt install python3 python3-dev",
      "macOS: brew install python@3.12",
      "Windows: Download installer from python.org",
      "Verify installation with: python3 --version"
    ]
    verification {
      command = "python3 --version"
      expect  = "Python 3.12"
    }
  }
}

# ========================================
# TOOL KIT - Package Manager (pip)
# ========================================
resource "tofukit_tool" "pip" {
  name        = "pip"
  description = "Python package installer and dependency manager"
  version     = "24.0"

  depends_on_refs = ["@tofukit_language.python"]
  depends_on      = [tofukit_language.python]

  requirement {
    name     = "Package Manager Setup"
    priority = 95
    instructions = [
      "Ensure pip is installed and updated",
      "python3 -m ensurepip --upgrade",
      "python3 -m pip install --upgrade pip"
    ]
    verification {
      command = "python3 -m pip --version"
      expect  = "pip"
    }
  }
}

# ========================================
# TOOL KIT - Virtual Environment Manager
# ========================================
resource "tofukit_tool" "venv" {
  name        = "venv"
  description = "Python virtual environment for dependency isolation"
  version     = "3.12"

  depends_on_refs = ["@tofukit_language.python"]
  depends_on      = [tofukit_language.python]

  requirement {
    name     = "Virtual Environment Creation"
    priority = 90
    instructions = [
      "Create isolated Python environment",
      "python3 -m venv .venv",
      "Activate: source .venv/bin/activate (Unix)",
      "Activate: .venv\\Scripts\\activate (Windows)"
    ]
    verification {
      command = "test -d .venv && echo 'venv exists'"
      expect  = "venv exists"
    }
  }
}

# ========================================
# FRAMEWORK KIT - Click CLI Framework
# ========================================
resource "tofukit_framework" "click" {
  name        = "click"
  description = "Composable command line interface toolkit"
  version     = "8.1.7"

  depends_on_refs = ["@tofukit_tool.pip"]
  depends_on      = [tofukit_tool.pip]

  requirement {
    name     = "CLI Framework Installation"
    priority = 85
    instructions = [
      "Install Click framework for CLI development",
      "pip install click==8.1.7",
      "Verify with: pip show click"
    ]
    verification {
      command = "pip show click | grep Version"
      expect  = "8.1.7"
    }
  }
}

# ========================================
# TESTING KIT - Pytest Framework
# ========================================
resource "tofukit_framework" "pytest" {
  name        = "pytest"
  description = "Python testing framework with fixtures and plugins"
  version     = "7.4.3"

  depends_on_refs = ["@tofukit_tool.pip"]
  depends_on      = [tofukit_tool.pip]

  requirement {
    name     = "Testing Framework Setup"
    priority = 80
    instructions = [
      "Install pytest for unit testing",
      "pip install pytest==7.4.3 pytest-cov==4.1.0",
      "Configure pytest.ini for project settings"
    ]
    verification {
      command = "pytest --version"
      expect  = "pytest 7.4.3"
    }
  }
}

# ========================================
# QUALITY KIT - Black Formatter
# ========================================
resource "tofukit_tool" "black" {
  name        = "black"
  description = "The uncompromising Python code formatter"
  version     = "23.11.0"

  depends_on_refs = ["@tofukit_tool.pip"]
  depends_on      = [tofukit_tool.pip]

  requirement {
    name     = "Black Formatter Installation"
    priority = 75
    instructions = [
      "Install Black code formatter",
      "pip install black==23.11.0",
      "Configure pyproject.toml with Black settings"
    ]
    verification {
      command = "black --version"
      expect  = "black, 23.11.0"
    }
  }
}

# ========================================
# QUALITY KIT - Ruff Linter
# ========================================
resource "tofukit_tool" "ruff" {
  name        = "ruff"
  description = "An extremely fast Python linter and formatter"
  version     = "0.1.6"

  depends_on_refs = ["@tofukit_tool.pip"]
  depends_on      = [tofukit_tool.pip]

  requirement {
    name     = "Ruff Linter Installation"
    priority = 74
    instructions = [
      "Install Ruff for fast Python linting",
      "pip install ruff==0.1.6",
      "Configure ruff.toml or pyproject.toml"
    ]
    verification {
      command = "ruff --version"
      expect  = "ruff 0.1.6"
    }
  }
}

# ========================================
# QUALITY KIT - MyPy Type Checker
# ========================================
resource "tofukit_tool" "mypy" {
  name        = "mypy"
  description = "Static type checker for Python"
  version     = "1.7.1"

  depends_on_refs = ["@tofukit_tool.pip"]
  depends_on      = [tofukit_tool.pip]

  requirement {
    name     = "MyPy Type Checker Installation"
    priority = 73
    instructions = [
      "Install MyPy for static type checking",
      "pip install mypy==1.7.1 types-click==7.1.8",
      "Configure mypy.ini with project settings"
    ]
    verification {
      command = "mypy --version"
      expect  = "mypy 1.7.1"
    }
  }
}

# ========================================
# METHODOLOGY KIT - Python Best Practices
# ========================================
resource "tofukit_methodology" "python_standards" {
  name        = "python-standards"
  description = "Python coding standards and best practices"
  version     = "2.0.0"

  requirement {
    name     = "PEP 8 Compliance"
    priority = 70
    instructions = [
      "Follow PEP 8 style guide",
      "4 spaces indentation, 79 char line limit",
      "snake_case for functions/variables",
      "PascalCase for classes",
      "Descriptive variable names"
    ]
    verification {
      command = "ruff check --select E,W,F --statistics"
      expect  = "0 errors"
    }
  }

  requirement {
    name     = "Type Annotations"
    priority = 65
    instructions = [
      "Use type hints for all functions",
      "Import from typing module",
      "Run mypy for type checking",
      "Document return types"
    ]
    verification {
      command = "mypy --strict app.py"
      expect  = "Success"
    }
  }

  requirement {
    name     = "Documentation Standards"
    priority = 60
    instructions = [
      "Write docstrings for all modules/functions",
      "Use Google or NumPy docstring format",
      "Include usage examples",
      "Document edge cases"
    ]
    verification {
      command = "python -m pydoc app"
      expect  = "Help on module"
    }
  }
}

# ========================================
# PACKAGING KIT - Build Tool
# ========================================
resource "tofukit_tool" "build" {
  name        = "build"
  description = "Python build frontend for PEP 517 projects"
  version     = "1.0.3"

  depends_on_refs = ["@tofukit_tool.pip"]
  depends_on      = [tofukit_tool.pip]

  requirement {
    name     = "Build Tool Installation"
    priority = 55
    instructions = [
      "Install Python build tool",
      "pip install build==1.0.3",
      "Create pyproject.toml for package config"
    ]
    verification {
      command = "python -m build --version"
      expect  = "build 1.0.3"
    }
  }
}

# ========================================
# PACKAGING KIT - Twine Upload Tool
# ========================================
resource "tofukit_tool" "twine" {
  name        = "twine"
  description = "Utility for publishing Python packages to PyPI"
  version     = "4.0.2"

  depends_on_refs = ["@tofukit_tool.pip"]
  depends_on      = [tofukit_tool.pip]

  requirement {
    name     = "Twine Installation"
    priority = 54
    instructions = [
      "Install Twine for PyPI uploads",
      "pip install twine==4.0.2",
      "Configure .pypirc for credentials"
    ]
    verification {
      command = "twine --version"
      expect  = "twine version 4.0.2"
    }
  }
}
