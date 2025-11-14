# Project inline file ordering consistency test configuration (dry run for prompt generation)
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
  name    = "file-ordering-test"
  version = "1.0.0"

  files = {
    "hi2.txt" = {
      content = "Hi from file 2\n"
    }
    "LICENSE.md" = {
      content = "MIT License\n"
    }
    "README.md" = {
      content = "Project README\n"
    }
  }
}
