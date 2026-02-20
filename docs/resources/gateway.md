---
page_title: "openclaw_gateway Resource - openclaw"
subcategory: ""
description: |-
  Manages the OpenClaw gateway server configuration.
---

# openclaw_gateway

Manages the OpenClaw gateway server configuration including port, bind address, authentication, and reload behavior.

This is a singleton resource -- only one `openclaw_gateway` block should exist per configuration.

## Example Usage

```hcl
resource "openclaw_gateway" "main" {
  port        = 18789
  bind        = "loopback"
  auth_mode   = "token"
  auth_token  = var.gateway_token
  reload_mode = "hybrid"
}
```

### Expose via Tailscale

```hcl
resource "openclaw_gateway" "main" {
  port           = 18789
  bind           = "all"
  tailscale_mode = "funnel"
  auth_mode      = "token"
  auth_token     = var.gateway_token
}
```

## Argument Reference

| Argument | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `port` | Int64 | No | `18789` | Gateway listen port. |
| `bind` | String | No | `"loopback"` | Bind address: `loopback` or `all`. |
| `auth_mode` | String | No | -- | Authentication mode: `token`, `password`, or `none`. |
| `auth_token` | String | No | -- | Gateway auth token. **Sensitive.** |
| `reload_mode` | String | No | `"hybrid"` | Config reload mode: `hybrid`, `hot`, `restart`, or `off`. |
| `tailscale_mode` | String | No | -- | Tailscale exposure: `off`, `serve`, or `funnel`. |

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Always `"gateway"`. |

## Import

```bash
terraform import openclaw_gateway.main gateway
```
