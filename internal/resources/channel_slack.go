package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ resource.Resource = &ChannelSlackResource{}
var _ resource.ResourceWithImportState = &ChannelSlackResource{}

type ChannelSlackResource struct {
	client client.Client
}

type ChannelSlackModel struct {
	ID                    types.String `tfsdk:"id"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	BotToken              types.String `tfsdk:"bot_token"`
	AppToken              types.String `tfsdk:"app_token"`
	DmPolicy              types.String `tfsdk:"dm_policy"`
	AllowFrom             types.List   `tfsdk:"allow_from"`
	AllowBots             types.Bool   `tfsdk:"allow_bots"`
	HistoryLimit          types.Int64  `tfsdk:"history_limit"`
	TextChunkLimit        types.Int64  `tfsdk:"text_chunk_limit"`
	ChunkMode             types.String `tfsdk:"chunk_mode"`
	MediaMaxMb            types.Int64  `tfsdk:"media_max_mb"`
	ReplyToMode           types.String `tfsdk:"reply_to_mode"`
	ReactionNotifications types.String `tfsdk:"reaction_notifications"`
}

func NewChannelSlackResource() resource.Resource {
	return &ChannelSlackResource{}
}

func (r *ChannelSlackResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_slack"
}

func (r *ChannelSlackResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw Slack channel configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable the Slack channel.",
				Optional:    true,
			},
			"bot_token": schema.StringAttribute{
				Description: "Slack bot token (xoxb-...). Sensitive. Falls back to SLACK_BOT_TOKEN.",
				Optional:    true,
				Sensitive:   true,
			},
			"app_token": schema.StringAttribute{
				Description: "Slack app token (xapp-...) for Socket Mode. Sensitive. Falls back to SLACK_APP_TOKEN.",
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
				Description: "Slack user IDs allowed to message the bot.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"allow_bots": schema.BoolAttribute{
				Description: "Allow messages from other bots. Default: false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"history_limit": schema.Int64Attribute{
				Description: "Max chat history messages. Default: 50.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(50),
			},
			"text_chunk_limit": schema.Int64Attribute{
				Description: "Max characters per chunk. Default: 4000.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4000),
			},
			"chunk_mode": schema.StringAttribute{
				Description: "Chunk mode: length or newline.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("length"),
			},
			"media_max_mb": schema.Int64Attribute{
				Description: "Max inbound media size in MB. Default: 20.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20),
			},
			"reply_to_mode": schema.StringAttribute{
				Description: "Reply-to behavior: off, first, all.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("off"),
			},
			"reaction_notifications": schema.StringAttribute{
				Description: "Reaction notification mode: off, own (default), all, allowlist.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("own"),
			},
		},
	}
}

func (r *ChannelSlackResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChannelSlackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ChannelSlackModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "slack"); err != nil {
		resp.Diagnostics.AddError("Failed to write Slack config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_slack")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelSlackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ChannelSlackModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "slack")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Slack config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("channel_slack")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelSlackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChannelSlackModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "slack"); err != nil {
		resp.Diagnostics.AddError("Failed to write Slack config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_slack")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelSlackResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "channels", "slack"); err != nil {
		resp.Diagnostics.AddError("Failed to delete Slack config", err.Error())
		return
	}
}

func (r *ChannelSlackResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "slack")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Slack config", err.Error())
		return
	}
	var state ChannelSlackModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("channel_slack")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelSlackResource) modelToMap(ctx context.Context, m ChannelSlackModel) map[string]any {
	d := make(map[string]any)
	setIfBool(d, "enabled", m.Enabled)
	setIfString(d, "botToken", m.BotToken)
	setIfString(d, "appToken", m.AppToken)
	setIfString(d, "dmPolicy", m.DmPolicy)
	setIfStringList(ctx, d, "allowFrom", m.AllowFrom)
	setIfBool(d, "allowBots", m.AllowBots)
	setIfInt64(d, "historyLimit", m.HistoryLimit)
	setIfInt64(d, "textChunkLimit", m.TextChunkLimit)
	setIfString(d, "chunkMode", m.ChunkMode)
	setIfInt64(d, "mediaMaxMb", m.MediaMaxMb)
	setIfString(d, "replyToMode", m.ReplyToMode)
	setIfString(d, "reactionNotifications", m.ReactionNotifications)
	return d
}

func (r *ChannelSlackResource) mapToModel(ctx context.Context, s map[string]any, m *ChannelSlackModel) {
	readBool(s, "enabled", &m.Enabled)
	readString(s, "dmPolicy", &m.DmPolicy)
	readStringList(ctx, s, "allowFrom", &m.AllowFrom)
	readBool(s, "allowBots", &m.AllowBots)
	readFloat64AsInt64(s, "historyLimit", &m.HistoryLimit)
	readFloat64AsInt64(s, "textChunkLimit", &m.TextChunkLimit)
	readString(s, "chunkMode", &m.ChunkMode)
	readFloat64AsInt64(s, "mediaMaxMb", &m.MediaMaxMb)
	readString(s, "replyToMode", &m.ReplyToMode)
	readString(s, "reactionNotifications", &m.ReactionNotifications)
}
