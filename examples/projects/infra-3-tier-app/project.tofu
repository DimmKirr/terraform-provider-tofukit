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
# DIAGRAM TOOL DEPENDENCY
# ========================================
resource "tofukit_tool" "drawio" {
  name        = "drawio"
  description = "Draw.io desktop CLI for diagram generation and export"
  version     = "latest"

  requirements = [
    {
      name = "Draw.io CLI Installation"

      instructions = [
        {
          prompt = <<-EOF
          Verify draw.io CLI is installed. If not installed, provide installation instructions:

          Installation options:

          Linux:
          1. Download from https://github.com/jgraph/drawio-desktop/releases
          2. Install package:
             - Debian/Ubuntu: sudo dpkg -i drawio-amd64-<version>.deb
             - RPM-based: sudo rpm -i drawio-<version>.rpm
          3. Verify: drawio --version

          macOS:
          1. Homebrew: brew install --cask drawio
          2. Or download from https://github.com/jgraph/drawio-desktop/releases
          3. Verify: drawio --version
          EOF

          constraints = [
            "drawio binary must be available in system PATH",
            "CLI must support -x (export), -f (format), and -o (output) flags",
            "Prefer native installation over Docker for better performance"
          ]
        }
      ]

      verifications = [
        {
          command = "drawio --version && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "which drawio && echo 'drawio found in PATH'"
          expect  = "drawio found in PATH"
        }
      ]
    }
  ]
}


# ========================================
# DIAGRAM GENERATION FEATURE
# ========================================
resource "tofukit_feature" "architecture_diagram" {
  name        = "architecture-diagram"
  description = "Auto-generated architecture diagram via introspection"

  requirements = [
    {
      name = "Generate Architecture Diagram"

      instructions = [
        {
          prompt = <<-EOF
          Generate architecture diagram by introspecting project_context.

          Show all features, integrations, and infrastructure components
          discovered in the project context.
          EOF
        }
      ]
    }
  ]

  files = {
    "architecture.drawio" = {
      instructions = [
        {
          prompt = "Generate draw.io diagram from project_context as described in requirements"
        }
      ]

      verifications = [
        {
          command = "test -f architecture.drawio && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "grep -q '<mxGraphModel' architecture.drawio && echo 'OK'"
          expect  = "OK"
        }
      ]
    }
  }

  kits = [tofukit_tool.drawio]
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
    "architecture_diagram" = tofukit_feature.architecture_diagram
  }
}
