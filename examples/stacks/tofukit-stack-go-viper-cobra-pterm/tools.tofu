# Go CLI Stack Tools
# Tool definitions for the Go CLI stack

# ========================================
# TOOL KITS
# ========================================
resource "tofukit_tool" "go_mod" {
  name        = "go-mod"
  description = "Go modules dependency management"
  version     = var.go_version

  depends_on = [tofukit_language.go]

  requirements = [
    {
      name = "Go Modules Initialization"

      instructions = [
        {
          prompt = "Initialize Go modules for the project: go mod init ${var.project_module_path}"
        }
      ]

      verifications = [
        {
          command = "test -f go.mod && echo 'go.mod exists'"
          expect  = "go.mod exists"
        }
      ]
    }
  ]
}

resource "tofukit_tool" "gotask" {
  name        = "gotask"
  description = "Task runner for build automation and task orchestration"
  version     = var.task_version

  depends_on = [tofukit_language.go]

  requirements = [
    {
      name = "Task Runner Installation"

      instructions = [
        {
          prompt = <<-EOF
          Verify Task runner v${var.task_version}+ is installed. If not installed, Task must be installed manually:

          Installation options:
          - macOS: brew install go-task
          - Linux: sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
          - Go install: go install github.com/go-task/task/v3/cmd/task@latest
          - Manual: Download from https://github.com/go-task/task/releases

          Verify installation with: task --version
          EOF

          constraints = [
            "Task binary must be available in system PATH",
            "Version must be 3.0.0 or higher for Taskfile v3 support"
          ]
        }
      ]

      verifications = [
        {
          command = "task --version"
          expect  = "Task version: v${var.task_version}"
        },
        {
          command = "which task && echo 'task found in PATH'"
          expect  = "task found in PATH"
        },
        {
          command = "test ! -f Makefile && echo 'no Makefile'"
          expect  = "no Makefile"
        }
      ]
    },
    {
      name = "Taskfile v3 Support"

      instructions = [
        {
          prompt = "Ensure Task runner supports Taskfile v3 schema with variables, includes, and advanced features"
          constraints = [
            "NO usage of deprecated Taskfile v2 syntax",
            "Support for dynamic variables and shell expansions",
            "Support for task dependencies and requires"
          ]
        }
      ]

      verifications = []
    }
  ]
}
