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

var _ resource.Resource = &SkillResource{}
var _ resource.ResourceWithImportState = &SkillResource{}

type SkillResource struct {
	client client.Client
}

type SkillModel struct {
	ID        types.String `tfsdk:"id"`
	SkillName types.String `tfsdk:"skill_name"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	APIKey    types.String `tfsdk:"api_key"`
	EnvJSON   types.String `tfsdk:"env_json"`
}

func NewSkillResource() resource.Resource {
	return &SkillResource{}
}

func (r *SkillResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_skill"
}

func (r *SkillResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenClaw skill entry.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"skill_name": schema.StringAttribute{
				Description: "Unique skill name. Used as the key under skills.entries.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable this skill.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "API key for the skill. Sensitive.",
				Optional:    true,
				Sensitive:   true,
			},
			"env_json": schema.StringAttribute{
				Description: "JSON object of environment variables to inject into the skill.",
				Optional:    true,
			},
		},
	}
}

func (r *SkillResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SkillResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SkillModel
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
		resp.Diagnostics.AddError("Invalid env_json", diags.Error())
		return
	}
	skillName := plan.SkillName.ValueString()
	if err := client.PatchNestedSection(ctx, r.client, m, cfg.Hash, "skills", "entries", skillName); err != nil {
		resp.Diagnostics.AddError("Failed to write skill config", err.Error())
		return
	}
	plan.ID = types.StringValue(skillName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SkillResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SkillModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	skillName := state.SkillName.ValueString()
	section, _, err := client.GetNestedSection(ctx, r.client, "skills", "entries", skillName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read skill config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(section, &state)
	state.ID = types.StringValue(skillName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SkillResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SkillModel
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
		resp.Diagnostics.AddError("Invalid env_json", diags.Error())
		return
	}
	skillName := plan.SkillName.ValueString()
	if err := client.PatchNestedSection(ctx, r.client, m, cfg.Hash, "skills", "entries", skillName); err != nil {
		resp.Diagnostics.AddError("Failed to write skill config", err.Error())
		return
	}
	plan.ID = types.StringValue(skillName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SkillResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SkillModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	skillName := state.SkillName.ValueString()
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "skills", "entries", skillName); err != nil {
		resp.Diagnostics.AddError("Failed to delete skill config", err.Error())
		return
	}
}

func (r *SkillResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	skillName := req.ID
	section, _, err := client.GetNestedSection(ctx, r.client, "skills", "entries", skillName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to import skill config", err.Error())
		return
	}
	var state SkillModel
	state.SkillName = types.StringValue(skillName)
	if section != nil {
		r.mapToModel(section, &state)
	}
	state.ID = types.StringValue(skillName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SkillResource) modelToMap(m SkillModel) (map[string]any, error) {
	d := make(map[string]any)
	setIfBool(d, "enabled", m.Enabled)
	setIfString(d, "apiKey", m.APIKey)
	if !m.EnvJSON.IsNull() && !m.EnvJSON.IsUnknown() {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(m.EnvJSON.ValueString()), &parsed); err != nil {
			return nil, fmt.Errorf("env_json must be a valid JSON object: %w", err)
		}
		d["env"] = parsed
	}
	return d, nil
}

func (r *SkillResource) mapToModel(s map[string]any, m *SkillModel) {
	readBool(s, "enabled", &m.Enabled)
	readString(s, "apiKey", &m.APIKey)
	if v, ok := s["env"].(map[string]any); ok && len(v) > 0 {
		b, _ := json.Marshal(v)
		m.EnvJSON = types.StringValue(string(b))
	}
}
