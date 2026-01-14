resource "tofukit_project" "hello_world" {
  name        = "hello-world"
  description = "A simple hello world project"
  version     = "1.0.0"


  # Simple files - no dependencies needed
  files = {
    "inline-deterministic.txt" = {
      content = "Inline file with determenistic content"
    }

    "inline-non-deterministic.txt" = {
      description = "LLM-Generated File"

      instructions = [
        {
          prompt = "Typical color of the sky?"
          constraints = [
            "Output in German language"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'blau' inline-non-deterministic.txt"
        }
      ]
    }

    "subdirectory/hello-subdirectory-deterministic.txt" = {
      content = "Subdirectory File Contents"
    }


    "hello-with-resource-link.txt" = {
      content = "More info in ${tofukit_file.readme.link}"
    }


    "hello-with-resource-link-in-prompt.txt" = {
      instructions = [
        {
          prompt = "Which book is ${tofukit_file.story.link}"
          constraints = [
            "Output just the book name"
          ]
        }
      ]

      verifications = [
        {
          command = "grep -q 'Animal' hello-with-resource-link-in-prompt.txt"
        }
      ]
    }

    "README.md" = tofukit_file.readme
    "STORY.md" = tofukit_file.story
  }
}

resource "tofukit_file" "story" {
  name = "story"
  path = "STORY.md"

  instructions = [
    {
      prompt = "First sentence of Orwell's Animal Farm book"
      constraints = [
        "Use all caps letters"
      ]
    }
  ]

  verifications = [
    {
      command = "grep -q 'JONES' STORY.md"
    }
  ]
}

resource "tofukit_file" "readme" {
  name = "readme"
  path = "README.md"

  instructions = [
    {
      prompt = "Simple README.md file for a Hello World project"
      constraints = [
        "Must not be longer than 100 characters"
      ]
    }
  ]

  verifications = [
    {
      command = "wc -c README.md | awk '{if ($1 < 100) print \"OK\"; else print \"error\"}'"
      expect  = "OK"
    }
  ]
}


# Terraform configuration for Click CLI example
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

# Provider configuration
provider "tofukit" {
  debug                 = true  # Enable debug output to see URI resolution
  output_format         = "json"
  claude_home_directory = "~/.claude"
}

