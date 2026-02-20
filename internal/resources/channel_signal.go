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

var _ resource.Resource = &ChannelSignalResource{}
var _ resource.ResourceWithImportState = &ChannelSignalResource{}

type ChannelSignalResource struct {
	client client.Client
}

type ChannelSignalModel struct {
	ID                    types.String `tfsdk:"id"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	DmPolicy              types.String `tfsdk:"dm_policy"`
	AllowFrom             types.List   `tfsdk:"allow_from"`
	ReactionNotifications types.String `tfsdk:"reaction_notifications"`
	HistoryLimit          types.Int64  `tfsdk:"history_limit"`
}

func NewChannelSignalResource() resource.Resource {
	return &ChannelSignalResource{}
}

func (r *ChannelSignalResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_signal"
}

func (r *ChannelSignalResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw Signal channel configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable the Signal channel.",
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
			"reaction_notifications": schema.StringAttribute{
				Description: "Reaction notification mode: own (default), all, none.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("own"),
			},
			"history_limit": schema.Int64Attribute{
				Description: "Max chat history messages to fetch. Default: 50.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(50),
			},
		},
	}
}

func (r *ChannelSignalResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChannelSignalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ChannelSignalModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "signal"); err != nil {
		resp.Diagnostics.AddError("Failed to write Signal config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_signal")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelSignalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ChannelSignalModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "signal")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Signal config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("channel_signal")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelSignalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChannelSignalModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "channels", "signal"); err != nil {
		resp.Diagnostics.AddError("Failed to write Signal config", err.Error())
		return
	}
	plan.ID = types.StringValue("channel_signal")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelSignalResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "channels", "signal"); err != nil {
		resp.Diagnostics.AddError("Failed to delete Signal config", err.Error())
		return
	}
}

func (r *ChannelSignalResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "signal")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import Signal config", err.Error())
		return
	}
	var state ChannelSignalModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("channel_signal")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelSignalResource) modelToMap(ctx context.Context, m ChannelSignalModel) map[string]any {
	d := make(map[string]any)
	setIfBool(d, "enabled", m.Enabled)
	setIfString(d, "dmPolicy", m.DmPolicy)
	setIfStringList(ctx, d, "allowFrom", m.AllowFrom)
	setIfString(d, "reactionNotifications", m.ReactionNotifications)
	setIfInt64(d, "historyLimit", m.HistoryLimit)
	return d
}

func (r *ChannelSignalResource) mapToModel(ctx context.Context, s map[string]any, m *ChannelSignalModel) {
	readBool(s, "enabled", &m.Enabled)
	readString(s, "dmPolicy", &m.DmPolicy)
	readStringList(ctx, s, "allowFrom", &m.AllowFrom)
	readString(s, "reactionNotifications", &m.ReactionNotifications)
	readFloat64AsInt64(s, "historyLimit", &m.HistoryLimit)
}
