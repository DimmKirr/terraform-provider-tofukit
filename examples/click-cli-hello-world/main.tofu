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
  debug                 = true   # Enable debug mode to see project JSON
  claude_home_directory = "~/.claude"
}

