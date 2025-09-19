package schemas

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RequirementModel represents a requirement block
type RequirementModel struct {
	Name         types.String       `tfsdk:"name"`
	Priority     types.Int64        `tfsdk:"priority"`
	Instructions []types.String     `tfsdk:"instructions"`
	Verification *VerificationModel `tfsdk:"verification"`
}

// VerificationModel represents a verification block
type VerificationModel struct {
	Command types.String `tfsdk:"command"`
	Expect  types.String `tfsdk:"expect"`
}

// ScaffoldModel represents a scaffold entry
type ScaffoldModel struct {
	Path     types.String `tfsdk:"path"`
	Content  types.String `tfsdk:"content"`
	Generate types.Bool   `tfsdk:"generate"`
	Template types.String `tfsdk:"template"`
}

// GetRequirementBlock returns the schema for requirement blocks
func GetRequirementBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: "Requirements for this component",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Name of the requirement",
					Required:            true,
				},
				"priority": schema.Int64Attribute{
					MarkdownDescription: "Priority of the requirement (higher number = higher priority, default: 0)",
					Optional:            true,
				},
				"instructions": schema.ListAttribute{
					MarkdownDescription: "List of instructions",
					Optional:            true,
					ElementType:         types.StringType,
				},
			},
			Blocks: map[string]schema.Block{
				"verification": schema.SingleNestedBlock{
					MarkdownDescription: "Verification for this requirement",
					Attributes: map[string]schema.Attribute{
						"command": schema.StringAttribute{
							MarkdownDescription: "Command to run for verification",
							Required:            true,
						},
						"expect": schema.StringAttribute{
							MarkdownDescription: "Expected output or pattern",
							Optional:            true,
						},
					},
				},
			},
		},
	}
}

// GetScaffoldBlock returns the schema for scaffold blocks
func GetScaffoldBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: "Scaffold entries for file generation",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"path": schema.StringAttribute{
					MarkdownDescription: "Path where the file should be created",
					Required:            true,
				},
				"content": schema.StringAttribute{
					MarkdownDescription: "Content of the file",
					Optional:            true,
				},
				"generate": schema.BoolAttribute{
					MarkdownDescription: "Whether to generate from template",
					Optional:            true,
				},
				"template": schema.StringAttribute{
					MarkdownDescription: "Template name to use for generation",
					Optional:            true,
				},
			},
		},
	}
}

// GetBaseComponentAttributes returns common attributes for all component types
func GetBaseComponentAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Resource identifier",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "Name of the component",
			Required:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Description of the component",
			Optional:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "Version of the component",
			Required:            true,
		},
		"depends_on_refs": schema.ListAttribute{
			MarkdownDescription: "String references to dependencies (e.g., '@language.python')",
			Optional:            true,
			ElementType:         types.StringType,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
	}
}
