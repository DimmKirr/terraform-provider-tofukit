resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"

  # Simple files - no dependencies needed
  file {
    path = "hello.txt"
    content = <<-EOF
hello world
EOF
  }

  file {
    path = "hello2.txt"
    content = <<-EOF
hello world2
EOF
  }

  file {
    path = "demo/hello4.txt"
    content = <<-EOF
hello world4
EOF
  }

#   file {
#     path = "hello3.txt"
#     content = <<-EOF
# hello world3
# EOF
#   }

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

