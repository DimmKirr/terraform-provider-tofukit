# Define the Hello World CLI project
resource "tofukit_project" "hello_world_cli" {
  name        = "hello-world-cli"
  description = "A simple Hello World CLI application using Python and Click"
  version     = "0.1.0"

  # Aggregate all kits with their requirements
  kits = {
    "language.python" = {
      id          = "language.python"
      type        = "language"
      name        = tofukit_language.python.name
      description = tofukit_language.python.description
      version     = tofukit_language.python.version
      requirements = tofukit_language.python.requirement
    }
    "tool.pip" = {
      id          = "tool.pip"
      type        = "tool"
      name        = tofukit_tool.pip.name
      description = tofukit_tool.pip.description
      version     = tofukit_tool.pip.version
      requirements = tofukit_tool.pip.requirement
    }
    "tool.venv" = {
      id          = "tool.venv"
      type        = "tool"
      name        = tofukit_tool.venv.name
      description = tofukit_tool.venv.description
      version     = tofukit_tool.venv.version
      requirements = tofukit_tool.venv.requirement
    }
    "framework.click" = {
      id          = "framework.click"
      type        = "framework"
      name        = tofukit_framework.click.name
      description = tofukit_framework.click.description
      version     = tofukit_framework.click.version
      requirements = tofukit_framework.click.requirement
    }
    "methodology.idiomatic" = {
      id          = "methodology.idiomatic"
      type        = "methodology"
      name        = tofukit_methodology.idiomatic.name
      description = tofukit_methodology.idiomatic.description
      version     = tofukit_methodology.idiomatic.version
      requirements = tofukit_methodology.idiomatic.requirement
    }
  }

  depends_on = [
    tofukit_stack.python_click_cli,
    tofukit_language.python,
    tofukit_framework.click,
    tofukit_tool.pip,
    tofukit_tool.venv,
    tofukit_methodology.idiomatic
  ]

  requirement {
    name = "Project Setup"
    priority = 100  # Critical - must be done first
    instructions = [
      "Create a new directory for the project",
      "Initialize git repository",
      "Set up Python virtual environment",
      "Install Click framework"
    ]
    verification {
      command = "ls -la .venv"
      expect  = ".venv"
    }
  }

  requirement {
    name = "Basic Hello World"
    priority = 90  # Core functionality
    instructions = [
      "Create a CLI application that prints 'Hello, World!'",
      "Use Click framework for command-line interface",
      "Follow Python best practices and PEP 8"
    ]
    verification {
      command = "python app.py"
      expect  = "Hello, World!"
    }
  }

  requirement {
    name = "Name Parameter"
    priority = 80  # Feature - high priority
    instructions = [
      "Add --name option to customize the greeting",
      "Default value should be 'World'",
      "Example: python app.py --name Alice → 'Hello, Alice!'"
    ]
    verification {
      command = "python app.py --name Test"
      expect  = "Hello, Test!"
    }
  }

  requirement {
    name = "Count Parameter"
    priority = 70  # Feature - medium priority
    instructions = [
      "Add --count option to repeat the greeting",
      "Must be a positive integer",
      "Default value should be 1"
    ]
    verification {
      command = "python app.py --count 2"
      expect  = "Hello, World!"
    }
  }

  requirement {
    name = "Shout Mode"
    instructions = [
      "Add --shout flag to output in uppercase",
      "When enabled, greeting should be in ALL CAPS"
    ]
    verification {
      command = "python app.py --shout"
      expect  = "HELLO, WORLD!"
    }
  }

  requirement {
    name = "Version Information"
    instructions = [
      "Add --version option to display version",
      "Should show 'hello-cli, version 0.1.0'"
    ]
    verification {
      command = "python app.py --version"
      expect  = "0.1.0"
    }
  }

  requirement {
    name = "Help Documentation"
    instructions = [
      "Ensure --help option shows usage information",
      "Document all options clearly",
      "Include examples in the docstring"
    ]
    verification {
      command = "python app.py --help"
      expect  = "Simple CLI that greets"
    }
  }

  requirement {
    name = "Testing"
    priority = 85  # Very important
    instructions = [
      "Write unit tests for all functionality",
      "Use pytest framework",
      "Achieve 100% code coverage for app.py"
    ]
    verification {
      command = "pytest test_app.py"
      expect  = "passed"
    }
  }

  requirement {
    name = "Code Quality"
    priority = 75  # Important
    instructions = [
      "Code must be formatted with Black",
      "Pass mypy type checking",
      "Pass ruff linting",
      "Include type hints for all functions"
    ]
    verification {
      command = "black --check app.py"
      expect  = "would be reformatted"
    }
  }

  requirement {
    name = "Documentation"
    priority = 60  # Nice to have
    instructions = [
      "Create comprehensive README.md",
      "Include installation instructions",
      "Provide usage examples",
      "Document development setup"
    ]
    verification {
      command = "ls README.md"
      expect  = "README.md"
    }
  }

  requirement {
    name = "Packaging"
    priority = 50  # Optional
    instructions = [
      "Create setup.py for pip installation",
      "Define entry point for CLI command",
      "Include all necessary metadata"
    ]
    verification {
      command = "ls setup.py"
      expect  = "setup.py"
    }
  }
}
