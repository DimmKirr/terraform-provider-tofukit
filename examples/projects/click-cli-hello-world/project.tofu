# ========================================
# TERRAFORM & PROVIDER CONFIGURATION
# ========================================
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  # output_path defaults to "./" (current directory)
  debug                 = true   # Enable debug mode to see project JSON
  claude_home_directory = "~/.claude"
}

# ========================================
# LOCALS
# ========================================
locals {
  project_name    = "python-click-cli"
  project_version = "1.0.0"
}

# ========================================
# MODULE: Python Click CLI Stack
# ========================================
module "click_app" {
  source = "../stacks/tofukit-stack-python-click-app"

  project_name    = local.project_name
  project_version = local.project_version
}

# ========================================
# PROJECT RESOURCE
# ========================================
resource "tofukit_project" "click_cli" {
  name        = local.project_name
  description = "Deterministic Python CLI application using Click framework - TEST"
  version     = local.project_version

  # Use Haiku model for fast, cheap execution
  model = "haiku"

  # Reference the click_app stack (includes kits, features, and files)
  # The stack provides a complete Python Click CLI application template
  stack = module.click_app.stack

  # Note: Stack includes features, so no separate features attribute needed
  # Project can add additional features here if needed (merged with stack features)

  # Add a status command feature (merges with stack's version and readme features)
  features = {
    "status" = {
      prompt      = "Add a status command that shows 'Status: OK'"
      description = "Status check command for the CLI"
      constraints = [
        "Use click.echo() to output the status",
        "Create a @cli.command() decorated function",
        "Keep it simple - just output 'Status: OK'"
      ]

      files = {
        "src/cli.py" = {
          instructions = [{
            prompt = <<-EOT
              Add a status command to the existing CLI:

              @cli.command()
              def status() -> None:
                  """Show the current status."""
                  click.echo("Status: OK")

              This should be added after the version command.
            EOT
            constraints = [
              "Do NOT remove existing commands",
              "Add the status command function after the version command",
              "Use click.echo() for output",
              "Include type hint (-> None)"
            ]
          }]

          verifications = [{
            command = "python3 src/cli.py status | grep -q 'Status: OK' && echo 'ok'"
            expect  = "ok"
          }]
        }
      }
    }
  }
}

# ========================================
# DATA SOURCES
# ========================================
data "tofukit_query" "ls" {
  instruction {
    prompt = "What color is sky?"
    constraints = [
      "Answer with exactly one word",
      "Use lowercase only",
      "Do NOT include any punctuation or additional text"
    ]
  }
}

# ========================================
# OUTPUTS
# ========================================
output "ls" {
  value = data.tofukit_query.ls
}
