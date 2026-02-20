---
page_title: "openclaw_channel_discord Resource - openclaw"
subcategory: ""
description: |-
  Manages the OpenClaw Discord channel.
---

# openclaw_channel_discord

Manages the Discord channel configuration including bot token, DM policy, message chunking, and action permissions (reactions, threads, pins, search).

## Example Usage

```hcl
resource "openclaw_channel_discord" "main" {
  enabled          = true
  token            = var.discord_bot_token
  dm_policy        = "allowlist"
  allow_from       = ["user123", "user456"]
  text_chunk_limit = 2000
  history_limit    = 30
  reply_to_mode    = "first"

  actions_reactions = true
  actions_messages  = true
  actions_threads   = true
  actions_pins      = false
  actions_search    = true
}
```

## Argument Reference

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `enabled` | Bool | No | -- | Enable or disable the Discord channel. |
| `token` | String | No | -- | Discord bot token. **Sensitive.** Falls back to `DISCORD_BOT_TOKEN`. |
| `dm_policy` | String | No | `"pairing"` | DM policy: `pairing`, `allowlist`, `open`, `disabled`. |
| `allow_from` | List(String) | No | -- | Allowed Discord user IDs or usernames. |
| `allow_bots` | Bool | No | `false` | Allow messages from other bots. |
| `media_max_mb` | Int64 | No | `8` | Max inbound media size in MB. |
| `text_chunk_limit` | Int64 | No | `2000` | Max characters per outbound message chunk. |
| `chunk_mode` | String | No | `"length"` | Chunk splitting: `length` or `newline`. |
| `history_limit` | Int64 | No | `20` | Max chat history messages to fetch. |
| `reply_to_mode` | String | No | `"off"` | Reply-to behavior: `off`, `first`, `all`. |
| `actions_reactions` | Bool | No | -- | Enable reaction actions. |
| `actions_messages` | Bool | No | -- | Enable message actions (read/send/edit/delete). |
| `actions_threads` | Bool | No | -- | Enable thread actions. |
| `actions_pins` | Bool | No | -- | Enable pin actions. |
| `actions_search` | Bool | No | -- | Enable search actions. |

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Always `"channel_discord"`. |

## Import

```bash
terraform import openclaw_channel_discord.main channel_discord
```
