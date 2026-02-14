resource "tofukit_feature" "ui_page_homepage_single_scroll" {
  name = "ui_page_homepage_single_scroll"
  description = "Single-page marketing homepage with vertical scroll sections for carrot delivery service"
  requirements = [
    {
      name = "Homepage Single-Scroll Template"
      instructions = [
        {
          prompt = <<-PROMPT
            **Page Metadata:**
            - **Type:** homepage
            - **Purpose:** Primary landing page for carrot delivery service marketing
            - **URL Pattern:** / (root)
            - **Layout Pattern:** Single-column vertical scroll with distinct sections
            - **Sections (in order):**
              1. Hero Section (carrot branding with logo, headline, subheadline, CTA button)
              2. Weather/Growing Conditions Widget (data visualization with weather metrics)
              3. Feature Section 1: Product Quality (image-text split, left image)
              4. Feature Section 2: Delivery Speed (image-text split, right image)
              5. Social Proof Section: Customer Testimonials (image with quotes)
              6. Lead Capture Form (contact form with name/email fields)
              7. Footer (navigation links, copyright)
            
            **Responsive Strategy:**
            - Desktop (>768px): Image-text sections use 50/50 split, alternating left/right
            - Mobile (<768px): Stack all sections vertically, full-width images
            - All sections: Full-width with centered content containers
            
            **Key Components Used:**
            - $${ui_component_button_cta} ("Get Your Carrots", "Contact Us")
            - $${ui_component_card_weather} (weather widget with metrics)
            - $${ui_component_card_feature} (product feature cards)
            - $${ui_component_form_contact} (lead capture form)
            - $${ui_component_testimonial_card} (customer quotes)
            
            **Key Images:**
            - $${ui_image_logo_carrot} (carrot icon in header)
            - $${ui_image_hero_background} (light green background pattern)
            - $${ui_image_product_carrots_bag} (fresh carrots in paper bag)
            - $${ui_image_character_rabbit_kitchen} (cartoon rabbit mascot)
            - $${ui_image_character_rabbit_family} (testimonial illustration)
            
            **Colors Applied:**
            - Hero background: Light cream/beige ($${ui_color_background_cream})
            - Primary CTA: Orange ($${ui_color_primary_orange})
            - Section backgrounds: Alternating light green ($${ui_color_background_mint}) and white
            - Footer: Dark green ($${ui_color_footer_green})
            - Text: Black for headlines, gray for body
            
            **Typography:**
            - Logo/Brand: Bold sans-serif ($${ui_font_brand_bold})
            - Headlines: Bold black sans-serif, large size ($${ui_font_heading_bold})
            - Subheadlines: Orange accent color ($${ui_font_subheading_orange})
            - Body text: Regular sans-serif, medium gray ($${ui_font_body_regular})
            
            **Implementation Structure:**
            ```html
            <div class="page-homepage">
              <section class="hero" style="background: cream">
                <div class="container">
                  <img class="logo" />
                  <h1>Carrot Labs</h1>
                  <h2 class="orange">Fresh Carrots Delivered to Your Burrow!</h2>
                  <p class="tagline">The happiest rabbits get the crunchiest carrots...</p>
                  <button class="cta-primary">Get Your Carrots</button>
                </div>
              </section>
              
              <section class="weather-widget" style="background: mint-green">
                <div class="container">
                  <h3>Weather is Always Good in the Burrow</h3>
                  <div class="weather-metrics">...</div>
                </div>
              </section>
              
              <section class="feature-alternating left-image" style="background: white">
                <div class="container split">
                  <img src="carrots-bag.jpg" />
                  <div class="content">
                    <h3>Garden-Fresh, Always Crunchy</h3>
                    <p>Our carrots are harvested at dawn...</p>
                  </div>
                </div>
              </section>
              
              <section class="feature-alternating right-image" style="background: mint-green">
                <div class="container split">
                  <div class="content">
                    <h3>Lightning-Fast Delivery</h3>
                    <p>We deliver fresh carrots within weeks...</p>
                  </div>
                  <img src="rabbit-kitchen.jpg" />
                </div>
              </section>
              
              <section class="testimonials" style="background: white">
                <div class="container">
                  <h3>Thousands of Happy Hoppers</h3>
                  <img src="rabbit-family.jpg" />
                  <div class="quotes">...</div>
                </div>
              </section>
              
              <section class="contact-form" style="background: mint-green">
                <div class="container">
                  <h2 class="orange">Ready for Fresh Carrots?</h2>
                  <form>...</form>
                </div>
              </section>
              
              <footer style="background: dark-green">
                <nav>...</nav>
                <p class="copyright">...</p>
              </footer>
            </div>
            ```
            
            **Page-Specific Styles:**
            - Section backgrounds alternate: cream → mint-green → white → mint-green → white → mint-green → dark-green
            - Full-width sections with max-width container (~1200px)
            - Generous vertical padding between sections (4-6rem)
            - Image-text sections use flexbox with alternating direction
            - Smooth scroll behavior for CTA interactions
            - Border radius on images and cards for friendly aesthetic
            
            **SEO Considerations:**
            - Hero H1: "Carrot Labs" (brand)
            - Hero H2: "Fresh Carrots Delivered to Your Burrow!" (value proposition)
            - Meta description: "The happiest rabbits get the crunchiest carrots. Join thousands of bunnies enjoying farm-fresh delivery every week!"
            - Open Graph image: $${ui_image_hero_background} or $${ui_image_product_carrots_bag}
          PROMPT
          constraints = [
            "Do not exceed 7 sections - optimal scroll depth for single-page landing",
            "Do not use autoplay animations in hero - performance and accessibility issue",
            "Do not make sections too tall - each section should fit in 1-1.5 viewports maximum",
            "Do not use complex parallax effects - janky on mobile devices",
            "Do not hide contact form behind modal - inline form increases conversions",
            "Do not use more than 3 testimonial quotes - keeps section scannable",
            "Do not alternate backgrounds randomly - follow consistent cream/mint-green/white pattern",
          ]
        },
      ]
    },
  ]
}