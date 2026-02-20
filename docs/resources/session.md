---
page_title: "openclaw_session Resource - openclaw"
subcategory: ""
description: |-
  Manages OpenClaw session lifecycle configuration.
---

# openclaw_session

Manages the session lifecycle configuration including scope, reset policy, and custom triggers.

This is a singleton resource.

## Example Usage

```hcl
resource "openclaw_session" "main" {
  dm_scope           = "per-peer"
  reset_mode         = "idle"
  reset_idle_minutes = 60
}
```

### Daily reset with custom triggers

```hcl
resource "openclaw_session" "main" {
  dm_scope       = "per-channel-peer"
  reset_mode     = "daily"
  reset_at_hour  = 3
  reset_triggers = ["/reset", "/new"]
}
```

## Argument Reference

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `dm_scope` | String | No | DM session scope: `main`, `per-peer`, `per-channel-peer`, `per-account-channel-peer`. |
| `reset_mode` | String | No | Reset mode: `daily` or `idle`. |
| `reset_at_hour` | Int64 | No | Hour of day (0-23) to reset sessions (for `daily` mode). |
| `reset_idle_minutes` | Int64 | No | Minutes of inactivity before reset (for `idle` mode). |
| `reset_triggers` | List(String) | No | Custom trigger phrases that reset the session. |

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Always `"session"`. |

## Import

```bash
terraform import openclaw_session.main session
```
