# Python Click Stack Framework
# Framework definition for the Python Click stack

# ========================================
# FRAMEWORK KITS
# ========================================
resource "tofukit_framework" "click" {
  name        = "click"
  description = "Composable command line interface toolkit"
  version     = var.click_version

  depends_on = [tofukit_tool.uv]

  requirements = [
    {
      name = "CLI Framework Installation"

      instructions = [
        {
          prompt      = "Install Click framework version ${var.click_version} using ${tofukit_tool.uv.link}: uv add click==${var.click_version}"
          constraints = [
            "Do NOT install any version other than ${var.click_version}",
            "Must use ${tofukit_tool.uv.link} for installation",
            "Install from PyPI only"
          ]
        }
      ]

      verifications = [
        {
          command = "uv pip show click | grep Version"
          expect  = "${var.click_version}"
        }
      ]
    },
    {
      name = "Idiomatic Click Usage"

      instructions = [
        {
          prompt = "Use Click idiomatically: @click.command decorators, @click.option for flags, click.echo for output, click.group for command groups"
          constraints = [
            "NO manual argument parsing - use Click decorators",
            "NO print() - use click.echo()",
            "Use click.group() for multi-command CLIs"
          ]
        }
      ]

      verifications = []
    }
  ]
}
