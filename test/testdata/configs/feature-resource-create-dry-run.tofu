# Feature resource create test configuration (dry run for prompt generation)
terraform {
  required_providers {
    tofukit = {
      source = "registry.terraform.io/DimmKirr/tofukit"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
  dry_run     = true
}

resource "tofukit_feature" "hello_cmd" {
  name        = "hello-command"
  description = "A simple hello command"

  requirements = [
    {
      name = "Hello Command"
      instructions = [
        {
          prompt = "Implement a hello command that prints 'Hello from feature!'"
        }
      ]
    }
  ]

  files = {
    "hello.txt" = {
      content = "Hello from feature!\n"
    }
  }

  verifications = [
    {
      command = "cat hello.txt"
      expect  = "Hello from feature!"
    }
  ]
}
