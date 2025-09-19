package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tofukit/opentofu-provider-tofukit/internal/registry"
	"github.com/tofukit/opentofu-provider-tofukit/internal/resources"
)

// Ensure TofukitProvider satisfies various provider interfaces.
var _ provider.Provider = &TofukitProvider{}

// TofukitProvider defines the provider implementation.
type TofukitProvider struct {
	// version is the provider version, set during build time.
	version string
}

// TofukitProviderModel describes the provider data model.
type TofukitProviderModel struct {
	OutputFormat types.String `tfsdk:"output_format"`
	OutputPath   types.String `tfsdk:"output_path"`
}

func (p *TofukitProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "tofukit"
	resp.Version = p.version
}

func (p *TofukitProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"output_format": schema.StringAttribute{
				MarkdownDescription: "Output format for generated contexts (json, yaml, markdown)",
				Optional:            true,
			},
			"output_path": schema.StringAttribute{
				MarkdownDescription: "Path where to write generated context files",
				Optional:            true,
			},
		},
	}
}

func (p *TofukitProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data TofukitProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values
	outputFormat := "json"
	outputPath := ".tofukit"

	if !data.OutputFormat.IsNull() {
		outputFormat = data.OutputFormat.ValueString()
	}

	if !data.OutputPath.IsNull() {
		outputPath = data.OutputPath.ValueString()
	}

	// Create provider data that will be passed to resources
	providerData := &ProviderData{
		OutputFormat: outputFormat,
		OutputPath:   outputPath,
		Registry:     registry.New(),
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData
}

func (p *TofukitProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewProjectResourceFinal,
		resources.NewStackResource,
		resources.NewBlueprintResource,
		resources.NewLanguageResource,
		resources.NewFrameworkResource,
		resources.NewToolResource,
		resources.NewMethodologyResource,
		resources.NewStyleResource,
		resources.NewInfrastructureResource,
		resources.NewIntegrationResource,
	}
}

func (p *TofukitProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// We'll add data sources later if needed
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TofukitProvider{
			version: version,
		}
	}
}

// ProviderData contains data that is passed to all resources
type ProviderData struct {
	OutputFormat string
	OutputPath   string
	Registry     *registry.Registry
}

// GetOutputPath returns the output path
func (p *ProviderData) GetOutputPath() string {
	return p.OutputPath
}

// GetOutputFormat returns the output format
func (p *ProviderData) GetOutputFormat() string {
	return p.OutputFormat
}

// GetRegistry returns the registry
func (p *ProviderData) GetRegistry() *registry.Registry {
	return p.Registry
}
