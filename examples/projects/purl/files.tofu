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
