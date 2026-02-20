---
page_title: "openclaw_plugin Resource - openclaw"
subcategory: ""
description: |-
  Manages an OpenClaw plugin entry.
---

# openclaw_plugin

Manages a plugin entry under `plugins.entries.<id>`. Plugins extend gateway functionality with custom behavior.

Changing `plugin_id` forces resource replacement.

## Example Usage

```hcl
resource "openclaw_plugin" "web_search" {
  plugin_id = "web_search"
  enabled   = true
  config_json = jsonencode({
    engine     = "google"
    max_results = 10
  })
}
```

## Argument Reference

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `plugin_id` | String | **Yes** | Unique plugin identifier. Used as the key under `plugins.entries`. Changing this forces replacement. |
| `enabled` | Bool | No | Enable or disable this plugin. |
| `config_json` | String | No | Raw JSON string with plugin-specific configuration. |

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Same as `plugin_id`. |

## Import

```bash
terraform import openclaw_plugin.web_search web_search
```
