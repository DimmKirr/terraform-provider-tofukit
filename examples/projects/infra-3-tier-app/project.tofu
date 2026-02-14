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
  debug                 = true
  claude_home_directory = "~/.claude"
}


# ========================================
# DIAGRAM MODULE
# ========================================
module "architecture_diagram" {
  source = "../../features/tofukit-feature-diagram-drawio"

  file_path = "architecture"
  style     = "AWS"
}


# ========================================
# TOOL MODULES
# ========================================
module "tool_opentofu" {
  source = "../../tools/tofukit-tool-opentofu"
}


# ========================================
# INFRASTRUCTURE FEATURES
# ========================================
resource "tofukit_feature" "infra_app_server" {
  name = "App Server"
  requirements = [
    {
      name = "Provision App Server"
      instructions = [{
        prompt = "Create app server infrastructure using ${module.tool_opentofu.link}"
      }]
    },
    {
      name = "Link DB"
      instructions = [{
        prompt = "Add a tag to the DB containing db host of ${tofukit_feature.infra_database.link}"
      }]
    }
  ]
}

resource "tofukit_feature" "infra_load_balancer" {
  name = "Load Balancer"
  requirements = [
    {
      name = "Create an ALB"
      instructions = [{
        prompt = "Create an ALB using ${module.tool_opentofu.link}"
      }]
    },
    {
      name = "Add App Host"
      instructions = [{
        prompt = "Add ${tofukit_feature.infra_app_server.link} to a target group"
      }]
    }
  ]
}

resource "tofukit_feature" "infra_database" {
  name = "Database"
  requirements = [
    {
      name = "Create an DB"
      instructions = [{
        prompt = "Create an rds serverless v2 using latest community module ALB using terraform-aws-modules/rds/aws latest module"
      }]
    }
  ]
}

# ========================================
# PROJECT RESOURCE
# ========================================
resource "tofukit_project" "three_tier_app" {
  name        = "infra-3-tier-app"
  description = "3-tier infrastructure diagram example with load balancer, web server, and database"
  version     = "1.0.0"

  model = "haiku"



  # Define infrastructure features inline
  features = {
    "load_balancer" = tofukit_feature.infra_load_balancer
    "web_server" = tofukit_feature.infra_app_server
    "database" = tofukit_feature.infra_database

    # Diagram feature auto-discovers all above features!
    "architecture_diagram" = module.architecture_diagram.feature
  }
}
