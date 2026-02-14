# Project inline file create test configuration (dry run for prompt generation)
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
  debug       = true # Writes prompt JSON to .debug/
  dry_run     = true # Skips LLM execution
}

resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  files = {
    "hello.txt" = {
      content = "hello world\n"
    }
    "hello2.txt" = {
      content = "hello world2\n"
    }
    "hello3.txt" = {
      content = "hello world3\n"
    }
  }
}
