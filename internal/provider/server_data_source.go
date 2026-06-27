package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/sophotechlabs/terraform-provider-hetzner-robot/internal/hrobot"
)

var (
	_ datasource.DataSource              = &serverDataSource{}
	_ datasource.DataSourceWithConfigure = &serverDataSource{}
)

func newServerDataSource() func() datasource.DataSource {
	return func() datasource.DataSource {
		return &serverDataSource{}
	}
}

type serverDataSource struct {
	client *hrobot.Client
}

type serverDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Number      types.Int64  `tfsdk:"number"`
	IP          types.String `tfsdk:"ip"`
	Name        types.String `tfsdk:"name"`
	Product     types.String `tfsdk:"product"`
	Datacenter  types.String `tfsdk:"datacenter"`
	Status      types.String `tfsdk:"status"`
	Traffic     types.String `tfsdk:"traffic"`
	Cancelled   types.Bool   `tfsdk:"cancelled"`
	PaidUntil   types.String `tfsdk:"paid_until"`
	IPv4        types.String `tfsdk:"ipv4"`
	IPv6Network types.String `tfsdk:"ipv6_network"`
	IPs         types.List   `tfsdk:"ips"`
}

func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads facts about a Hetzner Robot dedicated server. Look the server up by `number` (the Robot server-number) or by `ip` (any IPv4 address bound to the server). Exactly one of the two must be set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The Robot server-number as a string.",
				Computed:    true,
			},
			"number": schema.Int64Attribute{
				Description: "The Robot server-number. Set this to look the server up directly; left unset, it is resolved from `ip`.",
				Optional:    true,
				Computed:    true,
			},
			"ip": schema.StringAttribute{
				Description: "An IPv4 address bound to the server, used to resolve its server-number. Mutually exclusive with `number`.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The server name set in Robot.",
				Computed:    true,
			},
			"product": schema.StringAttribute{
				Description: "The product name (e.g. AX41, EQ 8).",
				Computed:    true,
			},
			"datacenter": schema.StringAttribute{
				Description: "The datacenter the server is located in (e.g. FSN1-DC8).",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The server status (e.g. ready, in process).",
				Computed:    true,
			},
			"traffic": schema.StringAttribute{
				Description: "The traffic limit for the server.",
				Computed:    true,
			},
			"cancelled": schema.BoolAttribute{
				Description: "Whether the server is cancelled.",
				Computed:    true,
			},
			"paid_until": schema.StringAttribute{
				Description: "The date the server is paid until (YYYY-MM-DD).",
				Computed:    true,
			},
			"ipv4": schema.StringAttribute{
				Description: "The primary IPv4 address of the server.",
				Computed:    true,
			},
			"ipv6_network": schema.StringAttribute{
				Description: "The IPv6 network assigned to the server.",
				Computed:    true,
			},
			"ips": schema.ListAttribute{
				Description: "All single IP addresses bound to the server.",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (d *serverDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*hrobot.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("expected *hrobot.Client, got %T. This is a provider bug.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasNumber := !data.Number.IsNull()
	hasIP := !data.IP.IsNull()
	if hasNumber == hasIP {
		resp.Diagnostics.AddError(
			"Invalid server lookup",
			"exactly one of `number` or `ip` must be set.",
		)
		return
	}

	var (
		server *hrobot.Server
		err    error
	)
	if hasNumber {
		server, err = d.client.GetServer(ctx, int(data.Number.ValueInt64()))
	} else {
		server, err = d.client.ServerByIP(ctx, data.IP.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Hetzner Robot server", err.Error())
		return
	}

	ips, diags := types.ListValueFrom(ctx, types.StringType, server.IPs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(strconv.Itoa(server.ServerNumber))
	data.Number = types.Int64Value(int64(server.ServerNumber))
	data.Name = types.StringValue(server.ServerName)
	data.Product = types.StringValue(server.Product)
	data.Datacenter = types.StringValue(server.DC)
	data.Status = types.StringValue(server.Status)
	data.Traffic = types.StringValue(server.Traffic)
	data.Cancelled = types.BoolValue(server.Cancelled)
	data.PaidUntil = types.StringValue(server.PaidUntil)
	data.IPv4 = types.StringValue(server.ServerIP)
	data.IPv6Network = types.StringValue(server.ServerIPv6Net)
	data.IPs = ips

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
