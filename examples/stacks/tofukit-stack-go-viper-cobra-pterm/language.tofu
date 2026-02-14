# Go CLI Stack Language
# Language definition for the Go CLI stack

# ========================================
# LANGUAGE KITS
# ========================================
resource "tofukit_language" "go" {
  name        = "go"
  description = "Go programming language"
  version     = var.go_version

  requirements = [
    {
      name = "Go Runtime Verification"

      instructions = [
        {
          prompt = "Verify Go ${var.go_version}+ is available on the system. If not installed, Go must be installed manually: Ubuntu/Debian: sudo apt install golang-go, macOS: brew install go, Windows: Download from golang.org"
        }
      ]

      verifications = [
        {
          command = "go version"
          expect  = "go${var.go_version}"
        }
      ]
    },
    {
      name = "Idiomatic Go Code Standards"

      instructions = [
        {
          prompt = <<-EOF
          ALL Go code MUST follow idiomatic, community-preferred, conventional Go patterns:

          - Use standard library patterns and idioms
          - Follow Effective Go guidelines (https://go.dev/doc/effective_go)
          - Use conventional project structure (cmd/, internal/, pkg/ when appropriate)
          - Prefer composition over inheritance
          - Use interfaces appropriately (accept interfaces, return structs)
          - Follow standard naming conventions (mixedCase for unexported, PascalCase for exported)
          - Use standard error handling patterns (return errors, don't panic)
          - Prefer simple, readable code over clever abstractions
          - Use context.Context for cancellation and request-scoped values
          - Follow Go proverbs: "Clear is better than clever", "Don't communicate by sharing memory, share memory by communicating"
          EOF

          constraints = [
            "NO non-standard patterns that deviate from community conventions",
            "NO unnecessary abstractions or over-engineering",
            "NO panic() in production code (except for unrecoverable programmer errors)",
            "NO ignoring errors silently",
            "NO deep inheritance hierarchies"
          ]
        }
      ]

      verifications = []
    }
  ]
}
