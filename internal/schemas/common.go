package schemas

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RequirementModel represents a requirement entry
// Note: For map-based requirements, the key is separate from the model
type RequirementModel struct {
	Name          types.String        `tfsdk:"name"`
	Instructions  []InstructionModel  `tfsdk:"instructions"`
	Verifications []VerificationModel `tfsdk:"verifications"`
}

// InstructionModel represents an instruction block
type InstructionModel struct {
	Prompt      types.String   `tfsdk:"prompt"`
	Constraints []types.String `tfsdk:"constraints"`
}

// VerificationModel represents a verification block
type VerificationModel struct {
	Command types.String `tfsdk:"command"`
	Expect  types.String `tfsdk:"expect"`
}

// ImageModel represents image generation configuration
type ImageModel struct {
	Size    types.String `tfsdk:"size"`    // Resolution in NNNNxNNNN format (e.g., "1024x1024")
	Quality types.String `tfsdk:"quality"` // Quality level (model-dependent: low/medium/high or standard/hd)
}

// FeatureModel represents a feature (inline or from resource reference)
type FeatureModel struct {
	Model         types.String        `tfsdk:"model"`        // Model to use for this feature's files
	Requirements  []RequirementModel  `tfsdk:"requirements"` // Uses common type!
	Files         types.Map           `tfsdk:"files"`
	Kits          types.Dynamic       `tfsdk:"kits"`
	Verifications []VerificationModel `tfsdk:"verifications"`
}

// FileModel represents a file entry
// Note: Path is now the map key, not a field in the struct
type FileModel struct {
	Content       types.String        `tfsdk:"content"`
	Instructions  []InstructionModel  `tfsdk:"instructions"`
	Verifications []VerificationModel `tfsdk:"verifications"`

	// Image generation configuration (optional, for image models)
	Image *ImageModel `tfsdk:"image"`

	// Computed drift detection fields
	ContentHash types.String `tfsdk:"content_hash"`
	FileHash    types.String `tfsdk:"file_hash"`
	FileModTime types.String `tfsdk:"file_modtime"`

	// Optional resource metadata fields (present when referencing tofukit_file)
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Link        types.String `tfsdk:"link"`
	Description types.String `tfsdk:"description"`
}

// ScaffoldModel is deprecated, use FileModel instead
// Kept for backward compatibility during migration
type ScaffoldModel = FileModel

// GetRequirementsListAttribute returns the schema for the requirements list attribute (optional)
func GetRequirementsListAttribute() schema.ListNestedAttribute {
	return getRequirementsListAttributeInternal(false)
}

// GetRequiredRequirementsListAttribute returns the schema for the requirements list attribute (required)
func GetRequiredRequirementsListAttribute() schema.ListNestedAttribute {
	return getRequirementsListAttributeInternal(true)
}

// getRequirementsListAttributeInternal is the internal implementation
func getRequirementsListAttributeInternal(required bool) schema.ListNestedAttribute {
	attr := schema.ListNestedAttribute{
		MarkdownDescription: "Requirements for this component (ordered list)",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Display name of the requirement",
					Required:            true,
				},
				"instructions": schema.ListNestedAttribute{
					MarkdownDescription: "Instruction steps for implementing this requirement (executed sequentially)",
					Required:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"prompt": schema.StringAttribute{
								MarkdownDescription: "LLM-facing instruction describing what to do",
								Required:            true,
							},
							"constraints": schema.ListAttribute{
								MarkdownDescription: "Constraints that must be respected (what NOT to do)",
								Optional:            true,
								ElementType:         types.StringType,
							},
						},
					},
				},
				"verifications": schema.ListNestedAttribute{
					MarkdownDescription: "Verification commands for this requirement",
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
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

	if required {
		attr.Required = true
	} else {
		attr.Optional = true
	}

	return attr
}

// GetRequirementBlock is deprecated - kept for backward compatibility
// Use GetRequirementsMapAttribute instead
func GetRequirementBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: "DEPRECATED: Use 'requirements' map attribute instead",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Name of the requirement",
					Required:            true,
				},
			},
			Blocks: map[string]schema.Block{
				"instruction": schema.ListNestedBlock{
					MarkdownDescription: "Instruction steps for implementing this requirement (executed sequentially)",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"prompt": schema.StringAttribute{
								MarkdownDescription: "LLM-facing instruction describing what to do",
								Required:            true,
							},
							"constraints": schema.ListAttribute{
								MarkdownDescription: "Constraints that must be respected (what NOT to do)",
								Optional:            true,
								ElementType:         types.StringType,
							},
						},
					},
				},
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

