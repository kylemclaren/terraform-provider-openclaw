---
page_title: "openclaw_gateway Data Source - openclaw"
subcategory: ""
description: |-
  Reads the current OpenClaw gateway server configuration.
---

# openclaw_gateway (Data Source)

Reads the current gateway server configuration without managing it. Useful for cross-stack references where one Terraform config manages the gateway and another reads from it.

## Example Usage

```hcl
data "openclaw_gateway" "current" {}

output "gateway_port" {
  value = data.openclaw_gateway.current.port
}

output "auth_enabled" {
  value = data.openclaw_gateway.current.auth_mode != "none"
}
```

### Conditional logic based on gateway config

```hcl
data "openclaw_gateway" "current" {}

resource "openclaw_channel_whatsapp" "main" {
  # Only enable if gateway is bound to all interfaces
  count     = data.openclaw_gateway.current.bind == "all" ? 1 : 0
  dm_policy = "pairing"
}
```

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Always `"gateway"`. |
| `port` | Int64 | Gateway listen port. |
| `bind` | String | Bind address: `loopback` or `all`. |
| `auth_mode` | String | Authentication mode: `token`, `password`, or `none`. |
| `reload_mode` | String | Config reload mode: `hybrid`, `hot`, `restart`, or `off`. |
| `tailscale_mode` | String | Tailscale exposure mode: `off`, `serve`, or `funnel`. |
| `mode` | String | Gateway mode (e.g. `local`). |
