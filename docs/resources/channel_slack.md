---
page_title: "openclaw_channel_slack Resource - openclaw"
subcategory: ""
description: |-
  Manages the OpenClaw Slack channel.
---

# openclaw_channel_slack

Manages the Slack channel configuration. Requires both a bot token (`xoxb-...`) and an app token (`xapp-...`) for Socket Mode.

## Example Usage

```hcl
resource "openclaw_channel_slack" "main" {
  enabled    = true
  bot_token  = var.slack_bot_token
  app_token  = var.slack_app_token
  dm_policy  = "allowlist"
  allow_from = ["U0123456789"]

  history_limit          = 50
  text_chunk_limit       = 4000
  reaction_notifications = "own"
}
```

## Argument Reference

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `enabled` | Bool | No | -- | Enable or disable the Slack channel. |
| `bot_token` | String | No | -- | Slack bot token (`xoxb-...`). **Sensitive.** Falls back to `SLACK_BOT_TOKEN`. |
| `app_token` | String | No | -- | Slack app token (`xapp-...`). **Sensitive.** Falls back to `SLACK_APP_TOKEN`. |
| `dm_policy` | String | No | `"pairing"` | DM policy: `pairing`, `allowlist`, `open`, `disabled`. |
| `allow_from` | List(String) | No | -- | Allowed Slack user IDs. |
| `allow_bots` | Bool | No | `false` | Allow messages from other bots. |
| `history_limit` | Int64 | No | `50` | Max chat history messages. |
| `text_chunk_limit` | Int64 | No | `4000` | Max characters per chunk. |
| `chunk_mode` | String | No | `"length"` | Chunk mode: `length` or `newline`. |
| `media_max_mb` | Int64 | No | `20` | Max inbound media size in MB. |
| `reply_to_mode` | String | No | `"off"` | Reply-to behavior: `off`, `first`, `all`. |
| `reaction_notifications` | String | No | `"own"` | Reaction notifications: `off`, `own`, `all`, `allowlist`. |

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Always `"channel_slack"`. |

## Import

```bash
terraform import openclaw_channel_slack.main channel_slack
```
