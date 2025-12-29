resource "tofukit_feature" "scroll_to_contact_form" {
  name = "scroll_to_contact_form"
  description = "User clicks CTA button to smoothly scroll down to contact form section"
  requirements = [
    {
      name = "Scroll to Contact Form Journey"
      instructions = [
        {
          prompt = <<-PROMPT
            **User Story:**
            As a user I am willing to click a call-to-action button to quickly navigate to the contact form. I am willing to wait for smooth scroll animation. When the scroll completes I am going to see the contact form centered in my viewport.
            
            **User Intent:**
            The user has consumed marketing content (value proposition, testimonials, product features) and is ready to take action. They want immediate access to the contact form without manually scrolling. This is a high-intent conversion action - the user has decided to engage and wants frictionless access to the next step.
            
            **Marketing Pattern (Awareness → Decision → Action):**
            This is the final stage of the conversion funnel:
            1. **Awareness** - User reads hero headline and value proposition
            2. **Interest** - User explores product features (freshness, delivery speed)
            3. **Decision** - User clicks "Get Your Carrots" CTA (commits to action)
            4. **Action** - User is taken directly to contact form to complete conversion
            
            Skipping manual scroll reduces friction and increases conversion rates by eliminating the cognitive burden of finding the form.
            
            **User Actions (Step by Step):**
            1. User reads hero headline: "Fresh Carrots Delivered to Your Burrow!"
            2. User evaluates value proposition: "Join thousands of bunnies enjoying farm-fresh delivery"
            3. User decides to take action
            4. User clicks orange CTA button: "Get Your Carrots 🥕"
            5. Browser smoothly scrolls to contact form section
            6. Contact form appears centered in viewport
            7. User proceeds to fill out form fields
            
            **User Inputs:**
            - **Click Action:** Single click on "Get Your Carrots 🥕" button
            - **No Form Input Yet:** This journey only handles navigation, not submission
            
            **User Expectations:**
            - Instant response to button click (no delay)
            - Smooth, animated scroll (not instant jump)
            - Form appears centered in viewport after scroll
            - No page refresh or new tab opening
            - Scroll animation completes within 1 second
            - User retains focus on form after scroll completes
            
            **User Success Criteria:**
            - User clicks CTA and scroll begins immediately
            - Smooth scroll animation provides visual continuity
            - Form section is perfectly centered after scroll
            - User can immediately start typing in form fields
            - No layout shift or jumping during scroll
            
            **User Failure Scenarios:**
            - Button doesn't respond to click → User clicks multiple times, gets frustrated
            - Instant jump instead of smooth scroll → Disorienting, user loses context
            - Form not centered after scroll → User has to manually scroll to see full form
            - Scroll animation too slow (>2 seconds) → User feels impatient and may abandon
            - Form fields not ready for input → User has to click again to focus
            
            **Feature Dependencies (Tofukit Links):**
            - **Implementation in:** JavaScript behavior defined in index.html
            - **Targets section:** #contact form section (describe target in text, not as link to avoid cycles)
          PROMPT
          constraints = [
            "Do not use instant scroll (scrollIntoView without smooth behavior) - creates jarring user experience",
            "Do not trigger scroll on page load - only on explicit user action (button click)",
            "Do not scroll if user is already at contact section - wastes user's time",
            "Do not forget to prevent default link behavior - prevents hash navigation conflicts",
            "Do not scroll too slowly (>2 seconds) - user patience threshold is low for navigation actions",
            "Do not scroll too quickly (<300ms) - animation feels abrupt and loses smooth quality",
          ]
        },
      ]
    },
  ]
}