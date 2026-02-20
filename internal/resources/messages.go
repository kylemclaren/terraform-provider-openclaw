package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ resource.Resource = &MessagesResource{}
var _ resource.ResourceWithImportState = &MessagesResource{}

type MessagesResource struct {
	client client.Client
}

type MessagesModel struct {
	ID                types.String `tfsdk:"id"`
	ResponsePrefix    types.String `tfsdk:"response_prefix"`
	AckReaction       types.String `tfsdk:"ack_reaction"`
	AckReactionScope  types.String `tfsdk:"ack_reaction_scope"`
	QueueMode         types.String `tfsdk:"queue_mode"`
	QueueDebounceMs   types.Int64  `tfsdk:"queue_debounce_ms"`
	QueueCap          types.Int64  `tfsdk:"queue_cap"`
	InboundDebounceMs types.Int64  `tfsdk:"inbound_debounce_ms"`
}

func NewMessagesResource() resource.Resource {
	return &MessagesResource{}
}

func (r *MessagesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_messages"
}

func (r *MessagesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw messages configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"response_prefix": schema.StringAttribute{
				Description: "Prefix prepended to every agent response.",
				Optional:    true,
			},
			"ack_reaction": schema.StringAttribute{
				Description: "Emoji reaction to acknowledge receipt of a message.",
				Optional:    true,
			},
			"ack_reaction_scope": schema.StringAttribute{
				Description: "Scope for ack reactions: group-mentions|group-all|direct|all.",
				Optional:    true,
			},
			"queue_mode": schema.StringAttribute{
				Description: "Queue processing mode: steer|followup|collect|steer-backlog|queue|interrupt.",
				Optional:    true,
			},
			"queue_debounce_ms": schema.Int64Attribute{
				Description: "Queue debounce in milliseconds. Default: 1000.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1000),
			},
			"queue_cap": schema.Int64Attribute{
				Description: "Max queued messages. Default: 20.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20),
			},
			"inbound_debounce_ms": schema.Int64Attribute{
				Description: "Inbound message debounce in milliseconds. Default: 2000.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2000),
			},
		},
	}
}

func (r *MessagesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MessagesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MessagesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchSection(ctx, r.client, "messages", r.modelToMap(plan), cfg.Hash); err != nil {
		resp.Diagnostics.AddError("Failed to write messages config", err.Error())
		return
	}

	plan.ID = types.StringValue("messages")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MessagesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MessagesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	section, _, err := client.GetSection(ctx, r.client, "messages")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read messages config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapToModel(section, &state)
	state.ID = types.StringValue("messages")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MessagesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MessagesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchSection(ctx, r.client, "messages", r.modelToMap(plan), cfg.Hash); err != nil {
		resp.Diagnostics.AddError("Failed to write messages config", err.Error())
		return
	}

	plan.ID = types.StringValue("messages")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MessagesResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.DeleteSection(ctx, r.client, "messages", cfg.Hash); err != nil {
		resp.Diagnostics.AddError("Failed to delete messages config", err.Error())
		return
	}
}

func (r *MessagesResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetSection(ctx, r.client, "messages")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import messages config", err.Error())
		return
	}

	var state MessagesModel
	if section != nil {
		r.mapToModel(section, &state)
	}
	state.ID = types.StringValue("messages")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ── model ↔ map conversion ──────────────────────────────────

func (r *MessagesResource) modelToMap(m MessagesModel) map[string]any {
	d := make(map[string]any)

	setIfString(d, "responsePrefix", m.ResponsePrefix)
	setIfString(d, "ackReaction", m.AckReaction)
	setIfString(d, "ackReactionScope", m.AckReactionScope)

	queue := make(map[string]any)
	setIfString(queue, "mode", m.QueueMode)
	setIfInt64(queue, "debounceMs", m.QueueDebounceMs)
	setIfInt64(queue, "cap", m.QueueCap)
	if len(queue) > 0 {
		d["queue"] = queue
	}

	inbound := make(map[string]any)
	setIfInt64(inbound, "debounceMs", m.InboundDebounceMs)
	if len(inbound) > 0 {
		d["inbound"] = inbound
	}

	return d
}

func (r *MessagesResource) mapToModel(s map[string]any, m *MessagesModel) {
	readString(s, "responsePrefix", &m.ResponsePrefix)
	readString(s, "ackReaction", &m.AckReaction)
	readString(s, "ackReactionScope", &m.AckReactionScope)

	if queue, ok := s["queue"].(map[string]any); ok {
		readString(queue, "mode", &m.QueueMode)
		readFloat64AsInt64(queue, "debounceMs", &m.QueueDebounceMs)
		readFloat64AsInt64(queue, "cap", &m.QueueCap)
	}

	if inbound, ok := s["inbound"].(map[string]any); ok {
		readFloat64AsInt64(inbound, "debounceMs", &m.InboundDebounceMs)
	}
}
