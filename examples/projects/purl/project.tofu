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
  project_description = "Universal ping utility supporting HTTP, ICMP, and DNS protocols."
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

  requirements = [
    {
      name = "Feature Integration"

      instructions = [{
        prompt = <<-EOF
        Ensure all features are properly integrated into a working application.

        INTEGRATION CHECKLIST:
        - ${tofukit_feature.protocol_abstraction.link} provides the pinger interface
        - ${tofukit_feature.protocol_detection.link} uses that interface for detection
        - ${tofukit_feature.http_ping.link}, ${tofukit_feature.icmp_ping.link}, ${tofukit_feature.dns_ping.link} implement the interface
        - ${tofukit_feature.user_config.link} provides timeout/interval configuration
        - cmd/root.go imports internal/pinger package and uses all components

        INTEGRATION POINTS:
        - cmd/root.go must accept target as Args[0] using cobra.ExactArgs(1)
        - cmd/root.go must read protocol flags (--http, --icmp, --dns, --auto)
        - cmd/root.go must call pinger.DetectProtocol() when --auto is set
        - cmd/root.go must create pinger using pinger.NewPinger()
        - cmd/root.go must implement continuous ping loop with configured interval
        - cmd/root.go must use pterm for colored output and time.RFC3339 for timestamps

        VERIFICATION:
        - Application must build successfully
        - Application must accept target argument
        - Protocol flags must appear in --help output
        EOF

        constraints = [
          "All feature files must be imported where needed",
          "cmd/root.go MUST import ${local.project_module_path}/internal/pinger",
          "Application must be runnable as: ${local.project_name} <target>",
          "Use cobra.ExactArgs(1) for target argument validation"
        ]
      }]

      verifications = [
        {
          command = "grep -q 'internal/pinger' cmd/root.go"
        },
        {
          command = "grep -q 'cobra\\.ExactArgs(1)' cmd/root.go"
        },
        {
          command = "grep -q 'Args.*\\[0\\]' cmd/root.go"
        }
      ]
    }
  ]


  # Project-specific features
  # Note: Stack features (version_command, viper_config, pterm_logger, taskfile, gitignore, readme)
  # are already included via the stack reference above
  features = {
    # Core protocol abstraction and detection
    "protocol_abstraction" = tofukit_feature.protocol_abstraction
    "protocol_detection"   = tofukit_feature.protocol_detection
    "user_config"          = tofukit_feature.user_config

    # Protocol implementations
    "http_ping" = tofukit_feature.http_ping
    "icmp_ping" = tofukit_feature.icmp_ping
    "dns_ping"  = tofukit_feature.dns_ping
  }
}
