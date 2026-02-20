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

var _ datasource.DataSource = &HealthDataSource{}

type HealthDataSource struct {
	client client.Client
}

type HealthDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	OK             types.Bool   `tfsdk:"ok"`
	Timestamp      types.Int64  `tfsdk:"timestamp"`
	DefaultAgentID types.String `tfsdk:"default_agent_id"`
	HeartbeatSecs  types.Int64  `tfsdk:"heartbeat_seconds"`
}

func NewHealthDataSource() datasource.DataSource {
	return &HealthDataSource{}
}

func (d *HealthDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_health"
}

func (d *HealthDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the health status of a running OpenClaw Gateway. Requires WebSocket mode.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"ok": schema.BoolAttribute{
				Description: "Whether the gateway health check passed.",
				Computed:    true,
			},
			"timestamp": schema.Int64Attribute{
				Description: "Server timestamp (Unix milliseconds) when health was checked.",
				Computed:    true,
			},
			"default_agent_id": schema.StringAttribute{
				Description: "The default agent ID configured on the gateway.",
				Computed:    true,
			},
			"heartbeat_seconds": schema.Int64Attribute{
				Description: "Heartbeat interval in seconds.",
				Computed:    true,
			},
		},
	}
}

func (d *HealthDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *HealthDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	health, err := d.client.Health(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Gateway health", err.Error())
		return
	}

	state := HealthDataSourceModel{
		ID:             types.StringValue("health"),
		OK:             types.BoolValue(health.OK),
		Timestamp:      types.Int64Value(health.Timestamp),
		DefaultAgentID: types.StringValue(health.DefaultAgentID),
		HeartbeatSecs:  types.Int64Value(health.HeartbeatSecs),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
