
provider "tofukit" {
  output_format = "json"
  output_path   = ".tofukit"
}

# Define the Python language component
resource "tofukit_language" "python" {
  name        = "python"
  description = "Python is a high-level programming language"
  version     = "3.12.6"

  requirement {
    name = "Install Python"
    instructions = [
      "Python must be installed using the system-level package manager",
      "For Ubuntu/Debian: sudo apt install python3",
      "For macOS: brew install python3",
      "For Windows: Download from python.org"
    ]
    verification {
      command = "python3 --version"
      expect  = "Python 3"
    }
  }
}
