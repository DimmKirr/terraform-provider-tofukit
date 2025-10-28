# Python Click CLI Stack Module
# This module provides a complete Python Click CLI application stack
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
# - language.tofu    : Language definition (Python)
# - tools.tofu       : Development tools (uv)
# - frameworks.tofu   : Framework (Click)
# - features.tofu    : Reusable features (hello command, readme)
# - stack.tofu       : Stack definition (click_app)
# - variables.tofu   : Input variables
# - outputs.tofu     : Output values
