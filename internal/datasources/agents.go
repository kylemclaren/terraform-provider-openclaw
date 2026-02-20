package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ datasource.DataSource = &AgentsDataSource{}

type AgentsDataSource struct {
	client client.Client
}

type AgentsDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	DefaultAgentID types.String `tfsdk:"default_agent_id"`
	AgentIDs       types.List   `tfsdk:"agent_ids"`
	Agents         types.List   `tfsdk:"agents"`
}

var agentObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"agent_id":      types.StringType,
		"name":          types.StringType,
		"is_default":    types.BoolType,
		"model":         types.StringType,
		"workspace":     types.StringType,
		"sandbox_mode":  types.StringType,
		"tools_profile": types.StringType,
	},
}

func NewAgentsDataSource() datasource.DataSource {
	return &AgentsDataSource{}
}

func (d *AgentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agents"
}

func (d *AgentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all configured OpenClaw agents.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"default_agent_id": schema.StringAttribute{
				Description: "The agent ID marked as default.",
				Computed:    true,
			},
			"agent_ids": schema.ListAttribute{
				Description: "List of all agent IDs.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"agents": schema.ListNestedAttribute{
				Description: "List of agents with their configuration.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"agent_id": schema.StringAttribute{
							Description: "Agent identifier.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Agent display name.",
							Computed:    true,
						},
						"is_default": schema.BoolAttribute{
							Description: "Whether this is the default agent.",
							Computed:    true,
						},
						"model": schema.StringAttribute{
							Description: "Model assigned to this agent.",
							Computed:    true,
						},
						"workspace": schema.StringAttribute{
							Description: "Workspace path for this agent.",
							Computed:    true,
						},
						"sandbox_mode": schema.StringAttribute{
							Description: "Sandbox mode for this agent.",
							Computed:    true,
						},
						"tools_profile": schema.StringAttribute{
							Description: "Tools profile for this agent.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *AgentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	pd, ok := req.ProviderData.(*shared.ProviderData)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *shared.ProviderData, got %T", req.ProviderData))
		return
	}
	d.client = pd.Client
}

func (d *AgentsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	section, _, err := client.GetSection(ctx, d.client, "agents")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read agents config", err.Error())
		return
	}

	state := AgentsDataSourceModel{
		ID: types.StringValue("agents"),
	}

	var agentIDs []string
	var agentObjects []attr.Value
	var defaultAgentID string

	if section != nil {
		// agents.list is an array of agent objects
		if list, ok := section["list"].([]any); ok {
			for _, item := range list {
				agent, ok := item.(map[string]any)
				if !ok {
					continue
				}

				agentID, _ := agent["id"].(string)
				if agentID == "" {
					continue
				}
				agentIDs = append(agentIDs, agentID)

				name, _ := agent["name"].(string)
				model, _ := agent["model"].(string)
				workspace, _ := agent["workspace"].(string)
				isDefault, _ := agent["default"].(bool)

				if isDefault {
					defaultAgentID = agentID
				}

				sandboxMode := ""
				if sb, ok := agent["sandbox"].(map[string]any); ok {
					sandboxMode, _ = sb["mode"].(string)
				}

				toolsProfile := ""
				if tools, ok := agent["tools"].(map[string]any); ok {
					toolsProfile, _ = tools["profile"].(string)
				}

				obj, diags := types.ObjectValue(agentObjectType.AttrTypes, map[string]attr.Value{
					"agent_id":      types.StringValue(agentID),
					"name":          stringOrNull(name),
					"is_default":    types.BoolValue(isDefault),
					"model":         stringOrNull(model),
					"workspace":     stringOrNull(workspace),
					"sandbox_mode":  stringOrNull(sandboxMode),
					"tools_profile": stringOrNull(toolsProfile),
				})
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				agentObjects = append(agentObjects, obj)
			}
		}
	}

	// Set agent_ids
	if len(agentIDs) > 0 {
		idList, diags := types.ListValueFrom(ctx, types.StringType, agentIDs)
		resp.Diagnostics.Append(diags...)
		state.AgentIDs = idList
	} else {
		state.AgentIDs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Set agents
	if len(agentObjects) > 0 {
		agentList, diags := types.ListValue(agentObjectType, agentObjects)
		resp.Diagnostics.Append(diags...)
		state.Agents = agentList
	} else {
		state.Agents = types.ListValueMust(agentObjectType, []attr.Value{})
	}

	// Set default agent ID
	if defaultAgentID != "" {
		state.DefaultAgentID = types.StringValue(defaultAgentID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
