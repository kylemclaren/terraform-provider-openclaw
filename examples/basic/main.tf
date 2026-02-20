terraform {
  required_providers {
    openclaw = {
      source = "registry.terraform.io/kylemclaren/openclaw"
    }
  }
}

# Connect to a running gateway (preferred)
# Or set OPENCLAW_GATEWAY_URL / OPENCLAW_GATEWAY_TOKEN env vars.
# Falls back to file mode at ~/.openclaw/openclaw.json when no URL is set.
provider "openclaw" {
  # gateway_url = "ws://127.0.0.1:18789"
  # token       = var.gateway_token
}

# ── Gateway ──────────────────────────────────────────────────

resource "openclaw_gateway" "main" {
  port        = 18789
  bind        = "loopback"
  reload_mode = "hybrid"
}

# ── Agent defaults ───────────────────────────────────────────

resource "openclaw_agent_defaults" "main" {
  workspace       = "~/.openclaw/workspace"
  model_primary   = "anthropic/claude-opus-4-6"
  model_fallbacks = ["openai/gpt-5.2"]

  thinking_default = "low"
  timeout_seconds  = 600
  max_concurrent   = 1

  heartbeat_every  = "30m"
  heartbeat_target = "last"

  sandbox_mode  = "non-main"
  sandbox_scope = "agent"
}

# ── WhatsApp ─────────────────────────────────────────────────

resource "openclaw_channel_whatsapp" "main" {
  dm_policy          = "pairing"
  allow_from         = ["+15555550123"]
  text_chunk_limit   = 4000
  send_read_receipts = true
  group_policy       = "allowlist"
}

# ── Telegram ─────────────────────────────────────────────────

variable "telegram_bot_token" {
  type      = string
  sensitive = true
}

resource "openclaw_channel_telegram" "main" {
  enabled       = true
  bot_token     = var.telegram_bot_token
  dm_policy     = "pairing"
  allow_from    = ["tg:123456789"]
  stream_mode   = "partial"
  reply_to_mode = "first"
}

# ── Data sources ─────────────────────────────────────────────

data "openclaw_config" "current" {}

output "config_hash" {
  value = data.openclaw_config.current.hash
}
