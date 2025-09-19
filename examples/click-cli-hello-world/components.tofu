# Define the pip tool
resource "tofukit_project" "test_prod" {
  name    = ""
  version = ""

}
resource "tofukit_tool" "pip" {
  name        = "pip"
  description = "Package installer for Python"
  version     = "25.2"

  depends_on_refs = ["@tofukit_language.python"]
  depends_on = [tofukit_language.python]

  requirement {
    name = "Install pip"
    instructions = [
      "Pip usually comes with Python installation",
      "If not installed: python3 -m ensurepip",
      "Or: sudo apt install python3-pip (Ubuntu/Debian)"
    ]
    verification {
      command = "pip3 --version"
      expect  = "pip"
    }
  }
}

# Define virtual environment tool
resource "tofukit_tool" "venv" {
  name        = "venv"
  description = "Python virtual environment tool"
  version     = "3.12.6"

  depends_on_refs = ["@tofukit_language.python"]
  depends_on = [tofukit_language.python]

  requirement {
    name = "Create virtual environment"
    instructions = [
      "python3 -m venv .venv"
    ]
    verification {
      command = "ls -la .venv"
      expect  = ".venv"
    }
  }

  requirement {
    name = "Activate virtual environment"
    instructions = [
      "source .venv/bin/activate (Linux/macOS)",
      ".venv\\Scripts\\activate (Windows)"
    ]
    verification {
      command = "which python"
      expect  = ".venv"
    }
  }
}

# Define the Click framework
resource "tofukit_framework" "click" {
  name        = "click"
  description = "Python package for creating beautiful command line interfaces"
  version     = "8.1.7"

  depends_on_refs = ["@tofukit_language.python", "@tofukit_tool.pip"]
  depends_on = [tofukit_language.python]

  requirement {
    name = "Install Click"
    instructions = [
      "pip install click==8.1.7"
    ]
    verification {
      command = "pip show click"
      expect  = "8.1.7"
    }
  }
}

# Define idiomatic methodology
resource "tofukit_methodology" "idiomatic" {
  name        = "idiomatic"
  description = "Follow Python idiomatic practices"
  version     = "1.0.0"

  requirement {
    name = "Follow PEP 8"
    instructions = [
      "Use 4 spaces for indentation",
      "Limit lines to 79 characters",
      "Use snake_case for functions and variables",
      "Use PascalCase for classes"
    ]
    verification {
      command = "echo 'PEP 8 standards applied'"
      expect  = "applied"
    }
  }

  requirement {
    name = "Use type hints"
    instructions = [
      "Add type hints to function signatures",
      "Use typing module for complex types"
    ]
    verification {
      command = "echo 'Type hints configured'"
      expect  = "configured"
    }
  }
}
