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
# PROJECT RESOURCE
# ========================================
resource "tofukit_project" "three_tier_app" {
  name        = "infra-3-tier-app"
  description = "3-tier infrastructure diagram example with load balancer, web server, and database"
  version     = "1.0.0"

  model = "haiku"

  # Define infrastructure features inline
  features = {
    "load_balancer" = {
      prompt = "Configure load balancer infrastructure"

      requirements = [
        {
          name = "Load Balancer Setup"
          instructions = [
            {
              prompt = "Document the load balancer configuration and setup"
            }
          ]
        }
      ]

      files = {
        "load-balancer.md" = {
          instructions = [
            {
              prompt = "Create documentation for the load balancer tier that distributes traffic across web servers"
            }
          ]
        }
      }
    }

    "web_server" = {
      prompt = "Setup web server tier"

      requirements = [
        {
          name = "Web Server Configuration"
          instructions = [
            {
              prompt = "Document the web server setup and configuration"
            }
          ]
        }
      ]

      files = {
        "web-server.md" = {
          instructions = [
            {
              prompt = "Create documentation for the web server tier that handles HTTP requests and serves content"
            }
          ]
        }
      }
    }

    "database" = {
      prompt = "Configure database tier"

      requirements = [
        {
          name = "Database Setup"
          instructions = [
            {
              prompt = "Document the database configuration"
            }
          ]
        }
      ]

      files = {
        "database.md" = {
          instructions = [
            {
              prompt = "Create documentation for the database server (SQL) that stores application data"
            }
          ]
        }
      }
    }

    # Diagram feature auto-discovers all above features!
    "architecture_diagram" = tofukit_feature.architecture_diagram
  }
}
