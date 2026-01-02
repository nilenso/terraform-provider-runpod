package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &RunpodProvider{}

// RunpodProvider defines the provider implementation
type RunpodProvider struct {
	version string
}

// RunpodProviderModel describes the provider data model
type RunpodProviderModel struct {
	APIKey types.String `tfsdk:"api_key"`
}

// New returns a new provider instance
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &RunpodProvider{version: version}
	}
}

func (p *RunpodProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "runpod"
	resp.Version = p.version
}

func (p *RunpodProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with RunPod API to manage GPU cloud resources.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "RunPod API key. Can also be set via RUNPOD_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func (p *RunpodProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config RunpodProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get API key from config or environment
	apiKey := os.Getenv("RUNPOD_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"Set the api_key value in configuration or use the RUNPOD_API_KEY environment variable.",
		)
		return
	}

	// Create and validate client
	client := NewClient(apiKey)
	if err := client.Ping(); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create RunPod API Client",
			"Error: "+err.Error(),
		)
		return
	}

	// Make client available to resources and data sources
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *RunpodProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPodResource,
	}
}

func (p *RunpodProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGpuTypesDataSource,
	}
}
