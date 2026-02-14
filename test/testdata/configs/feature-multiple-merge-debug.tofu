# Multiple features merge test configuration (actual execution with debug)
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
}

resource "tofukit_project" "test" {
  name    = "multi-feature-test"
  version = "0.1.0"

  features = {
    "logging" = {
      requirements = [
        {
          name = "Logging Capability"
          instructions = [
            {
              prompt = "Add logging capability"
            }
          ]
        }
      ]
      files = {
        "log.txt" = {
          content = "Logging enabled\n"
        }
      }
    }
    "config" = {
      requirements = [
        {
          name = "Configuration Capability"
          instructions = [
            {
              prompt = "Add configuration capability"
            }
          ]
        }
      ]
      files = {
        "config.txt" = {
          content = "Config loaded\n"
        }
      }
    }
    "metrics" = {
      requirements = [
        {
          name = "Metrics Capability"
          instructions = [
            {
              prompt = "Add metrics capability"
            }
          ]
        }
      ]
      files = {
        "metrics.txt" = {
          content = "Metrics tracking\n"
        }
      }
    }
  }
}
