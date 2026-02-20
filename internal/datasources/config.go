package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

var _ datasource.DataSource = &ConfigDataSource{}

type ConfigDataSource struct {
	client client.Client
}

type ConfigDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Raw  types.String `tfsdk:"raw"`
	Hash types.String `tfsdk:"hash"`
}

func NewConfigDataSource() datasource.DataSource {
	return &ConfigDataSource{}
}

func (d *ConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

func (d *ConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the full current OpenClaw configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"raw": schema.StringAttribute{
				Description: "The raw JSON config string.",
				Computed:    true,
			},
			"hash": schema.StringAttribute{
				Description: "Opaque hash for optimistic concurrency.",
				Computed:    true,
			},
		},
	}
}

func (d *ConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ConfigDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	cfg, err := d.client.GetConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read OpenClaw config", err.Error())
		return
	}

	state := ConfigDataSourceModel{
		ID:   types.StringValue("config"),
		Raw:  types.StringValue(cfg.Raw),
		Hash: types.StringValue(cfg.Hash),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
