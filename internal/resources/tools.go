package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ resource.Resource = &ToolsResource{}
var _ resource.ResourceWithImportState = &ToolsResource{}

type ToolsResource struct {
	client client.Client
}

type ToolsModel struct {
	ID              types.String `tfsdk:"id"`
	Profile         types.String `tfsdk:"profile"`
	Allow           types.List   `tfsdk:"allow"`
	Deny            types.List   `tfsdk:"deny"`
	ElevatedEnabled types.Bool   `tfsdk:"elevated_enabled"`
	BrowserEnabled  types.Bool   `tfsdk:"browser_enabled"`
}

func NewToolsResource() resource.Resource {
	return &ToolsResource{}
}

func (r *ToolsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tools"
}

func (r *ToolsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw tools configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"profile": schema.StringAttribute{
				Description: "Tools profile: minimal, coding, messaging, or full.",
				Optional:    true,
			},
			"allow": schema.ListAttribute{
				Description: "Explicit list of tool names to allow.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"deny": schema.ListAttribute{
				Description: "Explicit list of tool names to deny.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"elevated_enabled": schema.BoolAttribute{
				Description: "Enable elevated (privileged) tool execution.",
				Optional:    true,
			},
			"browser_enabled": schema.BoolAttribute{
				Description: "Enable browser-based tools.",
				Optional:    true,
			},
		},
	}
}

func (r *ToolsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ToolsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ToolsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "tools"); err != nil {
		resp.Diagnostics.AddError("Failed to write tools config", err.Error())
		return
	}
	plan.ID = types.StringValue("tools")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ToolsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ToolsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	section, _, err := client.GetNestedSection(ctx, r.client, "tools")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read tools config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(ctx, section, &state)
	state.ID = types.StringValue("tools")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ToolsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ToolsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(ctx, plan), cfg.Hash, "tools"); err != nil {
		resp.Diagnostics.AddError("Failed to write tools config", err.Error())
		return
	}
	plan.ID = types.StringValue("tools")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ToolsResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "tools"); err != nil {
		resp.Diagnostics.AddError("Failed to delete tools config", err.Error())
		return
	}
}

func (r *ToolsResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "tools")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import tools config", err.Error())
		return
	}
	var state ToolsModel
	if section != nil {
		r.mapToModel(ctx, section, &state)
	}
	state.ID = types.StringValue("tools")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ToolsResource) modelToMap(ctx context.Context, m ToolsModel) map[string]any {
	d := make(map[string]any)
	setIfString(d, "profile", m.Profile)
	setIfStringList(ctx, d, "allow", m.Allow)
	setIfStringList(ctx, d, "deny", m.Deny)
	if !m.ElevatedEnabled.IsNull() && !m.ElevatedEnabled.IsUnknown() {
		d["elevated"] = map[string]any{
			"enabled": m.ElevatedEnabled.ValueBool(),
		}
	}
	if !m.BrowserEnabled.IsNull() && !m.BrowserEnabled.IsUnknown() {
		d["browser"] = map[string]any{
			"enabled": m.BrowserEnabled.ValueBool(),
		}
	}
	return d
}

func (r *ToolsResource) mapToModel(ctx context.Context, s map[string]any, m *ToolsModel) {
	readString(s, "profile", &m.Profile)
	readStringList(ctx, s, "allow", &m.Allow)
	readStringList(ctx, s, "deny", &m.Deny)
	if elevated, ok := s["elevated"].(map[string]any); ok {
		readBool(elevated, "enabled", &m.ElevatedEnabled)
	}
	if browser, ok := s["browser"].(map[string]any); ok {
		readBool(browser, "enabled", &m.BrowserEnabled)
	}
}
