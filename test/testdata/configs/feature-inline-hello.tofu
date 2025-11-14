# Inline feature test configuration (actual execution with debug)
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
  name    = "inline-feature-test"
  version = "0.1.0"

  features = {
    "hello" = {
      requirements = [
        {
          name = "Hello File"
          instructions = [
            {
              prompt = "Create a hello.txt file with greeting"
            }
          ]
        }
      ]
      files = {
        "hello.txt" = {
          content = "Hello from inline feature.\n"
        }
      }
    }
  }
}
