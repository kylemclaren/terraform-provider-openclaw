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

var _ resource.Resource = &ChannelIMessageResource{}
var _ resource.ResourceWithImportState = &ChannelIMessageResource{}

type ChannelIMessageResource struct {
	client client.Client
}

type ChannelIMessageModel struct {
	ID           types.String `tfsdk:"id"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	DmPolicy     types.String `tfsdk:"dm_policy"`
	AllowFrom    types.List   `tfsdk:"allow_from"`
	HistoryLimit types.Int64  `tfsdk:"history_limit"`
	MediaMaxMb   types.Int64  `tfsdk:"media_max_mb"`
	Service      types.String `tfsdk:"service"`
	Region       types.String `tfsdk:"region"`
}

func NewChannelIMessageResource() resource.Resource {
	return &ChannelIMessageResource{}
}

func (r *ChannelIMessageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_imessage"
}

func (r *ChannelIMessageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw iMessage channel configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable the iMessage channel.",
				Optional:    true,
			},
			"dm_policy": schema.StringAttribute{
				Description: "DM policy: pairing (default), allowlist, open, disabled.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("pairing"),
			},
			"allow_from": schema.ListAttribute{
				Description: "Phone numbers or identifiers allowed to message.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"history_limit": schema.Int64Attribute{
				Description: "Max chat history messages to fetch. Default: 50.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(50),
			},
			"media_max_mb": schema.Int64Attribute{
				Description: "Max inbound media size in MB. Default: 16.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(16),
			},
			"service": schema.StringAttribute{
				Description: "iMessage service selection. Optional, defaults to auto.",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "Region for the iMessage channel. Optional.",
				Optional:    true,
			},
		},
	}
}

func (r *ChannelIMessageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChannelIMessageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ChannelIMessageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "imessage"); err != nil {
		resp.Diagnostics.AddError("Failed to write iMessage config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_imessage")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelIMessageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ChannelIMessageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "imessage")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read iMessage config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("channel_imessage")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelIMessageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChannelIMessageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "imessage"); err != nil {
		resp.Diagnostics.AddError("Failed to write iMessage config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_imessage")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelIMessageResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "channels", "imessage"); err != nil {
		resp.Diagnostics.AddError("Failed to delete iMessage config", err.Error())
		return
	}
}

func (r *ChannelIMessageResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "imessage")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import iMessage config", err.Error())
		return
	}
	var state ChannelIMessageModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("channel_imessage")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelIMessageResource) modelToMap(ctx context.Context, m ChannelIMessageModel) map[string]any {
	d := make(map[string]any)
	setIfBool(d, "enabled", m.Enabled)
	setIfString(d, "dmPolicy", m.DmPolicy)
	setIfStringList(ctx, d, "allowFrom", m.AllowFrom)
	setIfInt64(d, "historyLimit", m.HistoryLimit)
	setIfInt64(d, "mediaMaxMb", m.MediaMaxMb)
	setIfString(d, "service", m.Service)
	setIfString(d, "region", m.Region)
	return d
}

func (r *ChannelIMessageResource) mapToModel(ctx context.Context, s map[string]any, m *ChannelIMessageModel) {
	readBool(s, "enabled", &m.Enabled)
	readString(s, "dmPolicy", &m.DmPolicy)
	readStringList(ctx, s, "allowFrom", &m.AllowFrom)
	readFloat64AsInt64(s, "historyLimit", &m.HistoryLimit)
	readFloat64AsInt64(s, "mediaMaxMb", &m.MediaMaxMb)
	readString(s, "service", &m.Service)
	readString(s, "region", &m.Region)
}
