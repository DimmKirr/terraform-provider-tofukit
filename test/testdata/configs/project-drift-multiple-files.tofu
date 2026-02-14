# Project drift detection multiple files test configuration (dry run for prompt generation)
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
  name    = "drift-test"
  version = "1.0.0"

  files = {
    ".gitignore" = {
      content = "*.log\n"
    }
    "LICENSE" = {
      content = "MIT License\n"
    }
    "README.md" = {
      content = "# Project\n"
    }
  }
}
