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

variable "style" {
  description = "Diagram style: 'AWS' for AWS architecture best practices, 'generic' for general diagrams"
  type        = string
  default     = "generic"

  validation {
    condition     = contains(["AWS", "generic"], var.style)
    error_message = "Style must be either 'AWS' or 'generic'"
  }
}

variable "requirements" {
  description = "Additional requirements to append to base diagram generation requirements"
  type = list(object({
    name          = string
    instructions  = optional(list(object({
      prompt      = string
      constraints = optional(list(string))
    })))
    verifications = optional(list(object({
      command = string
      expect  = optional(string)
    })))
  }))
  default = []
}
