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
# FILE RESOURCES - Implementation Details
# ========================================

resource "tofukit_file" "cmd_status" {
  name = "cmd/status.go"

  instructions = [{
    prompt = "Implement ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} subcommand that displays Go runtime version using ${module.stack_go_viper_cobra_pterm.kits.framework.pterm.link} for formatting"

    constraints = [
      "Define statusCmd variable of type *cobra.Command",
      "Set Use field to 'status'",
      "Set Short description to 'Show system status information'",
      "Use runtime.Version() to get Go version",
      "Use ${module.stack_go_viper_cobra_pterm.kits.framework.pterm.link} Info() or DefaultBox.Println() for colored output",
      "Command should have no arguments or flags"
    ]
  }]

  verifications = [{
    command = "go build -o bin/${local.project_name} . && ./bin/${local.project_name} status 2>&1 | grep -iq 'go.*version' && echo 'OK' || echo 'FAIL'"
    expect  = "OK"
  }]
}

resource "tofukit_file" "cmd_ping" {
  name = "cmd/ping.go"

  instructions = [{
    prompt = "Implement ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} subcommand for continuous HTTP endpoint monitoring with response time tracking"

    constraints = [
      "Define pingCmd variable of type *cobra.Command",
      "Set Use field to 'ping <url>'",
      "Set Short description to 'Ping HTTP endpoints continuously'",
      "Accept URL as first positional argument via Args[0]",
      "Validate at least 1 argument is provided",
      "Use net/http.Get() for HTTP GET requests",
      "Loop indefinitely with time.Sleep(1 * time.Second) between requests",
      "For each request, print line: timestamp statusCode responseTime",
      "Use time.Now().Format(time.RFC3339) for timestamp",
      "Calculate response time in milliseconds using time.Since()",
      "Handle HTTP errors gracefully - print error but continue loop",
      "Use ${module.stack_go_viper_cobra_pterm.kits.framework.pterm.link} for colored status code output (green for 2xx, red for errors)"
    ]
  }]

  verifications = [
    {
      command = "go build -o bin/${local.project_name} . && echo 'build_ok'"
      expect  = "build_ok"
    },
    {
      command = "timeout 3s ./bin/${local.project_name} ping https://google.com 2>&1 | grep -qE '[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.*[0-9]{3}.*[0-9]+ms' && echo 'OK' || echo 'FAIL'"
      expect  = "OK"
    }
  ]
}

# ========================================
# FEATURE RESOURCES - Business Features
# ========================================

resource "tofukit_feature" "status_command" {
  name        = "status-command"
  description = "System status reporting feature that displays runtime environment information to help diagnose issues"

  # Link to implementation file
  files = {
    "cmd/status.go" = tofukit_file.cmd_status
  }

  # Feature-level requirement: integration with root command
  requirements = [{
    name = "Status Command Integration"

    instructions = [{
      prompt = "Register the status command with the ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} root CLI command by adding rootCmd.AddCommand(statusCmd) in the init() function of cmd/root.go"
    }]

    verifications = [{
      command = "grep -q 'AddCommand.*statusCmd' cmd/root.go && echo 'registered' || echo 'missing'"
      expect  = "registered"
    }]
  }]
}

resource "tofukit_feature" "http_ping" {
  name        = "http-ping"
  description = "HTTP endpoint monitoring feature for continuously pinging URLs and tracking response times and availability"

  # Link to implementation file
  files = {
    "cmd/ping.go" = tofukit_file.cmd_ping
  }

  # Feature-level requirement: integration with root command
  requirements = [{
    name = "Ping Command Integration"

    instructions = [{
      prompt = "Register the ping command with the ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} root CLI command by adding rootCmd.AddCommand(pingCmd) in the init() function of cmd/root.go"
    }]

    verifications = [{
      command = "grep -q 'AddCommand.*pingCmd' cmd/root.go && echo 'registered' || echo 'missing'"
      expect  = "registered"
    }]
  }]
}

# ========================================
# PROJECT RESOURCE - Composes Features
# ========================================

resource "tofukit_project" "this" {
  name        = local.project_name
  description = local.project_description
  version     = local.project_version

  # Use Haiku model for fast, cheap execution
  model = "haiku"

  # Reference the go_viper_cobra_pterm stack
  stack = module.stack_go_viper_cobra_pterm.stack

  # Project composes business features
  features = {
    "status_command" = tofukit_feature.status_command
    "http_ping"      = tofukit_feature.http_ping
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
  claude_home_directory = "~/.claude"
  debug                 = true
}
