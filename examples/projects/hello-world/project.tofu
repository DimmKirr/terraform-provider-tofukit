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

    "linked.txt" = {
      content = "More info in ${tofukit_file.target_file.link}"
    }

#     "hello3.txt" = {
#       content = "hello world3\n"
#     }
  }
}

resource "tofukit_file" "target_file" {
  name = "target_file.txt"
  content = "hello world linked\n"
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
  debug                 = true  # Enable debug output to see URI resolution
  output_format         = "json"
  output_path           = ".tofukit"
  claude_home_directory = "~/.claude"
}

