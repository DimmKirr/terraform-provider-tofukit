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
    name = "Status Command"

    instructions = [{
      prompt = <<-EOF
      Implement a "status" command that displays system status.

      BUSINESS REQUIREMENTS:
      - Command: status
      - Behavior: Show current system status
      - Output: "Status: OK" message
      - Integration: Must integrate with ${module.click_app.kits.framework.click.link} CLI framework

      IMPLEMENTATION LOCATION:
      This feature must be implemented in ${tofukit_file.src_cli_py.link}

      COMMAND SPECIFICATION:
      - Function name: status
      - Decorator: @cli.command()
      - Docstring: "Show the current status."
      - Type hint: -> None
      - Output: "Status: OK" (using click.echo)
      - Placement: After the version command provided by ${module.click_app.stack.link}
      EOF

      constraints = [
        "Use ${module.click_app.kits.framework.click.link} click.echo() to output the status",
        "Create a @cli.command() decorated function",
        "Keep it simple - just output 'Status: OK'",
        "Do NOT remove existing commands from ${module.click_app.stack.link}"
      ]
    }]
  }]
}
