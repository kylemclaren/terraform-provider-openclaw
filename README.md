# terraform-provider-openclaw

Terraform provider for [OpenClaw](https://github.com/openclaw/openclaw) -- declarative configuration management for the OpenClaw AI gateway.

OpenClaw connects chat apps (WhatsApp, Telegram, Discord, Slack, Signal, iMessage, Google Chat) to AI coding agents. This provider lets you manage the full gateway configuration as infrastructure-as-code.

## Quick Start

```hcl
terraform {
  required_providers {
    openclaw = {
      source = "registry.terraform.io/kylemclaren/openclaw"
    }
  }
}

# Connect to a running gateway over WebSocket
provider "openclaw" {
  gateway_url = "ws://127.0.0.1:18789"
  token       = var.gateway_token
}

# Or manage the config file directly (no running gateway needed)
# provider "openclaw" {
#   config_path = "~/.openclaw/openclaw.json"
# }

resource "openclaw_gateway" "main" {
  port        = 18789
  bind        = "loopback"
  reload_mode = "hybrid"
}

resource "openclaw_agent_defaults" "main" {
  model_primary   = "anthropic/claude-sonnet-4-20250514"
  timeout_seconds = 600
  heartbeat_every = "30m"
}

resource "openclaw_channel_whatsapp" "main" {
  dm_policy  = "pairing"
  allow_from = ["+15555550123"]
}
```

```bash
terraform init
terraform plan
terraform apply
```

## Provider Modes

| Mode | When to use | Config |
|------|-------------|--------|
| **WebSocket** | Gateway is running, live config patching via RPC | `gateway_url = "ws://..."` |
| **File** | Pre-provisioning before first boot, CI/CD pipelines | `config_path = "~/.openclaw/openclaw.json"` |

The provider auto-detects mode: if `gateway_url` is set (or `OPENCLAW_GATEWAY_URL` env var), it connects over WebSocket. Otherwise it reads/writes the JSON config file directly.

## Resources

| Resource | Description |
|----------|-------------|
| [`openclaw_gateway`](docs/resources/gateway.mdx) | Gateway server settings (port, bind, auth, reload) |
| [`openclaw_agent_defaults`](docs/resources/agent_defaults.mdx) | Default agent config (model, workspace, heartbeat, sandbox) |
| [`openclaw_agent`](docs/resources/agent.mdx) | Individual agent entry |
| [`openclaw_binding`](docs/resources/binding.mdx) | Multi-agent routing rules |
| [`openclaw_session`](docs/resources/session.mdx) | Session lifecycle (scope, reset policy) |
| [`openclaw_messages`](docs/resources/messages.mdx) | Message handling (queue, debounce, ack reactions) |
| [`openclaw_channel_whatsapp`](docs/resources/channel_whatsapp.mdx) | WhatsApp channel |
| [`openclaw_channel_telegram`](docs/resources/channel_telegram.mdx) | Telegram channel |
| [`openclaw_channel_discord`](docs/resources/channel_discord.mdx) | Discord channel |
| [`openclaw_channel_slack`](docs/resources/channel_slack.mdx) | Slack channel |
| [`openclaw_channel_signal`](docs/resources/channel_signal.mdx) | Signal channel |
| [`openclaw_channel_imessage`](docs/resources/channel_imessage.mdx) | iMessage channel |
| [`openclaw_channel_googlechat`](docs/resources/channel_googlechat.mdx) | Google Chat channel |
| [`openclaw_plugin`](docs/resources/plugin.mdx) | Plugin entry |
| [`openclaw_skill`](docs/resources/skill.mdx) | Skill entry |
| [`openclaw_hook`](docs/resources/hook.mdx) | Webhook configuration |
| [`openclaw_cron`](docs/resources/cron.mdx) | Cron job settings |
| [`openclaw_tools`](docs/resources/tools.mdx) | Tool access control |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| [`openclaw_gateway`](docs/data-sources/gateway.mdx) | Gateway settings (read-only) |
| [`openclaw_agent_defaults`](docs/data-sources/agent_defaults.mdx) | Agent default settings (read-only) |
| [`openclaw_agents`](docs/data-sources/agents.mdx) | All configured agents (read-only) |
| [`openclaw_channels`](docs/data-sources/channels.mdx) | All configured channels (read-only) |
| [`openclaw_config`](docs/data-sources/config.mdx) | Full raw config + hash |
| [`openclaw_health`](docs/data-sources/health.mdx) | Gateway health status (WebSocket mode only) |

## Documentation

See the [`docs/`](docs/index.mdx) directory for comprehensive documentation including:

- [Provider configuration](docs/provider.mdx)
- [Resource reference](docs/resources/) for all 18 resources
- [Data source reference](docs/data-sources/) for all 6 data sources
- [Full-stack example](examples/full-stack/main.tf) exercising every resource type

## Requirements

- [Terraform](https://www.terraform.io/downloads) >= 1.0
- [Go](https://go.dev/dl/) >= 1.23 (to build from source)
- [OpenClaw](https://github.com/openclaw/openclaw) >= 2026.2

## Building from Source

```bash
git clone https://github.com/kylemclaren/terraform-provider-openclaw.git
cd terraform-provider-openclaw
go build -o terraform-provider-openclaw .
```

To use a local build, add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/kylemclaren/openclaw" = "/path/to/binary/directory"
  }
  direct {}
}
```

## Testing

```bash
# Unit tests (no gateway needed)
go test ./internal/client/ -v

# File-mode acceptance tests (no gateway needed)
TF_ACC=1 go test ./internal/provider/ -v -run TestAccFileMode

# WS-mode tests (requires a running gateway)
TF_ACC=1 OPENCLAW_GATEWAY_TOKEN="your-token" go test ./... -v
```

### Docker-based Testing

No local Go, Terraform, or OpenClaw install required -- just Docker.

```bash
# Run the full acceptance test suite against a Dockerized gateway
./docker/test.sh

# Or use Make
make docker-test
```

The test harness builds the OpenClaw gateway from the [official repo](https://github.com/openclaw/openclaw),
starts it in a container, builds the provider from source, and runs Go acceptance tests over WebSocket.

Additional commands:

```bash
./docker/test.sh apply    # Terraform apply a test-stack against the gateway
./docker/test.sh plan     # Terraform plan only
./docker/test.sh shell    # Interactive shell with provider + terraform
./docker/test.sh logs     # Tail gateway logs
./docker/test.sh down     # Tear down containers and volumes
```

See [`docker/`](docker/) for the full Docker Compose setup.

## License

MIT
