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

var _ datasource.DataSource = &ChannelsDataSource{}

type ChannelsDataSource struct {
	client client.Client
}

type ChannelsDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Names    types.List   `tfsdk:"names"`
	Channels types.List   `tfsdk:"channels"`
}

var channelObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":      types.StringType,
		"enabled":   types.BoolType,
		"dm_policy": types.StringType,
	},
}

func NewChannelsDataSource() datasource.DataSource {
	return &ChannelsDataSource{}
}

func (d *ChannelsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channels"
}

func (d *ChannelsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all configured OpenClaw channels.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"names": schema.ListAttribute{
				Description: "List of configured channel names (e.g. whatsapp, telegram, discord).",
				Computed:    true,
				ElementType: types.StringType,
			},
			"channels": schema.ListNestedAttribute{
				Description: "List of channels with summary configuration.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Channel name (e.g. whatsapp, telegram, discord, slack, signal, imessage, googlechat).",
							Computed:    true,
						},
						"enabled": schema.BoolAttribute{
							Description: "Whether the channel is enabled. Channels without an explicit enabled field are considered enabled if configured.",
							Computed:    true,
						},
						"dm_policy": schema.StringAttribute{
							Description: "DM policy for this channel.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *ChannelsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ChannelsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	section, _, err := client.GetSection(ctx, d.client, "channels")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read channels config", err.Error())
		return
	}

	state := ChannelsDataSourceModel{
		ID: types.StringValue("channels"),
	}

	var names []string
	var channelObjects []attr.Value

	if section != nil {
		for name, val := range section {
			chMap, ok := val.(map[string]any)
			if !ok {
				continue
			}

			names = append(names, name)

			// Determine enabled state: explicit "enabled" field, or true if configured
			enabled := true
			if v, ok := chMap["enabled"].(bool); ok {
				enabled = v
			}

			// DM policy: look for dmPolicy or dm.policy
			dmPolicy := ""
			if v, ok := chMap["dmPolicy"].(string); ok {
				dmPolicy = v
			} else if dm, ok := chMap["dm"].(map[string]any); ok {
				if v, ok := dm["policy"].(string); ok {
					dmPolicy = v
				}
			}

			obj, diags := types.ObjectValue(channelObjectType.AttrTypes, map[string]attr.Value{
				"name":      types.StringValue(name),
				"enabled":   types.BoolValue(enabled),
				"dm_policy": stringOrNull(dmPolicy),
			})
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			channelObjects = append(channelObjects, obj)
		}
	}

	// Set names
	if len(names) > 0 {
		nameList, diags := types.ListValueFrom(ctx, types.StringType, names)
		resp.Diagnostics.Append(diags...)
		state.Names = nameList
	} else {
		state.Names = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Set channels
	if len(channelObjects) > 0 {
		chList, diags := types.ListValue(channelObjectType, channelObjects)
		resp.Diagnostics.Append(diags...)
		state.Channels = chList
	} else {
		state.Channels = types.ListValueMust(channelObjectType, []attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
