package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ datasource.DataSource = &GatewayDataSource{}

type GatewayDataSource struct {
	client client.Client
}

type GatewayDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Port          types.Int64  `tfsdk:"port"`
	Bind          types.String `tfsdk:"bind"`
	AuthMode      types.String `tfsdk:"auth_mode"`
	ReloadMode    types.String `tfsdk:"reload_mode"`
	TailscaleMode types.String `tfsdk:"tailscale_mode"`
	Mode          types.String `tfsdk:"mode"`
}

func NewGatewayDataSource() datasource.DataSource {
	return &GatewayDataSource{}
}

func (d *GatewayDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway"
}

func (d *GatewayDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the current OpenClaw gateway server configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"port": schema.Int64Attribute{
				Description: "Gateway listen port.",
				Computed:    true,
			},
			"bind": schema.StringAttribute{
				Description: "Bind address: loopback or all.",
				Computed:    true,
			},
			"auth_mode": schema.StringAttribute{
				Description: "Authentication mode: token, password, or none.",
				Computed:    true,
			},
			"reload_mode": schema.StringAttribute{
				Description: "Config reload mode: hybrid, hot, restart, or off.",
				Computed:    true,
			},
			"tailscale_mode": schema.StringAttribute{
				Description: "Tailscale exposure mode: off, serve, or funnel.",
				Computed:    true,
			},
			"mode": schema.StringAttribute{
				Description: "Gateway mode (e.g. local).",
				Computed:    true,
			},
		},
	}
}

func (d *GatewayDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*shared.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *shared.ProviderData, got %T", req.ProviderData))
		return
	}
	d.client = pd.Client
}

func (d *GatewayDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	section, _, err := client.GetSection(ctx, d.client, "gateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read gateway config", err.Error())
		return
	}

	state := GatewayDataSourceModel{
		ID: types.StringValue("gateway"),
	}

	if section != nil {
		if v, ok := section["port"].(float64); ok {
			state.Port = types.Int64Value(int64(v))
		}
		if v, ok := section["bind"].(string); ok {
			state.Bind = types.StringValue(v)
		}
		if v, ok := section["mode"].(string); ok {
			state.Mode = types.StringValue(v)
		}

		// Auth is nested under gateway.auth
		if auth, ok := section["auth"].(map[string]any); ok {
			if v, ok := auth["mode"].(string); ok {
				state.AuthMode = types.StringValue(v)
			}
		}

		// Reload is nested under gateway.reload
		if reload, ok := section["reload"].(map[string]any); ok {
			if v, ok := reload["mode"].(string); ok {
				state.ReloadMode = types.StringValue(v)
			}
		}

		// Tailscale mode
		if ts, ok := section["tailscale"].(map[string]any); ok {
			if v, ok := ts["mode"].(string); ok {
				state.TailscaleMode = types.StringValue(v)
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
