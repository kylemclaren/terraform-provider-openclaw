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

var _ resource.Resource = &AgentDefaultsResource{}
var _ resource.ResourceWithImportState = &AgentDefaultsResource{}

type AgentDefaultsResource struct {
	client client.Client
}

type AgentDefaultsResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Workspace       types.String `tfsdk:"workspace"`
	ModelPrimary    types.String `tfsdk:"model_primary"`
	ModelFallbacks  types.List   `tfsdk:"model_fallbacks"`
	ThinkingDefault types.String `tfsdk:"thinking_default"`
	VerboseDefault  types.String `tfsdk:"verbose_default"`
	TimeoutSeconds  types.Int64  `tfsdk:"timeout_seconds"`
	MaxConcurrent   types.Int64  `tfsdk:"max_concurrent"`
	UserTimezone    types.String `tfsdk:"user_timezone"`

	// Heartbeat
	HeartbeatEvery  types.String `tfsdk:"heartbeat_every"`
	HeartbeatTarget types.String `tfsdk:"heartbeat_target"`

	// Sandbox
	SandboxMode  types.String `tfsdk:"sandbox_mode"`
	SandboxScope types.String `tfsdk:"sandbox_scope"`
}

func NewAgentDefaultsResource() resource.Resource {
	return &AgentDefaultsResource{}
}

func (r *AgentDefaultsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_defaults"
}

func (r *AgentDefaultsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages OpenClaw agent defaults (model, workspace, heartbeat, sandbox, etc.).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"workspace": schema.StringAttribute{
				Description: "Default agent workspace path.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("~/.openclaw/workspace"),
			},
			"model_primary": schema.StringAttribute{
				Description: "Primary model in provider/model format (e.g. anthropic/claude-opus-4-6).",
				Optional:    true,
			},
			"model_fallbacks": schema.ListAttribute{
				Description: "Ordered list of fallback models.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"thinking_default": schema.StringAttribute{
				Description: "Default thinking level: off|minimal|low|medium|high|xhigh.",
				Optional:    true,
			},
			"verbose_default": schema.StringAttribute{
				Description: "Default verbose level: on|off.",
				Optional:    true,
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "Agent run timeout in seconds. Default: 600.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(600),
			},
			"max_concurrent": schema.Int64Attribute{
				Description: "Max parallel agent runs across sessions. Default: 1.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
			},
			"user_timezone": schema.StringAttribute{
				Description: "Timezone for system prompt context (e.g. America/Chicago).",
				Optional:    true,
			},
			"heartbeat_every": schema.StringAttribute{
				Description: "Heartbeat interval duration string (e.g. 30m, 2h). 0m disables.",
				Optional:    true,
			},
			"heartbeat_target": schema.StringAttribute{
				Description: "Heartbeat delivery target: last|whatsapp|telegram|discord|none.",
				Optional:    true,
			},
			"sandbox_mode": schema.StringAttribute{
				Description: "Sandbox mode: off|non-main|all.",
				Optional:    true,
			},
			"sandbox_scope": schema.StringAttribute{
				Description: "Sandbox scope: session|agent|shared.",
				Optional:    true,
			},
		},
	}
}

func (r *AgentDefaultsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AgentDefaultsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AgentDefaultsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaults := r.modelToMap(ctx, plan)

	_, hash, err := client.GetSection(ctx, r.client, "agents")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	patch := map[string]any{"agents": map[string]any{"defaults": defaults}}
	if err := r.client.PatchConfig(ctx, patch, hash); err != nil {
		resp.Diagnostics.AddError("Failed to write agent defaults", err.Error())
		return
	}

	plan.ID = types.StringValue("agent_defaults")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentDefaultsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentDefaultsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	section, _, err := client.GetNestedSection(ctx, r.client, "agents", "defaults")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read agent defaults", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("agent_defaults")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AgentDefaultsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AgentDefaultsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaults := r.modelToMap(ctx, plan)

	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	patch := map[string]any{"agents": map[string]any{"defaults": defaults}}
	if err := r.client.PatchConfig(ctx, patch, cfg.Hash); err != nil {
		resp.Diagnostics.AddError("Failed to write agent defaults", err.Error())
		return
	}

	plan.ID = types.StringValue("agent_defaults")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentDefaultsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	patch := map[string]any{"agents": map[string]any{"defaults": nil}}
	if err := r.client.PatchConfig(ctx, patch, cfg.Hash); err != nil {
		resp.Diagnostics.AddError("Failed to delete agent defaults", err.Error())
		return
	}
}

