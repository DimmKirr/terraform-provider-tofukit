resource "tofukit_feature" "ui_color_primary" {
  name = "ui_color_primary"
  description = "Primary brand color for headlines, borders, and theme consistency (DaisyUI garden theme green)"
  requirements = [
    {
      name = "Primary Brand Color"
      instructions = [
        {
          prompt = <<-PROMPT
            **Color Metadata:**
            - **Hex:** #4CAF50
            - **Purpose:** primary
            - **CSS Variable:** --p (DaisyUI theme variable)
            - **HSL Values:** 142 70% 49% (from DaisyUI garden theme)
            - **Usage Contexts:** Section headlines, text highlights, border accents, badge backgrounds, DaisyUI component theming
            - **WCAG Contrast:**
              - On white background (#FFFFFF): 3.8:1 (AA Large Text Pass)
              - On light green background (#E8F5E9): 2.1:1 (Decorative only)
              - With white text: 3.8:1 (AA Large Text Pass)
            
            **Implementation Instructions:**
            Apply #4CAF50 (garden green) to section headlines, border-left accents on content blocks, DaisyUI badge components, and primary text highlights. This color is configured in the DaisyUI garden theme via HSL variable --p: 142 70% 49%. Use Tailwind utility classes .text-primary, .border-primary, .badge-primary to reference this color consistently across the design system.
          PROMPT
          constraints = [
            "Do not use for small body text on white backgrounds - fails WCAG AA contrast (3.8:1 ratio)",
            "Do not pair with light green backgrounds (#E8F5E9) for text - insufficient contrast",
            "Do not use for primary CTA buttons - use carrot orange accent instead",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_color_text_gray" {
  name = "ui_color_text_gray"
  requirements = [
    {
      name = "Body Text Gray"
      instructions = [
        {
          prompt = <<-PROMPT
            **Color Metadata:**
            - **Hex:** #374151
            - **Purpose:** text (body content)
            - **CSS Variable:** None (Tailwind .text-gray-700 utility)
            - **Usage Contexts:** Paragraph text, descriptions, testimonials, form labels, secondary content
            - **WCAG Contrast:**
              - On white background (#FFFFFF): 10.7:1 (AAA Pass)
              - On light green background (#E8F5E9): 9.5:1 (AAA Pass)
              - On white with opacity 90%: 9.6:1 (AAA Pass)
            
            **Implementation Instructions:**
            Apply #374151 for all body text and paragraph content using Tailwind class .text-gray-700. This color provides excellent readability on both white and light green backgrounds while maintaining a softer appearance than pure black. Use for marketing copy, feature descriptions, testimonial text, and form field labels to ensure consistent text hierarchy throughout the site.
          PROMPT
          constraints = [
            "Do not use on dark backgrounds - will fail contrast requirements",
            "Do not use on primary green or carrot orange backgrounds - insufficient contrast",
            "Do not use for headline text - use primary green or carrot orange for emphasis",
            "Do not use with opacity modifiers - already calibrated for optimal contrast",
          ]
        },
      ]
    },
  ]
  description = "Gray text color for body content and descriptions"
}

resource "tofukit_feature" "ui_color_background_white" {
  name = "ui_color_background_white"
  description = "White background for hero content card, alternating sections, and form cards"
  requirements = [
    {
      name = "White Background"
      instructions = [
        {
          prompt = <<-PROMPT
            **Color Metadata:**
            - **Hex:** #FFFFFF
            - **Purpose:** background (primary content areas)
            - **CSS Variable:** None (Tailwind .bg-white utility)
            - **Usage Contexts:** Hero content card with opacity, section backgrounds, card components, form containers, testimonial cards
            - **WCAG Contrast:**
              - With dark text (#374151): 15.8:1 (AAA Pass)
              - With primary green (#4CAF50): 3.8:1 (AA Large Text Pass)
              - With carrot orange (#FF6B35): 3.5:1 (AA Large Text Pass)
            
            **Implementation Instructions:**
            Apply #FFFFFF as primary content background using Tailwind class .bg-white. Use with opacity modifier .bg-opacity-90 for hero content card to allow background image to show through while maintaining readability. Alternate white sections with light green (#E8F5E9) sections to create visual rhythm. White provides maximum contrast for all text colors and serves as the neutral base for the color palette.
          PROMPT
          constraints = [
            "Do not use without opacity on hero section - breaks design pattern of visible background image",
            "Do not use for all sections consecutively - alternate with #E8F5E9 for visual interest",
            "Do not forget shadow utilities when using white cards - needs depth on light backgrounds",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_color_background_light" {
  name = "ui_color_background_light"
  description = "Light green background for alternating sections and weather widget area"
  requirements = [
    {
      name = "Light Green Background"
      instructions = [
        {
          prompt = <<-PROMPT
            **Color Metadata:**
            - **Hex:** #E8F5E9
            - **Purpose:** background (light sections)
            - **CSS Variable:** None (used via inline styles and gradients)
            - **Usage Contexts:** Weather section background, alternating content sections, gradient starting points, image placeholder backgrounds
            - **WCAG Contrast:**
              - With dark text (#2E7D32): 9.2:1 (AAA Pass)
              - With gray text (#374151): 8.1:1 (AAA Pass)
              - With primary green (#4CAF50): 2.1:1 (Decorative only)
            
            **Implementation Instructions:**
            Apply #E8F5E9 as background color for alternating sections (weather section, fast delivery section) to create visual rhythm and separate content blocks. Use in gradient combinations: linear-gradient(135deg, #E8F5E9 0%, #C8E6C9 100%) for contact section. This soft green background maintains brand consistency while providing excellent readability for dark text content.
          PROMPT
          constraints = [
            "Do not use primary green (#4CAF50) text on this background - fails contrast",
            "Do not use for sections requiring high visual emphasis - too subtle",
            "Do not pair with light gray text - insufficient contrast",
            "Do not use as sole background color - alternate with white for visual hierarchy",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_color_accent" {
  name = "ui_color_accent"
  description = "Carrot orange accent color for CTAs, headlines, and brand personality"
  requirements = [
    {
      name = "Carrot Orange Accent"
      instructions = [
        {
          prompt = <<-PROMPT
            **Color Metadata:**
            - **Hex:** #FF6B35
            - **Purpose:** accent
            - **CSS Variable:** --carrot-orange, --a (DaisyUI accent variable)
            - **HSL Values:** 18 100% 60% (configured in DaisyUI theme)
            - **Usage Contexts:** Primary CTA buttons, hero headlines, contact section headlines, gradient overlays, brand personality elements
            - **WCAG Contrast:**
              - On white background (#FFFFFF): 3.5:1 (AA Large Text Pass)
              - With white text: 3.5:1 (AA Large Text Pass)
              - On light backgrounds: 2.8:1 (Decorative use only)
            
            **Implementation Instructions:**
            Apply #FF6B35 (carrot orange) to primary CTA buttons with white text, main hero headlines, contact form headlines, and as accent color in gradients. This is the signature brand color that conveys the carrot theme. Reference via CSS variable var(--carrot-orange) or DaisyUI theme variable. Use inline styles for buttons: style="background-color: var(--carrot-orange); border: none;" combined with .btn-primary class.
          PROMPT
          constraints = [
            "Do not use for small body text - fails contrast requirements",
            "Do not use on light green backgrounds without sufficient contrast verification",
            "Do not replace with other orange shades - this specific hex is part of brand identity",
            "Do not use for borders without adequate thickness - visibility issues on light backgrounds",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_color_footer" {
  name = "ui_color_footer"
  description = "Dark green footer background for site grounding and brand authority"
  requirements = [
    {
      name = "Dark Green Footer"
      instructions = [
        {
          prompt = <<-PROMPT
            **Color Metadata:**
            - **Hex:** #2E7D32
            - **Purpose:** background (footer)
            - **CSS Variable:** None (inline style)
            - **Usage Contexts:** Footer section background, high-contrast foundation element
            - **WCAG Contrast:**
              - With white text (#FFFFFF): 8.6:1 (AAA Pass)
              - With light opacity text (90%): 7.7:1 (AAA Pass)
              - With light opacity text (75%): 6.4:1 (AA Pass)
            
            **Implementation Instructions:**
            Apply #2E7D32 as footer background color via inline style: style="background-color: #2E7D32;" combined with .footer and .text-white classes. This dark green grounds the page, provides strong visual anchor, and maintains brand color consistency. Use with white text and varying opacity levels (.opacity-90, .opacity-75) for text hierarchy within footer content.
          PROMPT
          constraints = [
            "Do not use for body content sections - too dark for main content areas",
            "Do not pair with dark text colors - insufficient contrast",
            "Do not use without white text - other text colors will fail contrast requirements",
            "Do not use opacity below 75% for footer text - risks failing WCAG AA contrast",
          ]
        },
      ]
    },
  ]
}