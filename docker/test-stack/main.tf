# Smoke-test Terraform config for Docker-based integration testing.
#
# Exercises the provider against a live OpenClaw gateway running in Docker.
# Used by: ./docker/test.sh apply

terraform {
  required_providers {
    openclaw = {
      source = "registry.terraform.io/kylemclaren/openclaw"
    }
  }
}

# Connect to the gateway running in the Docker network.
# OPENCLAW_GATEWAY_URL and OPENCLAW_GATEWAY_TOKEN are set via env vars.
provider "openclaw" {}

# ── Gateway ──────────────────────────────────────────────────────

resource "openclaw_gateway" "test" {
  port        = 18789
  bind        = "lan"
  reload_mode = "hybrid"
}

# ── Agent defaults ───────────────────────────────────────────────

resource "openclaw_agent_defaults" "test" {
  model_primary    = "anthropic/claude-sonnet-4-20250514"
  timeout_seconds  = 300
  heartbeat_every  = "15m"
  thinking_default = "low"
  sandbox_mode     = "off"
}

# ── Agent ────────────────────────────────────────────────────────

resource "openclaw_agent" "research" {
  name           = "research"
  model_primary  = "anthropic/claude-sonnet-4-20250514"
  workspace      = "~/.openclaw/workspace/research"
  max_concurrent = 1
}

# ── Session ──────────────────────────────────────────────────────

resource "openclaw_session" "test" {
  scope        = "sender"
  reset_policy = "manual"
}

# ── Messages ─────────────────────────────────────────────────────

resource "openclaw_messages" "test" {
  queue_mode   = "debounce"
  debounce_ms  = 2000
  ack_reaction = true
}

# ── Tools ────────────────────────────────────────────────────────

resource "openclaw_tools" "test" {
  browser_enabled = false
}

# ── Data sources ─────────────────────────────────────────────────

data "openclaw_config" "current" {
  depends_on = [
    openclaw_gateway.test,
    openclaw_agent_defaults.test,
  ]
}

data "openclaw_health" "gw" {}

data "openclaw_gateway" "current" {
  depends_on = [openclaw_gateway.test]
}

data "openclaw_agent_defaults" "current" {
  depends_on = [openclaw_agent_defaults.test]
}

data "openclaw_agents" "all" {
  depends_on = [openclaw_agent.research]
}

data "openclaw_channels" "all" {}

# ── Outputs ──────────────────────────────────────────────────────

output "config_hash" {
  value = data.openclaw_config.current.hash
}

output "gateway_healthy" {
  value = data.openclaw_health.gw.healthy
}

output "gateway_port" {
  value = data.openclaw_gateway.current.port
}

output "agent_count" {
  value = length(data.openclaw_agents.all.agents)
}
