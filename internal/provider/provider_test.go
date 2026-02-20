package provider_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/kylemclaren/terraform-provider-openclaw/internal/provider"
)

// testAccProtoV6ProviderFactories creates provider factories for acceptance tests.
func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"openclaw": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// testConfigDir creates a temp directory with an empty config and returns the
// provider HCL block pointing at it. This isolates each test from the real config.
func testConfigDir(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "openclaw.json")
	os.WriteFile(cfgPath, []byte("{}"), 0o644)
	providerBlock := `
provider "openclaw" {
  config_path = "` + cfgPath + `"
}
`
	return cfgPath, providerBlock
}

// testWSProviderBlock returns a provider block pointing at the live gateway.
func testWSProviderBlock() string {
	url := os.Getenv("OPENCLAW_GATEWAY_URL")
	if url == "" {
		url = "ws://127.0.0.1:18789"
	}
	token := os.Getenv("OPENCLAW_GATEWAY_TOKEN")
	block := `
provider "openclaw" {
  gateway_url = "` + url + `"
`
	if token != "" {
		block += `  token = "` + token + `"
`
	}
	block += `}
`
	return block
}

// ── File-mode acceptance tests ──────────────────────────────
// These run without a live gateway, testing against a temp file.

func TestAccFileMode_GatewayResource(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_gateway" "test" {
  port        = 19000
  bind        = "loopback"
  reload_mode = "hot"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_gateway.test", "port", "19000"),
					resource.TestCheckResourceAttr("openclaw_gateway.test", "bind", "loopback"),
					resource.TestCheckResourceAttr("openclaw_gateway.test", "reload_mode", "hot"),
				),
			},
			// Update
			{
				Config: providerBlock + `
resource "openclaw_gateway" "test" {
  port        = 19001
  bind        = "all"
  reload_mode = "restart"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_gateway.test", "port", "19001"),
					resource.TestCheckResourceAttr("openclaw_gateway.test", "bind", "all"),
					resource.TestCheckResourceAttr("openclaw_gateway.test", "reload_mode", "restart"),
				),
			},
		},
	})
}

func TestAccFileMode_AgentDefaultsResource(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_agent_defaults" "test" {
  workspace        = "~/.openclaw/workspace-test"
  model_primary    = "anthropic/claude-opus-4-6"
  thinking_default = "low"
  timeout_seconds  = 300
  max_concurrent   = 2

  heartbeat_every  = "15m"
  heartbeat_target = "none"

  sandbox_mode  = "non-main"
  sandbox_scope = "agent"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_agent_defaults.test", "workspace", "~/.openclaw/workspace-test"),
					resource.TestCheckResourceAttr("openclaw_agent_defaults.test", "model_primary", "anthropic/claude-opus-4-6"),
					resource.TestCheckResourceAttr("openclaw_agent_defaults.test", "thinking_default", "low"),
					resource.TestCheckResourceAttr("openclaw_agent_defaults.test", "timeout_seconds", "300"),
					resource.TestCheckResourceAttr("openclaw_agent_defaults.test", "max_concurrent", "2"),
					resource.TestCheckResourceAttr("openclaw_agent_defaults.test", "heartbeat_every", "15m"),
					resource.TestCheckResourceAttr("openclaw_agent_defaults.test", "sandbox_mode", "non-main"),
				),
			},
		},
	})
}

func TestAccFileMode_ChannelWhatsApp(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_channel_whatsapp" "test" {
  dm_policy          = "allowlist"
  allow_from         = ["+15555550123", "+447700900123"]
  text_chunk_limit   = 3000
  send_read_receipts = false
  group_policy       = "open"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_channel_whatsapp.test", "dm_policy", "allowlist"),
					resource.TestCheckResourceAttr("openclaw_channel_whatsapp.test", "allow_from.#", "2"),
					resource.TestCheckResourceAttr("openclaw_channel_whatsapp.test", "allow_from.0", "+15555550123"),
					resource.TestCheckResourceAttr("openclaw_channel_whatsapp.test", "text_chunk_limit", "3000"),
					resource.TestCheckResourceAttr("openclaw_channel_whatsapp.test", "send_read_receipts", "false"),
					resource.TestCheckResourceAttr("openclaw_channel_whatsapp.test", "group_policy", "open"),
				),
			},
		},
	})
}

func TestAccFileMode_ChannelTelegram(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_channel_telegram" "test" {
  enabled       = true
  bot_token     = "123456:ABCDEF"
  dm_policy     = "open"
  allow_from    = ["tg:999"]
  stream_mode   = "block"
  reply_to_mode = "all"
  history_limit = 25
  media_max_mb  = 10
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_channel_telegram.test", "dm_policy", "open"),
					resource.TestCheckResourceAttr("openclaw_channel_telegram.test", "stream_mode", "block"),
					resource.TestCheckResourceAttr("openclaw_channel_telegram.test", "reply_to_mode", "all"),
					resource.TestCheckResourceAttr("openclaw_channel_telegram.test", "history_limit", "25"),
				),
			},
		},
	})
}

func TestAccFileMode_ChannelDiscord(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_channel_discord" "test" {
  enabled           = true
  token             = "test-discord-token"
  dm_policy         = "allowlist"
  allow_from        = ["user1", "user2"]
  history_limit     = 30
  reply_to_mode     = "first"
  actions_reactions = true
  actions_messages  = true
  actions_search    = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_channel_discord.test", "dm_policy", "allowlist"),
					resource.TestCheckResourceAttr("openclaw_channel_discord.test", "allow_from.#", "2"),
					resource.TestCheckResourceAttr("openclaw_channel_discord.test", "history_limit", "30"),
					resource.TestCheckResourceAttr("openclaw_channel_discord.test", "reply_to_mode", "first"),
					resource.TestCheckResourceAttr("openclaw_channel_discord.test", "actions_reactions", "true"),
					resource.TestCheckResourceAttr("openclaw_channel_discord.test", "actions_search", "false"),
				),
			},
		},
	})
}

func TestAccFileMode_SessionResource(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_session" "test" {
  dm_scope           = "per-channel-peer"
  reset_mode         = "daily"
  reset_at_hour      = 4
  reset_idle_minutes = 60
  reset_triggers     = ["/new", "/reset"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_session.test", "dm_scope", "per-channel-peer"),
					resource.TestCheckResourceAttr("openclaw_session.test", "reset_mode", "daily"),
					resource.TestCheckResourceAttr("openclaw_session.test", "reset_at_hour", "4"),
					resource.TestCheckResourceAttr("openclaw_session.test", "reset_idle_minutes", "60"),
					resource.TestCheckResourceAttr("openclaw_session.test", "reset_triggers.#", "2"),
				),
			},
		},
	})
}

func TestAccFileMode_MessagesResource(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_messages" "test" {
  response_prefix     = "[Bot]"
  ack_reaction        = "👀"
  ack_reaction_scope  = "all"
  queue_mode          = "collect"
  queue_debounce_ms   = 500
  queue_cap           = 10
  inbound_debounce_ms = 3000
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_messages.test", "response_prefix", "[Bot]"),
					resource.TestCheckResourceAttr("openclaw_messages.test", "ack_reaction", "👀"),
					resource.TestCheckResourceAttr("openclaw_messages.test", "queue_mode", "collect"),
					resource.TestCheckResourceAttr("openclaw_messages.test", "queue_cap", "10"),
					resource.TestCheckResourceAttr("openclaw_messages.test", "inbound_debounce_ms", "3000"),
				),
			},
		},
	})
}

func TestAccFileMode_CronResource(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_cron" "test" {
  enabled             = true
  max_concurrent_runs = 3
  session_retention   = "48h"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_cron.test", "enabled", "true"),
					resource.TestCheckResourceAttr("openclaw_cron.test", "max_concurrent_runs", "3"),
					resource.TestCheckResourceAttr("openclaw_cron.test", "session_retention", "48h"),
				),
			},
		},
	})
}

func TestAccFileMode_HookResource(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_hook" "test" {
  enabled             = true
  token               = "test-hook-secret"
  path                = "/webhooks"
  default_session_key = "hook:test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_hook.test", "enabled", "true"),
					resource.TestCheckResourceAttr("openclaw_hook.test", "path", "/webhooks"),
					resource.TestCheckResourceAttr("openclaw_hook.test", "default_session_key", "hook:test"),
				),
			},
		},
	})
}

func TestAccFileMode_ToolsResource(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_tools" "test" {
  profile          = "coding"
  deny             = ["canvas", "browser"]
  elevated_enabled = true
  browser_enabled  = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_tools.test", "profile", "coding"),
					resource.TestCheckResourceAttr("openclaw_tools.test", "deny.#", "2"),
					resource.TestCheckResourceAttr("openclaw_tools.test", "elevated_enabled", "true"),
					resource.TestCheckResourceAttr("openclaw_tools.test", "browser_enabled", "false"),
				),
			},
		},
	})
}

func TestAccFileMode_ConfigDataSource(t *testing.T) {
	cfgPath, providerBlock := testConfigDir(t)

	// Seed the config with some data
	os.WriteFile(cfgPath, []byte(`{"gateway":{"port":18789}}`), 0o644)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
data "openclaw_config" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.openclaw_config.test", "raw"),
					resource.TestCheckResourceAttrSet("data.openclaw_config.test", "hash"),
				),
			},
		},
	})
}

func TestAccFileMode_MultiResourceComposition(t *testing.T) {
	_, providerBlock := testConfigDir(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
resource "openclaw_gateway" "test" {
  port        = 19000
  bind        = "loopback"
  reload_mode = "hybrid"
}

resource "openclaw_agent_defaults" "test" {
  workspace       = "~/.openclaw/workspace-multi"
  model_primary   = "anthropic/claude-sonnet-4-5"
  timeout_seconds = 300
}

resource "openclaw_channel_whatsapp" "test" {
  dm_policy  = "pairing"
  allow_from = ["+15555550123"]
}

resource "openclaw_session" "test" {
  dm_scope   = "per-peer"
  reset_mode = "idle"
  reset_idle_minutes = 30
}

resource "openclaw_cron" "test" {
  enabled = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_gateway.test", "port", "19000"),
					resource.TestCheckResourceAttr("openclaw_agent_defaults.test", "model_primary", "anthropic/claude-sonnet-4-5"),
					resource.TestCheckResourceAttr("openclaw_channel_whatsapp.test", "dm_policy", "pairing"),
					resource.TestCheckResourceAttr("openclaw_session.test", "dm_scope", "per-peer"),
					resource.TestCheckResourceAttr("openclaw_cron.test", "enabled", "true"),
				),
			},
		},
	})
}

func TestAccFileMode_GatewayDataSource(t *testing.T) {
	cfgPath, providerBlock := testConfigDir(t)

	// Pre-populate config with gateway section
	os.WriteFile(cfgPath,
		[]byte(`{"gateway":{"port":19999,"bind":"loopback","mode":"local","auth":{"mode":"token"},"reload":{"mode":"hybrid"}}}`), 0o644)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
data "openclaw_gateway" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openclaw_gateway.test", "port", "19999"),
					resource.TestCheckResourceAttr("data.openclaw_gateway.test", "bind", "loopback"),
					resource.TestCheckResourceAttr("data.openclaw_gateway.test", "auth_mode", "token"),
					resource.TestCheckResourceAttr("data.openclaw_gateway.test", "reload_mode", "hybrid"),
					resource.TestCheckResourceAttr("data.openclaw_gateway.test", "mode", "local"),
				),
			},
		},
	})
}

func TestAccFileMode_AgentDefaultsDataSource(t *testing.T) {
	cfgPath, providerBlock := testConfigDir(t)

	os.WriteFile(cfgPath,
		[]byte(`{"agents":{"defaults":{"workspace":"~/ws","timeout":300,"heartbeat":{"every":"15m","target":"last"}}}}`), 0o644)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
data "openclaw_agent_defaults" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openclaw_agent_defaults.test", "workspace", "~/ws"),
					resource.TestCheckResourceAttr("data.openclaw_agent_defaults.test", "timeout_seconds", "300"),
					resource.TestCheckResourceAttr("data.openclaw_agent_defaults.test", "heartbeat_every", "15m"),
					resource.TestCheckResourceAttr("data.openclaw_agent_defaults.test", "heartbeat_target", "last"),
				),
			},
		},
	})
}

func TestAccFileMode_AgentsDataSource(t *testing.T) {
	cfgPath, providerBlock := testConfigDir(t)

	os.WriteFile(cfgPath,
		[]byte(`{"agents":{"list":[{"id":"main","default":true,"name":"Main Agent","model":"anthropic/claude-sonnet-4-20250514"},{"id":"research","name":"Research","model":"openai/gpt-4.1"}]}}`), 0o644)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
data "openclaw_agents" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openclaw_agents.test", "default_agent_id", "main"),
					resource.TestCheckResourceAttr("data.openclaw_agents.test", "agent_ids.#", "2"),
					resource.TestCheckResourceAttr("data.openclaw_agents.test", "agents.#", "2"),
				),
			},
		},
	})
}

func TestAccFileMode_ChannelsDataSource(t *testing.T) {
	cfgPath, providerBlock := testConfigDir(t)

	os.WriteFile(cfgPath,
		[]byte(`{"channels":{"whatsapp":{"dmPolicy":"pairing"},"telegram":{"enabled":true,"dmPolicy":"allowlist"},"discord":{"enabled":false}}}`), 0o644)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock + `
data "openclaw_channels" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openclaw_channels.test", "names.#", "3"),
					resource.TestCheckResourceAttr("data.openclaw_channels.test", "channels.#", "3"),
				),
			},
		},
	})
}

