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

  requirements = [
    {
      name = "Hello Command Implementation"

      instructions = [
        {
          prompt = "Implement a hello command that greets users by name, with 'World' as the default if no name is provided"

          constraints = [
            "Must integrate with Cobra CLI framework",
            "Use structured logging for output",
            "Handle optional name argument gracefully",
            "Follow Go community conventions for CLI commands"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'var helloCmd' cmd/hello.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'rootCmd\\.AddCommand' cmd/hello.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "test -f cmd/hello.go && echo 'exists'"
          expect  = "exists"
        }
      ]
    }
  ]

  files = {
    "cmd/hello.go" = {
      instructions = [
        {
          prompt = <<-EOF
          Create cmd/hello.go as a Cobra subcommand that greets users by name.

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
            "Use cobra.Command struct literal for command definition",
            "Must call rootCmd.AddCommand(helloCmd) in init() function",
            "Use logger.Info for output, not fmt.Println",
            "Handle optional name argument with default value 'World'"
          ]
        }
      ]
    }
  }
}

# ========================================
# IP COMMAND FEATURE
# ========================================
resource "tofukit_feature" "ip_command" {
  name        = "ip-command"
  description = "IP command to fetch and display public IP address"

  requirements = [
    {
      name = "IP Address Retrieval"

      instructions = [
        {
          prompt = "Implement an ip command that retrieves the user's public IP address from ipinfo.io/ip and displays it using styled output"

          constraints = [
            "Must integrate with Cobra CLI framework",
            "Use HTTP client for external API calls",
            "Display output using pterm for consistent styling",
            "Handle network errors gracefully with user-friendly messages",
            "Encapsulate API logic in internal package for testability"
          ]
        }
      ]

      verifications = [
        {
          command = "test -f cmd/ip.go && echo 'exists'"
          expect  = "exists"
        },
        {
          command = "test -f internal/ipinfo/client.go && echo 'exists'"
          expect  = "exists"
        },
        {
          command = "grep -q 'var ipCmd' cmd/ip.go && echo 'found'"
          expect  = "found"
        },
        {
          command = "grep -q 'rootCmd\\.AddCommand' cmd/ip.go && echo 'found'"
          expect  = "found"
        }
      ]
    }
  ]

  files = {
    "cmd/ip.go" = {
      instructions = [
        {
          prompt = <<-EOF
          Create a file as a Cobra subcommand that displays the user's public IP address.

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
            * On success: display IP using pterm.Info.Printfln("Your public IP: %s", ip)

          INIT() FUNCTION:
          - Register command: rootCmd.AddCommand(ipCmd)
          EOF

          constraints = [
            "Use cobra.Command struct literal for command definition",
            "Must call rootCmd.AddCommand(ipCmd) in init() function",
            "Use pterm.Info.Printfln for styled output",
            "Use logger.Error for error reporting",
            "No arguments should be accepted"
          ]
        }
      ]
    }

    "internal/ipinfo/client.go" = {
      instructions = [
        {
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
        }
      ]
    }
  }

  kits = []
}
