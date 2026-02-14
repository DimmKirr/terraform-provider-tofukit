resource "tofukit_feature" "ui_color_scheme" {
  name = "ui_color_scheme"
  description = "Warm accent colors with soft pastel backgrounds - friendly and approachable palette"
  requirements = [
    {
      name = "Color Palette Implementation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Color Scheme Metadata:**
            - **Palette Approach:** Warm primary accents + soft pastel backgrounds + neutrals
            - **Harmony Strategy:** Warm orange primary balanced with cool mint green sections, creating visual rhythm and section separation. High contrast CTAs against soft backgrounds.
            - **Semantic Usage:**
              - primary: #FF6B35 (warm orange) - CTAs, headlines, brand accent, key interactive elements
              - secondary: #4CAF50 (green) - positive indicators, success states, environmental/fresh theme
              - background-primary: #E8F5E9 (soft mint green) - alternating section backgrounds, creates visual separation
              - background-secondary: #F5F5F5 (light gray) - alternating section backgrounds, neutral content areas
              - background-card: #FFFFFF (white) - card backgrounds, form containers, elevated content
              - background-footer: #2E7D32 (dark green) - footer section, grounding element
              - text-primary: #212121 (near black) - headings, important text, high contrast
              - text-secondary: #757575 (medium gray) - body text, descriptions, supporting content
              - text-light: #FFFFFF (white) - text on dark backgrounds (footer, CTAs)
              - border: #E0E0E0 (light gray) - form inputs, subtle separators
            
            **Implementation Instructions:**
            Create color palette following $${tofukit_feature.ui_design_style.link} with warm, friendly accent colors:
            
            **Primary Colors:**
            - Use warm orange (#FF6B35) for all CTAs and brand moments
            - Apply green (#4CAF50) for positive states and fresh/natural indicators
            - Reserve dark green (#2E7D32) exclusively for footer grounding
            
            **Background Strategy:**
            - Alternate between mint green (#E8F5E9) and light gray (#F5F5F5) for section backgrounds
            - Use white (#FFFFFF) for elevated cards and form containers
            - Create visual rhythm through alternating colored sections
            
            **Text Hierarchy:**
            - Dark gray (#212121) for headings and emphasis
            - Medium gray (#757575) for body text and descriptions
            - White text on dark backgrounds (footer, buttons)
            - Ensure WCAG AA contrast ratios on all combinations
          PROMPT
          constraints = [
            "Do not use low-contrast color combinations - ensure minimum 4.5:1 contrast for body text",
            "Do not exceed 6 main colors (excluding shades) - maintain simple, friendly palette",
            "Do not use pure black (#000000) - use dark gray (#212121) for softer appearance",
            "Do not apply orange to large background areas - reserve for accents and CTAs only",
            "Do not mix warm and cool greens - use consistent green family for cohesion",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_layout_system" {
  name = "ui_layout_system"
  description = "Centered container with card-based modular sections and 8px spacing system"
  requirements = [
    {
      name = "Layout System Implementation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Layout System Metadata:**
            - **Grid Approach:** Flexbox + CSS Grid for card layouts, single-column mobile-first approach
            - **Spacing System:** 8px base unit (8px, 16px, 24px, 32px, 48px, 64px) for consistent rhythm
            - **Container Strategy:** Centered max-width container (max-w-7xl ~1280px), full-width colored sections
            - **Responsive Philosophy:** Mobile-first with progressive enhancement, single-column mobile, multi-column desktop
            
            **Layout Structure:**
            - **Container:** max-width 1280px, centered with auto margins, 16-24px horizontal padding on mobile
            - **Section Spacing:** 48-64px vertical spacing between major sections
            - **Card Spacing:** 24-32px padding within cards, 16-24px margin between cards
            - **Content Spacing:** 16px between related elements, 24-32px between content blocks
            
            **Grid Patterns:**
            - **Hero Section:** Single column, centered content, full-width background
            - **Content Sections:** Alternating image-text layouts (50/50 on desktop, stacked on mobile)
            - **Weather Cards:** CSS Grid with 2x2 layout on desktop, single column on mobile
            - **Testimonials:** Vertical stack of quote cards with consistent spacing
            - **Form Section:** Centered single-column form with max-width constraint
            
            **Responsive Breakpoints:**
            - Mobile: 0-767px (single column, stacked layouts)
            - Tablet: 768-1023px (2-column grids where appropriate)
            - Desktop: 1024px+ (full multi-column layouts, alternating image-text)
            
            **Implementation Instructions:**
            Create layout system following $${tofukit_feature.ui_design_style.link} with modular, card-based sections:
            
            **Container Setup:**
            - Main container: max-width 1280px, margin 0 auto, padding 0 16px (mobile) / 0 24px (desktop)
            - Full-width sections: width 100%, colored backgrounds extend edge-to-edge
            - Inner content: constrained to container max-width
            
            **Section Structure:**
            - Each section: padding 48-64px vertical, alternating background colors
            - Section inner: max-width container, centered alignment
            - Section spacing: consistent 48-64px rhythm between sections
            
            **Card Layouts:**
            - Card container: background white, border-radius 8-12px, padding 24-32px
            - Card shadow: subtle elevation (0 2px 8px rgba(0,0,0,0.1))
            - Card grid: CSS Grid or Flexbox with gap 16-24px
            - Responsive: stack cards vertically on mobile, grid on desktop
            
            **Spacing Scale:**
            - xs: 8px (tight spacing, icon gaps)
            - sm: 16px (related elements, card gaps)
            - md: 24px (content blocks, card padding)
            - lg: 32px (section padding, major separations)
            - xl: 48px (section vertical spacing)
            - 2xl: 64px (major section breaks)
            
            **Image-Text Alternating:**
            - Desktop: 50/50 split, image left/right alternates per section
            - Mobile: Image stacked above text, full-width
            - Alignment: Center-aligned on mobile, left-aligned on desktop
          PROMPT
          constraints = [
            "Do not use container width below 320px - minimum mobile support",
            "Do not exceed max-width 1280px for main container - maintains readability",
            "Do not use spacing values outside 8px scale - breaks visual rhythm",
            "Do not stack more than 3 levels of nested containers - keeps DOM clean",
            "Do not apply fixed heights to content containers - allow natural flow",
            "Do not use horizontal scroll - all content must fit viewport width",
            "Do not remove vertical spacing between sections - maintains breathing room",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_design_style" {
  name = "ui_design_style"
  description = "Modern minimalist design with playful editorial elements and bold accent colors"
  requirements = [
    {
      name = "Design System Implementation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Design Style Metadata:**
            - **Primary Style:** Hybrid: Minimalism + Editorial Design
            - **Influences:** Minimalism (clean layouts, whitespace, card-based sections), Editorial Design (image-text pairings, magazine-like content blocks), Flat Design (solid colors, minimal shadows)
            - **Characteristics:** Clean whitespace with generous padding, card-based modular layouts, bold sans-serif typography, playful illustrated content (bunny characters), warm color palette with orange accent, soft pastel backgrounds (mint green, light gray)
            - **Distinctiveness:** Combines professional minimalist structure with playful, friendly illustrations and warm color accents. Uses editorial content blocks with alternating image-text layouts. Soft, approachable aesthetic through rounded corners and friendly illustrations.
            - **Philosophy:** Clean and accessible with friendly, approachable personality. Professional structure meets playful content delivery. Focus on readability and visual hierarchy while maintaining warmth and character.
            
            **Common Web Design Styles Analysis:**
            - Minimalism markers present: Abundant whitespace, limited color palette (orange, green, neutrals), clean sans-serif typography, simple card layouts
            - Flat Design markers present: Solid colors, minimal shadows (subtle card elevation only), no gradients on main elements, simple geometric shapes
            - Editorial markers present: Magazine-style content blocks, alternating image-text layouts, strong typography hierarchy, content-focused design
            - NOT Neo-Brutalism: No thick black borders, uses rounded corners (not sharp), subtle shadows (not hard offset)
            - NOT Glassmorphism: No frosted glass blur, no transparency effects, solid backgrounds
            - NOT Material Design: No FAB buttons, not using Roboto, different elevation approach
            
            **Implementation Instructions:**
            Create design system following modern minimalist approach with editorial content blocks:
            
            **Layout Structure:**
            - Card-based modular sections with generous whitespace between
            - Alternating image-text layouts for content variety
            - Centered content with max-width container
            - Consistent padding within cards (24-32px)
            - Full-width sections with subtle background colors (mint green #E8F5E9, light gray #F5F5F5)
            
            **Visual Treatment:**
            - Rounded corners on cards and buttons (8-12px border-radius)
            - Subtle card elevation with soft shadows (0 2px 8px rgba(0,0,0,0.1))
            - Solid color backgrounds (no gradients on main elements)
            - White cards on colored section backgrounds for depth
            
            **Interactive Elements:**
            - Bold orange CTA buttons (#FF6B35 range) with white text
            - Rounded button styling (8px border-radius)
            - Hover states with subtle color shifts
            - Form inputs with light borders and rounded corners
            
            **Content Approach:**
            - Mix of photography (carrots, garden) and friendly illustrations (bunny characters)
            - Strong typography hierarchy (large headings, readable body text)
            - Editorial content blocks with balanced image-text ratios
            - Playful, friendly tone through illustrations and warm colors
          PROMPT
          constraints = [
            "Do not use harsh shadows or thick borders - maintain soft, approachable aesthetic",
            "Do not apply gradients to primary UI elements - keep backgrounds solid and clean",
            "Do not sacrifice whitespace for content density - generous spacing is core to the design",
            "Do not use more than 4 accent colors (orange, green, mint, plus neutrals) - maintain limited palette",
            "Do not mix illustration styles - keep consistent friendly character style",
            "Do not use sharp corners on interactive elements - rounded corners essential for friendly feel",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_typography_system" {
  name = "ui_typography_system"
  description = "Clean sans-serif hierarchy with strong headings and readable body text"
  requirements = [
    {
      name = "Typography System Implementation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Typography System Metadata:**
            - **Font Pairing:** Sans-serif for both headings and body (modern, clean, accessible)
            - **Hierarchy:** 32-36px (H1 hero) / 24-28px (H2 section) / 18-20px (H3 subsection) / 16px (body) / 14px (caption/meta)
            - **Scale Rhythm:** 1.33 modular scale (perfect fourth) for balanced progression
            - **Line Height:** 1.5-1.6 for body text (readability), 1.2-1.3 for headings (tighter, impactful)
            - **Font Weights:** Bold (700) for headings, Regular (400) for body, Medium (500) for emphasis
            
            **Typography Roles:**
            - **Hero Heading (H1):** 32-36px, bold (700), tight line-height (1.2), used for main page headline and primary CTAs
            - **Section Heading (H2):** 24-28px, bold (700), line-height (1.3), used for content block titles
            - **Subsection (H3):** 18-20px, bold (700), line-height (1.3), used for card titles and form labels
            - **Body Text:** 16px, regular (400), line-height (1.5-1.6), used for paragraphs and descriptions
            - **Caption/Meta:** 14px, regular (400), line-height (1.5), used for footer text and fine print
            - **Button Text:** 16px, medium/bold (500-700), used for CTAs and interactive elements
            
            **Implementation Instructions:**
            Create typography system following $${tofukit_feature.ui_design_style.link} with clean, accessible text hierarchy:
            
            **Font Selection:**
            - Primary sans-serif: Use system font stack or modern sans (Inter, Roboto, Open Sans)
            - Fallback: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif
            - No decorative or script fonts - maintain clean, readable aesthetic
            
            **Heading Treatment:**
            - Bold weight (700) for all headings
            - Dark gray color (#212121) for maximum contrast
            - Generous margin-bottom for breathing room (16-24px)
            - Tight line-height (1.2-1.3) for visual impact
            
            **Body Text:**
            - Regular weight (400) for comfortable reading
            - Medium gray (#757575) for reduced eye strain
            - Optimal line-height (1.5-1.6) for readability
            - Max-width 65-75 characters for optimal reading length
            
            **Responsive Scaling:**
            - Scale down heading sizes by 20-30% on mobile
            - Maintain body text at 16px minimum for accessibility
            - Adjust line-height slightly tighter on small screens
          PROMPT
          constraints = [
            "Do not use serif fonts - breaks minimalist aesthetic",
            "Do not set body text below 16px - ensures mobile readability",
            "Do not use more than 3 font weights - maintains simplicity",
            "Do not apply ALL CAPS to long text - reduces readability",
            "Do not use line-height below 1.2 for headings or 1.4 for body - compromises accessibility",
            "Do not exceed 75 characters per line - optimal reading length is 65-75 characters",
          ]
        },
      ]
    },
  ]
}