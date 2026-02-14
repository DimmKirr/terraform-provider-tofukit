output "features" {
  value = {
    marketing_content_reading = tofukit_feature.marketing_content_reading
    weather_info_viewing = tofukit_feature.weather_info_viewing
    contact_form_submission = tofukit_feature.contact_form_submission
    testimonial_reading = tofukit_feature.testimonial_reading
    page_session_tracking = tofukit_feature.page_session_tracking
    scroll_to_contact_form = tofukit_feature.scroll_to_contact_form
  }
}