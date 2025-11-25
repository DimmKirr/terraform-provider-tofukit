# ========================================
# FILE RESOURCES - Implementation Details
# ========================================

resource "tofukit_file" "src_cli_py" {
  name = "src/cli.py"

  instructions = [{
    prompt = <<-EOT
    Add a status command to the existing ${module.click_app.stack.link} CLI:

    @cli.command()
    def status() -> None:
        """Show the current status."""
        click.echo("Status: OK")

    This should be added after the version command.
    EOT

    constraints = [
      "Do NOT remove existing commands",
      "Add the status command function after the version command",
      "Use ${module.click_app.kits.framework.click.link} click.echo() for output",
      "Include type hint (-> None)"
    ]
  }]

  verifications = [{
    command = "python3 src/cli.py status | grep -q 'Status: OK' && echo 'ok'"
    expect  = "ok"
  }]
}
