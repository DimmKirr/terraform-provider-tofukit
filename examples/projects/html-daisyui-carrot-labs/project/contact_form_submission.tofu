resource "tofukit_feature" "contact_form_submission" {
  name = "contact_form_submission"
  description = "User submits name and email to express interest in carrot delivery service"
  requirements = [
    {
      name = "Contact Form Submission Journey"
      instructions = [
        {
          prompt = <<-PROMPT
            **User Story:**
            As a user I am willing to submit my contact information to receive more details about the carrot delivery service. I am willing to enter my name (required). I am willing to enter my email address (required). When submitted I am going to see a confirmation alert saying "Thanks for your interest! We'll be in touch soon! 🥕".
            
            **User Intent:**
            The user has been convinced by the marketing content (testimonials, product features, delivery promises) and wants to express interest or get more information. They expect the business to follow up via email. This is a low-friction lead capture - only essential information (name, email) is required, minimizing barriers to conversion.
            
            **Marketing Pattern (Interest → Trust → Conversion):**
            This completes the conversion funnel:
            1. **Interest** - User explored product features and value proposition
            2. **Trust** - User read testimonials and felt confident
            3. **Conversion** - User submits contact information to engage with business
            
            Simplicity is key - no phone number, no long forms. Just name and email to reduce abandonment.
            
            **User Actions (Step by Step):**
            1. User navigates to contact form (via CTA button scroll or manual scroll)
            2. User sees form headline: "Ready for Fresh Carrots?"
            3. User reads form description: "Drop us your details and we'll hop to it!"
            4. User clicks into "Your Name" field
            5. User types their full name (e.g., "Fluffy Cottontail")
            6. User clicks into "Email Address" field
            7. User types their email address (e.g., "fluffy@burrow.com")
            8. User reviews entered information
            9. User clicks orange "Contact Us" button
            10. Browser validates email format (HTML5 validation)
            11. Form submits (prevented default, shows alert)
            12. User sees success message: "Thanks for your interest! We'll be in touch soon! 🥕"
            13. Form fields clear automatically
            14. User can submit again if needed
            
            **User Inputs:**
            - **Name:** Full name (required) - text input, placeholder: "Fluffy Cottontail"
            - **Email:** Email address (required) - email input with HTML5 validation, placeholder: "fluffy@burrow.com"
            - **No Phone Number:** Intentionally omitted to reduce friction
            - **No Message Field:** Simple interest capture, not full inquiry form
            
            **User Expectations:**
            - Clear visual feedback when fields are focused (blue outline)
            - Immediate validation error if email format is invalid
            - Submit button is clearly visible and clickable
            - Confirmation message appears immediately after submission
            - Form clears after successful submission
            - No page reload or navigation away from current page
            - Simple, fast process (<30 seconds from start to finish)
            
            **User Success Criteria:**
            - User enters name and email without errors
            - HTML5 email validation catches format mistakes
            - Success message confirms submission was received
            - Form resets for potential re-submission
            - User feels confident their information was captured
            - No technical errors or broken functionality
            
            **User Failure Scenarios:**
            - Invalid email format (e.g., "fluffy@") → Browser shows validation error ("Please include an '@' in the email address")
            - Empty required fields → Browser prevents submission, highlights missing fields
            - JavaScript disabled → Form may not show custom alert, but validation still works
            - Network error → User sees no feedback (current implementation doesn't handle server errors)
            - Form doesn't clear after submission → User has to manually delete entries to submit again
            
            **Feature Dependencies (Tofukit Links):**
            - **Styled with:** $${var.libraries.tailwind_css.link} for form layout and utility classes
            - **Uses DaisyUI:** $${var.libraries.daisyui.link} for form components (.form-control, .input, .btn)
            - **Implementation in:** JavaScript form handler in index.html
            - **Uses color:** Orange CTA button uses carrot orange accent color (describe in text, not as link to avoid cycles)
          PROMPT
          constraints = [
            "Do not submit form without client-side validation - user needs immediate feedback on invalid email format",
            "Do not require phone number or additional fields - increases abandonment rate significantly",
            "Do not navigate away from page on submission - user loses context and may not see confirmation",
            "Do not show generic error messages - user needs specific guidance (e.g., \"Invalid email format\")",
            "Do not clear form on validation error - user loses their typed content and has to retype everything",
            "Do not disable submit button after submission - user may want to submit multiple times or retry",
            "Do not send data to server without user confirmation - privacy violation (current implementation uses alert)",
            "Do not use alert() in production - provides poor UX, use inline confirmation message instead",
          ]
        },
      ]
    },
  ]
}