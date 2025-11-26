# ========================================
# COMMON GO CLI PATTERNS (Stack-level)
# ========================================
# These patterns apply to ALL go-viper-cobra-pterm applications
# Projects can reference these patterns via module outputs

locals {
  # HTTP Client Best Practices Pattern
  http_client_pattern = {
    name = "HTTP Client Best Practices"
    prompt = <<-EOF
    When making HTTP requests, use net/http.Client with appropriate configuration.

    PATTERN:
    - Set reasonable timeout (e.g., 10 seconds for external APIs)
    - Reuse clients when making multiple requests
    - Always close response bodies (defer resp.Body.Close())
    - Check for nil response before accessing
    EOF
    constraints = [
      "Set timeout: &http.Client{Timeout: 10 * time.Second}",
      "Close response bodies to prevent resource leaks",
      "Handle both connection and HTTP-level errors"
    ]
  }

  # Error Handling Best Practices Pattern
  error_handling_pattern = {
    name = "Graceful Error Handling"
    prompt = <<-EOF
    Handle errors gracefully without crashing unless absolutely necessary.

    PATTERN:
    - Use logger package for error logging with context
    - Return errors up the stack when appropriate
    - Only exit on truly unrecoverable errors
    - Continue execution for transient failures when reasonable
    EOF
    constraints = [
      "Log errors with structured context (key-value pairs)",
      "Don't use panic() for expected error conditions",
      "Use logger.Fatal() only for unrecoverable errors"
    ]
  }

  # Cobra Argument Validation Pattern
  cobra_argument_pattern = {
    name = "Cobra Argument Validation"
    prompt = <<-EOF
    Use Cobra's built-in argument validators for command arguments.

    PATTERN:
    - Use Args field in cobra.Command struct
    - Common validators: cobra.ExactArgs(n), cobra.MinArgs(n), cobra.RangeArgs(min, max)
    - Combine with custom validation if needed
    EOF
    constraints = [
      "Prefer built-in validators over manual validation",
      "Provide clear error messages for invalid arguments"
    ]
  }

  # ISO 8601 Timestamp Pattern
  iso8601_timestamp_pattern = {
    name = "ISO 8601 Timestamp Formatting"
    prompt = <<-EOF
    Use standard ISO 8601 (RFC3339) format for timestamps.

    PATTERN:
    - Use time.RFC3339 constant for formatting
    - Always use UTC for consistency: time.Now().UTC()
    - Format: time.Now().UTC().Format(time.RFC3339)
    EOF
    constraints = [
      "Use time.RFC3339 format constant",
      "Convert to UTC before formatting"
    ]
  }

  # Pterm Output Pattern
  pterm_output_pattern = {
    name = "Pterm Colored Output"
    prompt = <<-EOF
    Use pterm for visually enhanced CLI output.

    PATTERN:
    - Success messages: pterm.Success.Println() or Printf()
    - Error messages: pterm.Error.Println() or Printf()
    - Info messages: pterm.Info.Println()
    - Warnings: pterm.Warning.Println()
    EOF
    constraints = [
      "Use appropriate pterm printer for message type",
      "Respect --plain flag for non-TTY output (already in logger)"
    ]
  }
}
