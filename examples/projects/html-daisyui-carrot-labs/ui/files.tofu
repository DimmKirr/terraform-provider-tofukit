resource "tofukit_file" "ui_weather_sun" {
  name = "ui_weather_sun"
  model = "openai/gpt-image-1"
  instructions = [
    {
      prompt = <<-PROMPT
        **Style:** Flat design geometric icon with modern minimalist aesthetic
        
        **Composition:**
        - Geometric composition: circular center with radiating rays
        - Orange/coral color (#FF6B35), no gradients
        - Circular sun body occupying 40% of icon space
        - Eight evenly-spaced triangular rays extending radially from center
        - Clean minimal design, no shadows or 3D effects
        - Flat design aesthetic, modern UI style
        - Scalable design maintaining clarity at multiple sizes
        - High contrast, readable on light backgrounds
        
        **Generation Prompt:**
        
        CRITICAL FORMATTING: Aspect ratio 1:1 square, generate at 1024x1024.
        
        Create a flat design weather icon representing a sun. Use a simple geometric composition with a solid circular center occupying 40% of the icon space in bright orange/coral color (similar to #FF6B35). Surround the center with eight evenly-spaced triangular rays extending radially outward in the same orange/coral color. The rays should be simple geometric triangular shapes pointing outward. Use no gradients, shadows, or 3D effects - just solid flat color. The design should be minimal and clean, following modern flat design UI principles. The icon should be scalable and maintain clarity at different sizes. Ensure the design has high contrast and is easily readable when placed on white or light backgrounds. The style should match modern weather app aesthetics - professional, friendly, and immediately recognizable as a sunny weather indicator.
      PROMPT
      constraints = [
        "Do not use gradients or 3D effects - flat design only",
        "Do not make icon too complex - must be readable at small sizes",
        "Do not use multiple colors - single orange/coral color for consistency",
      ]
    },
  ]
}

resource "tofukit_file" "ui_paper_bag_carrots" {
  name = "ui_paper_bag_carrots"
  instructions = [
    {
      prompt = <<-PROMPT
        **Style:** Natural lifestyle product photography with warm window lighting
        
        **Composition:**
        - Vertical portrait composition: 35% wooden surface, 40% paper bag, 25% green foliage
        - Warm natural window lighting from left creating soft shadows on right
        - Rustic light-toned wooden countertop with natural grain texture
        - Brown kraft paper bag standing upright, naturally crumpled
        - 6-8 vibrant orange carrots with visible texture and natural variations
        - Lush fresh green carrot tops overflowing abundantly from bag
        - Natural arrangement with carrots leaning inside bag at various angles
        - Shallow depth of field, sharp foreground, soft blurred background
        - Warm color temperature (5000-5500K) creating farm-fresh atmosphere
        - Blurred warm beige/cream background suggesting kitchen or farm stand setting
        - Some green leafy plants visible in soft-focus background
        
        **Generation Prompt:**
        
        CRITICAL FORMATTING: Aspect ratio 3:4 portrait (vertical orientation), generate at 768x1024.
        
        Create a professional lifestyle product photograph in vertical portrait orientation showing fresh orange carrots with vibrant green leafy tops overflowing from a brown kraft paper bag on a rustic wooden surface. The composition should be 35% light-toned wooden countertop in foreground with visible natural grain texture, 40% brown paper bag in middle ground, and 25% lush green carrot foliage in upper portion. Use warm natural window lighting from the left side creating soft, gentle shadows on the right side. The wooden surface should have a light honey-brown tone with visible grain patterns and natural character. Show a brown kraft paper grocery bag standing upright, slightly crumpled naturally for authenticity. Arrange 6-8 bright orange carrots with natural texture variations inside the bag with fresh green leafy tops overflowing abundantly and dramatically from the opening. The carrots should lean at natural angles inside the bag. Use shallow depth of field with sharp focus on the foreground carrots and greens, with a soft blur on the background. The background should include blurred green leafy plants and a neutral warm beige/cream wall suggesting a kitchen or farm stand setting. Apply warm color temperature around 5000-5500K to create a cozy, farm-fresh, inviting atmosphere. The overall style should be natural lifestyle product photography that feels authentic, wholesome, and appealing - similar to farm-to-table or organic food marketing imagery.
      PROMPT
      constraints = [
        "Do not use stock photos with watermarks - must feel authentic",
        "Do not crop out green carrot tops - they show freshness",
        "Do not use artificial/studio lighting - natural window light only",
        "Do not over-stage the carrots - arrangement must look natural and organic",
      ]
    },
  ]
  model = "openai/gpt-image-1"
}

resource "tofukit_file" "ui_rabbit_delivery" {
  name = "ui_rabbit_delivery"
  instructions = [
    {
      prompt = <<-PROMPT
        **Style:** 3D rendered character illustration with soft lighting and detailed texturing, similar to Pixar/DreamWorks animation style
        
        **Composition:**
        - Landscape composition: 40% warm indoor background, 60% rabbit character
        - Soft diffused lighting from upper left with gentle highlights on fur
        - Warm indoor kitchen/home setting with beige/tan walls
        - Window visible in background with warm natural light
        - Anthropomorphic rabbit character, 75% of frame height
        - Gray fur with realistic texture and soft shading
        - Large upright ears with pink/salmon inner ear details
        - Friendly expression with large expressive eyes, direct gaze toward viewer
        - Slight three-quarter view showing personality and approachability
        - Small potted carrot plant visible on windowsill in background
        - Warm palette: grays, pinks, beiges, orange carrot accents
        - Professional yet playful character design
        
        **Generation Prompt:**
        
        CRITICAL FORMATTING: Aspect ratio 4:3 landscape (horizontal orientation), generate at 1024x768.
        
        Create a 3D rendered character illustration of a friendly anthropomorphic rabbit in a landscape composition. The scene should be 40% warm indoor background showing a cozy kitchen/home setting with beige/tan walls and a window with warm natural light, and 60% character subject. Use soft diffused lighting from the upper left creating gentle highlights on the rabbit's fur and subtle shadows. The rabbit should have realistic gray fur with soft texture and natural shading transitions. Feature large upright ears with pink/salmon colored inner ear details. Give the character a friendly, welcoming facial expression with large expressive eyes looking directly toward the viewer with a warm, approachable gaze. Position the rabbit in a slight three-quarter view to show personality and depth. The character should occupy approximately 75% of the frame height, positioned slightly right of center. Include a small potted carrot plant on the windowsill in the background to reinforce the brand theme. Use a warm color palette of grays, pinks, beiges, and orange accents from the carrot plant. The overall style should be high-quality 3D rendering similar to Pixar or DreamWorks animation films - professional, approachable, playful, and appealing to a broad family audience. The atmosphere should feel warm, trustworthy, and inviting.
      PROMPT
      constraints = [
        "Do not use low-quality or pixelated rendering - must be high-quality 3D",
        "Do not make character look unfriendly or aggressive - welcoming expression required",
        "Do not use photorealistic style - stylized 3D animation aesthetic only",
        "Do not omit carrot elements - thematic branding is essential",
      ]
    },
  ]
  model = "openai/gpt-image-1"
}

resource "tofukit_file" "ui_logo_carrot" {
  name = "ui_logo_carrot"
  instructions = [
    {
      prompt = <<-PROMPT
        **Style:** Simple flat design icon with clean lines and minimal detail
        
        **Composition:**
        - Square composition with carrot positioned at slight diagonal angle
        - Orange carrot body occupying 60% of icon space
        - Green leafy tops occupying 30% of upper portion
        - White or transparent background
        - Simple geometric shapes with smooth edges
        - Flat design aesthetic with no gradients or shadows
        - Carrot angled from lower-left to upper-right
        - Two to three green leaves sprouting from top
        
        **Generation Prompt:**
        
        CRITICAL FORMATTING: Aspect ratio 1:1 square, generate at 512x512.
        
        Create a simple flat design icon of a carrot positioned at a slight diagonal angle from lower-left to upper-right. The carrot should be bright orange color with a smooth, simplified geometric shape occupying about 60% of the icon space. Add 2-3 simple green leafy tops sprouting from the top of the carrot, occupying about 30% of the upper portion. Use clean, minimal lines with smooth edges. The design should be flat with no gradients, shadows, or 3D effects - just solid orange and green colors on a transparent or white background. The overall style should be friendly, modern, and immediately recognizable as a carrot icon suitable for a food delivery/agriculture brand logo. Keep it simple enough to be legible at small sizes (32px) while maintaining a professional, polished appearance.
      PROMPT
      constraints = [
        "Do not use photorealistic style - must be simplified icon",
        "Do not add gradients or shadows - flat design only",
        "Do not make too detailed - must work at small sizes",
        "Do not use multiple shades - solid colors only",
      ]
    },
  ]
  model = "openai/gpt-image-1"
}

resource "tofukit_file" "ui_rabbit_family" {
  name = "ui_rabbit_family"
  instructions = [
    {
      prompt = <<-PROMPT
        **Style:** 3D rendered character illustration with soft lighting and detailed texturing, similar to Pixar/DreamWorks animation style
        
        **Composition:**
        - Vertical portrait composition: 30% warm indoor background, 70% rabbit family group
        - Soft diffused lighting from upper left with gentle highlights
        - Warm indoor home setting with beige/tan walls
        - Group of 4 anthropomorphic rabbits arranged together
        - Varying sizes suggesting family group (adult, teen, young children)
        - Different fur colors: gray, white/cream, brown/tan patterns
        - All characters with large expressive eyes and friendly expressions
        - Characters positioned close together showing warmth and connection
        - Front-facing group portrait arrangement
        - Warm palette: grays, whites, browns, pinks for inner ears
        - Professional yet playful character designs
        - Varying ear positions and sizes for character distinction
        
        **Generation Prompt:**
        
        CRITICAL FORMATTING: Aspect ratio 3:4 portrait (vertical orientation), generate at 768x1024.
        
        Create a 3D rendered character illustration of a family group of four friendly anthropomorphic rabbits in a vertical portrait composition. The scene should be 30% warm indoor background with beige/tan walls suggesting a cozy home setting, and 70% character subjects. Use soft diffused lighting from the upper left creating gentle highlights on the rabbits' fur and subtle shadows. The group should include 4 rabbits of varying sizes suggesting a family: one larger adult-sized rabbit, one medium teen-sized rabbit, and two smaller child-sized rabbits. Give them different fur colors for variety: one gray, one white/cream, and two with brown or tan patterns. All rabbits should have realistic fur texture with soft shading transitions. Feature large upright ears of varying sizes with pink/salmon colored inner ear details. Give all characters friendly, warm facial expressions with large expressive eyes looking toward the viewer. Position the rabbits close together in a group portrait arrangement showing warmth and family connection. The overall style should be high-quality 3D rendering similar to Pixar or DreamWorks animation films - professional, approachable, heartwarming, and appealing to a broad family audience. Use a warm color palette emphasizing family-friendly, trustworthy, welcoming atmosphere.
      PROMPT
      constraints = [
        "Do not use low-quality or pixelated rendering - must be high-quality 3D",
        "Do not make characters look unfriendly - welcoming expressions required for all",
        "Do not use photorealistic style - stylized 3D animation aesthetic only",
        "Do not make all rabbits look identical - variety in size and color required",
      ]
    },
  ]
  model = "openai/gpt-image-1"
}