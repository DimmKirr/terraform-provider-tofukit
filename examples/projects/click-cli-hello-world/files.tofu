# ========================================
# FILE RESOURCES - Implementation Details
# ========================================

resource "tofukit_file" "src_cli_py" {
  name = "src/cli.py"

  instructions = [{
    prompt = <<-EOT
    Implement ${module.click_app.kits.framework.click.link} command following Python CLI best practices.

    TECHNICAL CONSTRAINTS:
    - Use ${module.click_app.kits.framework.click.link} @cli.command() decorator
    - Use click.echo() for output (not print())
    - Include type hints (-> None)
    - Do NOT remove existing commands from ${module.click_app.stack.link}
    - Follow Python community conventions for CLI command structure
    - Place new command after existing commands
    EOT

    constraints = [
      "Do NOT remove existing commands",
      "Use ${module.click_app.kits.framework.click.link} click.echo() for output",
      "Include type hint (-> None)",
      "Follow Python CLI community conventions"
    ]
  }]

  verifications = [{
    command = "python3 src/cli.py status | grep -q 'Status: OK' && echo 'ok'"
    expect  = "ok"
  }]
}
