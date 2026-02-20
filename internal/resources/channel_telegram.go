package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ resource.Resource = &ChannelTelegramResource{}
var _ resource.ResourceWithImportState = &ChannelTelegramResource{}

type ChannelTelegramResource struct {
	client client.Client
}

type ChannelTelegramModel struct {
	ID           types.String `tfsdk:"id"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	BotToken     types.String `tfsdk:"bot_token"`
	DmPolicy     types.String `tfsdk:"dm_policy"`
	AllowFrom    types.List   `tfsdk:"allow_from"`
	StreamMode   types.String `tfsdk:"stream_mode"`
	ReplyToMode  types.String `tfsdk:"reply_to_mode"`
	LinkPreview  types.Bool   `tfsdk:"link_preview"`
	HistoryLimit types.Int64  `tfsdk:"history_limit"`
	MediaMaxMb   types.Int64  `tfsdk:"media_max_mb"`
	WebhookURL   types.String `tfsdk:"webhook_url"`
}

func NewChannelTelegramResource() resource.Resource {
	return &ChannelTelegramResource{}
}

func (r *ChannelTelegramResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_telegram"
}

func (r *ChannelTelegramResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw Telegram channel configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable the Telegram channel.",
				Optional:    true,
			},
			"bot_token": schema.StringAttribute{
				Description: "Telegram bot token. Sensitive. Falls back to TELEGRAM_BOT_TOKEN env var.",
				Optional:    true,
				Sensitive:   true,
			},
			"dm_policy": schema.StringAttribute{
				Description: "DM policy: pairing (default), allowlist, open, disabled.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("pairing"),
			},
			"allow_from": schema.ListAttribute{
				Description: "Telegram user IDs allowed to message the bot (e.g. tg:123456789).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"stream_mode": schema.StringAttribute{
				Description: "Stream preview mode: off, partial, block.",
				Optional:    true,
			},
			"reply_to_mode": schema.StringAttribute{
				Description: "Reply-to behavior: off, first, all.",
				Optional:    true,
			},
			"link_preview": schema.BoolAttribute{
				Description: "Enable link previews in outbound messages.",
				Optional:    true,
			},
			"history_limit": schema.Int64Attribute{
				Description: "Max chat history messages to fetch for context.",
				Optional:    true,
			},
			"media_max_mb": schema.Int64Attribute{
				Description: "Max inbound media size in MB.",
				Optional:    true,
			},
			"webhook_url": schema.StringAttribute{
				Description: "Webhook URL for Telegram webhook mode.",
				Optional:    true,
			},
		},
	}
}

func (r *ChannelTelegramResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChannelTelegramResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ChannelTelegramModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tg := r.modelToMap(ctx, plan)

	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchNestedSection(ctx, r.client, tg, cfg.Hash, "channels", "telegram"); err != nil {
		resp.Diagnostics.AddError("Failed to write Telegram config", err.Error())
		return
	}

	plan.ID = types.StringValue("channel_telegram")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelTelegramResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ChannelTelegramModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "telegram")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Telegram config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("channel_telegram")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelTelegramResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChannelTelegramModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tg := r.modelToMap(ctx, plan)

	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchNestedSection(ctx, r.client, tg, cfg.Hash, "channels", "telegram"); err != nil {
		resp.Diagnostics.AddError("Failed to write Telegram config", err.Error())
		return
	}

	plan.ID = types.StringValue("channel_telegram")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelTelegramResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "channels", "telegram"); err != nil {
		resp.Diagnostics.AddError("Failed to delete Telegram config", err.Error())
		return
	}
}

func (r *ChannelTelegramResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "telegram")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Telegram config", err.Error())
		return
	}

	var state ChannelTelegramModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("channel_telegram")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelTelegramResource) modelToMap(ctx context.Context, m ChannelTelegramModel) map[string]any {
	tg := make(map[string]any)

	if !m.Enabled.IsNull() && !m.Enabled.IsUnknown() {
		tg["enabled"] = m.Enabled.ValueBool()
	}
	if !m.BotToken.IsNull() && !m.BotToken.IsUnknown() {
		tg["botToken"] = m.BotToken.ValueString()
	}
	if !m.DmPolicy.IsNull() && !m.DmPolicy.IsUnknown() {
		tg["dmPolicy"] = m.DmPolicy.ValueString()
	}
	if !m.AllowFrom.IsNull() && !m.AllowFrom.IsUnknown() {
		var af []string
		m.AllowFrom.ElementsAs(ctx, &af, false)
		tg["allowFrom"] = af
	}
	if !m.StreamMode.IsNull() && !m.StreamMode.IsUnknown() {
		tg["streamMode"] = m.StreamMode.ValueString()
	}
	if !m.ReplyToMode.IsNull() && !m.ReplyToMode.IsUnknown() {
		tg["replyToMode"] = m.ReplyToMode.ValueString()
	}
	if !m.LinkPreview.IsNull() && !m.LinkPreview.IsUnknown() {
		tg["linkPreview"] = m.LinkPreview.ValueBool()
	}
	if !m.HistoryLimit.IsNull() && !m.HistoryLimit.IsUnknown() {
		tg["historyLimit"] = m.HistoryLimit.ValueInt64()
	}
	if !m.MediaMaxMb.IsNull() && !m.MediaMaxMb.IsUnknown() {
		tg["mediaMaxMb"] = m.MediaMaxMb.ValueInt64()
	}
	if !m.WebhookURL.IsNull() && !m.WebhookURL.IsUnknown() {
		tg["webhookUrl"] = m.WebhookURL.ValueString()
	}

	return tg
}

func (r *ChannelTelegramResource) mapToModel(ctx context.Context, section map[string]any, m *ChannelTelegramModel) {
	if v, ok := section["enabled"].(bool); ok {
		m.Enabled = types.BoolValue(v)
	}
	// Don't read back bot token for security
	if v, ok := section["dmPolicy"].(string); ok {
		m.DmPolicy = types.StringValue(v)
	}
	if v, ok := section["allowFrom"].([]any); ok {
		strs := make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				strs = append(strs, str)
			}
		}
		list, _ := types.ListValueFrom(ctx, types.StringType, strs)
		m.AllowFrom = list
	}
	if v, ok := section["streamMode"].(string); ok {
		m.StreamMode = types.StringValue(v)
	}
	if v, ok := section["replyToMode"].(string); ok {
		m.ReplyToMode = types.StringValue(v)
	}
	if v, ok := section["linkPreview"].(bool); ok {
		m.LinkPreview = types.BoolValue(v)
	}
	if v, ok := section["historyLimit"].(float64); ok {
		m.HistoryLimit = types.Int64Value(int64(v))
	}
	if v, ok := section["mediaMaxMb"].(float64); ok {
		m.MediaMaxMb = types.Int64Value(int64(v))
	}
	if v, ok := section["webhookUrl"].(string); ok {
		m.WebhookURL = types.StringValue(v)
	}
}
