# Inline feature basic test configuration
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
  dry_run     = true  # Skip LLM execution, just generate prompt
}

resource "tofukit_project" "test" {
  name    = "inline-feature-test"
  version = "0.1.0"

  features = {
    "hello" = {
      requirements = [
        {
          name = "Feature Implementation"
          instructions = [
            {
              prompt = "Add hello greeting capability"
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
