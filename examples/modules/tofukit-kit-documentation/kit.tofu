# Documentation Kit Module
# This module provides a reusable documentation kit that generates
# comprehensive project documentation

terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# ========================================
# DOCUMENTATION STACK
# ========================================
resource "tofukit_stack" "documentation" {
  name        = "comprehensive-documentation"
  description = "Generates comprehensive project documentation including README, setup instructions, and usage examples"

  # README.md with comprehensive documentation
  files = {
    "README.md" = {
      instructions = [
        {
          prompt = "Create comprehensive documentation for this project. Include the following sections: project overview, features, installation instructions, usage examples with command-line examples, configuration options, troubleshooting guide, and contributing guidelines. Make the documentation clear, professional, and easy to follow."
          constraints = [
            "Do NOT include emojis or badges",
            "Use clear section headers (##) for main sections",
            "Include code blocks with proper syntax highlighting",
            "Keep explanations concise but complete",
            "Must include project name '${var.project_name}' in the title",
            var.project_description != "" ? "Must include description: ${var.project_description}" : "Include a brief project description",
            "Include setup instructions specific to the tools and frameworks used",
            "Include usage examples that demonstrate key features",
            "Maximum 500 lines to keep it focused and maintainable"
          ]
        }
      ]

      verifications = [
        {
          command = "grep '${var.project_name}' README.md"
          expect  = var.project_name
        },
        {
          command = "test -f README.md && echo 'README exists'"
          expect  = "README exists"
        },
        {
          command = "grep -q '^# ' README.md && echo 'Has title'"
          expect  = "Has title"
        }
      ]
    }
  }
}
