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

var _ resource.Resource = &AgentResource{}
var _ resource.ResourceWithImportState = &AgentResource{}

type AgentResource struct {
	client client.Client
}

type AgentModel struct {
	ID              types.String `tfsdk:"id"`
	AgentID         types.String `tfsdk:"agent_id"`
	DefaultAgent    types.Bool   `tfsdk:"default_agent"`
	Name            types.String `tfsdk:"name"`
	Workspace       types.String `tfsdk:"workspace"`
	Model           types.String `tfsdk:"model"`
	IdentityName    types.String `tfsdk:"identity_name"`
	IdentityEmoji   types.String `tfsdk:"identity_emoji"`
	IdentityTheme   types.String `tfsdk:"identity_theme"`
	MentionPatterns types.List   `tfsdk:"mention_patterns"`
	SandboxMode     types.String `tfsdk:"sandbox_mode"`
	SandboxScope    types.String `tfsdk:"sandbox_scope"`
	ToolsProfile    types.String `tfsdk:"tools_profile"`
	ToolsAllow      types.List   `tfsdk:"tools_allow"`
	ToolsDeny       types.List   `tfsdk:"tools_deny"`
}

func NewAgentResource() resource.Resource {
	return &AgentResource{}
}

func (r *AgentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (r *AgentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an individual agent entry in agents.list[].",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"agent_id": schema.StringAttribute{
				Description: "Stable identifier for the agent (maps to 'id' in config).",
				Required:    true,
			},
			"default_agent": schema.BoolAttribute{
				Description: "Whether this is the default agent.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "Display name for the agent.",
				Optional:    true,
			},
			"workspace": schema.StringAttribute{
				Description: "Workspace path for this agent.",
				Optional:    true,
			},
			"model": schema.StringAttribute{
				Description: "Model for this agent (e.g. anthropic/claude-opus-4-6).",
				Optional:    true,
			},
			"identity_name": schema.StringAttribute{
				Description: "Agent identity display name.",
				Optional:    true,
			},
			"identity_emoji": schema.StringAttribute{
				Description: "Agent identity emoji.",
				Optional:    true,
			},
			"identity_theme": schema.StringAttribute{
				Description: "Agent identity theme color.",
				Optional:    true,
			},
			"mention_patterns": schema.ListAttribute{
				Description: "Patterns that mention this agent in group chats.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"sandbox_mode": schema.StringAttribute{
				Description: "Sandbox mode: off|non-main|all.",
				Optional:    true,
			},
			"sandbox_scope": schema.StringAttribute{
				Description: "Sandbox scope: session|agent|shared.",
				Optional:    true,
			},
			"tools_profile": schema.StringAttribute{
				Description: "Tools profile name.",
				Optional:    true,
			},
			"tools_allow": schema.ListAttribute{
				Description: "Allowed tool names.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"tools_deny": schema.ListAttribute{
				Description: "Denied tool names.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *AgentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ── helpers for reading/writing the agents.list array ────────

func (r *AgentResource) getAgentsList(ctx context.Context) ([]any, string, error) {
	agentsSection, hash, err := client.GetSection(ctx, r.client, "agents")
	if err != nil {
		return nil, "", err
	}
	if agentsSection == nil {
		return nil, hash, nil
	}
	raw, ok := agentsSection["list"]
	if !ok {
		return nil, hash, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, hash, fmt.Errorf("agents.list is not an array")
	}
	return list, hash, nil
}

func (r *AgentResource) findAgentIndex(list []any, agentID string) int {
	for i, item := range list {
		if m, ok := item.(map[string]any); ok {
			if id, ok := m["id"].(string); ok && id == agentID {
				return i
			}
		}
	}
	return -1
}

func (r *AgentResource) writeAgentsList(ctx context.Context, list []any, hash string) error {
	patch := map[string]any{"agents": map[string]any{"list": list}}
	return r.client.PatchConfig(ctx, patch, hash)
}

// ── CRUD ─────────────────────────────────────────────────────

func (r *AgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, hash, err := r.getAgentsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read agents list", err.Error())
		return
	}

	entry := r.modelToMap(ctx, plan)
	agentID := plan.AgentID.ValueString()

	idx := r.findAgentIndex(list, agentID)
	if idx >= 0 {
		list[idx] = entry
	} else {
		list = append(list, entry)
	}

	if err := r.writeAgentsList(ctx, list, hash); err != nil {
		resp.Diagnostics.AddError("Failed to write agents list", err.Error())
		return
	}

	plan.ID = types.StringValue(agentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, _, err := r.getAgentsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read agents list", err.Error())
		return
	}

	agentID := state.AgentID.ValueString()
	idx := r.findAgentIndex(list, agentID)
	if idx < 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	entry, ok := list[idx].(map[string]any)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapToModel(ctx, entry, &state)
	state.ID = types.StringValue(agentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, hash, err := r.getAgentsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read agents list", err.Error())
		return
	}

	entry := r.modelToMap(ctx, plan)
	agentID := plan.AgentID.ValueString()

	idx := r.findAgentIndex(list, agentID)
	if idx >= 0 {
		list[idx] = entry
	} else {
		list = append(list, entry)
	}

	if err := r.writeAgentsList(ctx, list, hash); err != nil {
		resp.Diagnostics.AddError("Failed to write agents list", err.Error())
		return
	}

	plan.ID = types.StringValue(agentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, hash, err := r.getAgentsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read agents list", err.Error())
		return
	}

	agentID := state.AgentID.ValueString()
	idx := r.findAgentIndex(list, agentID)
	if idx >= 0 {
		list = append(list[:idx], list[idx+1:]...)
	}

	if err := r.writeAgentsList(ctx, list, hash); err != nil {
		resp.Diagnostics.AddError("Failed to delete agent", err.Error())
		return
	}
}

func (r *AgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	agentID := req.ID

	list, _, err := r.getAgentsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read agents list", err.Error())
		return
	}

	idx := r.findAgentIndex(list, agentID)
	if idx < 0 {
		resp.Diagnostics.AddError("Agent not found", fmt.Sprintf("No agent with id %q in agents.list", agentID))
		return
	}

	entry, ok := list[idx].(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Agent entry is not an object", "")
		return
	}

	var state AgentModel
	state.AgentID = types.StringValue(agentID)
	r.mapToModel(ctx, entry, &state)
	state.ID = types.StringValue(agentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ── model ↔ map conversion ──────────────────────────────────

func (r *AgentResource) modelToMap(ctx context.Context, m AgentModel) map[string]any {
	d := make(map[string]any)

	setIfString(d, "id", m.AgentID)
	setIfBool(d, "default", m.DefaultAgent)
	setIfString(d, "name", m.Name)
	setIfString(d, "workspace", m.Workspace)
	setIfString(d, "model", m.Model)
	setIfString(d, "sandboxMode", m.SandboxMode)
	setIfString(d, "sandboxScope", m.SandboxScope)

	identity := make(map[string]any)
	setIfString(identity, "name", m.IdentityName)
	setIfString(identity, "emoji", m.IdentityEmoji)
	setIfString(identity, "theme", m.IdentityTheme)
	if len(identity) > 0 {
		d["identity"] = identity
	}

	groupChat := make(map[string]any)
	setIfStringList(ctx, groupChat, "mentionPatterns", m.MentionPatterns)
	if len(groupChat) > 0 {
		d["groupChat"] = groupChat
	}

	tools := make(map[string]any)
	setIfString(tools, "profile", m.ToolsProfile)
	setIfStringList(ctx, tools, "allow", m.ToolsAllow)
	setIfStringList(ctx, tools, "deny", m.ToolsDeny)
	if len(tools) > 0 {
		d["tools"] = tools
	}

	return d
}

func (r *AgentResource) mapToModel(ctx context.Context, s map[string]any, m *AgentModel) {
	readString(s, "id", &m.AgentID)
	readBool(s, "default", &m.DefaultAgent)
	readString(s, "name", &m.Name)
	readString(s, "workspace", &m.Workspace)
	readString(s, "model", &m.Model)
	readString(s, "sandboxMode", &m.SandboxMode)
	readString(s, "sandboxScope", &m.SandboxScope)

	if identity, ok := s["identity"].(map[string]any); ok {
		readString(identity, "name", &m.IdentityName)
		readString(identity, "emoji", &m.IdentityEmoji)
		readString(identity, "theme", &m.IdentityTheme)
	}

	if groupChat, ok := s["groupChat"].(map[string]any); ok {
		readStringList(ctx, groupChat, "mentionPatterns", &m.MentionPatterns)
	}

	if tools, ok := s["tools"].(map[string]any); ok {
		readString(tools, "profile", &m.ToolsProfile)
		readStringList(ctx, tools, "allow", &m.ToolsAllow)
		readStringList(ctx, tools, "deny", &m.ToolsDeny)
	}
}
