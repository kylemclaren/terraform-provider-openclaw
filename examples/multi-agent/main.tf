terraform {
  required_providers {
    openclaw = {
      source = "registry.terraform.io/kylemclaren/openclaw"
    }
  }
}

provider "openclaw" {
  gateway_url = var.gateway_url
  token       = var.gateway_token
}

variable "gateway_url" {
  type    = string
  default = "ws://127.0.0.1:18789"
}

variable "gateway_token" {
  type      = string
  sensitive = true
  default   = ""
}

# ── Gateway ──────────────────────────────────────────────────

resource "openclaw_gateway" "main" {
  port        = 18789
  bind        = "loopback"
  reload_mode = "hybrid"
  auth_mode   = "token"
  auth_token  = var.gateway_token
}

# ── Agent defaults (shared across all agents) ────────────────

resource "openclaw_agent_defaults" "shared" {
  model_primary    = "anthropic/claude-opus-4-6"
  timeout_seconds  = 600
  thinking_default = "low"

  sandbox_mode  = "non-main"
  sandbox_scope = "agent"
}

# ── WhatsApp (personal) ─────────────────────────────────────

resource "openclaw_channel_whatsapp" "personal" {
  dm_policy    = "pairing"
  allow_from   = ["+15555550123"]
  group_policy = "allowlist"
}

# ── Telegram (work) ─────────────────────────────────────────

variable "telegram_bot_token" {
  type      = string
  sensitive = true
}

resource "openclaw_channel_telegram" "work" {
  enabled       = true
  bot_token     = var.telegram_bot_token
  dm_policy     = "allowlist"
  allow_from    = ["tg:111222333"]
  stream_mode   = "partial"
  history_limit = 50
}

# ── Health check ─────────────────────────────────────────────

data "openclaw_health" "gw" {}

output "gateway_status" {
  value = data.openclaw_health.gw.status
}

output "gateway_version" {
  value = data.openclaw_health.gw.version
}
