# Test configuration for validating kit dependency requirements are included in prompt

locals {
  project_name        = "kit-dependency-test"
  project_version     = "1.0.0"
  project_module_path = "github.com/test/kit-dependency-test"
}

# Use the full stack with all kits
module "stack" {
  source = "../../../examples/stacks/tofukit-stack-go-viper-cobra-pterm"

  project_name        = local.project_name
  project_version     = local.project_version
  project_module_path = local.project_module_path
}

# Minimal project that uses the stack
resource "tofukit_project" "test" {
  name    = local.project_name
  version = local.project_version
  model   = "haiku"

  stack = module.stack.stack

  # Add one project-level file to test precedence
  files = {
    "test.txt" = {
      content = "project-level file\n"
    }
  }
}

terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  debug = true
}
