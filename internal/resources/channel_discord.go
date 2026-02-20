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

var _ resource.Resource = &ChannelDiscordResource{}
var _ resource.ResourceWithImportState = &ChannelDiscordResource{}

type ChannelDiscordResource struct {
	client client.Client
}

type ChannelDiscordModel struct {
	ID               types.String `tfsdk:"id"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Token            types.String `tfsdk:"token"`
	DmPolicy         types.String `tfsdk:"dm_policy"`
	AllowFrom        types.List   `tfsdk:"allow_from"`
	AllowBots        types.Bool   `tfsdk:"allow_bots"`
	MediaMaxMb       types.Int64  `tfsdk:"media_max_mb"`
	TextChunkLimit   types.Int64  `tfsdk:"text_chunk_limit"`
	ChunkMode        types.String `tfsdk:"chunk_mode"`
	HistoryLimit     types.Int64  `tfsdk:"history_limit"`
	ReplyToMode      types.String `tfsdk:"reply_to_mode"`
	ActionsReactions types.Bool   `tfsdk:"actions_reactions"`
	ActionsMessages  types.Bool   `tfsdk:"actions_messages"`
	ActionsThreads   types.Bool   `tfsdk:"actions_threads"`
	ActionsPins      types.Bool   `tfsdk:"actions_pins"`
	ActionsSearch    types.Bool   `tfsdk:"actions_search"`
}

func NewChannelDiscordResource() resource.Resource {
	return &ChannelDiscordResource{}
}

func (r *ChannelDiscordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_discord"
}

func (r *ChannelDiscordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw Discord channel configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable the Discord channel.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "Discord bot token. Sensitive. Falls back to DISCORD_BOT_TOKEN.",
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
				Description: "Discord user IDs or usernames allowed to message.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"allow_bots": schema.BoolAttribute{
				Description: "Allow messages from other bots. Default: false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"media_max_mb": schema.Int64Attribute{
				Description: "Max inbound media size in MB. Default: 8.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(8),
			},
			"text_chunk_limit": schema.Int64Attribute{
				Description: "Max characters per outbound message chunk. Default: 2000.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2000),
			},
			"chunk_mode": schema.StringAttribute{
				Description: "Chunk splitting mode: length or newline.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("length"),
			},
			"history_limit": schema.Int64Attribute{
				Description: "Max chat history messages to fetch. Default: 20.",
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
			"actions_reactions": schema.BoolAttribute{
				Description: "Enable reaction actions.",
				Optional:    true,
			},
			"actions_messages": schema.BoolAttribute{
				Description: "Enable message actions (read/send/edit/delete).",
				Optional:    true,
			},
			"actions_threads": schema.BoolAttribute{
				Description: "Enable thread actions.",
				Optional:    true,
			},
			"actions_pins": schema.BoolAttribute{
				Description: "Enable pin actions.",
				Optional:    true,
			},
			"actions_search": schema.BoolAttribute{
				Description: "Enable search actions.",
				Optional:    true,
			},
		},
	}
}

func (r *ChannelDiscordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChannelDiscordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ChannelDiscordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "discord"); err != nil {
		resp.Diagnostics.AddError("Failed to write Discord config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_discord")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelDiscordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ChannelDiscordModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "discord")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Discord config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("channel_discord")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelDiscordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChannelDiscordModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "discord"); err != nil {
		resp.Diagnostics.AddError("Failed to write Discord config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_discord")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelDiscordResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "channels", "discord"); err != nil {
		resp.Diagnostics.AddError("Failed to delete Discord config", err.Error())
		return
	}
}

func (r *ChannelDiscordResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "discord")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Discord config", err.Error())
		return
	}
	var state ChannelDiscordModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("channel_discord")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelDiscordResource) modelToMap(ctx context.Context, m ChannelDiscordModel) map[string]any {
	d := make(map[string]any)
	setIfBool(d, "enabled", m.Enabled)
	setIfString(d, "token", m.Token)
	setIfString(d, "dmPolicy", m.DmPolicy)
	setIfStringList(ctx, d, "allowFrom", m.AllowFrom)
	setIfBool(d, "allowBots", m.AllowBots)
	setIfInt64(d, "mediaMaxMb", m.MediaMaxMb)
	setIfInt64(d, "textChunkLimit", m.TextChunkLimit)
	setIfString(d, "chunkMode", m.ChunkMode)
	setIfInt64(d, "historyLimit", m.HistoryLimit)
	setIfString(d, "replyToMode", m.ReplyToMode)

	actions := make(map[string]any)
	setIfBool(actions, "reactions", m.ActionsReactions)
	setIfBool(actions, "messages", m.ActionsMessages)
	setIfBool(actions, "threads", m.ActionsThreads)
	setIfBool(actions, "pins", m.ActionsPins)
	setIfBool(actions, "search", m.ActionsSearch)
	if len(actions) > 0 {
		d["actions"] = actions
	}

	return d
}

func (r *ChannelDiscordResource) mapToModel(ctx context.Context, s map[string]any, m *ChannelDiscordModel) {
	readBool(s, "enabled", &m.Enabled)
	// Don't read back token
	readString(s, "dmPolicy", &m.DmPolicy)
	readStringList(ctx, s, "allowFrom", &m.AllowFrom)
	readBool(s, "allowBots", &m.AllowBots)
	readFloat64AsInt64(s, "mediaMaxMb", &m.MediaMaxMb)
	readFloat64AsInt64(s, "textChunkLimit", &m.TextChunkLimit)
	readString(s, "chunkMode", &m.ChunkMode)
	readFloat64AsInt64(s, "historyLimit", &m.HistoryLimit)
	readString(s, "replyToMode", &m.ReplyToMode)

	if actions, ok := s["actions"].(map[string]any); ok {
		readBool(actions, "reactions", &m.ActionsReactions)
		readBool(actions, "messages", &m.ActionsMessages)
		readBool(actions, "threads", &m.ActionsThreads)
		readBool(actions, "pins", &m.ActionsPins)
		readBool(actions, "search", &m.ActionsSearch)
	}
}
