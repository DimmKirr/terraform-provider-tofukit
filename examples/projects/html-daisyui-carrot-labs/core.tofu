# ========================================
# MODULE ORCHESTRATION
# ========================================
module "stack" {
  source = "./stack"
}

module "project" {
  source = "./project"

  files = {
    ui_weather_sun = module.ui.files.ui_weather_sun
    ui_paper_bag_carrots = module.ui.files.ui_paper_bag_carrots
    ui_rabbit_delivery = module.ui.files.ui_rabbit_delivery
    ui_logo_carrot = module.ui.files.ui_logo_carrot
    ui_rabbit_family = module.ui.files.ui_rabbit_family
  }

  integrations = {
    integration_open_meteo_api = tofukit_feature.integration_open_meteo_api
    integration_google_font_api = tofukit_feature.integration_google_font_api
  }

  libraries = {
    tailwind_css = module.stack.libraries.tailwind_css
    jquery = module.stack.libraries.jquery
    font_awesome = module.stack.libraries.font_awesome
    daisyui = module.stack.libraries.daisyui
  }

  ui = module.ui.features
}

module "ui" {
  source = "./ui"

  files = {
    ui_paper_bag_carrots = module.ui.files.ui_paper_bag_carrots
    ui_rabbit_delivery = module.ui.files.ui_rabbit_delivery
    ui_logo_carrot = module.ui.files.ui_logo_carrot
    ui_rabbit_family = module.ui.files.ui_rabbit_family
    ui_weather_sun = module.ui.files.ui_weather_sun
  }

  integrations = {
    integration_google_font_api = tofukit_feature.integration_google_font_api
    integration_open_meteo_api = tofukit_feature.integration_open_meteo_api
  }

  libraries = {
    tailwind_css = module.stack.libraries.tailwind_css
    jquery = module.stack.libraries.jquery
    font_awesome = module.stack.libraries.font_awesome
    daisyui = module.stack.libraries.daisyui
  }
}