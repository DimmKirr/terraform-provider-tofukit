variable "integrations" {
  type = map(object({
    link = string
  }))
  default = {}
  description = "Map of integration resources passed from root"
}

variable "libraries" {
  type = map(object({
    link = string
  }))
  default = {}
  description = "Map of stack library resources"
}

variable "ui" {
  type = any
  description = "Map of UI feature resources from ui module"
}

variable "files" {
  type = map(object({
    link = string
  }))
  default = {}
  description = "Map of file resources passed from root"
}
