# Variables for Basic Documentation Feature Module

variable "project_name" {
  description = "Name of the project for the README"
  type        = string
  default     = "my-project"
}

variable "project_description" {
  description = "Brief description of the project"
  type        = string
  default     = "A software project"
}

variable "features" {
  description = "Map of feature names to descriptions that will be documented in README"
  type        = map(string)
  default = {
    "feature1" = "First feature description"
    "feature2" = "Second feature description"
  }
}
