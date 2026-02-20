// Package provider implements the Terraform provider for OpenClaw.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/client"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/datasources"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/resources"
	"github.com/kylemclaren/terraform-provider-openclaw/internal/shared"
)

// Ensure the provider satisfies the interface.
var _ provider.Provider = &OpenClawProvider{}

// OpenClawProvider is the top-level Terraform provider for OpenClaw.
type OpenClawProvider struct {
	version string
}

// OpenClawProviderModel describes the provider HCL configuration.
type OpenClawProviderModel struct {
	GatewayURL types.String `tfsdk:"gateway_url"`
	Token      types.String `tfsdk:"token"`
	ConfigPath types.String `tfsdk:"config_path"`
}

// New returns a provider.Provider constructor for the given version string.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OpenClawProvider{
			version: version,
		}
	}
}

func (p *OpenClawProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openclaw"
	resp.Version = p.version
}

func (p *OpenClawProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for OpenClaw -- declarative configuration of the OpenClaw AI gateway.",
		Attributes: map[string]schema.Attribute{
			"gateway_url": schema.StringAttribute{
				Description: "WebSocket URL of the OpenClaw Gateway (e.g. ws://127.0.0.1:18789). " +
					"Takes precedence over config_path. Can also be set via OPENCLAW_GATEWAY_URL.",
				Optional: true,
			},
			"token": schema.StringAttribute{
				Description: "Authentication token for the Gateway WebSocket API. " +
					"Can also be set via OPENCLAW_GATEWAY_TOKEN.",
				Optional:  true,
				Sensitive: true,
			},
			"config_path": schema.StringAttribute{
				Description: "Path to the openclaw.json config file for local/file-based management. " +
					"Used when no gateway_url is set. Defaults to ~/.openclaw/openclaw.json. " +
					"Can also be set via OPENCLAW_CONFIG_PATH.",
				Optional: true,
			},
		},
	}
}

func (p *OpenClawProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config OpenClawProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve values: HCL > env > defaults.
	gatewayURL := stringValueOrEnv(config.GatewayURL, "OPENCLAW_GATEWAY_URL", "")
	token := stringValueOrEnv(config.Token, "OPENCLAW_GATEWAY_TOKEN", "")
	configPath := stringValueOrEnv(config.ConfigPath, "OPENCLAW_CONFIG_PATH", "~/.openclaw/openclaw.json")

	var c client.Client
	var err error

	if gatewayURL != "" {
		c, err = client.NewWSClient(ctx, client.WSClientConfig{
			URL:   gatewayURL,
			Token: token,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to connect to OpenClaw Gateway",
				"Could not establish WebSocket connection to "+gatewayURL+": "+err.Error(),
			)
			return
		}
	} else {
		c, err = client.NewFileClient(configPath)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to initialize file client",
				"Could not set up file-based config at "+configPath+": "+err.Error(),
			)
			return
		}
	}

	pd := &shared.ProviderData{Client: c}
	resp.DataSourceData = pd
	resp.ResourceData = pd
}

func (p *OpenClawProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Core
		resources.NewGatewayResource,
		resources.NewAgentDefaultsResource,
		resources.NewAgentResource,
		resources.NewBindingResource,
		resources.NewSessionResource,
		resources.NewMessagesResource,

		// Channels
		resources.NewChannelWhatsAppResource,
		resources.NewChannelTelegramResource,
		resources.NewChannelDiscordResource,
		resources.NewChannelSlackResource,
		resources.NewChannelSignalResource,
		resources.NewChannelIMessageResource,
		resources.NewChannelGoogleChatResource,

		// Automation & tools
		resources.NewPluginResource,
		resources.NewSkillResource,
		resources.NewHookResource,
		resources.NewCronResource,
		resources.NewToolsResource,
	}
}

func (p *OpenClawProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewConfigDataSource,
		datasources.NewHealthDataSource,
		datasources.NewGatewayDataSource,
		datasources.NewAgentDefaultsDataSource,
		datasources.NewAgentsDataSource,
		datasources.NewChannelsDataSource,
	}
}

func stringValueOrEnv(val types.String, envKey, fallback string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}
