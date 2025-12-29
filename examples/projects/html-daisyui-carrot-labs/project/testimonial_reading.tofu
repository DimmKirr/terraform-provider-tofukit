resource "tofukit_feature" "testimonial_reading" {
  name = "testimonial_reading"
  description = "User reads customer testimonials to evaluate service credibility and quality"
  requirements = [
    {
      name = "Testimonial Reading Journey"
      instructions = [
        {
          prompt = <<-PROMPT
            **User Story:**
            As a user I am willing to read testimonials from other customers to evaluate if I can trust this carrot delivery service. After reading testimonials, I am more likely to proceed to the contact form. My goal is to understand if real customers have had positive experiences before I commit my contact information.
            
            **Marketing Pattern (Trust Building → Social Proof → Conversion):**
            This is a classic marketing conversion pattern leveraging social proof:
            1. **Social Proof** - User reads reviews from "Fluffy", "Thumper", and "Cottontail"
            2. **Credibility Check** - User evaluates authenticity (names, locations, specific details)
            3. **Trust Building** - Positive reviews reduce perceived risk
            4. **Decision Point** - User decides service is credible
            5. **Conversion** - User proceeds to contact form with higher confidence
            
            Testimonials are positioned strategically BEFORE the contact form to maximize conversion impact.
            
            **User Intent:**
            The user is in the evaluation phase of their customer journey. They've read the value proposition ("Fresh Carrots Delivered to Your Burrow!") and want validation from other customers before taking action. Testimonials reduce perceived risk and increase conversion rates by showing that real people have tried and enjoyed the service.
            
            **User Actions (Step by Step):**
            1. User scrolls past product features (freshness, delivery speed)
            2. User reaches "Thousands of Happy Hoppers" section
            3. User sees social proof headline validating popularity
            4. User reads section description: "Don't just take our word for it! Rabbits across the meadow are raving about our premium carrots."
            5. User scans three testimonial cards:
               - Card 1: "Best carrots ever! My kits can't get enough!" - Fluffy, Meadow Hills
               - Card 2: "Crunchy, sweet, perfect!" - Thumper, Burrow Lane
               - Card 3: "Delivery is so fast, carrots are still warm from the sun!" - Cottontail, Warren District
            6. User evaluates testimonial authenticity:
               - Real names? ✓ (Fluffy, Thumper, Cottontail)
               - Specific locations? ✓ (Meadow Hills, Burrow Lane, Warren District)
               - Specific details? ✓ ("warm from the sun", "kits can't get enough")
            7. User's perception shifts: skeptical → confident
            8. User decides to proceed to contact form
            9. User scrolls down and submits contact information
            
            **User Inputs:**
            - **None** - This is a passive content consumption feature
            - **Scroll/Read** - User navigates through testimonials at their own pace
            - **Visual Processing** - User evaluates card layout, badges, and testimonial content
            
            **User Expectations:**
            - Testimonials appear authentic (not obviously fake or generic)
            - Multiple testimonials from different customer types (variety builds trust)
            - Specific details that feel genuine ("warm from the sun" vs. "great service")
            - Attribution with names and locations (increases credibility)
            - Visual hierarchy makes testimonials easy to scan
            - Testimonials are recent or timeless (no outdated dates that reduce trust)
            
            **User Success Criteria:**
            - User reads at least 2-3 testimonials
            - User feels more confident about service quality
            - User proceeds to contact form (conversion indicator)
            - User references testimonials in their decision-making ("I saw others liked it...")
            - Testimonials overcome objections ("too expensive" → "worth it for quality")
            
            **User Failure Scenarios:**
            - Testimonials seem fake (generic praise, no details) → User loses trust and leaves site
            - No location attribution → User doubts authenticity ("these could be made up")
            - Too many testimonials (10+) → Overwhelming, user skips section entirely
            - Testimonials contradict product claims → User notices inconsistency, trust breaks
            - All testimonials are 5-star perfect → Feels inauthentic, users expect some variation
            - No testimonials visible → User has no social proof, hesitates to proceed
            
            **Feature Dependencies (Tofukit Links):**
            - **Styled with:** $${var.libraries.daisyui.link} for card components (.card, .card-body, .badge)
            - **Uses Tailwind:** $${var.libraries.tailwind_css.link} for spacing and layout utilities
            - **Leads to:** Contact form section (user proceeds after reading, describe in text not as link)
          PROMPT
          constraints = [
            "Do not show testimonials without customer attribution - anonymous testimonials reduce credibility significantly",
            "Do not overwhelm users with too many testimonials - 3-5 is optimal for conversion without cognitive overload",
            "Do not place testimonials far from contact form - user momentum is lost if they have to search for next step",
            "Do not use obviously stock photos or generic avatars - users recognize fake testimonials immediately",
            "Do not hide negative feedback entirely - addressing concerns builds more trust than perfect 5-star reviews",
            "Do not use generic praise (\"great service\") - specific details (\"warm from the sun\") feel more authentic",
            "Do not omit location information - location attribution increases perceived authenticity",
          ]
        },
      ]
    },
  ]
}