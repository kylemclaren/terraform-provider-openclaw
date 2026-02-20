---
page_title: "openclaw_cron Resource - openclaw"
subcategory: ""
description: |-
  Manages the OpenClaw cron configuration.
---

# openclaw_cron

Manages the cron job configuration including concurrency limits and session retention.

This is a singleton resource.

## Example Usage

```hcl
resource "openclaw_cron" "main" {
  enabled             = true
  max_concurrent_runs = 3
  session_retention   = "48h"
}
```

## Argument Reference

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `enabled` | Bool | No | -- | Enable or disable cron jobs. |
| `max_concurrent_runs` | Int64 | No | `2` | Maximum number of concurrent cron runs. |
| `session_retention` | String | No | `"24h"` | How long to retain cron session data (e.g. `24h`, `7d`). |

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Always `"cron"`. |

## Import

```bash
terraform import openclaw_cron.main cron
```
