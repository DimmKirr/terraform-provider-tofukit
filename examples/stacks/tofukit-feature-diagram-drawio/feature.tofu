# Draw.io Diagram Generation Feature
# Generates architecture diagrams via project context introspection

terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = "0.1.0"
    }
  }
}

resource "tofukit_feature" "diagram_drawio" {
  name        = "diagram-drawio"
  description = "Auto-generate draw.io architecture diagrams from project introspection"

  requirements = [
    {
      name = "Draw.io Diagram Generation via Project Introspection"

      instructions = [
        {
          prompt = <<-EOF
          Generate a professional architecture diagram in draw.io XML format by introspecting the project context.

          INTROSPECTION INSTRUCTIONS:
          1. Read the project_context field from the specification JSON
          2. Extract all components:
             - features: Application capabilities and their files
             - integrations: External APIs and services
             - kits: Languages, frameworks, and tools
             - requirements: High-level architectural requirements
          3. Analyze relationships between components (which features use which integrations/kits)

          DIAGRAM STRUCTURE:
          1. Create valid draw.io XML with <mxGraphModel> root element
          2. Organize components in logical tiers (top to bottom):
             - External Integrations (cloud shapes)
             - Application Features (rounded rectangles)
             - Infrastructure/Kits (cylinders or rectangles)
          3. Add arrows showing dependencies and data flow:
             - Feature → Integration: API calls
             - Feature → Kit: Uses framework/tool
             - Feature → Feature: Internal dependencies
          4. Label each component with:
             - Name (bold)
             - Type/description (smaller text)
          5. Use professional color scheme:
             - Features: Blue tones
             - Integrations: Orange/cloud colors
             - Kits: Gray/infrastructure colors

          OUTPUT FILE: ${var.file_path}.drawio

          The diagram should comprehensively visualize the entire project architecture
          without requiring manual component specification.
          EOF

          constraints = [
            "MUST read project_context field from specification",
            "MUST include ALL features found in project_context.features",
            "MUST include ALL integrations found in project_context.integrations (if present)",
            "MUST include ALL kits found in project_context.kits",
            "Output must be valid draw.io XML format",
            "File must be saved to exact path: ${var.file_path}.drawio",
            "Use standard draw.io shape library (no custom shapes)",
            "Arrange components logically with clear visual hierarchy",
            "Show dependencies with arrows (shortest path, above all blocks)",
            "NO manual text descriptions - use draw.io native labels",
            "Diagram must be viewable in draw.io desktop application"
          ]
        }
      ]
    }
  ]

  files = {
    "${var.file_path}.drawio" = {
      instructions = [
        {
          prompt = "Generate draw.io XML diagram via project introspection as specified in requirements"
        }
      ]

      verifications = [
        {
          command = "test -f ${var.file_path}.drawio && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "grep -q '<mxGraphModel' ${var.file_path}.drawio && echo 'OK'"
          expect  = "OK"
        },
        {
          command = "drawio -x -f png --scale 2.5 -o ${var.file_path}.drawio.png ${var.file_path}.drawio && echo 'OK'"
          expect  = "OK"
        }
      ]
    }
  }

  kits = [tofukit_tool.drawio]
}
