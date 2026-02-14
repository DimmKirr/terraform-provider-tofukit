resource "tofukit_feature" "page_session_tracking" {
  name = "page_session_tracking"
  description = "System tracks user session data using cookies, localStorage, and sessionStorage for analytics and personalization"
  requirements = [
    {
      name = "Session and Preference Tracking"
      instructions = [
        {
          prompt = <<-PROMPT
            **User Story:**
            As a user I am automatically tracked when I visit the page. The system sets cookies, localStorage, and sessionStorage without requiring my consent. I may not be aware this tracking is happening. When I return to the page I am recognized as a returning visitor.
            
            **System Intent:**
            The application wants to track user behavior for analytics and potential personalization. This includes:
            - **Cookies:** Store long-term preference ("carrot_happines": true for 365 days)
            - **LocalStorage:** Store visit count, preferences (favorite carrot type, delivery frequency)
            - **SessionStorage:** Store session ID, browsing start time, page views
            
            This data could be used for:
            - Analytics (tracking returning visitors, session duration)
            - Personalization (remembering user preferences)
            - A/B testing (identifying user cohorts)
            
            **CRITICAL PRIVACY NOTE:**
            This tracking happens WITHOUT user consent, which violates GDPR, CCPA, and other privacy regulations. Production implementation MUST include:
            - Cookie consent banner before setting non-essential cookies
            - Opt-in for analytics tracking
            - Privacy policy disclosure
            - Data retention and deletion mechanisms
            
            **User Actions (Step by Step):**
            1. User loads the page for the first time
            2. JavaScript executes on $(document).ready()
            3. System sets cookie: `carrot_happines=true` (expires in 365 days)
            4. System writes to localStorage:
               - `carrot_preferences`: {favorite_carrot: "orange", delivery_frequency: "weekly", happy_customer: true, visits: 1}
               - `carrot_visits`: "1"
               - `last_visit`: "2025-12-27T21:30:00.000Z"
            5. System writes to sessionStorage:
               - `carrot_session_id`: "CARROT-SESSION-1735333800000-ABC123"
               - `carrot_session_data`: {session_id, user_type: "rabbit", browsing_start, cart_items: 0, page_views: 1}
               - `session_active`: "true"
            6. System logs to console (developer visibility):
               - "🥕 Carrot happiness cookie set!"
               - "💾 LocalStorage set!"
               - "🔐 SessionStorage set!"
            7. User continues browsing (unaware of background tracking)
            8. User closes browser (sessionStorage cleared)
            9. User returns next day (localStorage and cookies persist)
            10. System increments visit count: `carrot_visits=2`
            
            **User Inputs:**
            - **None** - Tracking happens automatically without user interaction
            - **Consent NOT Required** - Current implementation violates privacy regulations
            
            **User Expectations:**
            - User expects transparency about data collection (missing in current implementation)
            - User expects control over tracking (no opt-out mechanism provided)
            - User expects data security (no encryption or protection described)
            - User expects compliance with privacy laws (GDPR, CCPA) - currently non-compliant
            
            **User Success Criteria:**
            - System successfully tracks returning visitors
            - Visit count increments correctly on each return
            - Session data is isolated per browser session
            - Cookies persist across browser restarts
            - LocalStorage survives browser close/reopen
            
            **User Failure Scenarios:**
            - Third-party cookies blocked → Cookie tracking fails silently
            - Private browsing mode → localStorage and sessionStorage don't persist
            - User clears browser data → All tracking data lost, user appears as new visitor
            - Multiple devices → User appears as different visitors (no cross-device tracking)
            - Cookie/localStorage disabled → Tracking fails completely, no fallback mechanism
            
            **Privacy Compliance Requirements (MISSING IN CURRENT IMPLEMENTATION):**
            - ❌ No cookie consent banner before setting cookies
            - ❌ No privacy policy disclosure about data collection
            - ❌ No opt-out mechanism for analytics tracking
            - ❌ No data retention policy or automatic deletion
            - ❌ No encryption or security for stored data
            - ❌ No user access to view/delete their data (GDPR right to erasure)
            
            **Feature Dependencies (Tofukit Links):**
            - **Implemented in:** JavaScript on page load (index.html lines 454-555)
            - **Uses jQuery:** $${var.libraries.jquery.link} for $(document).ready() event binding
            - **Storage Mechanisms:** Browser APIs (document.cookie, localStorage, sessionStorage)
          PROMPT
          constraints = [
            "Do not set cookies without user consent - violates GDPR, CCPA, and other privacy regulations",
            "Do not track users without disclosure - violates FTC transparency requirements",
            "Do not store personal data without security measures - risks data breach liability",
            "Do not use tracking for essential functionality - cookies should only be necessary ones without consent",
            "Do not ignore privacy regulations - implement cookie consent banner before production deployment",
            "Do not log sensitive data to console - visible in browser developer tools",
            "Do not assume localStorage/sessionStorage is always available - some browsers disable it",
            "Do not store unencrypted personal data - localStorage is visible in browser storage inspection",
          ]
        },
      ]
    },
  ]
}