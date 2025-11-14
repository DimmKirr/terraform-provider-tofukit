# Project inline file verification test configuration (dry run for prompt generation)
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
}

resource "tofukit_project" "test" {
  name    = "verification-test"
  version = "1.0.0"

  files = {
    "hello.txt" = {
      content = "hello world"

      verifications = [{
        command = "cat hello.txt"
        expect  = "hello world"
      }]
    }
  }
}
