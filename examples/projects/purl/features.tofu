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
