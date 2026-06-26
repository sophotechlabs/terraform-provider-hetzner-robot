package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &HrobotProvider{}

type HrobotProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HrobotProvider{version: version}
	}
}

func (p *HrobotProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hrobot"
	resp.Version = p.version
}

func (p *HrobotProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Hetzner Robot dedicated-server infrastructure as code via the Robot webservice API.",
	}
}

func (p *HrobotProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *HrobotProvider) Resources(_ context.Context) []func() resource.Resource {
	return nil
}

func (p *HrobotProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newMetaDataSource(p.version),
	}
}
