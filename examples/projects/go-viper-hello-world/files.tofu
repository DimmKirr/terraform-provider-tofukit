# ========================================
# FILE RESOURCES - Implementation Details
# ========================================
# Reusable file resources for the go-viper-hello-world project

resource "tofukit_file" "cmd_hello" {
  name = "cmd/hello.go"

  instructions = [{
    prompt = <<-EOF
    Create cmd/hello.go as a ${module.go_viper_cobra_pterm.kits.framework.cobra.link} subcommand that greets users by name.

    PACKAGE & IMPORTS:
    - Package: cmd
    - Imports: dimmiirr.com/go-viper-hello-world/internal/logger, github.com/spf13/cobra

    HELLO COMMAND (var helloCmd):
    - Use: "go-viper-hello-world [name]"
    - Short: "Say hello to someone"
    - Long: "Greet someone by name, or the world if no name is provided"
    - Args: cobra.MaximumNArgs(1)
    - Run: Extract name from args (default "World"), log with logger.Info: "Hello, <name>!"

    INIT() FUNCTION:
    - Register command: rootCmd.AddCommand(helloCmd)
    EOF

    constraints = [
      "Use ${module.go_viper_cobra_pterm.kits.framework.cobra.link} Command struct literal for command definition",
      "Must call rootCmd.AddCommand(helloCmd) in init() function",
      "Use logger.Info for output, not fmt.Println",
      "Handle optional name argument with default value 'World'"
    ]
  }]
}

resource "tofukit_file" "cmd_ip" {
  name = "cmd/ip.go"

  instructions = [{
    prompt = <<-EOF
    Create a file as a ${module.go_viper_cobra_pterm.kits.framework.cobra.link} subcommand that displays the user's public IP address.

    PACKAGE & IMPORTS:
    - Package: cmd
    - Imports:
      * dimmiirr.com/go-viper-hello-world/internal/ipinfo
      * dimmiirr.com/go-viper-hello-world/internal/logger
      * github.com/spf13/cobra
      * github.com/pterm/pterm

    IP COMMAND (var ipCmd):
    - Use: "ip"
    - Short: "Display your public IP address"
    - Long: "Fetches and displays your public IP address from ipinfo.io"
    - Args: cobra.NoArgs
    - Run:
      * Call ipinfo.GetPublicIP() to fetch IP
      * On error: log error using logger.Error and return
      * On success: display IP using ${module.go_viper_cobra_pterm.kits.framework.pterm.link} Info.Printfln("Your public IP: %s", ip)

    INIT() FUNCTION:
    - Register command: rootCmd.AddCommand(ipCmd)
    EOF

    constraints = [
      "Use ${module.go_viper_cobra_pterm.kits.framework.cobra.link} Command struct literal for command definition",
      "Must call rootCmd.AddCommand(ipCmd) in init() function",
      "Use ${module.go_viper_cobra_pterm.kits.framework.pterm.link} Info.Printfln for styled output",
      "Use logger.Error for error reporting",
      "No arguments should be accepted"
    ]
  }]
}

resource "tofukit_file" "internal_ipinfo_client" {
  name = "internal/ipinfo/client.go"

  instructions = [{
    prompt = <<-EOF
    Create a file with API client logic for fetching public IP.

    PACKAGE & IMPORTS:
    - Package: ipinfo
    - Imports:
      * net/http
      * io
      * strings
      * fmt
      * time

    CONSTANTS:
    - ipInfoURL = "https://ipinfo.io/ip"
    - requestTimeout = 10 * time.Second

    FUNCTION GetPublicIP() (string, error):
    - Create HTTP client with timeout (requestTimeout)
    - Create GET request to ipInfoURL
    - Execute request using client.Do()
    - Read response body using io.ReadAll
    - Return trimmed string (strings.TrimSpace) or error
    - Handle all error cases: request creation, execution, body reading
    - Close response body properly (defer)
    EOF

    constraints = [
      "Must use context-aware HTTP client with timeout",
      "Handle all error cases explicitly",
      "Response body must be closed with defer",
      "Trim whitespace from response",
      "Use meaningful error messages",
      "No external dependencies beyond standard library"
    ]
  }]
}
