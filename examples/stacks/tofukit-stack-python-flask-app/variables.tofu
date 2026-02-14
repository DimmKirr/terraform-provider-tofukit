variable "project_name" {
  description = "Name of the project (used in Flask application and file templates)"
  type        = string
}

variable "project_version" {
  description = "Version of the project"
  type        = string
}

variable "python_version" {
  description = "Python version"
  type        = string
  default     = "3.12.6"
}

variable "flask_version" {
  description = "Flask framework version"
  type        = string
  default     = "3.0.0"
}

variable "flask_cors_version" {
  description = "Flask-CORS extension version"
  type        = string
  default     = "4.0.0"
}

variable "uv_version" {
  description = "uv package manager version"
  type        = string
  default     = "0.4.27"
}
