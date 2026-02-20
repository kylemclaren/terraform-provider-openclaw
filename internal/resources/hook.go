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

var _ resource.Resource = &HookResource{}
var _ resource.ResourceWithImportState = &HookResource{}

type HookResource struct {
	client client.Client
}

type HookModel struct {
	ID                types.String `tfsdk:"id"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	Token             types.String `tfsdk:"token"`
	Path              types.String `tfsdk:"path"`
	DefaultSessionKey types.String `tfsdk:"default_session_key"`
}

func NewHookResource() resource.Resource {
	return &HookResource{}
}

func (r *HookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hook"
}

func (r *HookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw hooks configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable hooks.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "Authentication token for hooks. Sensitive.",
				Optional:    true,
				Sensitive:   true,
			},
			"path": schema.StringAttribute{
				Description: "URL path prefix for hooks. Default: /hooks.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/hooks"),
			},
			"default_session_key": schema.StringAttribute{
				Description: "Default session key used when no key is specified in the hook request.",
				Optional:    true,
			},
		},
	}
}

func (r *HookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *HookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan HookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(plan), cfg.Hash, "hooks"); err != nil {
		resp.Diagnostics.AddError("Failed to write hooks config", err.Error())
		return
	}
	plan.ID = types.StringValue("hooks")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state HookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	section, _, err := client.GetNestedSection(ctx, r.client, "hooks")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read hooks config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(section, &state)
	state.ID = types.StringValue("hooks")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan HookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(plan), cfg.Hash, "hooks"); err != nil {
		resp.Diagnostics.AddError("Failed to write hooks config", err.Error())
		return
	}
	plan.ID = types.StringValue("hooks")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *HookResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "hooks"); err != nil {
		resp.Diagnostics.AddError("Failed to delete hooks config", err.Error())
		return
	}
}

func (r *HookResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "hooks")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import hooks config", err.Error())
		return
	}
	var state HookModel
	if section != nil {
		r.mapToModel(section, &state)
	}
	state.ID = types.StringValue("hooks")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *HookResource) modelToMap(m HookModel) map[string]any {
	d := make(map[string]any)
	setIfBool(d, "enabled", m.Enabled)
	setIfString(d, "token", m.Token)
	setIfString(d, "path", m.Path)
	setIfString(d, "defaultSessionKey", m.DefaultSessionKey)
	return d
}

func (r *HookResource) mapToModel(s map[string]any, m *HookModel) {
	readBool(s, "enabled", &m.Enabled)
	readString(s, "token", &m.Token)
	readString(s, "path", &m.Path)
	readString(s, "defaultSessionKey", &m.DefaultSessionKey)
}
