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

var _ resource.Resource = &ChannelGoogleChatResource{}
var _ resource.ResourceWithImportState = &ChannelGoogleChatResource{}

type ChannelGoogleChatResource struct {
	client client.Client
}

type ChannelGoogleChatModel struct {
	ID          types.String `tfsdk:"id"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	WebhookPath types.String `tfsdk:"webhook_path"`
	BotUser     types.String `tfsdk:"bot_user"`
	DmPolicy    types.String `tfsdk:"dm_policy"`
	DmAllowFrom types.List   `tfsdk:"dm_allow_from"`
	GroupPolicy types.String `tfsdk:"group_policy"`
	MediaMaxMb  types.Int64  `tfsdk:"media_max_mb"`
}

func NewChannelGoogleChatResource() resource.Resource {
	return &ChannelGoogleChatResource{}
}

func (r *ChannelGoogleChatResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_googlechat"
}

func (r *ChannelGoogleChatResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw Google Chat channel configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable the Google Chat channel.",
				Optional:    true,
			},
			"webhook_path": schema.StringAttribute{
				Description: "Webhook path for incoming messages.",
				Optional:    true,
			},
			"bot_user": schema.StringAttribute{
				Description: "Bot user identifier for Google Chat.",
				Optional:    true,
			},
			"dm_policy": schema.StringAttribute{
				Description: "DM policy: pairing (default), allowlist, open, disabled.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("pairing"),
			},
			"dm_allow_from": schema.ListAttribute{
				Description: "User identifiers allowed to send direct messages.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"group_policy": schema.StringAttribute{
				Description: "Group message policy: allowlist (default), open, disabled.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("allowlist"),
			},
			"media_max_mb": schema.Int64Attribute{
				Description: "Max inbound media size in MB. Default: 20.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(20),
			},
		},
	}
}

func (r *ChannelGoogleChatResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChannelGoogleChatResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ChannelGoogleChatModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "googlechat"); err != nil {
		resp.Diagnostics.AddError("Failed to write Google Chat config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_googlechat")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelGoogleChatResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ChannelGoogleChatModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "googlechat")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Google Chat config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("channel_googlechat")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelGoogleChatResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChannelGoogleChatModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "googlechat"); err != nil {
		resp.Diagnostics.AddError("Failed to write Google Chat config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_googlechat")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelGoogleChatResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "channels", "googlechat"); err != nil {
		resp.Diagnostics.AddError("Failed to delete Google Chat config", err.Error())
		return
	}
}

func (r *ChannelGoogleChatResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "googlechat")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Google Chat config", err.Error())
		return
	}
	var state ChannelGoogleChatModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("channel_googlechat")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelGoogleChatResource) modelToMap(ctx context.Context, m ChannelGoogleChatModel) map[string]any {
	d := make(map[string]any)
	setIfBool(d, "enabled", m.Enabled)
	setIfString(d, "webhookPath", m.WebhookPath)
	setIfString(d, "botUser", m.BotUser)

	dm := make(map[string]any)
	setIfString(dm, "policy", m.DmPolicy)
	setIfStringList(ctx, dm, "allowFrom", m.DmAllowFrom)
	if len(dm) > 0 {
		d["dm"] = dm
	}

	setIfString(d, "groupPolicy", m.GroupPolicy)
	setIfInt64(d, "mediaMaxMb", m.MediaMaxMb)
	return d
}

func (r *ChannelGoogleChatResource) mapToModel(ctx context.Context, s map[string]any, m *ChannelGoogleChatModel) {
	readBool(s, "enabled", &m.Enabled)
	readString(s, "webhookPath", &m.WebhookPath)
	readString(s, "botUser", &m.BotUser)

	if dm, ok := s["dm"].(map[string]any); ok {
		readString(dm, "policy", &m.DmPolicy)
		readStringList(ctx, dm, "allowFrom", &m.DmAllowFrom)
	}

	readString(s, "groupPolicy", &m.GroupPolicy)
	readFloat64AsInt64(s, "mediaMaxMb", &m.MediaMaxMb)
}
