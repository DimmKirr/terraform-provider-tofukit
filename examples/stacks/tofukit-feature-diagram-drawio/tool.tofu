# Draw.io CLI Tool
# Used for PNG export and diagram validation

resource "tofukit_tool" "drawio" {
  name        = "drawio"
  description = "Draw.io desktop CLI for diagram generation and export"
  version     = "latest"

  requirements = [
    {
      name = "Draw.io CLI Installation"

      instructions = [
        {
          prompt = <<-EOF
          Verify draw.io CLI is installed. If not installed, provide installation instructions:

          Installation options:

          Linux:
          1. Download from https://github.com/jgraph/drawio-desktop/releases
          2. Install package:
             - Debian/Ubuntu: sudo dpkg -i drawio-amd64-<version>.deb
             - RPM-based: sudo rpm -i drawio-<version>.rpm
          3. Verify: drawio --version

          macOS:
          1. Homebrew: brew install --cask drawio
          2. Or download from https://github.com/jgraph/drawio-desktop/releases
          3. Verify: drawio --version

          Windows:
          1. Download installer from https://github.com/jgraph/drawio-desktop/releases
          2. Run drawio-<version>.exe
          3. Add to PATH if using CLI

          Alternative (Cross-platform):
          Use Docker: docker run -it --rm -v $(pwd):/data jgraph/drawio-desktop --help
          EOF

          constraints = [
            "drawio binary must be available in system PATH",
            "CLI must support -x (export), -f (format), and -o (output) flags",
            "Prefer native installation over Docker for better performance"
          ]
        }
      ]

      verifications = [
        {
          command = "drawio --version && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "which drawio && echo 'drawio found in PATH'"
          expect  = "drawio found in PATH"
        }
      ]
    }
  ]
}
