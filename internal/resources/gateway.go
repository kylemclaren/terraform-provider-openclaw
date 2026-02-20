package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ resource.Resource = &GatewayResource{}
var _ resource.ResourceWithImportState = &GatewayResource{}

type GatewayResource struct {
	client client.Client
}

type GatewayResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Port          types.Int64  `tfsdk:"port"`
	Bind          types.String `tfsdk:"bind"`
	AuthMode      types.String `tfsdk:"auth_mode"`
	AuthToken     types.String `tfsdk:"auth_token"`
	ReloadMode    types.String `tfsdk:"reload_mode"`
	TailscaleMode types.String `tfsdk:"tailscale_mode"`
}

func NewGatewayResource() resource.Resource {
	return &GatewayResource{}
}

func (r *GatewayResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway"
}

func (r *GatewayResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw Gateway server configuration (port, bind, auth, reload, Tailscale).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier (always 'gateway').",
				Computed:    true,
			},
			"port": schema.Int64Attribute{
				Description: "Gateway listen port. Default: 18789.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(18789),
			},
			"bind": schema.StringAttribute{
				Description: "Bind address: 'loopback' (default) or 'all'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("loopback"),
			},
			"auth_mode": schema.StringAttribute{
				Description: "Authentication mode: 'token', 'password', or 'none'.",
				Optional:    true,
			},
			"auth_token": schema.StringAttribute{
				Description: "Gateway auth token. Sensitive.",
				Optional:    true,
				Sensitive:   true,
			},
			"reload_mode": schema.StringAttribute{
				Description: "Config reload mode: 'hybrid' (default), 'hot', 'restart', or 'off'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("hybrid"),
			},
			"tailscale_mode": schema.StringAttribute{
				Description: "Tailscale exposure mode: 'off' (default), 'serve', or 'funnel'.",
				Optional:    true,
			},
		},
	}
}

func (r *GatewayResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*shared.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *provider.ProviderData, got %T", req.ProviderData))
		return
	}
	r.client = pd.Client
}

func (r *GatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gw := r.modelToMap(plan)

	_, hash, err := client.GetSection(ctx, r.client, "gateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchSection(ctx, r.client, "gateway", gw, hash); err != nil {
		resp.Diagnostics.AddError("Failed to write gateway config", err.Error())
		return
	}

	plan.ID = types.StringValue("gateway")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	section, _, err := client.GetSection(ctx, r.client, "gateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read gateway config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapToModel(section, &state)
	state.ID = types.StringValue("gateway")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	gw := r.modelToMap(plan)

	_, hash, err := client.GetSection(ctx, r.client, "gateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchSection(ctx, r.client, "gateway", gw, hash); err != nil {
		resp.Diagnostics.AddError("Failed to write gateway config", err.Error())
		return
	}

	plan.ID = types.StringValue("gateway")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GatewayResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, hash, err := client.GetSection(ctx, r.client, "gateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.DeleteSection(ctx, r.client, "gateway", hash); err != nil {
		resp.Diagnostics.AddError("Failed to delete gateway config", err.Error())
		return
	}
}

func (r *GatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetSection(ctx, r.client, "gateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import gateway config", err.Error())
		return
	}

	var state GatewayResourceModel
	if section != nil {
		r.mapToModel(section, &state)
	}
	state.ID = types.StringValue("gateway")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GatewayResource) modelToMap(m GatewayResourceModel) map[string]any {
	gw := make(map[string]any)

	if !m.Port.IsNull() && !m.Port.IsUnknown() {
		gw["port"] = m.Port.ValueInt64()
	}
	if !m.Bind.IsNull() && !m.Bind.IsUnknown() {
		gw["bind"] = m.Bind.ValueString()
	}
	if !m.ReloadMode.IsNull() && !m.ReloadMode.IsUnknown() {
		gw["reload"] = map[string]any{"mode": m.ReloadMode.ValueString()}
	}

	auth := make(map[string]any)
	if !m.AuthMode.IsNull() && !m.AuthMode.IsUnknown() {
		auth["mode"] = m.AuthMode.ValueString()
	}
	if !m.AuthToken.IsNull() && !m.AuthToken.IsUnknown() {
		auth["token"] = m.AuthToken.ValueString()
	}
	if len(auth) > 0 {
		gw["auth"] = auth
	}

	if !m.TailscaleMode.IsNull() && !m.TailscaleMode.IsUnknown() {
		gw["tailscale"] = map[string]any{"mode": m.TailscaleMode.ValueString()}
	}

	return gw
}

func (r *GatewayResource) mapToModel(section map[string]any, m *GatewayResourceModel) {
	if v, ok := section["port"]; ok {
		if f, ok := v.(float64); ok {
			m.Port = types.Int64Value(int64(f))
		}
	}
	if v, ok := section["bind"]; ok {
		if s, ok := v.(string); ok {
			m.Bind = types.StringValue(s)
		}
	}
	if v, ok := section["reload"]; ok {
		if rm, ok := v.(map[string]any); ok {
			if mode, ok := rm["mode"]; ok {
				if s, ok := mode.(string); ok {
					m.ReloadMode = types.StringValue(s)
				}
			}
		}
	}
	if v, ok := section["auth"]; ok {
		if am, ok := v.(map[string]any); ok {
			if mode, ok := am["mode"]; ok {
				if s, ok := mode.(string); ok {
					m.AuthMode = types.StringValue(s)
				}
			}
			// Don't read back auth token from config for security.
		}
	}
	if v, ok := section["tailscale"]; ok {
		if ts, ok := v.(map[string]any); ok {
			if mode, ok := ts["mode"]; ok {
				if s, ok := mode.(string); ok {
					m.TailscaleMode = types.StringValue(s)
				}
			}
		}
	}
}
