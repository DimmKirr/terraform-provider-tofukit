# Python Click Stack Tools
# Tool definitions for the Python Click stack

# ========================================
# TOOL KITS
# ========================================
resource "tofukit_tool" "uv" {
  name        = "uv"
  description = "Fast Python package installer and resolver written in Rust"
  version     = var.uv_version

  depends_on = [tofukit_language.python]

  requirements = [
    {
      name = "uv Installation"

      instructions = [
        {
          prompt = <<-EOF
          Verify uv v${var.uv_version}+ is installed. If not installed, uv must be installed:

          Installation options:
          - macOS/Linux: curl -LsSf https://astral.sh/uv/install.sh | sh
          - Windows: powershell -c "irm https://astral.sh/uv/install.ps1 | iex"
          - Homebrew: brew install uv
          - pipx: pipx install uv

          Verify installation with: uv --version
          EOF

          constraints = [
            "uv binary must be available in system PATH",
            "Version must be 0.4.0 or higher"
          ]
        }
      ]

      verifications = [
        {
          command = "uv --version"
          expect  = "uv ${var.uv_version}"
        },
        {
          command = "which uv && echo 'uv found in PATH'"
          expect  = "uv found in PATH"
        }
      ]
    },
    {
      name = "uv Project Initialization"

      instructions = [
        {
          prompt = "Initialize Python project with uv: uv init --no-readme --python ${var.python_version}"
          constraints = [
            "Do NOT create README (will be provided by feature)",
            "Use specified Python version",
            "Initialize in current directory"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f pyproject.toml && echo 'pyproject.toml exists'"
          expect  = "pyproject.toml exists"
        }
      ]
    }
  ]
}
