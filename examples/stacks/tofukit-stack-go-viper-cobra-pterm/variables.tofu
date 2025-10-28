variable "project_name" {
  description = "Name of the CLI application (used in binary name and commands)"
  type        = string
}

variable "project_version" {
  description = "Version of the project (used in --version output)"
  type        = string
}

variable "project_module_path" {
  description = "Go module path (e.g., github.com/username/project)"
  type        = string
}

variable "go_version" {
  description = "Go language version"
  type        = string
  default     = "1.23.1"
}

variable "cobra_version" {
  description = "Cobra framework version"
  type        = string
  default     = "1.8.1"
}

variable "viper_version" {
  description = "Viper config version"
  type        = string
  default     = "1.19.0"
}

variable "pterm_version" {
  description = "pterm UI version"
  type        = string
  default     = "0.12.80"
}

variable "task_version" {
  description = "Task runner (go-task) version"
  type        = string
  default     = "3.39.2"
}
