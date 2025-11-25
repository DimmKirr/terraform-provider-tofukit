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
    name = "Hello Command Integration"

    instructions = [{
      prompt = "Implement a hello command that greets users by name, with 'World' as the default if no name is provided. Integrate with ${module.go_viper_cobra_pterm.kits.framework.cobra.link} CLI framework."

      constraints = [
        "Must integrate with ${module.go_viper_cobra_pterm.kits.framework.cobra.link} CLI framework",
        "Use structured logging for output",
        "Handle optional name argument gracefully",
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
    "cmd/ip.go"                  = tofukit_file.cmd_ip
    "internal/ipinfo/client.go" = tofukit_file.internal_ipinfo_client
  }

  # Feature-level requirement: integration with root command
  requirements = [{
    name = "IP Address Retrieval"

    instructions = [{
      prompt = "Implement an ip command that retrieves the user's public IP address from ipinfo.io/ip and displays it using ${module.go_viper_cobra_pterm.kits.framework.pterm.link} styled output. Integrate with ${module.go_viper_cobra_pterm.kits.framework.cobra.link} CLI framework."

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
