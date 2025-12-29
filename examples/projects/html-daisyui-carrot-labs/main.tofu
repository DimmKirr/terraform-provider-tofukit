# ========================================
# MAIN PROJECT RESOURCE
# ========================================
resource "tofukit_project" "labs" {
  name        = "labs"
  description = "Generated from kitcut analysis"
  version     = "1.0.0"

  model = "anthropic/claude-haiku"

  # Link to all features from modules
  features = {
    "weather_info_viewing" = module.project.features.weather_info_viewing
    "ui_typography_system" = module.ui.features.ui_typography_system
    "ui_color_primary" = module.ui.features.ui_color_primary
    "ui_rabbit_delivery" = module.ui.features.ui_rabbit_delivery
    "ui_font_inter_400" = module.ui.features.ui_font_inter_400
    "ui_animation_hover_button_primary" = module.ui.features.ui_animation_hover_button_primary
    "marketing_content_reading" = module.project.features.marketing_content_reading
    "contact_form_submission" = module.project.features.contact_form_submission
    "page_session_tracking" = module.project.features.page_session_tracking
    "ui_color_text_gray" = module.ui.features.ui_color_text_gray
    "ui_font_poppins_300" = module.ui.features.ui_font_poppins_300
    "ui_font_poppins_400" = module.ui.features.ui_font_poppins_400
    "ui_design_style" = module.ui.features.ui_design_style
    "ui_color_background_white" = module.ui.features.ui_color_background_white
    "ui_font_inter_300" = module.ui.features.ui_font_inter_300
    "ui_font_poppins_600" = module.ui.features.ui_font_poppins_600
    "ui_font_poppins_700" = module.ui.features.ui_font_poppins_700
    "integration_google_font_api" = tofukit_feature.integration_google_font_api
    "ui_animation_hover_card_lift" = module.ui.features.ui_animation_hover_card_lift
    "integration_open-meteo-api" = tofukit_feature.integration_open_meteo_api
    "ui_color_footer" = module.ui.features.ui_color_footer
    "ui_font_inter_600" = module.ui.features.ui_font_inter_600
    "ui_animation_fade_in_scroll" = module.ui.features.ui_animation_fade_in_scroll
    "ui_layout_system" = module.ui.features.ui_layout_system
    "ui_rabbit_family" = module.ui.features.ui_rabbit_family
    "ui_weather_sun" = module.ui.features.ui_weather_sun
    "ui_page_homepage_single_scroll" = module.ui.features.ui_page_homepage_single_scroll
    "testimonial_reading" = module.project.features.testimonial_reading
    "scroll_to_contact_form" = module.project.features.scroll_to_contact_form
    "ui_font_inter_500" = module.ui.features.ui_font_inter_500
    "ui_logo_carrot" = module.ui.features.ui_logo_carrot
    "ui_paper_bag_carrots" = module.ui.features.ui_paper_bag_carrots
    "ui_animation_stagger_list_items" = module.ui.features.ui_animation_stagger_list_items
    "ui_animation_input_focus_ring" = module.ui.features.ui_animation_input_focus_ring
    "ui_color_background_light" = module.ui.features.ui_color_background_light
    "ui_color_accent" = module.ui.features.ui_color_accent
    "ui_color_scheme" = module.ui.features.ui_color_scheme
    "ui_animation_hero_logo_spin" = module.ui.features.ui_animation_hero_logo_spin
  }
}
