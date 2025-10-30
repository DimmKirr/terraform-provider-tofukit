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
