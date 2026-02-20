terraform {
  required_providers {
    openclaw = {
      source = "registry.terraform.io/kylemclaren/openclaw"
    }
  }
}

variable "gateway_token" {
  type      = string
  sensitive = true
  default   = ""
}

variable "telegram_bot_token" {
  type      = string
  sensitive = true
}

variable "discord_bot_token" {
  type      = string
  sensitive = true
}

variable "slack_bot_token" {
  type      = string
  sensitive = true
}

variable "slack_app_token" {
  type      = string
  sensitive = true
}

variable "gemini_api_key" {
  type      = string
  sensitive = true
}

provider "openclaw" {
  # gateway_url = "ws://127.0.0.1:18789"
  # token       = var.gateway_token
}

# ── Gateway ──────────────────────────────────────────────────

resource "openclaw_gateway" "main" {
  port           = 18789
  bind           = "loopback"
  reload_mode    = "hybrid"
  tailscale_mode = "serve"
}

# ── Agent defaults ───────────────────────────────────────────

resource "openclaw_agent_defaults" "shared" {
  workspace        = "~/.openclaw/workspace"
  model_primary    = "anthropic/claude-opus-4-6"
  model_fallbacks  = ["openai/gpt-5.2"]
  thinking_default = "low"
  timeout_seconds  = 600
  max_concurrent   = 2

  heartbeat_every  = "30m"
  heartbeat_target = "last"

  sandbox_mode  = "non-main"
  sandbox_scope = "agent"
}

# ── Agents ───────────────────────────────────────────────────

resource "openclaw_agent" "home" {
  agent_id      = "home"
  default_agent = true
  name          = "Molty"
  workspace     = "~/.openclaw/workspace-home"
  model         = "anthropic/claude-opus-4-6"

  identity_name  = "Molty"
  identity_emoji = "🦞"
  identity_theme = "helpful space lobster"

  mention_patterns = ["@openclaw", "molty"]
}

resource "openclaw_agent" "work" {
  agent_id  = "work"
  name      = "Work Agent"
  workspace = "~/.openclaw/workspace-work"
  model     = "anthropic/claude-sonnet-4-5"

  sandbox_mode  = "all"
  sandbox_scope = "session"

  tools_profile = "coding"
  tools_deny    = ["canvas"]
}

# ── Bindings (multi-agent routing) ───────────────────────────

resource "openclaw_binding" "home_wa" {
  agent_id         = openclaw_agent.home.agent_id
  match_channel    = "whatsapp"
  match_account_id = "personal"
}

resource "openclaw_binding" "work_tg" {
  agent_id      = openclaw_agent.work.agent_id
  match_channel = "telegram"
}

# ── Channels ─────────────────────────────────────────────────

resource "openclaw_channel_whatsapp" "main" {
  dm_policy          = "pairing"
  allow_from         = ["+15555550123"]
  send_read_receipts = true
  group_policy       = "allowlist"
}

resource "openclaw_channel_telegram" "main" {
  enabled       = true
  bot_token     = var.telegram_bot_token
  dm_policy     = "pairing"
  allow_from    = ["tg:123456789"]
  stream_mode   = "partial"
  reply_to_mode = "first"
  history_limit = 50
}

resource "openclaw_channel_discord" "main" {
  enabled        = true
  token          = var.discord_bot_token
  dm_policy      = "pairing"
  allow_from     = ["steipete", "1234567890123"]
  history_limit  = 20
  reply_to_mode  = "off"

  actions_reactions = true
  actions_messages  = true
  actions_threads   = true
  actions_pins      = true
  actions_search    = true
}

resource "openclaw_channel_slack" "main" {
  enabled        = true
  bot_token      = var.slack_bot_token
  app_token      = var.slack_app_token
  dm_policy      = "pairing"
  allow_from     = ["U123", "U456"]
  history_limit  = 50
  reply_to_mode  = "off"
  reaction_notifications = "own"
}

resource "openclaw_channel_signal" "main" {
  enabled                = true
  dm_policy              = "pairing"
  reaction_notifications = "own"
  history_limit          = 50
}

# ── Session ──────────────────────────────────────────────────

resource "openclaw_session" "config" {
  dm_scope           = "per-channel-peer"
  reset_mode         = "daily"
  reset_at_hour      = 4
  reset_idle_minutes = 120
  reset_triggers     = ["/new", "/reset"]
}

# ── Messages ─────────────────────────────────────────────────

resource "openclaw_messages" "config" {
  response_prefix    = "🦞"
  ack_reaction       = "👀"
  ack_reaction_scope = "group-mentions"
  queue_mode         = "collect"
  queue_debounce_ms  = 1000
  queue_cap          = 20
  inbound_debounce_ms = 2000
}

# ── Plugins ──────────────────────────────────────────────────

resource "openclaw_plugin" "voice_call" {
  plugin_id = "voice-call"
  enabled   = true
  config_json = jsonencode({
    provider = "twilio"
  })
}

# ── Skills ───────────────────────────────────────────────────

resource "openclaw_skill" "nano_banana" {
  skill_name = "nano-banana-pro"
  enabled    = true
  api_key    = var.gemini_api_key
}

# ── Hooks ────────────────────────────────────────────────────

resource "openclaw_hook" "ingress" {
  enabled             = true
  token               = "shared-webhook-secret"
  path                = "/hooks"
  default_session_key = "hook:ingress"
}

# ── Cron ─────────────────────────────────────────────────────

resource "openclaw_cron" "config" {
  enabled             = true
  max_concurrent_runs = 2
  session_retention   = "24h"
}

# ── Tools ────────────────────────────────────────────────────

resource "openclaw_tools" "config" {
  profile          = "coding"
  deny             = ["canvas"]
  elevated_enabled = true
  browser_enabled  = true
}

# ── Data Sources ─────────────────────────────────────────────

data "openclaw_config" "current" {}
data "openclaw_health" "gw" {}

output "config_hash" {
  value = data.openclaw_config.current.hash
}

output "gateway_version" {
  value = data.openclaw_health.gw.version
}
