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
# FILE RESOURCES - Implementation Details
# ========================================

# Cobra root command extended with HTTP ping functionality
resource "tofukit_file" "cmd_root" {
  name        = "cmd/root.go"
  description = "Cobra root command extended with HTTP ping functionality"

  instructions = [
    # INSTRUCTION 1: Base Structure
    {
      prompt = <<-EOF
      Extend the Cobra root command structure from the stack.

      BASE STRUCTURE:
      - Start with ${module.stack_go_viper_cobra_pterm.files.cobra_root_cmd.link}
      - Keep all standard elements:
        * Package declaration: package cmd
        * Imports: fmt, os, internal/config, internal/logger, cobra
        * var rootCmd structure
        * Execute() function
        * init() function with cobra.OnInitialize
        * Standard flags: --config, --log-level, --quiet
      EOF

      constraints = [
        "Keep all standard flags and initialization from ${module.stack_go_viper_cobra_pterm.files.cobra_root_cmd.link}",
        "Don't remove or modify existing initialization logic"
      ]
    },

    # INSTRUCTION 2: Command Arguments
    {
      prompt = <<-EOF
      Configure rootCmd to accept URL argument.

      IMPLEMENTATION:
      - Add Args field to rootCmd: Args: cobra.ExactArgs(1)
      - Update Use field: Use: "${local.project_name} [url]"
      - Update Short description to reflect HTTP ping purpose
      EOF

      constraints = [
        "Use cobra.ExactArgs(1) for argument validation",
        "URL is args[0] in RunE function"
      ]
    },

    # INSTRUCTION 3: HTTP Request Loop
    {
      prompt = <<-EOF
      Implement the HTTP request loop in RunE function.

      IMPLEMENTATION STEPS:
      1. Get URL from args[0]
      2. Create HTTP client with 10-second timeout
      3. Enter infinite for loop:
         a. Record start time: start := time.Now()
         b. Make HTTP GET request
         c. Calculate duration: duration := time.Since(start)
         d. Handle response or error
         e. Display output
         f. Sleep for 1 second

      HTTP CLIENT CONFIGURATION:
      - Use net/http.Client with 10-second timeout
      - Always close response bodies: defer resp.Body.Close()
      - Continue loop on errors - don't exit
      EOF

      constraints = [
        "Use &http.Client{Timeout: 10 * time.Second}",
        "Use time.Sleep(1 * time.Second) for interval",
        "Close response bodies to prevent resource leaks",
        "Handle both connection and HTTP-level errors",
        "Continue execution for transient failures"
      ]
    },

    # INSTRUCTION 4: Output Formatting
    {
      prompt = <<-EOF
      Format and display ping results.

      IMPLEMENTATION:
      - For successful requests:
        * Format timestamp: time.Now().UTC().Format(time.RFC3339)
        * Get status code: resp.StatusCode
        * Get duration in ms: duration.Milliseconds()
        * Display with pterm.Success.Printf()

      - For failed requests:
        * Format timestamp: time.Now().UTC().Format(time.RFC3339)
        * Display error with pterm.Error.Printf()

      OUTPUT FORMAT: "timestamp statuscode durationms"
      Example: 2025-11-01T12:34:56Z 200 45ms

      PTERM USAGE:
      - Success messages (2xx status): pterm.Success.Printf()
      - Error messages (non-2xx or connection failures): pterm.Error.Printf()
      EOF

      constraints = [
        "Use time.RFC3339 format constant for timestamps",
        "Convert to UTC before formatting: time.Now().UTC()",
        "Convert duration to milliseconds: duration.Milliseconds()",
        "Use pterm.Success for 2xx status codes",
        "Use pterm.Error for errors and non-2xx status codes"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q '^package cmd' cmd/root.go"
    },
    {
      command = "grep -q 'cobra\\.ExactArgs(1)' cmd/root.go"
    },
    {
      command = "grep -q 'net/http' cmd/root.go"
    },
    {
      command = "grep -q 'time\\.Sleep' cmd/root.go"
    },
    {
      command = "grep -q 'StatusCode' cmd/root.go"
    },
    {
      command = "grep -q 'func Execute()' cmd/root.go"
    },
    {
      command = "grep -q 'pterm\\.(Success|Error)' cmd/root.go"
    }
  ]
}

# ========================================
# FEATURE RESOURCES - Business Features
# ========================================

# HTTP ping feature with focused requirements
resource "tofukit_feature" "http_ping" {
  name        = "http-ping"
  description = "HTTP ping utility for monitoring URL availability"

  files = {
    "cmd/root.go" = tofukit_file.cmd_root
  }

  requirements = [
    # REQUIREMENT 1: Command Line Interface
    {
      name = "Command Line Interface"

      instructions = [
        {
          prompt = <<-EOF
          Define the primary command interface for HTTP pinging.

          BUSINESS SPECIFICATION:
          - Command usage: `${local.project_name} <URL>`
          - URL is a required positional argument
          - Accept exactly one URL argument

          Example: ${local.project_name} http://google.com
          EOF

          constraints = [
            "URL must be a positional argument (not a flag)",
            "Exactly one URL required",
            "Use ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} with Args: cobra.ExactArgs(1)"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'cobra\\.ExactArgs(1)' cmd/root.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "test -f cmd/root.go && echo 'exists'"
          expect  = "exists"
        }
      ]
    },

    # REQUIREMENT 2: Ping Execution Behavior
    {
      name = "Ping Execution Behavior"

      instructions = [
        {
          prompt = <<-EOF
          Implement continuous HTTP request behavior.

          BUSINESS SPECIFICATION:
          - Issue HTTP GET requests in a loop to the provided URL
          - Request interval: 1 second between requests (default)
          - Execution mode: Run continuously until user interrupts (Ctrl+C)
          - Resilience: Continue running even if individual requests fail

          The tool should be resilient and keep pinging even when the server is temporarily unavailable.
          EOF

          constraints = [
            "Default interval: 1 second between requests",
            "Run in infinite loop until interrupt",
            "Don't exit on individual request failures",
            "Use ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} command structure"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'net/http' cmd/root.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'time\\.Sleep' cmd/root.go && echo 'found'"
          expect  = "found"
        }
      ]
    },

    # REQUIREMENT 3: Output Format
    {
      name = "Output Format"

      instructions = [
        {
          prompt = <<-EOF
          Display ping results in a structured format.

          BUSINESS SPECIFICATION:
          For each request, display a single line with:
          1. ISO 8601 timestamp (e.g., 2025-11-01T12:34:56Z)
          2. HTTP status code (e.g., 200, 404, 500)
          3. Response time in milliseconds (e.g., 45ms)

          EXPECTED OUTPUT EXAMPLE:
          2025-11-01T12:34:56Z 200 45ms
          2025-11-01T12:34:57Z 200 42ms
          2025-11-01T12:34:58Z 500 150ms

          Color coding:
          - Success responses (2xx): green
          - Error responses (4xx, 5xx): red
          - Connection failures: red
          EOF

          constraints = [
            "Use ISO 8601 (RFC3339) timestamp format",
            "Display status code and response time in milliseconds",
            "Use ${module.stack_go_viper_cobra_pterm.kits.framework.pterm.link} for colored output"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'StatusCode' cmd/root.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "go build -o ${local.project_name} && timeout 3s ./${local.project_name} https://google.com 2>&1 | grep -qE '[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z [0-9]{3} [0-9]+ms' && echo 'OK' || echo 'FAILED'"
          expect  = "OK"
        }
      ]
    },

    # REQUIREMENT 4: Stack Integration
    {
      name = "Stack Integration"

      instructions = [
        {
          prompt = <<-EOF
          Integrate with the stack's base command structure.

          TECHNICAL INTEGRATION:
          - This extends ${module.stack_go_viper_cobra_pterm.files.cobra_root_cmd.link}
          - Use ${module.stack_go_viper_cobra_pterm.kits.framework.cobra.link} for command structure
          - Use ${module.stack_go_viper_cobra_pterm.kits.framework.pterm.link} for colored output
          - Keep all standard flags and initialization from the stack
          EOF

          constraints = [
            "Don't replace stack's rootCmd - extend it",
            "Preserve standard flags: --config, --log-level, --quiet",
            "Maintain init() function and cobra.OnInitialize"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'func Execute()' cmd/root.go && echo 'found'"
          expect  = "found"
        }
      ]
    }
  ]

  kits = [
    module.stack_go_viper_cobra_pterm.kits.framework.cobra,
    module.stack_go_viper_cobra_pterm.kits.framework.pterm
  ]

  verifications = []
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
