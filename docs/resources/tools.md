---
page_title: "openclaw_tools Resource - openclaw"
subcategory: ""
description: |-
  Manages the OpenClaw tool access control configuration.
---

# openclaw_tools

Manages tool access control including profiles, allow/deny lists, and elevated/browser tool toggles.

This is a singleton resource.

## Example Usage

### Use a preset profile

```hcl
resource "openclaw_tools" "main" {
  profile         = "coding"
  elevated_enabled = false
  browser_enabled  = true
}
```

### Fine-grained allow/deny lists

```hcl
resource "openclaw_tools" "main" {
  allow = ["bash", "read", "write", "glob", "grep", "edit"]
  deny  = ["rm", "curl"]

  elevated_enabled = false
  browser_enabled  = false
}
```

## Argument Reference

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `profile` | String | No | Tools profile: `minimal`, `coding`, `messaging`, or `full`. |
| `allow` | List(String) | No | Explicit list of tool names to allow. |
| `deny` | List(String) | No | Explicit list of tool names to deny. |
| `elevated_enabled` | Bool | No | Enable elevated (privileged) tool execution. |
| `browser_enabled` | Bool | No | Enable browser-based tools. |

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Always `"tools"`. |

## Import

```bash
terraform import openclaw_tools.main tools
```
