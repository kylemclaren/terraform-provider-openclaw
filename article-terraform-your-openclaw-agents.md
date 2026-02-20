# Terraform Your OpenClaw Agents

OpenClaw went from a weekend WhatsApp relay script to [180,000+ GitHub stars](https://x.com/TKVResearch/status/2018910347164529084) in under two months. People are calling it the [death of SaaS lock-in](https://x.com/fabulouzfab/status/2022025007212241395). The ClawIndex just crossed [245 apps](https://x.com/0xSammy/status/2023414421402300654). And with OpenAI [bringing on its creator](https://x.com/levie/status/2023152367366222151), the ecosystem is only accelerating.

But here's the thing nobody's talking about: **configuring OpenClaw is still a manual JSON editing nightmare.**

Your `openclaw.json` file quietly becomes the most important config in your stack. It controls which AI models your agents use, which chat channels they respond on, who's allowed to talk to them, how sessions reset, what tools are available, and how multi-agent routing works. One wrong edit and your agent goes silent -- or worse, starts responding to people it shouldn't.

Today I'm releasing **[terraform-provider-openclaw](https://github.com/kylemclaren/terraform-provider-openclaw)** -- a Terraform provider that lets you manage your entire OpenClaw gateway configuration as infrastructure-as-code.

## Why Terraform?

The OpenClaw community has already built Terraform modules for [deploying the gateway itself on Hetzner](https://github.com/andreesg/openclaw-terraform-hetzner), and Akamai published a guide on [moving your agent to the cloud](https://www.akamai.com/blog/developers/openclaw-agent-doesnt-sleep-laptop-does-move-cloud). But deploying the *server* is only half the problem. What about the *configuration running on it*?

That's what this provider solves. Instead of hand-editing JSON, you declare your desired state:

```hcl
resource "openclaw_agent" "home" {
  agent_id      = "home"
  default_agent = true
  name          = "Molty"
  model         = "anthropic/claude-opus-4-6"

  identity_name  = "Molty"
  identity_emoji = "🦞"
  identity_theme = "helpful space lobster"

  mention_patterns = ["@openclaw", "molty"]
}

resource "openclaw_agent" "work" {
  agent_id  = "work"
  name      = "Work Agent"
  model     = "anthropic/claude-sonnet-4-5"

  sandbox_mode  = "all"
  sandbox_scope = "session"
  tools_profile = "coding"
  tools_deny    = ["canvas"]
}

resource "openclaw_binding" "home_wa" {
  agent_id      = openclaw_agent.home.agent_id
  match_channel = "whatsapp"
}

resource "openclaw_binding" "work_tg" {
  agent_id      = openclaw_agent.work.agent_id
  match_channel = "telegram"
}
```

`terraform plan` shows you exactly what will change. `terraform apply` makes it so. Git tracks every revision. PRs gate every update.

## 18 Resources, Two Modes

The provider covers every section of the OpenClaw config:

**Core** -- gateway settings, agent defaults, individual agents, multi-agent bindings, session lifecycle, message handling.

**Channels** -- WhatsApp, Telegram, Discord, Slack, Signal, iMessage, Google Chat. Each with full control over DM policies, allowlists, streaming modes, history limits, and platform-specific features.

**Extensions** -- plugins, skills, hooks, cron jobs, tool access control.

It also ships 6 read-only **data sources** for introspecting your running gateway -- useful for health checks, config auditing, or feeding values into other Terraform resources.

## Live or Offline -- Your Call

The provider auto-detects how to connect:

| Mode | When | How |
|------|------|-----|
| **WebSocket** | Gateway is running | Patches config via JSON-RPC. Changes take effect immediately. |
| **File** | Pre-provisioning, CI/CD | Reads/writes `openclaw.json` directly. No running gateway needed. |

File mode is the killer feature for anyone running OpenClaw in production. Build and validate your entire config in a CI pipeline, ship the JSON artifact, then start the gateway. No boot-time surprises.

## The Multi-Agent Story

This is where Terraform's declarative model really shines. OpenClaw's [multi-agent routing](https://deepwiki.com/openclaw/openclaw/4.2-configuration-management) lets you run specialized agents on different channels -- a personal assistant on WhatsApp, a coding agent on Telegram, a team bot on Discord. But wiring that up by hand means coordinating agent definitions, binding rules, channel configs, and tool permissions across a sprawling JSON file.

With the Terraform provider, those relationships are explicit, typed, and cross-referenced:

```hcl
resource "openclaw_binding" "home_wa" {
  agent_id         = openclaw_agent.home.agent_id
  match_channel    = "whatsapp"
  match_account_id = "personal"
}

resource "openclaw_binding" "work_tg" {
  agent_id      = openclaw_agent.work.agent_id
  match_channel = "telegram"
}
```

Add a new agent? Terraform tells you if you forgot a binding. Remove a channel? It warns you about orphaned routes. Rename an agent ID? One variable change propagates everywhere.

## Getting Started

```bash
terraform init    # pulls the provider from the registry
terraform plan    # shows what would change
terraform apply   # applies the config
```

The provider is on the [Terraform Registry](https://registry.terraform.io/providers/kylemclaren/openclaw/latest) at `registry.terraform.io/kylemclaren/openclaw`. For existing setups, every resource supports `terraform import`:

```bash
terraform import openclaw_gateway.main gateway
terraform import openclaw_agent.research research
terraform import openclaw_channel_whatsapp.main channel_whatsapp
```

If you want to kick the tires without installing anything locally, the repo includes a full Docker-based test harness that spins up an OpenClaw gateway and runs the provider against it:

```bash
git clone https://github.com/kylemclaren/terraform-provider-openclaw.git
cd terraform-provider-openclaw
make docker-test
```

## What's Next

The provider currently covers the full OpenClaw config surface as of v2026.2. As OpenClaw continues to evolve -- new channel types, new agent capabilities, the foundation governance model -- the provider will track it.

If you're already running OpenClaw (and given those star counts, a lot of you are), stop editing JSON by hand. Your AI agents deserve the same rigor as the rest of your infrastructure.

**GitHub:** [kylemclaren/terraform-provider-openclaw](https://github.com/kylemclaren/terraform-provider-openclaw)
**Registry:** [registry.terraform.io/kylemclaren/openclaw](https://registry.terraform.io/providers/kylemclaren/openclaw/latest)
**License:** MIT
