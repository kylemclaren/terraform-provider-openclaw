---
page_title: "openclaw_skill Resource - openclaw"
subcategory: ""
description: |-
  Manages an OpenClaw skill entry.
---

# openclaw_skill

Manages a skill entry under `skills.entries.<name>`. Skills are executable capabilities that agents can invoke.

Changing `skill_name` forces resource replacement.

## Example Usage

```hcl
resource "openclaw_skill" "calculator" {
  skill_name = "calculator"
  enabled    = true
  api_key    = var.calculator_api_key
  env_json = jsonencode({
    PRECISION = "high"
    MAX_DEPTH = "10"
  })
}
```

## Argument Reference

| Argument | Type | Required | Description |
|----------|------|----------|-------------|
| `skill_name` | String | **Yes** | Unique skill name. Used as the key under `skills.entries`. Changing this forces replacement. |
| `enabled` | Bool | No | Enable or disable this skill. |
| `api_key` | String | No | API key for the skill. **Sensitive.** |
| `env_json` | String | No | JSON object of environment variables to inject into the skill. |

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Same as `skill_name`. |

## Import

```bash
terraform import openclaw_skill.calculator calculator
```
