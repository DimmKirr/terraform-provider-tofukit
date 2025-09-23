
locals {
  project_name = "python-click-cli"
  project_version = "1.0.0"
}

resource "tofukit_project" "click_cli" {
  name        = local.project_name
  description = "Deterministic Python CLI application using Click framework - TEST"
  version     = local.project_version

  # Aggregate all kits with proper references
  kits = {
    # Language Kit
    "language.python" = {
      id          = "language.python"
      type        = "language"
      name        = tofukit_language.python.name
      description = tofukit_language.python.description
      version     = tofukit_language.python.version
      requirements = tofukit_language.python.requirement
    }

    # Tool Kits
    "tool.pip" = {
      id          = "tool.pip"
      type        = "tool"
      name        = tofukit_tool.pip.name
      description = tofukit_tool.pip.description
      version     = tofukit_tool.pip.version
      requirements = tofukit_tool.pip.requirement
    }

    # Framework Kits
    "framework.click" = {
      id          = "framework.click"
      type        = "framework"
      name        = tofukit_framework.click.name
      description = tofukit_framework.click.description
      version     = tofukit_framework.click.version
      requirements = tofukit_framework.click.requirement
    }
  }

  # Stack reference - removed as not supported in schema

  # Project dependencies
  depends_on = [
    tofukit_language.python,
    tofukit_tool.pip,
    tofukit_tool.venv,
    tofukit_framework.click,
    tofukit_stack.click_app
  ]
}

# ========================================
# APPLICATION STACK - Scaffolding & Files
# ========================================
resource "tofukit_stack" "click_app" {
  name        = "click-application-stack"
  description = "Complete Python Click CLI application with testing and packaging"

  depends_on = [
    tofukit_language.python,
    tofukit_framework.click,
  ]

  # Main CLI Application
  file {
    path = "src/cli.py"
    content = <<-EOF
    #!/usr/bin/env python3
    """
    Simple CLI application using Click framework.
    """
    import click


    @click.group()
    @click.version_option(version='${local.project_version}', prog_name='${local.project_name}')
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

    verification {
      command = "python3 src/cli.py hello"
      expect  = "world 1"
    }

    verification {
      command = "python3 src/cli.py --version"
      expect  = "${local.project_version}"
    }
  }


  # Requirements Files
  file {
    path = "requirements.txt"
    content = <<-EOF
    click==${tofukit_framework.click.version}
    EOF

    verification {
      command = "grep -q 'click==${tofukit_framework.click.version}' requirements.txt"
    }
    verification {
      command = "test -f requirements.txt"
    }
  }


  # demo txt
  file {
    path = "demo.txt"
    content = <<-EOF
    hello world
    EOF

    verification {
      command = "cat demo.txt"
      expect  = "hello world"
    }
  }

  file {
    path = "README.md"
    generate = true
    instructions = [
      "Create readme file typical for this project. Must contain project name",
      "It should contain sections: ## Quickstart, ## Installation, ## Usage, ## Contributing, ## License"
    ]
    verification {
      command = "grep '${local.project_name}' README.md"
      expect  = "${local.project_name}"
    }
    verification {
      command = "grep -c '^## ' README.md"
      expect  = "5"
    }
  }

  # demo txt
  file {
    path = "demo.txt"
    content = <<-EOF
    hello world
    EOF

    verification {
      command = "cat demo.txt"
      expect  = "hello world"
    }
  }
}

