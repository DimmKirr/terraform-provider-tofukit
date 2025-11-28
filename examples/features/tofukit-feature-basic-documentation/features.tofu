# Basic Documentation Feature
# Provides a README.md file that documents project features from a variable map

terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = ">= 0.1.0"
    }
  }
}

# ========================================
# DOCUMENTATION FEATURE
# ========================================
resource "tofukit_feature" "basic_documentation" {
  name        = "basic-documentation"
  description = "Basic README documentation with dynamic feature list"

  requirements = [
    {
      name = "README Creation"
      instructions = [
        {
          prompt = "Create project documentation as specified in the README.md file resource"
        }
      ]
    }
  ]

  files = {
    "README.md" = tofukit_file.readme_md
  }

  kits = []
}

resource "tofukit_file" "readme_md" {
  name = "README.md"

  instructions = [
    {
      prompt = <<-EOF
      Create a simple README.md documentation file.

      PROJECT DETAILS:
      - Project Name: ${var.project_name}
      - Description: ${var.project_description}

      FEATURES TO DOCUMENT:
      ${join("\n", [for key, desc in var.features : "- ${key}: ${desc}"])}

      STRUCTURE:
      - Title with project name
      - Brief overview/description
      - Features section listing all features with their descriptions
      - Installation section (basic)
      - Usage section (basic)

      Keep the documentation clear, concise, and professional.
      EOF

      constraints = [
        "Use proper markdown formatting",
        "Include all features from the list",
        "Keep descriptions concise",
        "NO emojis or excessive styling"
      ]
    }
  ]

  verifications = [
    {
      command = "test -f README.md && echo 'found'"
      expect  = "found"
    },
    {
      command = "grep -q '${var.project_name}' README.md && echo 'has project name'"
      expect  = "has project name"
    },
    {
      command = "grep -q '^# ' README.md && echo 'has title'"
      expect  = "has title"
    }
  ]
}
