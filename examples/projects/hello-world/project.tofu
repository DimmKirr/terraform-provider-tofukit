resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  # Simple files - no dependencies needed
  files = {
    "hello.txt" = {
      content = "hello world\n"
    }

    "hello2.txt" = {
      content = "hello world2\n"
    }

    "demo/hello4.txt" = {
      content = "hello world4\n"
    }

#     "hello3.txt" = {
#       content = "hello world3\n"
#     }
  }



}


# Terraform configuration for Click CLI example
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration
provider "tofukit" {
  output_format         = "json"
  output_path           = ".tofukit"
  claude_home_directory = "~/.claude"
}

