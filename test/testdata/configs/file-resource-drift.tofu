# File resource drift detection test configuration
terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

provider "tofukit" {
  output_path = "output"
  debug       = true
  dry_run     = true
}

resource "tofukit_file" "hello" {
  name    = "hello"
  path    = "hello.txt"
  content = "Hello World"
}

resource "tofukit_project" "test" {
  name    = "file-drift-test"
  version = "1.0.0"

  files = {
    "hello.txt" = tofukit_file.hello
  }
}
