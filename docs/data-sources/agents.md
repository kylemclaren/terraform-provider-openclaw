---
page_title: "openclaw_agents Data Source - openclaw"
subcategory: ""
description: |-
  Lists all configured OpenClaw agents.
---

# openclaw_agents (Data Source)

Lists all configured agents with their settings. Returns both a flat list of agent IDs and a structured list with per-agent details.

## Example Usage

```hcl
data "openclaw_agents" "all" {}

output "agent_count" {
  value = length(data.openclaw_agents.all.agent_ids)
}

output "default_agent" {
  value = data.openclaw_agents.all.default_agent_id
}

output "agent_ids" {
  value = data.openclaw_agents.all.agent_ids
}
```

### Create bindings for each agent

```hcl
data "openclaw_agents" "all" {}

resource "openclaw_binding" "telegram" {
  for_each      = toset(data.openclaw_agents.all.agent_ids)
  agent_id      = each.value
  match_channel = "telegram"
}
```

### Check if a specific agent exists

```hcl
data "openclaw_agents" "all" {}

locals {
  has_research_agent = contains(data.openclaw_agents.all.agent_ids, "research")
}
```

## Attribute Reference

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | Always `"agents"`. |
| `default_agent_id` | String | The agent ID marked as default. |
| `agent_ids` | List(String) | List of all agent IDs. |
| `agents` | List(Object) | List of agents with their configuration. |

### Nested `agents` Object

| Attribute | Type | Description |
|-----------|------|-------------|
| `agent_id` | String | Agent identifier. |
| `name` | String | Agent display name. |
| `is_default` | Bool | Whether this is the default agent. |
| `model` | String | Model assigned to this agent. |
| `workspace` | String | Workspace path for this agent. |
| `sandbox_mode` | String | Sandbox mode for this agent. |
| `tools_profile` | String | Tools profile for this agent. |
