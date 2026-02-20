package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ resource.Resource = &SessionResource{}
var _ resource.ResourceWithImportState = &SessionResource{}

type SessionResource struct {
	client client.Client
}

type SessionModel struct {
	ID               types.String `tfsdk:"id"`
	DmScope          types.String `tfsdk:"dm_scope"`
	ResetMode        types.String `tfsdk:"reset_mode"`
	ResetAtHour      types.Int64  `tfsdk:"reset_at_hour"`
	ResetIdleMinutes types.Int64  `tfsdk:"reset_idle_minutes"`
	ResetTriggers    types.List   `tfsdk:"reset_triggers"`
}

func NewSessionResource() resource.Resource {
	return &SessionResource{}
}

func (r *SessionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_session"
}

func (r *SessionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw session configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"dm_scope": schema.StringAttribute{
				Description: "DM session scope: main|per-peer|per-channel-peer|per-account-channel-peer.",
				Optional:    true,
			},
			"reset_mode": schema.StringAttribute{
				Description: "Session reset mode: daily|idle.",
				Optional:    true,
			},
			"reset_at_hour": schema.Int64Attribute{
				Description: "Hour of day to reset (for daily mode).",
				Optional:    true,
			},
			"reset_idle_minutes": schema.Int64Attribute{
				Description: "Minutes of inactivity before reset (for idle mode).",
				Optional:    true,
			},
			"reset_triggers": schema.ListAttribute{
				Description: "Custom trigger phrases that reset the session.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *SessionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*shared.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *shared.ProviderData, got %T", req.ProviderData))
		return
	}
	r.client = pd.Client
}

func (r *SessionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SessionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchSection(ctx, r.client, "session", r.modelToMap(ctx, plan), cfg.Hash); err != nil {
		resp.Diagnostics.AddError("Failed to write session config", err.Error())
		return
	}

	plan.ID = types.StringValue("session")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SessionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SessionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	section, _, err := client.GetSection(ctx, r.client, "session")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read session config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("session")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SessionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SessionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchSection(ctx, r.client, "session", r.modelToMap(ctx, plan), cfg.Hash); err != nil {
		resp.Diagnostics.AddError("Failed to write session config", err.Error())
		return
	}

	plan.ID = types.StringValue("session")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SessionResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.DeleteSection(ctx, r.client, "session", cfg.Hash); err != nil {
		resp.Diagnostics.AddError("Failed to delete session config", err.Error())
		return
	}
}

func (r *SessionResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetSection(ctx, r.client, "session")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import session config", err.Error())
		return
	}

	var state SessionModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("session")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ── model ↔ map conversion ──────────────────────────────────

func (r *SessionResource) modelToMap(ctx context.Context, m SessionModel) map[string]any {
	d := make(map[string]any)

	setIfString(d, "dmScope", m.DmScope)
	setIfStringList(ctx, d, "resetTriggers", m.ResetTriggers)

	reset := make(map[string]any)
	setIfString(reset, "mode", m.ResetMode)
	setIfInt64(reset, "atHour", m.ResetAtHour)
	setIfInt64(reset, "idleMinutes", m.ResetIdleMinutes)
	if len(reset) > 0 {
		d["reset"] = reset
	}

	return d
}

func (r *SessionResource) mapToModel(ctx context.Context, s map[string]any, m *SessionModel) {
	readString(s, "dmScope", &m.DmScope)
	readStringList(ctx, s, "resetTriggers", &m.ResetTriggers)

	if reset, ok := s["reset"].(map[string]any); ok {
		readString(reset, "mode", &m.ResetMode)
		readFloat64AsInt64(reset, "atHour", &m.ResetAtHour)
		readFloat64AsInt64(reset, "idleMinutes", &m.ResetIdleMinutes)
	}
}
