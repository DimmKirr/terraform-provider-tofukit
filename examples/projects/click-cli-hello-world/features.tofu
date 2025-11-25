# ========================================
# PROJECT FEATURES
# ========================================

resource "tofukit_feature" "status_command" {
  name        = "status-command"
  description = "Status command to demonstrate CLI functionality"

  # Link to implementation file
  files = {
    "src/cli.py" = tofukit_file.src_cli_py
  }

  # Feature-level requirement
  requirements = [{
    name = "Status Command Implementation"

    instructions = [{
      prompt = "Add a status command that shows 'Status: OK' using ${module.click_app.kits.framework.click.link} framework"

      constraints = [
        "Use ${module.click_app.kits.framework.click.link} click.echo() to output the status",
        "Create a @cli.command() decorated function",
        "Keep it simple - just output 'Status: OK'"
      ]
    }]
  }]
}
