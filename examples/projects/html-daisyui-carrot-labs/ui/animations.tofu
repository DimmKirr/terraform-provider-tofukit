resource "tofukit_feature" "ui_animation_input_focus_ring" {
  name = "ui_animation_input_focus_ring"
  description = "Form input focus ring animation with smooth border transition"
  requirements = [
    {
      name = "Input Focus Ring Animation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Animation Metadata:**
            - **Type:** focus state micro-interaction
            - **Effect:** border color transition + subtle shadow glow
            - **Trigger:** :focus, :focus-visible states
            - **Duration:** 200ms
            - **Easing:** ease-out
            - **Applied To:** Text inputs in contact form (Name, Email Address)
            
            **CSS:**
            ```css
            .input {
              border: 2px solid transparent;
              background-color: rgba(255, 255, 255, 0.9);
              transition: border-color 200ms ease-out, box-shadow 200ms ease-out;
            }
            
            .input:focus, .input:focus-visible {
              border-color: #FF6B35;
              box-shadow: 0 0 0 3px rgba(255, 107, 53, 0.1);
              outline: none;
            }
            
            .input:hover:not(:focus) {
              border-color: rgba(255, 107, 53, 0.3);
            }
            ```
            
            **Accessibility:**
            - Clear focus indicator for keyboard navigation
            - Color contrast ratio meets WCAG AA standards
            - Works with :focus-visible for keyboard-only focus styling
            
            **Performance:** Only animates border and shadow (composited properties)
            
            **Usage Context:**
            - Contact form text inputs
            - Email subscription fields
            
            **Related Components:**
            - Applied to form input fields in contact section
          PROMPT
          constraints = [
            "Do not remove outline without providing alternative focus indicator - accessibility violation",
            "Do not use box-shadow values that obscure nearby content",
            "Do not exceed 250ms - input feels unresponsive",
            "Do not use only color change - insufficient for users with color blindness",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_animation_hover_button_primary" {
  name = "ui_animation_hover_button_primary"
  description = "Primary button hover effect with scale and brightness increase"
  requirements = [
    {
      name = "Primary Button Hover Animation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Animation Metadata:**
            - **Type:** hover effect / micro-interaction
            - **Effect:** subtle scale up (scale(1.02)) + brightness increase + shadow enhancement
            - **Trigger:** :hover, :focus states
            - **Duration:** 200ms
            - **Easing:** ease-out
            - **Applied To:** Orange CTA buttons ("Get Your Carrots", "Contact Us")
            
            **CSS:**
            ```css
            .btn-primary {
              background-color: #FF6B35;
              color: white;
              transition: transform 200ms ease-out, box-shadow 200ms ease-out, filter 200ms ease-out;
              box-shadow: 0 2px 4px rgba(255, 107, 53, 0.2);
            }
            
            .btn-primary:hover, .btn-primary:focus {
              transform: scale(1.02);
              filter: brightness(1.1);
              box-shadow: 0 4px 12px rgba(255, 107, 53, 0.4);
            }
            
            .btn-primary:active {
              transform: scale(0.98);
              box-shadow: 0 2px 4px rgba(255, 107, 53, 0.3);
            }
            
            .btn-primary:focus-visible {
              outline: 2px solid #FF6B35;
              outline-offset: 2px;
            }
            ```
            
            **Accessibility:** Works with keyboard focus (:focus-visible), not just mouse hover
            
            **Performance:** GPU-accelerated properties only (transform, filter are composited)
          PROMPT
          constraints = [
            "Do not use scale >1.05 - too dramatic and feels unprofessional",
            "Do not skip :active state - lacks tactile feedback for clicks",
            "Do not omit :focus-visible - keyboard users need clear visual feedback",
            "Do not exceed 250ms duration - button feels unresponsive",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_animation_hover_card_lift" {
  name = "ui_animation_hover_card_lift"
  description = "Card lift effect on hover with shadow elevation"
  requirements = [
    {
      name = "Card Hover Lift Animation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Animation Metadata:**
            - **Type:** hover effect
            - **Effect:** translateY(-4px) + shadow elevation increase
            - **Trigger:** :hover state on interactive cards
            - **Duration:** 250ms
            - **Easing:** ease-out
            - **Applied To:** Weather card, feature image cards, testimonial cards
            
            **CSS:**
            ```css
            .card {
              transition: transform 250ms ease-out, box-shadow 250ms ease-out;
              box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
            }
            
            .card:hover {
              transform: translateY(-4px);
              box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
            }
            
            @media (prefers-reduced-motion: reduce) {
              .card {
                transition: box-shadow 250ms ease-out;
              }
              .card:hover {
                transform: none;
              }
            }
            ```
            
            **Performance:**
            - GPU-accelerated translateY
            - Shadow uses rgba for smooth transition
            - No layout shifts
            
            **Usage Contexts:**
            - Weather information card
            - Feature section cards with images
            - Testimonial card section
            
            **Accessibility:** Transform disabled for prefers-reduced-motion, shadow change remains for visual feedback
          PROMPT
          constraints = [
            "Do not use translateY values >8px - card feels like it's floating away",
            "Do not animate on mobile touch devices - conflicts with scroll gesture",
            "Do not add cursor: pointer unless card is actually clickable",
            "Do not skip box-shadow transition - causes jarring visual jump",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_animation_hero_logo_spin" {
  name = "ui_animation_hero_logo_spin"
  description = "Carrot logo emoji subtle rotation animation on page load"
  requirements = [
    {
      name = "Hero Logo Spin Animation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Animation Metadata:**
            - **Type:** entrance animation / attention-grabbing micro-interaction
            - **Effect:** single 360-degree rotation on page load
            - **Trigger:** Page load (animation-delay for after hero content fades in)
            - **Duration:** 800ms
            - **Easing:** ease-in-out (smooth start and end)
            - **Delay:** 300ms (wait for hero section to appear first)
            - **Transform:** rotate(0deg) → rotate(360deg)
            - **Applied To:** Carrot emoji logo in hero section
            - **Library/Framework:** CSS @keyframes animation
            
            **CSS Implementation:**
            ```css
            @keyframes spin-once {
              from { transform: rotate(0deg); }
              to { transform: rotate(360deg); }
            }
            
            .hero-logo {
              display: inline-block;
              animation: spin-once 800ms ease-in-out 300ms;
              animation-fill-mode: both;
            }
            
            @media (prefers-reduced-motion: reduce) {
              .hero-logo {
                animation: none;
              }
            }
            ```
            
            **Performance:**
            - GPU-accelerated transform property
            - Single execution (not infinite loop)
            - will-change: transform applied only during animation
            
            **Accessibility:** Respects prefers-reduced-motion
            
            **Usage Context:**
            - Applied to carrot emoji (🥕) in hero section logo
            - Draws attention to brand identity without being distracting
          PROMPT
          constraints = [
            "Do not make infinite loop - becomes annoying and distracting",
            "Do not exceed 1000ms - single rotation should feel snappy",
            "Do not apply to multiple page loads in SPA - only on initial page load",
            "Do not skip prefers-reduced-motion check - accessibility requirement",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_animation_stagger_list_items" {
  name = "ui_animation_stagger_list_items"
  description = "Staggered fade-in animation for weather metrics and testimonial quotes"
  requirements = [
    {
      name = "Staggered List Items Animation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Animation Metadata:**
            - **Type:** entrance animation with stagger effect
            - **Effect:** fade in + slide up with sequential delay
            - **Trigger:** Intersection Observer when section becomes visible
            - **Duration:** 400ms per item
            - **Easing:** ease-out
            - **Delay:** Stagger 100ms per item (nth-child based)
            - **Transform:** opacity 0 → 1, translateY(15px) → translateY(0)
            - **Applied To:** Weather metrics (humidity, wind, feels like, pressure), testimonial quotes
            
            **CSS Implementation:**
            ```css
            .stagger-item {
              opacity: 0;
              transform: translateY(15px);
              transition: opacity 400ms ease-out, transform 400ms ease-out;
            }
            
            .stagger-container.is-visible .stagger-item:nth-child(1) {
              transition-delay: 0ms;
            }
            
            .stagger-container.is-visible .stagger-item:nth-child(2) {
              transition-delay: 100ms;
            }
            
            .stagger-container.is-visible .stagger-item:nth-child(3) {
              transition-delay: 200ms;
            }
            
            .stagger-container.is-visible .stagger-item:nth-child(4) {
              transition-delay: 300ms;
            }
            
            .stagger-container.is-visible .stagger-item {
              opacity: 1;
              transform: translateY(0);
            }
            
            @media (prefers-reduced-motion: reduce) {
              .stagger-item {
                opacity: 1;
                transform: none;
                transition: none;
              }
            }
            ```
            
            **JavaScript Trigger:**
            ```javascript
            const observer = new IntersectionObserver((entries) => {
              entries.forEach(entry => {
                if (entry.isIntersecting) {
                  entry.target.classList.add('is-visible');
                }
              });
            }, { threshold: 0.3 });
            
            document.querySelectorAll('.stagger-container').forEach(el => observer.observe(el));
            ```
            
            **Performance:** GPU-accelerated, minimal reflow
            
            **Usage Contexts:**
            - Weather metrics grid (humidity: 70%, wind: 7 km/h, feels like: 22°, pressure: 1019 hPa)
            - Customer testimonial quotes in testimonial section
            
            **Visual Effect:** Creates cascading reveal that guides eye through content naturally
          PROMPT
          constraints = [
            "Do not exceed 150ms stagger delay - total animation takes too long",
            "Do not apply to more than 6-8 items - becomes tedious to watch",
            "Do not use for critical information that needs immediate visibility",
            "Do not skip nth-child limits - could cause excessive delays for long lists",
          ]
        },
      ]
    },
  ]
}

resource "tofukit_feature" "ui_animation_fade_in_scroll" {
  name = "ui_animation_fade_in_scroll"
  description = "Fade-in animation triggered when content sections enter viewport during scroll"
  requirements = [
    {
      name = "Fade In on Scroll Animation"
      instructions = [
        {
          prompt = <<-PROMPT
            **Animation Metadata:**
            - **Type:** entrance animation
            - **Effect:** fade in (opacity 0 → 1) with subtle upward movement (translateY(20px) → translateY(0))
            - **Trigger:** Intersection Observer (element enters viewport at 20% threshold)
            - **Duration:** 600ms
            - **Easing:** ease-out (cubic-bezier(0.16, 1, 0.3, 1))
            - **Delay:** Stagger 150ms per section for sequential reveal
            - **Transform:** translateY(20px) → translateY(0) + opacity 0 → 1
            - **Applied To:** Content sections (weather card, feature sections, testimonial section, form section)
            - **Library/Framework:** Custom CSS + Intersection Observer API
            - **CSS Implementation:**
            ```css
            .fade-in-scroll {
              opacity: 0;
              transform: translateY(20px);
              transition: opacity 600ms cubic-bezier(0.16, 1, 0.3, 1), transform 600ms cubic-bezier(0.16, 1, 0.3, 1);
            }
            
            .fade-in-scroll.is-visible {
              opacity: 1;
              transform: translateY(0);
            }
            ```
            - **JavaScript Trigger:**
            ```javascript
            const observer = new IntersectionObserver((entries) => {
              entries.forEach(entry => {
                if (entry.isIntersecting) {
                  entry.target.classList.add('is-visible');
                }
              });
            }, { threshold: 0.2 });
            
            document.querySelectorAll('.fade-in-scroll').forEach(el => observer.observe(el));
            ```
            - **Performance:**
              - Uses GPU-accelerated properties (opacity, transform)
              - will-change: transform, opacity for actively scrolling elements
              - Respects prefers-reduced-motion media query
            - **Accessibility:**
            ```css
            @media (prefers-reduced-motion: reduce) {
              .fade-in-scroll {
                transition: none;
                opacity: 1;
                transform: none;
              }
            }
            ```
            
            **Usage Contexts:**
            - Weather information card
            - Feature sections (Garden-Fresh, Lightning-Fast Delivery)
            - Testimonial section (Thousands of Happy Hoppers)
            - Contact form section
            
            **Related Components:**
            - Applied to: section containers, card components, content blocks
          PROMPT
          constraints = [
            "Do not animate layout properties (width, height, margin, padding) - causes reflow and performance issues",
            "Do not exceed 600ms duration - feels sluggish and breaks user flow",
            "Do not disable for prefers-reduced-motion users - accessibility violation",
            "Do not apply to all elements simultaneously - loses visual hierarchy and feels jarring",
          ]
        },
      ]
    },
  ]
}