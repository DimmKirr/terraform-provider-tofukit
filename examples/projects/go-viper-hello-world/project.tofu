# ========================================
# LOCALS
# ========================================
locals {
  project_name        = "go-viper-hello-world"
  project_version     = "1.0.0"
  project_module_path = "dimmiirr.com/go-viper-hello-world"
}

# ========================================
# MODULE: Go CLI Stack
# ========================================
module "go_viper_cobra_pterm" {
  source = "../../stacks/tofukit-stack-go-viper-cobra-pterm"

  project_name        = local.project_name
  project_version     = local.project_version
  project_module_path = local.project_module_path

  # Optional: Override default versions
  # go_version     = "1.23.1"
  # cobra_version  = "1.8.1"
  # viper_version  = "1.19.0"
  # pterm_version  = "0.12.80"
}

# ========================================
# PROJECT RESOURCE
# ========================================
resource "tofukit_project" "go_hello" {
  name        = local.project_name
  description = "Simple Go CLI application using Cobra, Viper, and pterm"
  version     = local.project_version

  # Use Haiku model for fast, cheap execution
  model = "haiku"

  # Reference the go_cli stack (includes Go, Cobra, Viper, pterm, app files, and README)
  # The stack provides a complete Go CLI application template
  # Stack contains: kits, features, and files - everything needed for the project
  stack = module.go_viper_cobra_pterm.stack

  # Project-specific features (reference standalone feature resources)
  features = {
    "hello_command" = tofukit_feature.hello_command
  }

  # Note: Terraform automatically creates dependencies from resource references
  # (stack reference creates implicit dependency on the module)
}

# ========================================
# OUTPUTS
# ========================================
output "project_info" {
  value = {
    name    = tofukit_project.go_hello.name
    version = tofukit_project.go_hello.version
    path    = tofukit_project.go_hello.project_path
  }
}



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
  # output_path defaults to "./" (current directory)
  debug                 = true   # Enable debug mode to see project JSON
  claude_home_directory = "~/.claude"
}
