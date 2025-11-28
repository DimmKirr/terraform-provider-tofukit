# Draw.io Diagram Generation Feature
# Generates architecture diagrams via project context introspection

terraform {
  required_providers {
    tofukit = {
      source  = "registry.terraform.io/DimmKirr/tofukit"
      version = ">= 0.1.0"
    }
  }
}

locals {
  # Complete AWS diagram template (from reference)
  aws_diagram_template = <<-TEMPLATE
<?xml version="1.0" encoding="UTF-8"?>
<mxGraphModel dx="746" dy="532" grid="1" gridSize="10" guides="1" tooltips="1" connect="1" arrows="1" fold="1" page="1" pageScale="1" pageWidth="1169" pageHeight="827" math="0" shadow="0">
  <root>
    <mxCell id="0" />
    <mxCell id="1" parent="0" />

    <!-- Route 53 (outside AWS Cloud) -->
    <mxCell id="route53" value="Route 53" style="sketch=0;points=[[0,0,0],[0.25,0,0],[0.5,0,0],[0.75,0,0],[1,0,0],[0,1,0],[0.25,1,0],[0.5,1,0],[0.75,1,0],[1,1,0],[0,0.25,0],[0,0.5,0],[0,0.75,0],[1,0.25,0],[1,0.5,0],[1,0.75,0]];outlineConnect=0;fontColor=#232F3E;fillColor=#8C4FFF;strokeColor=#ffffff;dashed=0;verticalLabelPosition=bottom;verticalAlign=top;align=center;html=1;fontSize=12;fontStyle=0;aspect=fixed;shape=mxgraph.aws4.resourceIcon;resIcon=mxgraph.aws4.route_53;" vertex="1" parent="1">
      <mxGeometry x="320" y="20" width="40" height="40" as="geometry" />
    </mxCell>

    <!-- AWS Cloud Container -->
    <mxCell id="aws-cloud" value="AWS Cloud" style="points=[[0,0],[0.25,0],[0.5,0],[0.75,0],[1,0],[1,0.25],[1,0.5],[1,0.75],[1,1],[0.75,1],[0.5,1],[0.25,1],[0,1],[0,0.75],[0,0.5],[0,0.25]];outlineConnect=0;gradientColor=none;html=1;whiteSpace=wrap;fontSize=12;fontStyle=0;container=0;pointerEvents=0;collapsible=0;recursiveResize=0;shape=mxgraph.aws4.group;grIcon=mxgraph.aws4.group_aws_cloud;strokeColor=#232F3E;fillColor=none;verticalAlign=top;align=left;spacingLeft=30;fontColor=#232F3E;dashed=0;" vertex="1" parent="1">
      <mxGeometry x="200" y="80" width="720" height="440" as="geometry" />
    </mxCell>

    <!-- VPC -->
    <mxCell id="vpc" value="VPC" style="sketch=0;outlineConnect=0;gradientColor=none;html=1;whiteSpace=wrap;fontSize=12;fontStyle=0;container=0;pointerEvents=0;collapsible=0;recursiveResize=0;shape=mxgraph.aws4.group;grIcon=mxgraph.aws4.group_vpc;strokeColor=#248814;fillColor=none;verticalAlign=top;align=left;spacingLeft=30;fontColor=#AAB7B8;dashed=0;" vertex="1" parent="1">
      <mxGeometry x="280" y="160" width="600" height="320" as="geometry" />
    </mxCell>

    <!-- Availability Zone A -->
    <mxCell id="az-a" value="Availability Zone A" style="fillColor=none;strokeColor=#147EBA;dashed=1;verticalAlign=top;fontStyle=0;fontColor=#147EBA;whiteSpace=wrap;html=1;" vertex="1" parent="1">
      <mxGeometry x="320" y="200" width="240" height="240" as="geometry" />
    </mxCell>

    <!-- Public Subnet A -->
    <mxCell id="pub-subnet-a" value="Public Subnet A" style="points=[[0,0],[0.25,0],[0.5,0],[0.75,0],[1,0],[1,0.25],[1,0.5],[1,0.75],[1,1],[0.75,1],[0.5,1],[0.25,1],[0,1],[0,0.75],[0,0.5],[0,0.25]];outlineConnect=0;gradientColor=none;html=1;whiteSpace=wrap;fontSize=12;fontStyle=0;container=0;collapsible=0;recursiveResize=0;shape=mxgraph.aws4.group;grIcon=mxgraph.aws4.group_security_group;grStroke=0;strokeColor=#248814;fillColor=#E9F3E6;verticalAlign=top;align=left;spacingLeft=30;fontColor=#248814;dashed=0;" vertex="1" parent="1">
      <mxGeometry x="340" y="240" width="200" height="180" as="geometry" />
    </mxCell>

    <!-- Availability Zone B -->
    <mxCell id="az-b" value="Availability Zone B" style="fillColor=none;strokeColor=#147EBA;dashed=1;verticalAlign=top;fontStyle=0;fontColor=#147EBA;whiteSpace=wrap;html=1;" vertex="1" parent="1">
      <mxGeometry x="600" y="200" width="240" height="240" as="geometry" />
    </mxCell>

    <!-- Public Subnet B -->
    <mxCell id="pub-subnet-b" value="Public Subnet B" style="points=[[0,0],[0.25,0],[0.5,0],[0.75,0],[1,0],[1,0.25],[1,0.5],[1,0.75],[1,1],[0.75,1],[0.5,1],[0.25,1],[0,1],[0,0.75],[0,0.5],[0,0.25]];outlineConnect=0;gradientColor=none;html=1;whiteSpace=wrap;fontSize=12;fontStyle=0;container=0;collapsible=0;recursiveResize=0;shape=mxgraph.aws4.group;grIcon=mxgraph.aws4.group_security_group;grStroke=0;strokeColor=#248814;fillColor=#E9F3E6;verticalAlign=top;align=left;spacingLeft=30;fontColor=#248814;dashed=0;" vertex="1" parent="1">
      <mxGeometry x="620" y="240" width="200" height="180" as="geometry" />
    </mxCell>

    <!-- Internet Gateway -->
    <mxCell id="igw" value="Internet Gateway" style="sketch=0;outlineConnect=0;fontColor=#232F3E;fillColor=#8C4FFF;strokeColor=#ffffff;dashed=0;verticalLabelPosition=bottom;verticalAlign=top;align=center;html=1;fontSize=12;fontStyle=0;aspect=fixed;pointerEvents=1;shape=mxgraph.aws4.internet_gateway;" vertex="1" parent="1">
      <mxGeometry x="560" y="120" width="40" height="40" as="geometry" />
    </mxCell>

    <!-- Application Load Balancer -->
    <mxCell id="alb" value="Application Load Balancer" style="sketch=0;points=[[0,0,0],[0.25,0,0],[0.5,0,0],[0.75,0,0],[1,0,0],[0,1,0],[0.25,1,0],[0.5,1,0],[0.75,1,0],[1,1,0],[0,0.25,0],[0,0.5,0],[0,0.75,0],[1,0.25,0],[1,0.5,0],[1,0.75,0]];outlineConnect=0;fontColor=#232F3E;fillColor=#ED7100;strokeColor=#ffffff;dashed=0;verticalLabelPosition=bottom;verticalAlign=top;align=center;html=1;fontSize=12;fontStyle=0;aspect=fixed;shape=mxgraph.aws4.resourceIcon;resIcon=mxgraph.aws4.application_load_balancer;" vertex="1" parent="1">
      <mxGeometry x="560" y="270" width="40" height="40" as="geometry" />
    </mxCell>

    <!-- EC2 Instance 1 -->
    <mxCell id="ec2-a" value="EC2 Instance 1" style="sketch=0;points=[[0,0,0],[0.25,0,0],[0.5,0,0],[0.75,0,0],[1,0,0],[0,1,0],[0.25,1,0],[0.5,1,0],[0.75,1,0],[1,1,0],[0,0.25,0],[0,0.5,0],[0,0.75,0],[1,0.25,0],[1,0.5,0],[1,0.75,0]];outlineConnect=0;fontColor=#232F3E;fillColor=#ED7100;strokeColor=#ffffff;dashed=0;verticalLabelPosition=bottom;verticalAlign=top;align=center;html=1;fontSize=12;fontStyle=0;aspect=fixed;shape=mxgraph.aws4.resourceIcon;resIcon=mxgraph.aws4.ec2;" vertex="1" parent="1">
      <mxGeometry x="420" y="350" width="40" height="40" as="geometry" />
    </mxCell>

    <!-- EC2 Instance 2 -->
    <mxCell id="ec2-b" value="EC2 Instance 2" style="sketch=0;points=[[0,0,0],[0.25,0,0],[0.5,0,0],[0.75,0,0],[1,0,0],[0,1,0],[0.25,1,0],[0.5,1,0],[0.75,1,0],[1,1,0],[0,0.25,0],[0,0.5,0],[0,0.75,0],[1,0.25,0],[1,0.5,0],[1,0.75,0]];outlineConnect=0;fontColor=#232F3E;fillColor=#ED7100;strokeColor=#ffffff;dashed=0;verticalLabelPosition=bottom;verticalAlign=top;align=center;html=1;fontSize=12;fontStyle=0;aspect=fixed;shape=mxgraph.aws4.resourceIcon;resIcon=mxgraph.aws4.ec2;" vertex="1" parent="1">
      <mxGeometry x="700" y="350" width="40" height="40" as="geometry" />
    </mxCell>

    <!-- Connections -->
    <mxCell id="conn-route53-alb" style="endArrow=classicThin;html=1;rounded=0;strokeColor=#147EBA;strokeWidth=1;edgeStyle=orthogonalEdgeStyle;dashed=1;curved=1;" edge="1" parent="1" source="route53" target="alb">
      <mxGeometry relative="1" as="geometry" />
    </mxCell>

    <mxCell id="conn-igw-alb" style="endArrow=classicThin;html=1;rounded=0;strokeColor=#147EBA;strokeWidth=1;edgeStyle=orthogonalEdgeStyle;" edge="1" parent="1" source="igw" target="alb">
      <mxGeometry relative="1" as="geometry" />
    </mxCell>

    <mxCell id="conn-alb-ec2a" style="edgeStyle=orthogonalEdgeStyle;endArrow=classicThin;html=1;rounded=0;strokeColor=#147EBA;strokeWidth=1;" edge="1" parent="1" source="alb" target="ec2-a">
      <mxGeometry relative="1" as="geometry" />
    </mxCell>

    <mxCell id="conn-alb-ec2b" style="edgeStyle=orthogonalEdgeStyle;endArrow=classicThin;html=1;rounded=0;strokeColor=#147EBA;strokeWidth=1;" edge="1" parent="1" source="alb" target="ec2-b">
      <mxGeometry relative="1" as="geometry" />
    </mxCell>
  </root>
</mxGraphModel>
TEMPLATE

  # Diagram style definitions
  diagram_styles = {
    generic = <<-EOF
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
       **CRITICAL ARROW ROUTING RULES:**
       - Use edgeStyle=orthogonalEdgeStyle for all arrows (right-angle routing)
       - Set exitX/exitY and entryX/entryY connection points to avoid box centers
       - Exit from bottom of source box: exitX=0.5;exitY=1
       - Enter from top of target box: entryX=0.5;entryY=0
       - For side connections: exitX=1;exitY=0.5 (right) or exitX=0;exitY=0.5 (left)
       - Add waypoints (mxPoint) to route around boxes if needed
       - Keep arrows ABOVE or BELOW boxes, never crossing box centers
    4. Label each component with:
       - Name (bold)
       - Type/description (smaller text)
    5. Use professional color scheme:
       - Features: Blue tones
       - Integrations: Orange/cloud colors
       - Kits: Gray/infrastructure colors
    EOF

    AWS = <<-EOF
    AWS ARCHITECTURE DIAGRAM:

    **IMPORTANT: Use AWS-style icons and colors for professional architecture diagrams.**

    Follow these AWS architecture best practices:
    1. Use official AWS service icons (mxgraph.aws4 shape library)
    2. Apply AWS standard colors: Purple (networking), Orange (compute), Blue (database), Green (storage)
    3. Create containment hierarchy: AWS Cloud → VPC → Availability Zone → Subnet → Resources
    4. Use solid blue lines for physical connections, dotted blue lines for logical connections
    5. Place DNS services (Route 53) outside the AWS Cloud boundary

    **SAMPLE AWS DIAGRAM - FOLLOW THIS STYLING PRECISELY:**

    ${local.aws_diagram_template}

    **When generating your diagram:**
    - Start by reading project_context to identify all components
    - Use the EXACT style syntax from the sample above for each AWS resource type
    - Copy the style strings character-for-character (they're complex but necessary for icons)
    - Maintain the same visual hierarchy and color scheme
    - Adapt the layout and components to match your specific architecture
    EOF
  }

  # Base requirement for diagram generation
  base_requirements = [
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

          ${local.diagram_styles[var.style]}

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
            "Use edgeStyle=orthogonalEdgeStyle for ALL arrows to enable right-angle routing",
            "Set exitX/exitY and entryX/entryY on all edges to connect to box edges (NOT centers)",
            "Arrows must NOT cross through the middle of boxes - route around them",
            "Exit from bottom (exitY=1) and enter from top (entryY=0) for vertical flows",
            "NO manual text descriptions - use draw.io native labels",
            "Diagram must be viewable in draw.io desktop application"
          ]
        }
      ]
    }
  ]
}

resource "tofukit_feature" "diagram_drawio" {
  name        = "diagram-drawio"
  description = "Auto-generate draw.io architecture diagrams from project introspection"

  # Concatenate base requirements with user-provided additional requirements
  requirements = concat(local.base_requirements, var.requirements)

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