// ── WS-mode acceptance tests ────────────────────────────────
// These run against a live OpenClaw gateway.

func TestAccWSMode_HealthDataSource(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Set TF_ACC=1 to run acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testWSProviderBlock() + `
data "openclaw_health" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openclaw_health.test", "ok", "true"),
					resource.TestCheckResourceAttrSet("data.openclaw_health.test", "timestamp"),
					resource.TestCheckResourceAttrSet("data.openclaw_health.test", "default_agent_id"),
				),
			},
		},
	})
}

func TestAccWSMode_ConfigDataSource(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Set TF_ACC=1 to run acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testWSProviderBlock() + `
data "openclaw_config" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.openclaw_config.test", "raw"),
					resource.TestCheckResourceAttrSet("data.openclaw_config.test", "hash"),
				),
			},
		},
	})
}

func TestAccWSMode_GatewayAndChannelPatch(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Set TF_ACC=1 to run acceptance tests")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testWSProviderBlock() + `
resource "openclaw_cron" "test" {
  enabled             = true
  max_concurrent_runs = 1
  session_retention   = "12h"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openclaw_cron.test", "enabled", "true"),
					resource.TestCheckResourceAttr("openclaw_cron.test", "max_concurrent_runs", "1"),
					resource.TestCheckResourceAttr("openclaw_cron.test", "session_retention", "12h"),
				),
			},
		},
	})
}