func (r *AgentDefaultsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "agents", "defaults")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import agent defaults", err.Error())
		return
	}

	var state AgentDefaultsResourceModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("agent_defaults")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AgentDefaultsResource) modelToMap(_ context.Context, m AgentDefaultsResourceModel) map[string]any {
	d := make(map[string]any)

	if !m.Workspace.IsNull() && !m.Workspace.IsUnknown() {
		d["workspace"] = m.Workspace.ValueString()
	}
	if !m.TimeoutSeconds.IsNull() && !m.TimeoutSeconds.IsUnknown() {
		d["timeoutSeconds"] = m.TimeoutSeconds.ValueInt64()
	}
	if !m.MaxConcurrent.IsNull() && !m.MaxConcurrent.IsUnknown() {
		d["maxConcurrent"] = m.MaxConcurrent.ValueInt64()
	}
	if !m.UserTimezone.IsNull() && !m.UserTimezone.IsUnknown() {
		d["userTimezone"] = m.UserTimezone.ValueString()
	}
	if !m.ThinkingDefault.IsNull() && !m.ThinkingDefault.IsUnknown() {
		d["thinkingDefault"] = m.ThinkingDefault.ValueString()
	}
	if !m.VerboseDefault.IsNull() && !m.VerboseDefault.IsUnknown() {
		d["verboseDefault"] = m.VerboseDefault.ValueString()
	}

	// Model
	model := make(map[string]any)
	if !m.ModelPrimary.IsNull() && !m.ModelPrimary.IsUnknown() {
		model["primary"] = m.ModelPrimary.ValueString()
	}
	if !m.ModelFallbacks.IsNull() && !m.ModelFallbacks.IsUnknown() {
		var fallbacks []string
		m.ModelFallbacks.ElementsAs(context.Background(), &fallbacks, false)
		model["fallbacks"] = fallbacks
	}
	if len(model) > 0 {
		d["model"] = model
	}

	// Heartbeat
	hb := make(map[string]any)
	if !m.HeartbeatEvery.IsNull() && !m.HeartbeatEvery.IsUnknown() {
		hb["every"] = m.HeartbeatEvery.ValueString()
	}
	if !m.HeartbeatTarget.IsNull() && !m.HeartbeatTarget.IsUnknown() {
		hb["target"] = m.HeartbeatTarget.ValueString()
	}
	if len(hb) > 0 {
		d["heartbeat"] = hb
	}

	// Sandbox
	sb := make(map[string]any)
	if !m.SandboxMode.IsNull() && !m.SandboxMode.IsUnknown() {
		sb["mode"] = m.SandboxMode.ValueString()
	}
	if !m.SandboxScope.IsNull() && !m.SandboxScope.IsUnknown() {
		sb["scope"] = m.SandboxScope.ValueString()
	}
	if len(sb) > 0 {
		d["sandbox"] = sb
	}

	return d
}

func (r *AgentDefaultsResource) mapToModel(_ context.Context, section map[string]any, m *AgentDefaultsResourceModel) {
	if v, ok := section["workspace"].(string); ok {
		m.Workspace = types.StringValue(v)
	}
	if v, ok := section["timeoutSeconds"].(float64); ok {
		m.TimeoutSeconds = types.Int64Value(int64(v))
	}
	if v, ok := section["maxConcurrent"].(float64); ok {
		m.MaxConcurrent = types.Int64Value(int64(v))
	}
	if v, ok := section["userTimezone"].(string); ok {
		m.UserTimezone = types.StringValue(v)
	}
	if v, ok := section["thinkingDefault"].(string); ok {
		m.ThinkingDefault = types.StringValue(v)
	}
	if v, ok := section["verboseDefault"].(string); ok {
		m.VerboseDefault = types.StringValue(v)
	}

	if model, ok := section["model"].(map[string]any); ok {
		if v, ok := model["primary"].(string); ok {
			m.ModelPrimary = types.StringValue(v)
		}
		if v, ok := model["fallbacks"].([]any); ok {
			fallbacks := make([]string, 0, len(v))
			for _, f := range v {
				if s, ok := f.(string); ok {
					fallbacks = append(fallbacks, s)
				}
			}
			list, _ := types.ListValueFrom(context.Background(), types.StringType, fallbacks)
			m.ModelFallbacks = list
		}
	} else if model, ok := section["model"].(string); ok {
		// Simple string form
		m.ModelPrimary = types.StringValue(model)
	}

	if hb, ok := section["heartbeat"].(map[string]any); ok {
		if v, ok := hb["every"].(string); ok {
			m.HeartbeatEvery = types.StringValue(v)
		}
		if v, ok := hb["target"].(string); ok {
			m.HeartbeatTarget = types.StringValue(v)
		}
	}

	if sb, ok := section["sandbox"].(map[string]any); ok {
		if v, ok := sb["mode"].(string); ok {
			m.SandboxMode = types.StringValue(v)
		}
		if v, ok := sb["scope"].(string); ok {
			m.SandboxScope = types.StringValue(v)
		}
	}
}
