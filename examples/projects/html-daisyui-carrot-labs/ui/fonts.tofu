resource "tofukit_feature" "ui_font_poppins_300" {
  name = "ui_font_poppins_300"
  description = "Poppins Light (300) for subtle text and secondary content"
  requirements = [
    {
      name = "Poppins Font Weight 300"
      instructions = [
        {
          prompt = <<-PROMPT
            **Font Metadata:**
            - **Font Family:** Poppins
            - **Font Weight:** 300 (Light)
            - **Source:** google_fonts
            - **Usage Contexts:** secondary descriptions, caption text, subtle labels
            
            **CSS Implementation:**
            ```css
            font-family: 'Poppins', sans-serif;
            font-weight: 300;
            ```
            
            **Integration Reference:**
            Load from Google Fonts: $${var.integrations.integration_google_fonts.link}
            Ensure CSS includes: Poppins:wght@300
            
            **CRITICAL: Use Module Variable Reference**
            - Fonts are in the `ui` module, which receives integrations as input variables
            - MUST use `var.integrations.integration_google_fonts` NOT `tofukit_feature.integration_google_fonts`
            - The integration resource is defined at root level and passed to the module via `integrations` variable
            
            **When to Use:**
            - Subtle secondary text and descriptions
            - Caption text for images
            - Tertiary information that should be de-emphasized
            
            **When to Use Other Weights:**
            - Use Poppins 400 for standard body text
            - Use Poppins 600 for medium emphasis subheadings
            - Use Poppins 700 for bold headings and CTAs
          PROMPT
          constraints = [
            "Do not use for primary headings - too light and lacks visual hierarchy",
            "Do not use for CTA buttons - insufficient weight for calls to action",
            "Ensure sufficient contrast ratio for accessibility (WCAG AA minimum 4.5:1)",
            "Must ensure Google Fonts CSS includes Poppins:wght@300 before use",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_font_poppins_400" {
  name = "ui_font_poppins_400"
  description = "Poppins Regular (400) for body text and standard content"
  requirements = [
    {
      name = "Poppins Font Weight 400"
      instructions = [
        {
          prompt = <<-PROMPT
            **Font Metadata:**
            - **Font Family:** Poppins
            - **Font Weight:** 400 (Regular)
            - **Source:** google_fonts
            - **Usage Contexts:** body text, paragraphs, standard descriptions, general content
            
            **CSS Implementation:**
            ```css
            font-family: 'Poppins', sans-serif;
            font-weight: 400;
            ```
            
            **Integration Reference:**
            Load from Google Fonts: $${var.integrations.integration_google_fonts.link}
            Ensure CSS includes: Poppins:wght@400
            
            **CRITICAL: Use Module Variable Reference**
            - Fonts are in the `ui` module, which receives integrations as input variables
            - MUST use `var.integrations.integration_google_fonts` NOT `tofukit_feature.integration_google_fonts`
            - The integration resource is defined at root level and passed to the module via `integrations` variable
            
            **When to Use:**
            - Main body text for paragraphs
            - Standard content descriptions (e.g., "The happiest rabbits get the crunchiest carrots...")
            - General marketing copy
            - Footer text
            
            **When to Use Other Weights:**
            - Use Poppins 300 for subtle secondary text
            - Use Poppins 600 for medium emphasis headings
            - Use Poppins 700 for primary headings and CTAs
          PROMPT
          constraints = [
            "Do not use for primary headings - insufficient weight for visual hierarchy",
            "Do not use for CTAs - buttons need stronger emphasis with weight 700",
            "Ensure line-height of at least 1.5 for readability",
            "Must ensure Google Fonts CSS includes Poppins:wght@400 before use",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_font_poppins_600" {
  name = "ui_font_poppins_600"
  description = "Poppins SemiBold (600) for subheadings and medium emphasis"
  requirements = [
    {
      name = "Poppins Font Weight 600"
      instructions = [
        {
          prompt = <<-PROMPT
            **Font Metadata:**
            - **Font Family:** Poppins
            - **Font Weight:** 600 (SemiBold)
            - **Font Source:** google_fonts
            - **Usage Contexts:** subheadings, form labels, emphasized content, medium-weight headings
            
            **CSS Implementation:**
            ```css
            font-family: 'Poppins', sans-serif;
            font-weight: 600;
            ```
            
            **Integration Reference:**
            Load from Google Fonts: $${var.integrations.integration_google_fonts.link}
            Ensure CSS includes: Poppins:wght@600
            
            **CRITICAL: Use Module Variable Reference**
            - Fonts are in the `ui` module, which receives integrations as input variables
            - MUST use `var.integrations.integration_google_fonts` NOT `tofukit_feature.integration_google_fonts`
            - The integration resource is defined at root level and passed to the module via `integrations` variable
            
            **When to Use:**
            - H3 and H4 subheadings
            - Form field labels ("Your Name", "Email Address")
            - Section content that needs emphasis
            - Badge text and labels
            
            **When to Use Other Weights:**
            - Use Poppins 400 for standard body text
            - Use Poppins 700 for primary headings (H1, H2) and CTAs
            - Use Poppins 300 for de-emphasized content
          PROMPT
          constraints = [
            "Do not use for primary hero headings - use Poppins 700 for stronger impact",
            "Do not use for long-form body text - use Poppins 400 for better readability",
            "Ensure sufficient line-height (1.4+) for comfortable reading",
            "Must ensure Google Fonts CSS includes Poppins:wght@600 before use",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_font_poppins_700" {
  name = "ui_font_poppins_700"
  description = "Poppins Bold (700) for primary headings and call-to-action buttons"
  requirements = [
    {
      name = "Poppins Font Weight 700"
      instructions = [
        {
          prompt = <<-PROMPT
            **Font Metadata:**
            - **Font Family:** Poppins
            - **Font Weight:** 700 (Bold)
            - **Source:** google_fonts
            - **Usage Contexts:** primary headings (H1, H2), CTAs, hero titles, strong emphasis
            
            **CSS Implementation:**
            ```css
            font-family: 'Poppins', sans-serif;
            font-weight: 700;
            ```
            
            **Integration Reference:**
            Load from Google Fonts: $${var.integrations.integration_google_fonts.link}
            Ensure CSS includes: Poppins:wght@700
            
            **CRITICAL: Use Module Variable Reference**
            - Fonts are in the `ui` module, which receives integrations as input variables
            - MUST use `var.integrations.integration_google_fonts` NOT `tofukit_feature.integration_google_fonts`
            - The integration resource is defined at root level and passed to the module via `integrations` variable
            
            **When to Use:**
            - Main hero headings: "Carrot Labs", "Fresh Carrots Delivered to Your Burrow!"
            - Section headings: "Weather is Always Good in the Burrow", "Garden-Fresh, Always Crunchy"
            - Call-to-action button text: "Get Your Carrots", "Contact Us"
            - Footer heading: "Carrot Labs - Fresh Carrots for Happy Rabbots"
            - Strong emphasis text requiring maximum visual weight
            
            **When to Use Other Weights:**
            - Use Poppins 400 for body text
            - Use Poppins 600 for subheadings and medium emphasis
            - Use Poppins 300 for subtle secondary content
          PROMPT
          constraints = [
            "Do not use for body text - too heavy and impacts readability",
            "Do not use for small text below 14px - becomes difficult to read at bold weight",
            "Ensure sufficient line-height (1.2-1.4) to prevent text from appearing cramped",
            "Must ensure Google Fonts CSS includes Poppins:wght@700 before use",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_font_inter_500" {
  name = "ui_font_inter_500"
  description = "Inter Medium (500) for emphasized UI elements and data values"
  requirements = [
    {
      name = "Inter Font Weight 500"
      instructions = [
        {
          prompt = <<-PROMPT
            **Font Metadata:**
            - **Font Family:** Inter
            - **Font Weight:** 500 (Medium)
            - **Source:** google_fonts
            - **Usage Contexts:** emphasized UI text, data values, button text, active states
            
            **CSS Implementation:**
            ```css
            font-family: 'Inter', sans-serif;
            font-weight: 500;
            ```
            
            **Integration Reference:**
            Load from Google Fonts: $${var.integrations.integration_google_fonts.link}
            Ensure CSS includes: Inter:wght@500
            
            **CRITICAL: Use Module Variable Reference**
            - Fonts are in the `ui` module, which receives integrations as input variables
            - MUST use `var.integrations.integration_google_fonts` NOT `tofukit_feature.integration_google_fonts`
            - The integration resource is defined at root level and passed to the module via `integrations` variable
            
            **When to Use:**
            - Weather widget data values (temperature, humidity percentages, wind speed)
            - Active navigation states
            - Secondary button text
            - Badge labels and emphasized UI components
            - Data display requiring medium emphasis
            
            **When to Use Other Weights:**
            - Use Inter 400 for standard UI text
            - Use Inter 600 for strong emphasis
            - Use Inter 300 for de-emphasized labels
          PROMPT
          constraints = [
            "Do not use for long-form text - use Inter 400 for better readability",
            "Do not use for primary CTAs - use Poppins 700 for stronger impact",
            "Ensure sufficient contrast especially in smaller sizes",
            "Must ensure Google Fonts CSS includes Inter:wght@500 before use",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_font_inter_300" {
  name = "ui_font_inter_300"
  description = "Inter Light (300) for subtle interface text and metadata"
  requirements = [
    {
      name = "Inter Font Weight 300"
      instructions = [
        {
          prompt = <<-PROMPT
            **Font Metadata:**
            - **Font Family:** Inter
            - **Font Weight:** 300 (Light)
            - **Source:** google_fonts
            - **Usage Contexts:** metadata labels, timestamps, subtle UI text, secondary information
            
            **CSS Implementation:**
            ```css
            font-family: 'Inter', sans-serif;
            font-weight: 300;
            ```
            
            **Integration Reference:**
            Load from Google Fonts: $${var.integrations.integration_google_fonts.link}
            Ensure CSS includes: Inter:wght@300
            
            **CRITICAL: Use Module Variable Reference**
            - Fonts are in the `ui` module, which receives integrations as input variables
            - MUST use `var.integrations.integration_google_fonts` NOT `tofukit_feature.integration_google_fonts`
            - The integration resource is defined at root level and passed to the module via `integrations` variable
            
            **When to Use:**
            - Weather widget metadata labels ("Humidity", "Wind Speed", "Feels Like", "Pressure")
            - Timestamps and date information
            - Secondary UI labels that should be de-emphasized
            - Small caption text
            
            **When to Use Other Weights:**
            - Use Inter 400 for standard UI text
            - Use Inter 500 for medium-weight interface elements
            - Use Inter 600 for emphasized UI components
          PROMPT
          constraints = [
            "Do not use for primary content - too light for main text",
            "Do not use below 12px font size - becomes illegible at light weight",
            "Ensure sufficient contrast ratio (WCAG AA: 4.5:1 minimum)",
            "Must ensure Google Fonts CSS includes Inter:wght@300 before use",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_font_inter_400" {
  name = "ui_font_inter_400"
  description = "Inter Regular (400) for standard UI text and interface elements"
  requirements = [
    {
      name = "Inter Font Weight 400"
      instructions = [
        {
          prompt = <<-PROMPT
            **Font Metadata:**
            - **Font Family:** Inter
            - **Font Weight:** 400 (Regular)
            - **Source:** google_fonts
            - **Usage Contexts:** UI text, form inputs, standard interface text, general application content
            
            **CSS Implementation:**
            ```css
            font-family: 'Inter', sans-serif;
            font-weight: 400;
            ```
            
            **Integration Reference:**
            Load from Google Fonts: $${var.integrations.integration_google_fonts.link}
            Ensure CSS includes: Inter:wght@400
            
            **CRITICAL: Use Module Variable Reference**
            - Fonts are in the `ui` module, which receives integrations as input variables
            - MUST use `var.integrations.integration_google_fonts` NOT `tofukit_feature.integration_google_fonts`
            - The integration resource is defined at root level and passed to the module via `integrations` variable
            
            **When to Use:**
            - Form input text and placeholders
            - Standard UI labels and descriptions
            - Navigation menu items
            - Footer links and general UI text
            - Interface components requiring neutral weight
            
            **When to Use Other Weights:**
            - Use Inter 300 for de-emphasized metadata
            - Use Inter 500 for medium emphasis UI elements
            - Use Inter 600 for strong UI emphasis
          PROMPT
          constraints = [
            "Do not use for primary marketing headings - use Poppins 700 instead",
            "Do not mix with Poppins for body content - maintain consistent font family per content type",
            "Ensure line-height of at least 1.5 for comfortable reading",
            "Must ensure Google Fonts CSS includes Inter:wght@400 before use",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_font_inter_600" {
  name = "ui_font_inter_600"
  description = "Inter SemiBold (600) for strong UI emphasis and important data"
  requirements = [
    {
      name = "Inter Font Weight 600"
      instructions = [
        {
          prompt = <<-PROMPT
            **Font Metadata:**
            - **Font Family:** Inter
            - **Font Weight:** 600 (SemiBold)
            - **Source:** google_fonts
            - **Usage Contexts:** strong UI emphasis, important data values, highlighted elements, critical information
            
            **CSS Implementation:**
            ```css
            font-family: 'Inter', sans-serif;
            font-weight: 600;
            ```
            
            **Integration Reference:**
            Load from Google Fonts: $${var.integrations.integration_google_fonts.link}
            Ensure CSS includes: Inter:wght@600
            
            **CRITICAL: Use Module Variable Reference**
            - Fonts are in the `ui` module, which receives integrations as input variables
            - MUST use `var.integrations.integration_google_fonts` NOT `tofukit_feature.integration_google_fonts`
            - The integration resource is defined at root level and passed to the module via `integrations` variable
            
            **When to Use:**
            - Weather temperature display ("21°")
            - Critical data values requiring strong emphasis
            - Important UI notifications
            - Selected/active state indicators
            - Key metrics and statistics
            
            **When to Use Other Weights:**
            - Use Inter 400 for standard UI text
            - Use Inter 500 for medium emphasis data
            - Use Inter 300 for subtle metadata labels
          PROMPT
          constraints = [
            "Do not overuse - reserve for truly important information to maintain hierarchy",
            "Do not use for marketing headings - use Poppins 700 for brand consistency",
            "Do not use for body text - creates visual fatigue with heavy weight",
            "Must ensure Google Fonts CSS includes Inter:wght@600 before use",
          ]
        },
      ]
    },
  ]
}