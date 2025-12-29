variable "files" {
  type = map(object({
    link = string
  }))
  default = {}
  description = "Map of tofukit_file resources passed from root"
}

variable "integrations" {
  type = map(object({
    link = string
  }))
  default = {}
  description = "Map of tofukit_integration resources passed from root"
}

variable "libraries" {
  type = map(object({
    link = string
  }))
  default = {}
  description = "Map of tofukit_library resources from stack module"
}
