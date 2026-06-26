package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &metaDataSource{}

func newMetaDataSource(version string) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &metaDataSource{version: version}
	}
}

type metaDataSource struct {
	version string
}

type metaDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Note    types.String `tfsdk:"note"`
	Version types.String `tfsdk:"version"`
}

func (d *metaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_meta"
}

func (d *metaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns metadata about the hetzner-robot provider. Scaffolding data source used to validate provider wiring; superseded by real Hetzner Robot data sources (hrobot_server and the declarative resources) in later versions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Static identifier for this metadata lookup.",
				Computed:    true,
			},
			"note": schema.StringAttribute{
				Description: "Optional note echoed back unchanged in the response.",
				Optional:    true,
			},
			"version": schema.StringAttribute{
				Description: "The running provider version.",
				Computed:    true,
			},
		},
	}
}

func (d *metaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data metaDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue("meta")
	data.Version = types.StringValue(d.version)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
