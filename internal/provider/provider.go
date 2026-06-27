package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/sophotechlabs/terraform-provider-hetzner-robot/internal/hrobot"
)

var _ provider.Provider = &HrobotProvider{}

type HrobotProvider struct {
	version      string
	mockEndpoint string
}

type HrobotProviderModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Endpoint types.String `tfsdk:"endpoint"`
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
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: "Robot webservice user (Robot UI -> Settings -> Web service and app settings, not the main account login). Can also be set via the HROBOT_USERNAME environment variable.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "Password for the Robot webservice user. Can also be set via the HROBOT_PASSWORD environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"endpoint": schema.StringAttribute{
				Description: "Base URL of the Robot webservice. Defaults to https://robot-ws.your-server.de.",
				Optional:    true,
			},
		},
	}
}

func (p *HrobotProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config HrobotProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := os.Getenv("HROBOT_USERNAME")
	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	password := os.Getenv("HROBOT_PASSWORD")
	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	endpoint := hrobot.DefaultEndpoint
	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Robot webservice username",
			"username must be set in provider config or the HROBOT_USERNAME environment variable.",
		)
	}
	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Robot webservice password",
			"password must be set in provider config or the HROBOT_PASSWORD environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client := hrobot.NewClient(endpoint, username, password)
	if p.mockEndpoint != "" {
		client.Endpoint = p.mockEndpoint
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *HrobotProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newSSHKeyResource,
	}
}

func (p *HrobotProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newServerDataSource(),
	}
}
