---
page_title: "Provider: openclaw"
subcategory: ""
description: |-
  The OpenClaw Terraform provider enables declarative, version-controlled management of your OpenClaw AI gateway configuration.
---

# OpenClaw Terraform Provider

The OpenClaw Terraform provider enables declarative, version-controlled management of your [OpenClaw](https://github.com/openclaw/openclaw) AI gateway configuration. Instead of editing `openclaw.json` by hand, define your gateway, channels, agents, and routing rules as Terraform resources.

## Why Terraform for OpenClaw?

- **Version control** -- track every config change in git
- **Review workflows** -- PR-based approval for gateway changes
- **Reproducibility** -- spin up identical configurations across environments
- **Composition** -- combine OpenClaw config with cloud infrastructure in a single plan
- **Drift detection** -- `terraform plan` shows exactly what changed outside Terraform

## Architecture

The provider has two transport backends:

```
                          +------------------+
                          |   Terraform CLI  |
                          +--------+---------+
                                   |
                          +--------v---------+
                          | openclaw provider|
                          +---+-----------+--+
                              |           |
                    +---------v--+   +----v--------+
                    |  WSClient  |   | FileClient  |
                    | (live RPC) |   | (JSON file) |
                    +-----+------+   +------+------+
                          |                 |
                +---------v------+   +------v-----------+
                | OpenClaw       |   | ~/.openclaw/     |
                | Gateway :18789 |   |  openclaw.json   |
                +----------------+   +------------------+
```

**WebSocket mode** connects to a running gateway and patches config via the `config.patch` RPC. Changes take effect immediately (depending on `reload_mode`).

**File mode** reads and writes the JSON config file directly. Useful for pre-provisioning a config before the gateway starts, or in CI/CD pipelines.

## Example Usage

### WebSocket Mode (Recommended)

Connect to a running OpenClaw gateway for live configuration:

```hcl
provider "openclaw" {
  gateway_url = "ws://127.0.0.1:18789"
  token       = var.gateway_token
}
```

### File Mode

Manage the config file directly without a running gateway:

```hcl
provider "openclaw" {
  config_path = "~/.openclaw/openclaw.json"
}
```

### Environment Variables Only

All provider attributes can be set via environment variables, allowing a zero-config provider block:

```hcl
provider "openclaw" {}
```

```bash
export OPENCLAW_GATEWAY_URL="ws://127.0.0.1:18789"
export OPENCLAW_GATEWAY_TOKEN="your-secret-token"
terraform apply
```

## Argument Reference

| Argument | Type | Description | Env Var | Default |
|----------|------|-------------|---------|---------|
| `gateway_url` | String | WebSocket URL of the OpenClaw gateway. When set, the provider uses WebSocket mode. | `OPENCLAW_GATEWAY_URL` | -- |
| `token` | String, Sensitive | Authentication token for the gateway WebSocket API. | `OPENCLAW_GATEWAY_TOKEN` | -- |
| `config_path` | String | Path to the `openclaw.json` config file. Used when `gateway_url` is not set. | `OPENCLAW_CONFIG_PATH` | `~/.openclaw/openclaw.json` |

## Mode Selection

The provider automatically selects its transport mode:

1. If `gateway_url` is set (or `OPENCLAW_GATEWAY_URL`), **WebSocket mode** is used. The provider connects to the gateway's WS RPC API and applies changes via `config.patch`.
2. Otherwise, **File mode** is used. The provider reads and writes the JSON config file at `config_path`.

### WebSocket Mode

- Requires a running OpenClaw gateway
- Changes are applied via the `config.patch` RPC
- Config reloads happen according to the gateway's `reload_mode` setting
- The `openclaw_health` data source is only available in this mode
- Supports authentication via `token`

### File Mode

- No running gateway required
- Reads and writes `openclaw.json` directly
- Uses a mutex to safely handle parallel resource operations
- The `openclaw_health` data source will return an error in this mode
- Useful for pre-provisioning configs before deploying the gateway

## Authentication

When the gateway has `gateway.auth.mode` set to `"token"`, you must provide the matching token:

```hcl
variable "gateway_token" {
  type      = string
  sensitive = true
}

provider "openclaw" {
  gateway_url = "ws://127.0.0.1:18789"
  token       = var.gateway_token
}
```

Or via environment variable:

```bash
export OPENCLAW_GATEWAY_TOKEN="your-secret-token"
```

If the gateway has no auth configured (`auth.mode = "none"`), the `token` argument can be omitted.

## Getting Started

### 1. Install OpenClaw

```bash
npm install -g openclaw
```

### 2. Start the gateway

```bash
openclaw gateway --port 18789
```

### 3. Write your Terraform config

```hcl
terraform {
  required_providers {
    openclaw = {
      source = "registry.terraform.io/kylemclaren/openclaw"
    }
  }
}

provider "openclaw" {
  gateway_url = "ws://127.0.0.1:18789"
  token       = var.gateway_token
}

resource "openclaw_gateway" "main" {
  port        = 18789
  bind        = "loopback"
  reload_mode = "hybrid"
}

resource "openclaw_agent_defaults" "main" {
  model_primary   = "anthropic/claude-sonnet-4-20250514"
  workspace       = "~/.openclaw/workspace"
  timeout_seconds = 600
  heartbeat_every = "30m"
}

resource "openclaw_channel_whatsapp" "main" {
  dm_policy  = "pairing"
  allow_from = ["+15555550123"]
}
```

### 4. Apply

```bash
terraform init
terraform plan
terraform apply
```

### 5. Verify

```bash
cat ~/.openclaw/openclaw.json
```

## Import

All resources support `terraform import`. Singleton resources use a fixed ID:

```bash
terraform import openclaw_gateway.main gateway
terraform import openclaw_session.main session
```

Array-based resources use their identifier:

```bash
terraform import openclaw_agent.research research
terraform import openclaw_binding.discord_research "research/discord"
terraform import openclaw_plugin.web_search web_search
terraform import openclaw_skill.calculator calculator
```

## Examples

See the [`examples/`](https://github.com/kylemclaren/terraform-provider-openclaw/tree/main/examples) directory:

- **[basic](https://github.com/kylemclaren/terraform-provider-openclaw/tree/main/examples/basic)** -- Single gateway with WhatsApp and Telegram
- **[multi-agent](https://github.com/kylemclaren/terraform-provider-openclaw/tree/main/examples/multi-agent)** -- Multiple agents with channel-based routing
- **[full-stack](https://github.com/kylemclaren/terraform-provider-openclaw/tree/main/examples/full-stack)** -- Every resource type exercised
