package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ resource.Resource = &PluginResource{}
var _ resource.ResourceWithImportState = &PluginResource{}

type PluginResource struct {
	client client.Client
}

type PluginModel struct {
	ID         types.String `tfsdk:"id"`
	PluginID   types.String `tfsdk:"plugin_id"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	ConfigJSON types.String `tfsdk:"config_json"`
}

func NewPluginResource() resource.Resource {
	return &PluginResource{}
}

func (r *PluginResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugin"
}

func (r *PluginResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenClaw plugin entry.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"plugin_id": schema.StringAttribute{
				Description: "Unique plugin identifier. Used as the key under plugins.entries.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable this plugin.",
				Optional:    true,
			},
			"config_json": schema.StringAttribute{
				Description: "Raw JSON string containing plugin-specific configuration.",
				Optional:    true,
			},
		},
	}
}

func (r *PluginResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PluginResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PluginModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	m, diags := r.modelToMap(plan)
	if diags != nil {
		resp.Diagnostics.AddError("Invalid config_json", diags.Error())
		return
	}
	pluginID := plan.PluginID.ValueString()
	if err := client.PatchNestedSection(ctx, r.client, m, cfg.Hash, "plugins", "entries", pluginID); err != nil {
		resp.Diagnostics.AddError("Failed to write plugin config", err.Error())
		return
	}
	plan.ID = types.StringValue(pluginID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PluginResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PluginModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pluginID := state.PluginID.ValueString()
	section, _, err := client.GetNestedSection(ctx, r.client, "plugins", "entries", pluginID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read plugin config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(section, &state)
	state.ID = types.StringValue(pluginID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PluginResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PluginModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	m, diags := r.modelToMap(plan)
	if diags != nil {
		resp.Diagnostics.AddError("Invalid config_json", diags.Error())
		return
	}
	pluginID := plan.PluginID.ValueString()
	if err := client.PatchNestedSection(ctx, r.client, m, cfg.Hash, "plugins", "entries", pluginID); err != nil {
		resp.Diagnostics.AddError("Failed to write plugin config", err.Error())
		return
	}
	plan.ID = types.StringValue(pluginID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PluginResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PluginModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	pluginID := state.PluginID.ValueString()
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "plugins", "entries", pluginID); err != nil {
		resp.Diagnostics.AddError("Failed to delete plugin config", err.Error())
		return
	}
}

func (r *PluginResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pluginID := req.ID
	section, _, err := client.GetNestedSection(ctx, r.client, "plugins", "entries", pluginID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import plugin config", err.Error())
		return
	}
	var state PluginModel
	state.PluginID = types.StringValue(pluginID)
	if section != nil {
		r.mapToModel(section, &state)
	}
	state.ID = types.StringValue(pluginID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PluginResource) modelToMap(m PluginModel) (map[string]any, error) {
	d := make(map[string]any)
	setIfBool(d, "enabled", m.Enabled)
	if !m.ConfigJSON.IsNull() && !m.ConfigJSON.IsUnknown() {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(m.ConfigJSON.ValueString()), &parsed); err != nil {
			return nil, fmt.Errorf("config_json must be a valid JSON object: %w", err)
		}
		for k, v := range parsed {
			d[k] = v
		}
	}
	return d, nil
}

func (r *PluginResource) mapToModel(s map[string]any, m *PluginModel) {
	readBool(s, "enabled", &m.Enabled)
	// Rebuild config_json from the section, excluding the "enabled" key.
	extra := make(map[string]any)
	for k, v := range s {
		if k == "enabled" {
			continue
		}
		extra[k] = v
	}
	if len(extra) > 0 {
		b, _ := json.Marshal(extra)
		m.ConfigJSON = types.StringValue(string(b))
	}
}
