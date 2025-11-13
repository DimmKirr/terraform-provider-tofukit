# Python Flask Stack Language
# Language definition for the Python Flask stack

# ========================================
# LANGUAGE KITS
# ========================================
resource "tofukit_language" "python" {
  name        = "python"
  description = "Python high-level programming language runtime"
  version     = var.python_version

  requirements = [
    {
      name = "Python Runtime Verification"

      instructions = [
        {
          prompt = "Verify Python 3.12+ is available on the system. If not installed, Python must be installed manually: Ubuntu/Debian: sudo apt install python3 python3-dev, macOS: brew install python@3.12, Windows: Download from python.org"
        }
      ]

      verifications = [
        {
          command = "python3 --version"
          expect  = "Python 3.12"
        }
      ]
    }
  ]
}
