# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Terraform provider for OpenClaw — an AI gateway that connects chat platforms (WhatsApp, Telegram, Discord, Slack, Signal, iMessage, Google Chat) to AI coding agents. Written in Go using the HashiCorp Terraform Plugin Framework.

**Module:** `github.com/kylemclaren/terraform-provider-openclaw`
**Go version:** 1.25.1
**Registry address:** `registry.terraform.io/kylemclaren/openclaw`

## Common Commands

```bash
make build          # Build provider binary
make test           # Unit tests (no gateway needed)
make testacc        # Full acceptance tests (TF_ACC=1, needs gateway)
make lint           # golangci-lint
make fmt            # gofmt -s -w .
make generate       # go generate ./...
make docker-test    # Acceptance tests against Dockerized gateway
make docker-shell   # Interactive shell with provider + terraform + gateway
```

Run a single test:
```bash
go test ./internal/client/ -v -run TestFileClient
TF_ACC=1 go test ./internal/provider/ -v -run TestAccFileMode -timeout 120m
```

## Architecture

### Dual-Mode Client

The provider operates in two modes, auto-selected by configuration precedence (`gateway_url` > `OPENCLAW_GATEWAY_URL` env > file mode):

- **WebSocket mode** (`internal/client/ws.go`): Connects to a running OpenClaw gateway via WS JSON-RPC. Used for live config management.
- **File mode** (`internal/client/file.go`): Reads/writes the JSON config file directly. No running gateway needed. Default path: `~/.openclaw/openclaw.json`.

Both implement `client.Client` interface in `internal/client/client.go`. All CRUD operations go through `GetConfig` → `PatchConfig` with optimistic concurrency via `baseHash`.

### Resource Pattern

Every resource in `internal/resources/` follows the same structure:
1. `*Resource` struct holding a `client.Client`
2. `*ResourceModel` struct with `tfsdk` tags
3. Standard CRUD + `ImportState` methods
4. `modelToMap()` — converts TF model → `map[string]any` for config patching
5. `mapToModel()` — converts config map → TF model for state reads

Helper functions in `internal/resources/helpers.go` (`setIfString`, `readString`, `readFloat64AsInt64`, etc.) handle the TF types ↔ Go types conversion. JSON numbers unmarshal as `float64`, so integer fields use `readFloat64AsInt64`.

Channel resources (e.g., `channel_whatsapp.go`) use nested paths via `client.GetNestedSection` / `client.PatchNestedSection` under the `"channels"` config key.

### Config Operations

`internal/client/client.go` provides section-level helpers used by all resources:
- `GetSection` / `GetNestedSection` — read a config section by key path
- `PatchSection` / `PatchNestedSection` — merge-patch a section
- `DeleteSection` — set a key to `null` to remove it

### Package Layout

- `internal/provider/` — Provider setup, mode detection, resource/datasource registration
- `internal/client/` — Transport layer (WS + file implementations)
- `internal/resources/` — 18 Terraform resources (core, channels, automation)
- `internal/datasources/` — 6 read-only data sources (config, health, agents, channels, etc.)
- `internal/shared/` — `ProviderData` struct shared between resources and data sources
- `docker/` — Docker-based integration test harness (`test.sh`, compose, Dockerfiles)
- `docs/` — Terraform registry documentation (`.mdx`)
- `examples/` — Example HCL configurations (basic, full-stack, multi-agent)

### Resources (18 total)

Core: `gateway`, `agent_defaults`, `agent`, `binding`, `session`, `messages`
Channels: `channel_whatsapp`, `channel_telegram`, `channel_discord`, `channel_slack`, `channel_signal`, `channel_imessage`, `channel_google_chat`
Automation: `plugin`, `skill`, `hook`, `cron`, `tools`

### Data Sources (6 total)

`config`, `health`, `gateway`, `agent_defaults`, `agents`, `channels`

## Environment Variables

- `OPENCLAW_GATEWAY_URL` — WebSocket URL (triggers WS mode)
- `OPENCLAW_GATEWAY_TOKEN` — Auth token for WS connection
- `OPENCLAW_CONFIG_PATH` — Config file path (file mode, default `~/.openclaw/openclaw.json`)
- `TF_ACC=1` — Required for acceptance tests

## Adding a New Resource

1. Create `internal/resources/<name>.go` with struct, model, CRUD, and `modelToMap`/`mapToModel`
2. Add `NewXxxResource` constructor and register it in `internal/provider/provider.go` `Resources()` method
3. Add documentation in `docs/resources/<name>.mdx`
4. Add test coverage (file-mode acceptance test at minimum)
