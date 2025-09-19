# Example of project configuration with kits manually specified
# This shows what the structure should look like

resource "tofukit_project" "hello_world_cli_v2" {
  name        = "hello-world-cli"
  description = "A simple Hello World CLI application using Python and Click"
  version     = "0.1.0"

  # Manual specification of kits - this would need to be automated
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

  requirement {
    name = "Project Setup"
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
}
