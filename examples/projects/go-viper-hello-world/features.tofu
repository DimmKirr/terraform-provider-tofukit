# ========================================
# PROJECT FEATURES
# ========================================
# Features specific to the go-viper-hello-world project

# ========================================
# HELLO COMMAND FEATURE
# ========================================
resource "tofukit_feature" "hello_command" {
  name        = "hello-command"
  description = "Hello command to demonstrate basic CLI functionality"

  # Link to implementation file
  files = {
    "cmd/hello.go" = tofukit_file.cmd_hello
  }

  # Feature-level requirement: integration with root command
  requirements = [{
    name = "Hello Command"

    instructions = [{
      prompt = <<-EOF
      Implement a "hello" command that greets users.

      BUSINESS REQUIREMENTS:
      - Command: hello [name]
      - Behavior: Greet the user by name, default to "World" if no name provided
      - Output: "Hello, <name>!" message
      - Integration: Must integrate with ${module.go_viper_cobra_pterm.kits.framework.cobra.link} CLI framework

      IMPLEMENTATION LOCATION:
      This feature must be implemented in ${tofukit_file.cmd_hello.link}

      COMMAND SPECIFICATION:
      - Use: "go-viper-hello-world [name]"
      - Short description: "Say hello to someone"
      - Long description: "Greet someone by name, or the world if no name is provided"
      - Arguments: Accept optional name (max 1 argument)
      - Output: Use structured logging to display "Hello, <name>!"
      EOF

      constraints = [
        "Must integrate with ${module.go_viper_cobra_pterm.kits.framework.cobra.link} CLI framework",
        "Use structured logging for output",
        "Handle optional name argument gracefully (default to 'World')",
        "Follow Go community conventions for CLI commands"
      ]
    }]

    verifications = [{
      command = "grep -q 'var helloCmd' cmd/hello.go && echo 'found'"
      expect  = "found"
    }, {
      command = "grep -q 'rootCmd\\.AddCommand' cmd/hello.go && echo 'found'"
      expect  = "found"
    }, {
      command = "test -f cmd/hello.go && echo 'exists'"
      expect  = "exists"
    }]
  }]
}

# ========================================
# IP COMMAND FEATURE
# ========================================
resource "tofukit_feature" "ip_command" {
  name        = "ip-command"
  description = "IP command to fetch and display public IP address"

  # Link to implementation files
  files = {
    "cmd/ip.go"                 = tofukit_file.cmd_ip
    "internal/ipinfo/client.go" = tofukit_file.internal_ipinfo_client
  }

  # Feature-level requirement: integration with root command
  requirements = [{
    name = "Public IP Address Display"

    instructions = [{
      prompt = <<-EOF
      Implement an "ip" command that displays the user's public IP address.

      BUSINESS REQUIREMENTS:
      - Command: ip
      - Behavior: Fetch and display user's public IP address
      - Data Source: ipinfo.io/ip API endpoint
      - Output: Display IP address with styled output using ${module.go_viper_cobra_pterm.kits.framework.pterm.link}
      - Integration: Must integrate with ${module.go_viper_cobra_pterm.kits.framework.cobra.link} CLI framework

      IMPLEMENTATION LOCATION:
      - CLI command implementation: ${tofukit_file.cmd_ip.link}
      - API client logic: ${tofukit_file.internal_ipinfo_client.link}

      COMMAND SPECIFICATION:
      - Use: "ip"
      - Short description: "Display your public IP address"
      - Long description: "Fetches and displays your public IP address from ipinfo.io"
      - Arguments: None (cobra.NoArgs)
      - API Endpoint: https://ipinfo.io/ip
      - Output format: "Your public IP: <ip_address>"
      - Timeout: 10 seconds for HTTP request

      API CLIENT REQUIREMENTS (${tofukit_file.internal_ipinfo_client.link}):
      - Function: GetPublicIP() (string, error)
      - HTTP GET request to ipinfo.io/ip
      - Timeout: 10 seconds
      - Error handling: Request creation, execution, body reading
      - Response: Trimmed string (no whitespace)
      EOF

      constraints = [
        "Must integrate with ${module.go_viper_cobra_pterm.kits.framework.cobra.link} CLI framework",
        "Use HTTP client for external API calls",
        "Display output using ${module.go_viper_cobra_pterm.kits.framework.pterm.link} for consistent styling",
        "Handle network errors gracefully with user-friendly messages",
        "Encapsulate API logic in internal package for testability"
      ]
    }]

    verifications = [{
      command = "test -f cmd/ip.go && echo 'exists'"
      expect  = "exists"
    }, {
      command = "test -f internal/ipinfo/client.go && echo 'exists'"
      expect  = "exists"
    }, {
      command = "grep -q 'var ipCmd' cmd/ip.go && echo 'found'"
      expect  = "found"
    }, {
      command = "grep -q 'rootCmd\\.AddCommand' cmd/ip.go && echo 'found'"
      expect  = "found"
    }]
  }]
}
