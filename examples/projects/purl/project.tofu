# ========================================
# TERRAFORM & PROVIDER CONFIGURATION
# ========================================
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
  claude_home_directory = "~/.claude"
  debug                 = true
}

# ========================================
# LOCALS
# ========================================
locals {
  project_name        = "purl"
  project_version     = "0.1.0"
  project_description = "A tool that combines multi-protocol ping utils."
  project_module_path = "github.com/DimmKirr/purl"
}

# ========================================
# MODULE: Go CLI Stack
# ========================================
module "stack_go_viper_cobra_pterm" {
  source = "../../stacks/tofukit-stack-go-viper-cobra-pterm"

  project_name        = local.project_name
  project_version     = local.project_version
  project_module_path = local.project_module_path
}

# ========================================
# PROJECT RESOURCE
# ========================================
resource "tofukit_project" "this" {
  name        = local.project_name
  description = local.project_description
  version     = local.project_version

  # Use Haiku model for fast, cheap execution
  model = "haiku"

  # Reference the go_viper_cobra_pterm stack
  stack = module.stack_go_viper_cobra_pterm.stack

  # Project-specific features
  # Note: Stack features (version_command, viper_config, pterm_logger, taskfile, gitignore, readme)
  # are already included via the stack reference above
  features = {
    "http_ping" = tofukit_feature.http_ping
  }
}
