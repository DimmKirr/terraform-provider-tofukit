# ========================================
# FILE RESOURCES - Implementation Details
# ========================================
# Reusable file resources for the go-viper-hello-world project

resource "tofukit_file" "cmd_hello" {
  name = "cmd/hello.go"

  instructions = [{
    prompt = <<-EOF
    Implement ${module.go_viper_cobra_pterm.kits.framework.cobra.link} subcommand following Go CLI best practices.

    TECHNICAL CONSTRAINTS:
    - Package: cmd
    - Required imports: internal/logger, github.com/spf13/cobra
    - Use ${module.go_viper_cobra_pterm.kits.framework.cobra.link} Command struct literal
    - Register command in init() function with rootCmd.AddCommand()
    - Use logger.Info for output (not fmt.Println)
    - Follow Go community conventions for CLI command structure
    EOF

    constraints = [
      "Use ${module.go_viper_cobra_pterm.kits.framework.cobra.link} Command struct literal for command definition",
      "Must call rootCmd.AddCommand() in init() function",
      "Use logger.Info for output, not fmt.Println",
      "Follow Go CLI community conventions"
    ]
  }]
}

resource "tofukit_file" "cmd_ip" {
  name = "cmd/ip.go"

  instructions = [{
    prompt = <<-EOF
    Implement ${module.go_viper_cobra_pterm.kits.framework.cobra.link} subcommand following Go CLI best practices.

    TECHNICAL CONSTRAINTS:
    - Package: cmd
    - Required imports:
      * internal/ipinfo (for API client)
      * internal/logger (for logging)
      * github.com/spf13/cobra (for CLI framework)
      * github.com/pterm/pterm (for styled output)
    - Use ${module.go_viper_cobra_pterm.kits.framework.cobra.link} Command struct literal
    - Register command in init() function with rootCmd.AddCommand()
    - Use ${module.go_viper_cobra_pterm.kits.framework.pterm.link} Info.Printfln for styled output
    - Use logger.Error for error reporting
    - Follow Go CLI community conventions
    EOF

    constraints = [
      "Use ${module.go_viper_cobra_pterm.kits.framework.cobra.link} Command struct literal for command definition",
      "Must call rootCmd.AddCommand() in init() function",
      "Use ${module.go_viper_cobra_pterm.kits.framework.pterm.link} Info.Printfln for styled output",
      "Use logger.Error for error reporting",
      "Follow Go CLI community conventions"
    ]
  }]
}

resource "tofukit_file" "internal_ipinfo_client" {
  name = "internal/ipinfo/client.go"

  instructions = [{
    prompt = <<-EOF
    Implement HTTP API client following Go standard library best practices.

    TECHNICAL CONSTRAINTS:
    - Package: ipinfo
    - Use only Go standard library (net/http, io, strings, fmt, time)
    - Use context-aware HTTP client with timeout
    - Handle all error cases explicitly with meaningful messages
    - Close response body with defer
    - Trim whitespace from API responses (strings.TrimSpace)
    - Follow Go community conventions for HTTP client implementation
    EOF

    constraints = [
      "Use context-aware HTTP client with timeout",
      "Handle all error cases explicitly",
      "Response body must be closed with defer",
      "Trim whitespace from response",
      "Use meaningful error messages",
      "Only use Go standard library"
    ]
  }]
}
