package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ resource.Resource = &BindingResource{}
var _ resource.ResourceWithImportState = &BindingResource{}

type BindingResource struct {
	client client.Client
}

type BindingModel struct {
	ID             types.String `tfsdk:"id"`
	AgentID        types.String `tfsdk:"agent_id"`
	MatchChannel   types.String `tfsdk:"match_channel"`
	MatchAccountID types.String `tfsdk:"match_account_id"`
	MatchPeerKind  types.String `tfsdk:"match_peer_kind"`
	MatchPeerID    types.String `tfsdk:"match_peer_id"`
}

func NewBindingResource() resource.Resource {
	return &BindingResource{}
}

func (r *BindingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_binding"
}

func (r *BindingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an individual binding entry in bindings[].",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"agent_id": schema.StringAttribute{
				Description: "Agent ID this binding routes to.",
				Required:    true,
			},
			"match_channel": schema.StringAttribute{
				Description: "Channel to match (e.g. discord, telegram, whatsapp).",
				Required:    true,
			},
			"match_account_id": schema.StringAttribute{
				Description: "Account ID to match.",
				Optional:    true,
			},
			"match_peer_kind": schema.StringAttribute{
				Description: "Peer kind to match (e.g. dm, group).",
				Optional:    true,
			},
			"match_peer_id": schema.StringAttribute{
				Description: "Peer ID to match.",
				Optional:    true,
			},
		},
	}
}

func (r *BindingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ── composite key ────────────────────────────────────────────

func bindingCompositeKey(agentID, channel, accountID string) string {
	return agentID + "/" + channel + "/" + accountID
}

func bindingKeyFromModel(m BindingModel) string {
	accountID := ""
	if !m.MatchAccountID.IsNull() && !m.MatchAccountID.IsUnknown() {
		accountID = m.MatchAccountID.ValueString()
	}
	return bindingCompositeKey(m.AgentID.ValueString(), m.MatchChannel.ValueString(), accountID)
}

func bindingKeyFromMap(entry map[string]any) string {
	agentID, _ := entry["agentId"].(string)
	channel := ""
	accountID := ""
	if match, ok := entry["match"].(map[string]any); ok {
		channel, _ = match["channel"].(string)
		accountID, _ = match["accountId"].(string)
	}
	return bindingCompositeKey(agentID, channel, accountID)
}

// ── helpers for reading/writing the bindings array ───────────

func (r *BindingResource) getBindingsList(ctx context.Context) ([]any, string, error) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("reading config: %w", err)
	}

	parsed, err := parseRawJSONHelper(cfg.Raw)
	if err != nil {
		return nil, cfg.Hash, err
	}

	raw, ok := parsed["bindings"]
	if !ok {
		return nil, cfg.Hash, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, cfg.Hash, fmt.Errorf("bindings is not an array")
	}
	return list, cfg.Hash, nil
}

func (r *BindingResource) findBindingIndex(list []any, key string) int {
	for i, item := range list {
		if m, ok := item.(map[string]any); ok {
			if bindingKeyFromMap(m) == key {
				return i
			}
		}
	}
	return -1
}

func (r *BindingResource) writeBindingsList(ctx context.Context, list []any, hash string) error {
	patch := map[string]any{"bindings": list}
	return r.client.PatchConfig(ctx, patch, hash)
}

// ── CRUD ─────────────────────────────────────────────────────

func (r *BindingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, hash, err := r.getBindingsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read bindings", err.Error())
		return
	}

	entry := r.modelToMap(plan)
	key := bindingKeyFromModel(plan)

	idx := r.findBindingIndex(list, key)
	if idx >= 0 {
		list[idx] = entry
	} else {
		list = append(list, entry)
	}

	if err := r.writeBindingsList(ctx, list, hash); err != nil {
		resp.Diagnostics.AddError("Failed to write bindings", err.Error())
		return
	}

	plan.ID = types.StringValue(key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BindingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, _, err := r.getBindingsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read bindings", err.Error())
		return
	}

	key := bindingKeyFromModel(state)
	idx := r.findBindingIndex(list, key)
	if idx < 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	entry, ok := list[idx].(map[string]any)
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapToModel(entry, &state)
	state.ID = types.StringValue(key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BindingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BindingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, hash, err := r.getBindingsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read bindings", err.Error())
		return
	}

	entry := r.modelToMap(plan)
	key := bindingKeyFromModel(plan)

	idx := r.findBindingIndex(list, key)
	if idx >= 0 {
		list[idx] = entry
	} else {
		list = append(list, entry)
	}

	if err := r.writeBindingsList(ctx, list, hash); err != nil {
		resp.Diagnostics.AddError("Failed to write bindings", err.Error())
		return
	}

	plan.ID = types.StringValue(key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BindingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BindingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	list, hash, err := r.getBindingsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read bindings", err.Error())
		return
	}

	key := bindingKeyFromModel(state)
	idx := r.findBindingIndex(list, key)
	if idx >= 0 {
		list = append(list[:idx], list[idx+1:]...)
	}

	if err := r.writeBindingsList(ctx, list, hash); err != nil {
		resp.Diagnostics.AddError("Failed to delete binding", err.Error())
		return
	}
}

func (r *BindingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: agentId/channel/accountId
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) < 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: agentId/channel or agentId/channel/accountId")
		return
	}

	agentID := parts[0]
	channel := parts[1]
	accountID := ""
	if len(parts) == 3 {
		accountID = parts[2]
	}
	key := bindingCompositeKey(agentID, channel, accountID)

	list, _, err := r.getBindingsList(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read bindings", err.Error())
		return
	}

	idx := r.findBindingIndex(list, key)
	if idx < 0 {
		resp.Diagnostics.AddError("Binding not found", fmt.Sprintf("No binding with key %q in bindings[]", key))
		return
	}

	entry, ok := list[idx].(map[string]any)
	if !ok {
		resp.Diagnostics.AddError("Binding entry is not an object", "")
		return
	}

	var state BindingModel
	r.mapToModel(entry, &state)
	state.ID = types.StringValue(key)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ── model ↔ map conversion ──────────────────────────────────

func (r *BindingResource) modelToMap(m BindingModel) map[string]any {
	d := make(map[string]any)

	setIfString(d, "agentId", m.AgentID)

	match := make(map[string]any)
	setIfString(match, "channel", m.MatchChannel)
	setIfString(match, "accountId", m.MatchAccountID)

	peer := make(map[string]any)
	setIfString(peer, "kind", m.MatchPeerKind)
	setIfString(peer, "id", m.MatchPeerID)
	if len(peer) > 0 {
		match["peer"] = peer
	}

	if len(match) > 0 {
		d["match"] = match
	}

	return d
}

func (r *BindingResource) mapToModel(s map[string]any, m *BindingModel) {
	readString(s, "agentId", &m.AgentID)

	if match, ok := s["match"].(map[string]any); ok {
		readString(match, "channel", &m.MatchChannel)
		readString(match, "accountId", &m.MatchAccountID)

		if peer, ok := match["peer"].(map[string]any); ok {
			readString(peer, "kind", &m.MatchPeerKind)
			readString(peer, "id", &m.MatchPeerID)
		}
	}
}

// parseRawJSONHelper is a local helper to parse raw JSON config.
func parseRawJSONHelper(raw string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}
	return result, nil
}
