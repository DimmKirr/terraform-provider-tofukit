# Project inline file verification retry test configuration (dry run for prompt generation)
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
  dry_run     = true
  max_retries = 3 # Retry setting doesn't affect prompt generation
}

resource "tofukit_project" "test" {
  name    = "verification-retry-test"
  version = "1.0.0"

  files = {
    "greeting.txt" = {
      instructions = [{
        prompt = "Create a text file containing 'hello world'"
        constraints = [
          "Content must be exactly: hello world",
          "Keep it simple"
        ]
      }]
      verifications = [{
        command = "cat greeting.txt"
        expect  = "hello world"
      }]
    }
  }
}
