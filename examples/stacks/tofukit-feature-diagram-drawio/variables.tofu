# Variables for Draw.io Diagram Generation Feature

variable "file_path" {
  description = "Output path for diagram file (without .drawio extension)"
  type        = string
  default     = "architecture"

  validation {
    condition     = !endswith(var.file_path, ".drawio")
    error_message = "file_path should not include .drawio extension (it will be added automatically)"
  }
}
