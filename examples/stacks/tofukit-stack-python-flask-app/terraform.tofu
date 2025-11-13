# ========================================
# TERRAFORM CONFIGURATION FOR STACK MODULE
# ========================================
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}