// GetFilesMapAttribute returns the schema for the files map attribute
// Accepts both inline file definitions and tofukit_file resource references
func GetFilesMapAttribute() schema.MapNestedAttribute {
	return schema.MapNestedAttribute{
		MarkdownDescription: "Files to generate and manage, keyed by file path. Accepts either inline file definitions or tofukit_file resource references.",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				// Model attribute for per-file model override
				"model": schema.StringAttribute{
					MarkdownDescription: "Model to use for this file in provider/model format (e.g., 'anthropic/claude-haiku', 'openai/gpt-image-1'). Overrides feature, stack, project, and provider-level model settings.",
					Optional:            true,
				},
				// Core file attributes (used by both inline and resource references)
				"content": schema.StringAttribute{
					MarkdownDescription: "Static content of the file (mutually exclusive with instructions)",
					Optional:            true,
				},
				"instructions": schema.ListNestedAttribute{
					MarkdownDescription: "Instructions for generating the file content (mutually exclusive with content)",
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"prompt": schema.StringAttribute{
								MarkdownDescription: "LLM-facing instruction describing what to generate",
								Required:            true,
							},
							"constraints": schema.ListAttribute{
								MarkdownDescription: "Constraints that must be respected (what NOT to do)",
								Optional:            true,
								ElementType:         types.StringType,
							},
						},
					},
				},
				"verifications": schema.ListNestedAttribute{
					MarkdownDescription: "Verification commands for this file",
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
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
				// Image generation configuration (optional, for image models)
				"image": schema.SingleNestedAttribute{
					MarkdownDescription: "Image generation configuration. Only applicable when using an image generation model (e.g., openai/gpt-image-1-mini, openai/dall-e-3).",
					Optional:            true,
					Attributes: map[string]schema.Attribute{
						"size": schema.StringAttribute{
							MarkdownDescription: "Image resolution in NNNNxNNNN format (e.g., '1024x1024', '1792x1024'). Default varies by model.",
							Optional:            true,
						},
						"quality": schema.StringAttribute{
							MarkdownDescription: "Image quality level. Valid values depend on model: 'low'/'medium'/'high' for gpt-image-1*, 'standard'/'hd' for dall-e-3.",
							Optional:            true,
						},
					},
				},
				// Resource metadata attributes (present when referencing tofukit_file resources)
				// These are optional and ignored for inline definitions
				// NOTE: Not marked as Computed because Terraform provides these when passing
				// resource references, the provider doesn't compute them
				"id": schema.StringAttribute{
					MarkdownDescription: "Resource identifier (present when referencing tofukit_file resource)",
					Optional:            true,
				},
				"name": schema.StringAttribute{
					MarkdownDescription: "File name (present when referencing tofukit_file resource)",
					Optional:            true,
				},
				"link": schema.StringAttribute{
					MarkdownDescription: "URI link to the file resource (present when referencing tofukit_file resource)",
					Optional:            true,
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "File description (present when referencing tofukit_file resource)",
					Optional:            true,
				},
				"content_hash": schema.StringAttribute{
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "SHA256 hash of the file specification (content or instructions JSON) for drift detection",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"file_hash": schema.StringAttribute{
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "SHA256 hash of the actual file on disk for drift detection",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"file_modtime": schema.StringAttribute{
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "File modification time in RFC3339 format (optimization for drift detection)",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
			},
		},
	}
}

// GetFeaturesMapAttribute returns the schema for the features map
func GetFeaturesMapAttribute() schema.MapNestedAttribute {
	return schema.MapNestedAttribute{
		MarkdownDescription: "Features to implement (capabilities bundled with files, kits, and verifications)",
		Optional:            true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"requirements": GetRequirementsListAttribute(),
				"files":        GetFilesMapAttribute(),
				"kits": schema.DynamicAttribute{
					MarkdownDescription: "Kit dependencies for this feature",
					Optional:            true,
				},
				"verifications": schema.ListNestedAttribute{
					MarkdownDescription: "Verification commands for this feature",
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
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

// GetFileBlock is deprecated - kept for backward compatibility
// Use GetFilesMapAttribute instead
func GetFileBlock() schema.ListNestedBlock {
	return schema.ListNestedBlock{
		MarkdownDescription: "DEPRECATED: Use 'files' map attribute instead",
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
			},
			Blocks: map[string]schema.Block{
				"instruction": schema.ListNestedBlock{
					MarkdownDescription: "Instructions for generating content",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"prompt": schema.StringAttribute{
								MarkdownDescription: "Instruction prompt",
								Required:            true,
							},
							"constraints": schema.ListAttribute{
								MarkdownDescription: "Constraints",
								Optional:            true,
								ElementType:         types.StringType,
							},
						},
					},
				},
				"verification": schema.ListNestedBlock{
					MarkdownDescription: "Verification commands",
					NestedObject: schema.NestedBlockObject{
						Attributes: map[string]schema.Attribute{
							"command": schema.StringAttribute{
								MarkdownDescription: "Verification command",
								Required:            true,
							},
							"expect": schema.StringAttribute{
								MarkdownDescription: "Expected output",
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
		"link": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "URI link to this resource for cross-referencing (e.g., tofukit://kit/language/name or tofukit://kit/library/name)",
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
