variable "project_name" {
  description = "Name of the project (used in CLI application and file templates)"
  type        = string
}

variable "project_version" {
  description = "Version of the project (used in CLI --version output)"
  type        = string
}

variable "python_version" {
  description = "Python version"
  type        = string
  default     = "3.12.6"
}

variable "click_version" {
  description = "Click framework version"
  type        = string
  default     = "8.1.7"
}

variable "uv_version" {
  description = "uv package manager version"
  type        = string
  default     = "0.4.27"
}
