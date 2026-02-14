# ========================================
# COMMON GO CLI PATTERNS (Stack-level)
# ========================================
# These patterns apply to ALL go-viper-cobra-pterm applications
# Projects can reference these patterns via ${module.stack.features.pattern_name.link}

# HTTP Client Best Practices Pattern
resource "tofukit_feature" "http_client_pattern" {
  name        = "http-client-pattern"
  description = "HTTP Client Best Practices"

  requirements = [{
    name = "HTTP Client Best Practices"
    instructions = [{
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
    }]
  }]
}

# Error Handling Best Practices Pattern
resource "tofukit_feature" "error_handling_pattern" {
  name        = "error-handling-pattern"
  description = "Graceful Error Handling"

  requirements = [{
    name = "Graceful Error Handling"
    instructions = [{
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
    }]
  }]
}

# Cobra Argument Validation Pattern
resource "tofukit_feature" "cobra_argument_pattern" {
  name        = "cobra-argument-pattern"
  description = "Cobra Argument Validation"

  requirements = [{
    name = "Cobra Argument Validation"
    instructions = [{
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
    }]
  }]
}

# ISO 8601 Timestamp Pattern
resource "tofukit_feature" "iso8601_timestamp_pattern" {
  name        = "iso8601-timestamp-pattern"
  description = "ISO 8601 Timestamp Formatting"

  requirements = [{
    name = "ISO 8601 Timestamp Formatting"
    instructions = [{
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
    }]
  }]
}

# Pterm Output Pattern
resource "tofukit_feature" "pterm_output_pattern" {
  name        = "pterm-output-pattern"
  description = "Pterm Colored Output"

  requirements = [{
    name = "Pterm Colored Output"
    instructions = [{
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
    }]
  }]
}
