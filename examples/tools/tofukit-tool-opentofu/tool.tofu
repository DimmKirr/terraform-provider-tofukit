terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = ">= 0.1.0"
    }
  }
}

resource "tofukit_tool" "opentofu" {
  name        = "opentofu"
  description = "OpenTofu infrastructure-as-code tool for provisioning and managing cloud resources"
  version     = "latest"

  requirements = [
    {
      name = "OpenTofu Installation"

      instructions = [
        {
          prompt = <<-EOF
          Verify OpenTofu CLI is installed. If not installed, provide installation instructions:

          Installation options:

          Linux:
          1. Install via package manager:
             - Arch: sudo pacman -S opentofu
             - Homebrew: brew install opentofu
          2. Or download from https://opentofu.org/docs/intro/install/
          3. Verify: tofu --version

          macOS:
          1. Homebrew: brew install opentofu
          2. Or download from https://opentofu.org/docs/intro/install/
          3. Verify: tofu --version

          Windows:
          1. Chocolatey: choco install opentofu
          2. Or download from https://opentofu.org/docs/intro/install/
          3. Verify: tofu --version
          EOF

          constraints = [
            "OpenTofu binary must be available in system PATH as 'tofu'",
            "Version should be 1.6.0 or higher for full compatibility",
            "Prefer native installation over Docker for better developer experience"
          ]
        }
      ]

      verifications = [
        {
          command = "tofu --version && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "which tofu && echo 'tofu found in PATH'"
          expect  = "tofu found in PATH"
        }
      ]
    }
  ]
}
