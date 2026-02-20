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

var _ resource.Resource = &ChannelWhatsAppResource{}
var _ resource.ResourceWithImportState = &ChannelWhatsAppResource{}

type ChannelWhatsAppResource struct {
	client client.Client
}

type ChannelWhatsAppModel struct {
	ID               types.String `tfsdk:"id"`
	DmPolicy         types.String `tfsdk:"dm_policy"`
	AllowFrom        types.List   `tfsdk:"allow_from"`
	TextChunkLimit   types.Int64  `tfsdk:"text_chunk_limit"`
	ChunkMode        types.String `tfsdk:"chunk_mode"`
	MediaMaxMb       types.Int64  `tfsdk:"media_max_mb"`
	SendReadReceipts types.Bool   `tfsdk:"send_read_receipts"`
	GroupPolicy      types.String `tfsdk:"group_policy"`
}

func NewChannelWhatsAppResource() resource.Resource {
	return &ChannelWhatsAppResource{}
}

func (r *ChannelWhatsAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_whatsapp"
}

func (r *ChannelWhatsAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw WhatsApp channel configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"dm_policy": schema.StringAttribute{
				Description: "DM policy: pairing (default), allowlist, open, disabled.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("pairing"),
			},
			"allow_from": schema.ListAttribute{
				Description: "Phone numbers allowed to message the bot (e.g. +15555550123).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"text_chunk_limit": schema.Int64Attribute{
				Description: "Max characters per outbound message chunk. Default: 4000.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4000),
			},
			"chunk_mode": schema.StringAttribute{
				Description: "Chunk splitting mode: length or newline.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("length"),
			},
			"media_max_mb": schema.Int64Attribute{
				Description: "Max inbound media size in MB. Default: 50.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(50),
			},
			"send_read_receipts": schema.BoolAttribute{
				Description: "Send read receipts (blue ticks). Default: true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"group_policy": schema.StringAttribute{
				Description: "Group policy: allowlist (default), open, disabled.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("allowlist"),
			},
		},
	}
}

func (r *ChannelWhatsAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ChannelWhatsAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ChannelWhatsAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wa := r.modelToMap(ctx, plan)

	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchNestedSection(ctx, r.client, wa, cfg.Hash, "channels", "whatsapp"); err != nil {
		resp.Diagnostics.AddError("Failed to write WhatsApp config", err.Error())
		return
	}

	plan.ID = types.StringValue("channel_whatsapp")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelWhatsAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ChannelWhatsAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "whatsapp")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read WhatsApp config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("channel_whatsapp")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelWhatsAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ChannelWhatsAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wa := r.modelToMap(ctx, plan)

	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchNestedSection(ctx, r.client, wa, cfg.Hash, "channels", "whatsapp"); err != nil {
		resp.Diagnostics.AddError("Failed to write WhatsApp config", err.Error())
		return
	}

	plan.ID = types.StringValue("channel_whatsapp")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ChannelWhatsAppResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}

	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "channels", "whatsapp"); err != nil {
		resp.Diagnostics.AddError("Failed to delete WhatsApp config", err.Error())
		return
	}
}

func (r *ChannelWhatsAppResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "channels", "whatsapp")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import WhatsApp config", err.Error())
		return
	}

	var state ChannelWhatsAppModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("channel_whatsapp")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ChannelWhatsAppResource) modelToMap(ctx context.Context, m ChannelWhatsAppModel) map[string]any {
	wa := make(map[string]any)

	if !m.DmPolicy.IsNull() && !m.DmPolicy.IsUnknown() {
		wa["dmPolicy"] = m.DmPolicy.ValueString()
	}
	if !m.AllowFrom.IsNull() && !m.AllowFrom.IsUnknown() {
		var af []string
		m.AllowFrom.ElementsAs(ctx, &af, false)
		wa["allowFrom"] = af
	}
	if !m.TextChunkLimit.IsNull() && !m.TextChunkLimit.IsUnknown() {
		wa["textChunkLimit"] = m.TextChunkLimit.ValueInt64()
	}
	if !m.ChunkMode.IsNull() && !m.ChunkMode.IsUnknown() {
		wa["chunkMode"] = m.ChunkMode.ValueString()
	}
	if !m.MediaMaxMb.IsNull() && !m.MediaMaxMb.IsUnknown() {
		wa["mediaMaxMb"] = m.MediaMaxMb.ValueInt64()
	}
	if !m.SendReadReceipts.IsNull() && !m.SendReadReceipts.IsUnknown() {
		wa["sendReadReceipts"] = m.SendReadReceipts.ValueBool()
	}
	if !m.GroupPolicy.IsNull() && !m.GroupPolicy.IsUnknown() {
		wa["groupPolicy"] = m.GroupPolicy.ValueString()
	}

	return wa
}

func (r *ChannelWhatsAppResource) mapToModel(ctx context.Context, section map[string]any, m *ChannelWhatsAppModel) {
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
	if v, ok := section["textChunkLimit"].(float64); ok {
		m.TextChunkLimit = types.Int64Value(int64(v))
	}
	if v, ok := section["chunkMode"].(string); ok {
		m.ChunkMode = types.StringValue(v)
	}
	if v, ok := section["mediaMaxMb"].(float64); ok {
		m.MediaMaxMb = types.Int64Value(int64(v))
	}
	if v, ok := section["sendReadReceipts"].(bool); ok {
		m.SendReadReceipts = types.BoolValue(v)
	}
	if v, ok := section["groupPolicy"].(string); ok {
		m.GroupPolicy = types.StringValue(v)
	}
}
