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

var _ datasource.DataSource = &AgentDefaultsDataSource{}

type AgentDefaultsDataSource struct {
	client client.Client
}

type AgentDefaultsDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Workspace       types.String `tfsdk:"workspace"`
	ModelPrimary    types.String `tfsdk:"model_primary"`
	ThinkingDefault types.String `tfsdk:"thinking_default"`
	TimeoutSeconds  types.Int64  `tfsdk:"timeout_seconds"`
	MaxConcurrent   types.Int64  `tfsdk:"max_concurrent"`
	UserTimezone    types.String `tfsdk:"user_timezone"`
	HeartbeatEvery  types.String `tfsdk:"heartbeat_every"`
	HeartbeatTarget types.String `tfsdk:"heartbeat_target"`
	SandboxMode     types.String `tfsdk:"sandbox_mode"`
	SandboxScope    types.String `tfsdk:"sandbox_scope"`
}

func NewAgentDefaultsDataSource() datasource.DataSource {
	return &AgentDefaultsDataSource{}
}

func (d *AgentDefaultsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_defaults"
}

func (d *AgentDefaultsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the current OpenClaw agent default configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"workspace": schema.StringAttribute{
				Description: "Default agent workspace path.",
				Computed:    true,
			},
			"model_primary": schema.StringAttribute{
				Description: "Primary model in provider/model format.",
				Computed:    true,
			},
			"thinking_default": schema.StringAttribute{
				Description: "Default thinking level.",
				Computed:    true,
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "Agent run timeout in seconds.",
				Computed:    true,
			},
			"max_concurrent": schema.Int64Attribute{
				Description: "Max parallel agent runs across sessions.",
				Computed:    true,
			},
			"user_timezone": schema.StringAttribute{
				Description: "Timezone for system prompt context.",
				Computed:    true,
			},
			"heartbeat_every": schema.StringAttribute{
				Description: "Heartbeat interval duration string.",
				Computed:    true,
			},
			"heartbeat_target": schema.StringAttribute{
				Description: "Heartbeat delivery target.",
				Computed:    true,
			},
			"sandbox_mode": schema.StringAttribute{
				Description: "Sandbox mode: off, non-main, all.",
				Computed:    true,
			},
			"sandbox_scope": schema.StringAttribute{
				Description: "Sandbox scope: session, agent, shared.",
				Computed:    true,
			},
		},
	}
}

func (d *AgentDefaultsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AgentDefaultsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	section, _, err := client.GetNestedSection(ctx, d.client, "agents", "defaults")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read agent defaults config", err.Error())
		return
	}

	state := AgentDefaultsDataSourceModel{
		ID: types.StringValue("agent_defaults"),
	}

	if section != nil {
		if v, ok := section["workspace"].(string); ok {
			state.Workspace = types.StringValue(v)
		}
		if v, ok := section["userTimezone"].(string); ok {
			state.UserTimezone = types.StringValue(v)
		}
		if v, ok := section["timeout"].(float64); ok {
			state.TimeoutSeconds = types.Int64Value(int64(v))
		}
		if v, ok := section["maxConcurrent"].(float64); ok {
			state.MaxConcurrent = types.Int64Value(int64(v))
		}

		// Model is nested under agents.defaults.model or agents.defaults.modelPrimary
		if v, ok := section["modelPrimary"].(string); ok {
			state.ModelPrimary = types.StringValue(v)
		} else if model, ok := section["model"].(map[string]any); ok {
			if v, ok := model["primary"].(string); ok {
				state.ModelPrimary = types.StringValue(v)
			}
		}

		// Thinking
		if v, ok := section["thinkingDefault"].(string); ok {
			state.ThinkingDefault = types.StringValue(v)
		} else if thinking, ok := section["thinking"].(map[string]any); ok {
			if v, ok := thinking["default"].(string); ok {
				state.ThinkingDefault = types.StringValue(v)
			}
		}

		// Heartbeat
		if hb, ok := section["heartbeat"].(map[string]any); ok {
			if v, ok := hb["every"].(string); ok {
				state.HeartbeatEvery = types.StringValue(v)
			}
			if v, ok := hb["target"].(string); ok {
				state.HeartbeatTarget = types.StringValue(v)
			}
		}

		// Sandbox
		if sb, ok := section["sandbox"].(map[string]any); ok {
			if v, ok := sb["mode"].(string); ok {
				state.SandboxMode = types.StringValue(v)
			}
			if v, ok := sb["scope"].(string); ok {
				state.SandboxScope = types.StringValue(v)
			}
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
