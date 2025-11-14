# Kit verification enforcement test configuration
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_format = "json"
  output_path   = "output"
  debug         = true
}

resource "tofukit_tool" "fake_tool" {
  name        = "nonexistent-tool"
  description = "A tool that doesn't exist"
  version     = "1.0.0"

  requirements = [
    {
      name = "Fake Tool Installation"

      instructions = [{
        prompt = "This tool doesn't exist and can't be installed"
      }]

      verifications = [
        {
          command = "false"
          expect  = ""
        }
      ]
    }
  ]
}

resource "tofukit_project" "test" {
  name        = "kit-verification-test"
  description = "Test that kit verifications are enforced"
  version     = "1.0.0"

  kits = [tofukit_tool.fake_tool]

  files = {
    "test.txt" = {
      content = "hello\n"
    }
  }
}
