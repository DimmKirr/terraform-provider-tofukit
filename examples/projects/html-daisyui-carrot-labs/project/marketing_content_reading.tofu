resource "tofukit_feature" "marketing_content_reading" {
  name = "marketing_content_reading"
  description = "User reads marketing content sections to learn about product features and benefits"
  requirements = [
    {
      name = "Marketing Content Reading Journey"
      instructions = [
        {
          prompt = <<-PROMPT
            **User Story:**
            As a user I am willing to scroll through multiple content sections to learn about the carrot delivery service. I am willing to read headlines and supporting copy. I can tolerate image placeholders if actual photos don't load. When I finish reading I am going to understand the three main value propositions: freshness, fast delivery, and customer satisfaction.
            
            **User Intent:**
            The user wants to evaluate if this carrot delivery service meets their needs. They're comparing value propositions (freshness, delivery speed) against their expectations. They need to understand what makes this service different from competitors or grocery stores. This is information-gathering mode - user hasn't decided to buy yet, they're still evaluating.
            
            **Marketing Funnel (Awareness → Interest → Evaluation):**
            This content moves users from awareness to interest:
            1. **Awareness** - User discovers site (from ad, search, referral)
            2. **Interest** - User reads hero headline: "Fresh Carrots Delivered to Your Burrow!"
            3. **Exploration** - User scrolls through three feature sections:
               - Section 1: "Garden-Fresh, Always Crunchy" (quality promise)
               - Section 2: "Lightning-Fast Delivery" (speed promise)
               - Section 3: "Thousands of Happy Hoppers" (social proof)
            4. **Evaluation** - User compares features against their needs
            5. **Decision** - User proceeds to testimonials and contact form (next steps)
            
            **User Actions (Step by Step):**
            1. User lands on page (from marketing campaign or search)
            2. User reads hero section value proposition
            3. User scrolls down to first content section: "Garden-Fresh, Always Crunchy"
            4. User reads copy: "Our carrots are harvested at dawn and delivered by lunch. No wilted greens, no sad carrots—just pure orange perfection straight from local farms to your paws."
            5. User processes benefit: freshness promise (harvested at dawn, delivered by lunch)
            6. User continues scrolling to second section: "Lightning-Fast Delivery"
            7. User reads copy: "Our friendly beaver delivery team works around the clock. Order by noon, carrots at your door by dinner. We know you can't wait to munch!"
            8. User processes benefit: speed promise (same-day delivery guarantee)
            9. User continues scrolling to third section: "Thousands of Happy Hoppers"
            10. User reads copy: "Don't just take our word for it! Rabbits across the meadow are raving about our premium carrots. Join the happiest bunny community today!"
            11. User processes benefit: social proof (thousands of customers)
            12. User forms mental model: "Fresh + Fast + Popular = Worth Trying"
            13. User decides to read testimonials or contact form
            
            **User Inputs:**
            - **Scroll/Read** - User navigates through content at their own pace
            - **Visual Processing** - User scans headlines, reads copy, views images
            - **No Form Input** - This is passive content consumption
            
            **User Expectations:**
            - Content is scannable (headlines stand out, short paragraphs)
            - Value propositions are clear and specific (not vague)
            - Visual hierarchy guides attention (large headlines, readable body text)
            - Images support the narrative (carrots, delivery, happy customers)
            - Content flows logically (awareness → features → social proof)
            - Sections are visually separated (alternating backgrounds help)
            
            **User Success Criteria:**
            - User understands the three main benefits (freshness, speed, satisfaction)
            - User feels the service addresses their pain points (stale produce, slow delivery)
            - User continues to testimonials and contact form (engagement indicator)
            - User spends 30-60 seconds per section (sign of reading, not skimming)
            - User remembers key details ("harvested at dawn", "order by noon")
            
            **User Failure Scenarios:**
            - Headlines too generic → User skims without absorbing information
            - Copy too long → User skips sections, misses value propositions
            - No visual separation → User feels overwhelmed by wall of text
            - Vague claims ("best carrots") → User doesn't believe marketing hype
            - Missing images → User loses visual engagement, bounces
            - Content contradicts itself → User notices inconsistency, loses trust
            
            **Feature Dependencies (Tofukit Links):**
            - **Layout uses:** $${var.libraries.tailwind_css.link} for responsive grid and spacing
            - **Styled with:** Alternating backgrounds (white, light green) for visual rhythm
            - **Typography hierarchy:** Large bold headlines with readable body text (describe in text, not as links)
          PROMPT
          constraints = [
            "Do not use vague marketing claims (\"best\", \"amazing\") - user needs specific, believable benefits",
            "Do not create walls of text - break content into short paragraphs (2-3 sentences max)",
            "Do not hide key benefits below the fold - user may not scroll if value isn't clear immediately",
            "Do not use generic stock photos - users recognize inauthentic imagery and lose trust",
            "Do not contradict value propositions across sections - inconsistency breaks credibility",
            "Do not omit visual separation between sections - user gets lost in content hierarchy",
            "Do not sacrifice readability for design - ensure sufficient contrast and font size (16px minimum)",
          ]
        },
      ]
    },
  ]
}