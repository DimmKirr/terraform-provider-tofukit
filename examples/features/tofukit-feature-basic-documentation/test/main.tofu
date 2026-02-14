# Example: Using the Basic Documentation Feature Module
# This file demonstrates how to consume the basic-documentation feature in a project

terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format         = "json"
  debug                 = true
  claude_home_directory = "~/.claude"
  output_path           = "output"
}

# ========================================
# USE THE DOCUMENTATION FEATURE MODULE
# ========================================
module "documentation" {
  source = "../" # Points to the parent directory feature module

  project_name        = "my-awesome-app"
  project_description = "A CLI application that does awesome things"

  features = {
    "authentication" = "User authentication with OAuth2 support"
    "data-export"    = "Export data to CSV and JSON formats"
    "logging"        = "Structured logging with multiple output backends"
    "api-client"     = "REST API client with retry logic"
  }
}

# ========================================
# CREATE A PROJECT USING THE FEATURE
# ========================================
resource "tofukit_project" "example" {
  name        = "example-with-docs"
  description = "Example project using the basic documentation feature"
  version     = "1.0.0"

  # Use the documentation feature from the module
  features = {
    "documentation" = module.documentation.feature
  }

  # Additional project requirements
  requirements = [
    {
      name = "Main Application"
      instructions = [
        {
          prompt = "Create a simple main application file"
        }
      ]
    }
  ]

  # Additional files
  files = {
    "main.py" = {
      instructions = [
        {
          prompt = "Create a simple main.py with a hello world function"
        }
      ]
    }
  }
}

# ========================================
# OUTPUTS
# ========================================
output "documentation_feature_link" {
  description = "URI link to the documentation feature"
  value       = module.documentation.feature_link
}
