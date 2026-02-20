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

var _ resource.Resource = &CronResource{}
var _ resource.ResourceWithImportState = &CronResource{}

type CronResource struct {
	client client.Client
}

type CronModel struct {
	ID                types.String `tfsdk:"id"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	MaxConcurrentRuns types.Int64  `tfsdk:"max_concurrent_runs"`
	SessionRetention  types.String `tfsdk:"session_retention"`
}

func NewCronResource() resource.Resource {
	return &CronResource{}
}

func (r *CronResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cron"
}

func (r *CronResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the OpenClaw cron configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable cron jobs.",
				Optional:    true,
			},
			"max_concurrent_runs": schema.Int64Attribute{
				Description: "Maximum number of concurrent cron runs. Default: 2.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"session_retention": schema.StringAttribute{
				Description: "How long to retain cron session data. Default: 24h.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("24h"),
			},
		},
	}
}

func (r *CronResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CronResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CronModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(plan), cfg.Hash, "cron"); err != nil {
		resp.Diagnostics.AddError("Failed to write cron config", err.Error())
		return
	}
	plan.ID = types.StringValue("cron")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CronResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CronModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	section, _, err := client.GetNestedSection(ctx, r.client, "cron")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read cron config", err.Error())
		return
	}
	if section == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.mapToModel(section, &state)
	state.ID = types.StringValue("cron")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CronResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan CronModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, r.modelToMap(plan), cfg.Hash, "cron"); err != nil {
		resp.Diagnostics.AddError("Failed to write cron config", err.Error())
		return
	}
	plan.ID = types.StringValue("cron")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CronResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	cfg, err := r.client.GetConfig(ctx)
	if err != nil {
		if isConnectionClosed(err) {
			resp.Diagnostics.AddWarning("Gateway connection lost during delete", "The gateway may have restarted. The delete was likely applied.")
			return
		}
		resp.Diagnostics.AddError("Failed to read config", err.Error())
		return
	}
	if err := client.PatchNestedSection(ctx, r.client, nil, cfg.Hash, "cron"); err != nil {
		if isConnectionClosed(err) {
			resp.Diagnostics.AddWarning("Gateway connection lost during delete", "The gateway may have restarted. The delete was likely applied.")
			return
		}
		resp.Diagnostics.AddError("Failed to delete cron config", err.Error())
		return
	}
}

func (r *CronResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	section, _, err := client.GetNestedSection(ctx, r.client, "cron")
	if err != nil {
		resp.Diagnostics.AddError("Failed to import cron config", err.Error())
		return
	}
	var state CronModel
	if section != nil {
		r.mapToModel(section, &state)
	}
	state.ID = types.StringValue("cron")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *CronResource) modelToMap(m CronModel) map[string]any {
	d := make(map[string]any)
	setIfBool(d, "enabled", m.Enabled)
	setIfInt64(d, "maxConcurrentRuns", m.MaxConcurrentRuns)
	setIfString(d, "sessionRetention", m.SessionRetention)
	return d
}

func (r *CronResource) mapToModel(s map[string]any, m *CronModel) {
	readBool(s, "enabled", &m.Enabled)
	readFloat64AsInt64(s, "maxConcurrentRuns", &m.MaxConcurrentRuns)
	readString(s, "sessionRetention", &m.SessionRetention)
}
