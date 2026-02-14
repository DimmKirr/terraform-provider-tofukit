# Feature precedence test configuration
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
  dry_run     = true
}

resource "tofukit_project" "test" {
  name    = "precedence-test"
  version = "0.1.0"

  features = {
    "base_config" = {
      requirements = [
        {
          name = "Base Configuration"
          instructions = [
            {
              prompt = "Provide base configuration"
            }
          ]
        }
      ]
      files = {
        "config.txt" = {
          content = "Config from feature\n"
        }
      }
    }
  }

  files = {
    "config.txt" = {
      content = "Config from project\n"
    }
  }
}
