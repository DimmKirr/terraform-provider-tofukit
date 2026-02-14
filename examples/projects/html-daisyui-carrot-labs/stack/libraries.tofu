resource "tofukit_library" "tailwind_css" {
  name = "tailwind_css"
  version = "latest"  # Version unknown during identification
}

resource "tofukit_library" "jquery" {
  name = "jquery"
  version = var.jquery_version
}

resource "tofukit_library" "font_awesome" {
  name = "font_awesome"
  version = var.font_awesome_version
}

resource "tofukit_library" "daisyui" {
  name = "daisyui"
  version = var.daisyui_version
}