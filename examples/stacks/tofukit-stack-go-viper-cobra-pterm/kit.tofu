# Go CLI Stack Module (Cobra + Viper + pterm)
# This module provides a complete Go CLI application stack
# with all necessary kits and file scaffolding

terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# ========================================
# STACK ORGANIZATION
# ========================================
# This stack is organized into modular files:
#
# - language.tofu    : Language definition (Go)
# - tools.tofu       : Development tools (go-mod)
# - frameworks.tofu   : Frameworks (Cobra, Viper, pterm)
# - features.tofu    : Reusable features (config, version, logger, etc.)
# - stack.tofu       : Stack definition (go_cli_app)
# - variables.tofu   : Input variables
# - outputs.tofu     : Output values
