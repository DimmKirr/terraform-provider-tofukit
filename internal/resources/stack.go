package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/tofukit/opentofu-provider-tofukit/internal/scaffolds"
	"github.com/tofukit/opentofu-provider-tofukit/internal/schemas"
)

var _ resource.Resource = &StackResource{}

func NewStackResource() resource.Resource {
	return &StackResource{}
}

type StackResource struct {
	BaseComponent
}

type StackResourceModel struct {
	ID            types.String            `tfsdk:"id"`
	Name          types.String            `tfsdk:"name"`
	Description   types.String            `tfsdk:"description"`
	DependsOnRefs []types.String          `tfsdk:"depends_on_refs"`
	Scaffolds     []schemas.ScaffoldModel `tfsdk:"scaffold"`
	OutputPath    types.String            `tfsdk:"output_path"` // Track where scaffolds are created
}

func (r *StackResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stack"
}

func (r *StackResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Stack resource for tofukit",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the stack",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the stack",
				Optional:            true,
			},
			"depends_on_refs": schema.ListAttribute{
				MarkdownDescription: "String references to dependencies",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"output_path": schema.StringAttribute{
				MarkdownDescription: "Path where scaffold files are created",
				Computed:            true,
			},
		},

		Blocks: map[string]schema.Block{
			"scaffold": schemas.GetScaffoldBlock(),
		},
	}
}

func (r *StackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StackResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("stack.%s", data.Name.ValueString()))

	// Get provider configuration for output path
	outputPath := ".tofukit"
	isDryRun := true
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
		GetOutputPath() string
	}); ok {
		outputPath = provData.GetOutputPath()
		isDryRun = provData.GetDryRun()
	}

	// Store the output path for later use
	stackDir := filepath.Join(outputPath, "stacks", data.Name.ValueString())
	data.OutputPath = types.StringValue(stackDir)

	// Create scaffolds if not in dry-run mode
	if !isDryRun && len(data.Scaffolds) > 0 {
		// Ensure stack directory exists
		if err := os.MkdirAll(stackDir, 0755); err != nil {
			tflog.Error(ctx, "Failed to create stack directory", map[string]interface{}{
				"stack_id": data.ID.ValueString(),
				"path": stackDir,
				"error": err.Error(),
			})
		} else {
			scaffoldMgr := scaffolds.NewManager(stackDir)
			for _, scaffold := range data.Scaffolds {
				if err := scaffoldMgr.WriteScaffold(ctx, scaffold); err != nil {
					tflog.Error(ctx, "Failed to create scaffold", map[string]interface{}{
						"stack_id": data.ID.ValueString(),
						"scaffold_path": scaffold.Path.ValueString(),
						"error": err.Error(),
					})
				}
			}
		}
	}

	tflog.Trace(ctx, fmt.Sprintf("created stack resource: %s", data.ID.ValueString()))
	r.SaveToRegistry(ctx, data.ID.ValueString(), data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data StackResourceModel
	var state StackResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get provider configuration
	isDryRun := true
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
	}); ok {
		isDryRun = provData.GetDryRun()
	}

	// Preserve output path from state
	data.OutputPath = state.OutputPath

	// Handle scaffold changes if not in dry-run mode
	if !isDryRun && !state.OutputPath.IsNull() && state.OutputPath.ValueString() != "" {
		stackDir := state.OutputPath.ValueString()
		scaffoldMgr := scaffolds.NewManager(stackDir)

		// Process scaffold changes (removes deleted scaffolds, creates new ones)
		if err := scaffoldMgr.ProcessScaffoldChanges(ctx, state.Scaffolds, data.Scaffolds); err != nil {
			tflog.Error(ctx, "Failed to process scaffold changes", map[string]interface{}{
				"stack_id": data.ID.ValueString(),
				"error": err.Error(),
			})
		}
	}

	r.SaveToRegistry(ctx, data.ID.ValueString(), data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *StackResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get provider configuration
	isDryRun := true
	if provData, ok := r.ProviderData.(interface {
		GetDryRun() bool
	}); ok {
		isDryRun = provData.GetDryRun()
	}

	// Clean up scaffold files if not in dry-run mode
	if !isDryRun && !data.OutputPath.IsNull() && data.OutputPath.ValueString() != "" {
		stackDir := data.OutputPath.ValueString()
		scaffoldMgr := scaffolds.NewManager(stackDir)

		// Remove all scaffold files
		scaffoldMgr.RemoveAllScaffolds(ctx, data.Scaffolds)

		// Try to remove the stack directory if it's empty
		if err := os.Remove(stackDir); err != nil {
			if !os.IsNotExist(err) {
				tflog.Debug(ctx, "Stack directory not empty or already removed", map[string]interface{}{
					"stack_id": data.ID.ValueString(),
					"path": stackDir,
				})
			}
		} else {
			tflog.Info(ctx, "Removed stack directory", map[string]interface{}{
				"stack_id": data.ID.ValueString(),
				"path": stackDir,
			})
		}
	}

	r.RemoveFromRegistry(ctx, data.ID.ValueString())
}
