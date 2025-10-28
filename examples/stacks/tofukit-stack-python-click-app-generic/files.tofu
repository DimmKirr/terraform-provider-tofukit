# File Resources for Python Click CLI Stack
# These reusable file resources can be referenced by projects using this stack

# Python CLI application entry point
resource "tofukit_file" "python_click_cli" {
  name        = "python-click-cli-main"
  description = "Main CLI application file using Click framework"

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
      command = "python3 src/cli.py --version 2>&1 | grep -o '${var.project_version}'"
      expect  = var.project_version
    }
  ]
}

# Python requirements file
resource "tofukit_file" "python_requirements" {
  name        = "python-click-requirements"
  description = "Python dependencies file for Click CLI"

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

# Demo file for testing
resource "tofukit_file" "demo_file" {
  name        = "demo-hello-world"
  description = "Simple demo file with hello world content"

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

# README documentation (instructions-based generation)
resource "tofukit_file" "readme_docs" {
  name        = "python-click-readme"
  description = "Comprehensive README documentation for Python Click CLI project"

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
      command = "grep -q '${var.project_name}' README.md && echo 'found'"
      expect  = "found"
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
