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

resource "tofukit_tool" "pip" {
  name        = "pip"
  description = "Python package installer and dependency manager"
  version     = "24.0"

  depends_on = [tofukit_language.python]

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


resource "tofukit_tool" "venv" {
  name        = "venv"
  description = "Python virtual environment for dependency isolation"
  version     = "3.12"

  depends_on = [tofukit_language.python]

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


resource "tofukit_framework" "click" {
  name        = "click"
  description = "Composable command line interface toolkit"
  version     = "8.1.7"

  depends_on = [tofukit_tool.pip]

  requirement {
    name     = "CLI Framework Installation"
    priority = 85
    instructions = [
      "Install Click framework for CLI development",
      "pip install click==8.1.7 or make it a part the selected package management system",
    ]
    verification {
      command = "pip show click | grep Version"
      expect  = "8.1.7"
    }
  }
}

