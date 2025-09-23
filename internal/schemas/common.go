package schemas

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RequirementModel represents a requirement block
type RequirementModel struct {
	Name         types.String        `tfsdk:"name"`
	Priority     types.Int64         `tfsdk:"priority"`
	Instructions []types.String      `tfsdk:"instructions"`
	Verification []VerificationModel `tfsdk:"verification"`
}

// VerificationModel represents a verification block
type VerificationModel struct {
	Command types.String `tfsdk:"command"`
	Expect  types.String `tfsdk:"expect"`
}

// FileModel represents a file entry
type FileModel struct {
	Path         types.String        `tfsdk:"path"`
	Content      types.String        `tfsdk:"content"`
	Generate     types.Bool          `tfsdk:"generate"`
	Instructions []types.String      `tfsdk:"instructions"`
	Verification []VerificationModel `tfsdk:"verification"`
}

// ScaffoldModel is deprecated, use FileModel instead
// Kept for backward compatibility during migration
type ScaffoldModel = FileModel

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
				"verification": schema.ListNestedBlock{
					MarkdownDescription: "Verification commands for this requirement",
					NestedObject: schema.NestedBlockObject{
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
		},
	}
}

// GetFileBlock returns the schema for file blocks
func GetFileBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: "File entries for generation and management",
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
					MarkdownDescription: "Whether to generate content based on instructions",
					Optional:            true,
				},
				"instructions": schema.ListAttribute{
					MarkdownDescription: "Instructions for generating the file content (used with generate=true)",
					Optional:            true,
					ElementType:         types.StringType,
				},
			},
			Blocks: map[string]schema.Block{
				"verification": schema.ListNestedBlock{
					MarkdownDescription: "Verification commands for this file",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"command": schema.StringAttribute{
								MarkdownDescription: "Command to run for verification (e.g., 'python src/cli.py --version')",
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
		},
	}
}

// GetBaseComponentAttributes returns common attributes for all component types
// GetScaffoldBlock is deprecated, use GetFileBlock instead
// Kept for backward compatibility during migration
func GetScaffoldBlock() schema.ListNestedBlock {
	return GetFileBlock()
}

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
	}
}
