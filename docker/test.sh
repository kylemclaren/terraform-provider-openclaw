#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────
# test.sh — One-command integration test for the OpenClaw TF provider
#
# Usage:
#   ./docker/test.sh              # Run file-mode acceptance tests (default)
#   ./docker/test.sh test-ws      # Run WS-mode tests (needs gateway)
#   ./docker/test.sh test-all     # Run all acceptance tests (file + WS)
#   ./docker/test.sh apply        # Build + terraform apply the test-stack
#   ./docker/test.sh shell        # Drop into an interactive shell
#   ./docker/test.sh down         # Tear everything down
#
# All services use --network host so the gateway binds to real loopback.
# This ensures WS device identity is auto-approved (no NOT_PAIRED errors).
#
# Environment:
#   OPENCLAW_GATEWAY_TOKEN  Gateway auth token (default: test-token-for-ci)
#   OPENCLAW_GATEWAY_PORT   Host port for gateway (default: 18789)
# ──────────────────────────────────────────────────────────────────
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"
COMPOSE="docker compose -f $COMPOSE_FILE --project-directory $PROJECT_ROOT"

export OPENCLAW_GATEWAY_TOKEN="${OPENCLAW_GATEWAY_TOKEN:-test-token-for-ci}"
export OPENCLAW_GATEWAY_PORT="${OPENCLAW_GATEWAY_PORT:-18789}"

TF_IMAGE="terraform-provider-openclaw-terraform"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}==>${NC} $*"; }
warn()  { echo -e "${YELLOW}==>${NC} $*"; }
error() { echo -e "${RED}==>${NC} $*" >&2; }

usage() {
  cat <<EOF
Usage: $0 [COMMAND]

Commands:
  test       Run file-mode acceptance tests (default, no gateway needed)
  test-ws    Run WS-mode tests against a live gateway
  test-all   Run all acceptance tests (file + WS)
  apply      Build the provider and terraform apply the test-stack
  plan       Build the provider and terraform plan the test-stack
  shell      Start an interactive shell with provider + terraform available
  down       Tear down all containers and volumes
  logs       Tail gateway logs

Environment:
  OPENCLAW_GATEWAY_TOKEN  Auth token (default: test-token-for-ci)
  OPENCLAW_GATEWAY_PORT   Host port (default: 18789)
EOF
}

# ── Build helpers ──────────────────────────────────────────────

cmd_build_terraform() {
  info "Building terraform image..."
  $COMPOSE build terraform
}

cmd_build_all() {
  info "Building gateway and terraform images..."
  $COMPOSE build
}

# ── Gateway lifecycle ─────────────────────────────────────────

start_gateway() {
  # All services use network_mode: host (set in docker-compose.yml).
  # The gateway binds to loopback so WS clients get auto-approved.
  $COMPOSE up -d gateway

  info "Waiting for gateway to start..."
  local retries=30
  while [ $retries -gt 0 ]; do
    if curl -sf "http://127.0.0.1:${OPENCLAW_GATEWAY_PORT}/" >/dev/null 2>&1; then
      info "Gateway is ready."
      return 0
    fi
    sleep 2
    retries=$((retries - 1))
  done
  error "Gateway failed to start within 60s"
  $COMPOSE logs gateway 2>&1 | tail -20
  return 1
}

run_tf_tests() {
  local run_pattern="$1"
  shift
  local extra_env=("$@")

  # Try compose first (all on host network, gateway at 127.0.0.1)
  if $COMPOSE run --rm --no-deps "${extra_env[@]}" terraform \
    go test ./internal/provider/ -v -timeout 120m -count=1 -run "$run_pattern" 2>/dev/null; then
    return 0
  fi

  # Fallback: direct docker run (e.g. if compose run has issues)
  warn "Compose run failed, falling back to docker run."
  docker run --rm --network host \
    -e TF_ACC=1 \
    "${extra_env[@]}" \
    "$TF_IMAGE" \
    go test ./internal/provider/ -v -timeout 120m -count=1 -run "$run_pattern"
}

# ── Test commands ─────────────────────────────────────────────

cmd_test() {
  cmd_build_terraform
  info "Running file-mode acceptance tests (no gateway needed)..."
  run_tf_tests "TestAccFileMode" \
    -e OPENCLAW_GATEWAY_URL= \
    -e OPENCLAW_GATEWAY_TOKEN=
  info "File-mode tests complete."
}

run_ws_tests() {
  local run_pattern="${1:-TestAccWSMode}"
  # All on host network — gateway at ws://127.0.0.1:18789
  if $COMPOSE run --rm \
    -e OPENCLAW_GATEWAY_URL="ws://127.0.0.1:${OPENCLAW_GATEWAY_PORT}" \
    -e OPENCLAW_GATEWAY_TOKEN="$OPENCLAW_GATEWAY_TOKEN" \
    terraform \
    go test ./internal/provider/ -v -timeout 120m -count=1 -run "$run_pattern" 2>/dev/null; then
    return 0
  fi

  warn "Compose run failed, falling back to docker run."
  docker run --rm --network host \
    -e TF_ACC=1 \
    -e OPENCLAW_GATEWAY_URL="ws://127.0.0.1:${OPENCLAW_GATEWAY_PORT}" \
    -e OPENCLAW_GATEWAY_TOKEN="$OPENCLAW_GATEWAY_TOKEN" \
    "$TF_IMAGE" \
    go test ./internal/provider/ -v -timeout 120m -count=1 -run "$run_pattern"
}

cmd_test_ws() {
  cmd_build_all
  start_gateway
  info "Running WS-mode acceptance tests..."
  run_ws_tests "TestAccWSMode"
  info "WS-mode tests complete."
}

cmd_test_all() {
  cmd_build_all

  info "Running file-mode tests (no gateway)..."
  run_tf_tests "TestAccFileMode" \
    -e OPENCLAW_GATEWAY_URL= \
    -e OPENCLAW_GATEWAY_TOKEN=
  info "File-mode tests passed."

  start_gateway
  info "Running WS-mode tests..."
  run_ws_tests "TestAccWSMode"
  info "All tests complete."
}

cmd_apply() {
  cmd_build_all
  start_gateway
  info "Running terraform apply on test-stack..."
  $COMPOSE run --rm terraform sh -c '
      cd /work/docker/test-stack &&
      terraform plan -out=tfplan &&
      terraform apply tfplan &&
      echo "" &&
      echo "=== Terraform outputs ===" &&
      terraform output
    '
  info "Apply complete."
}

cmd_plan() {
  cmd_build_all
  start_gateway
  info "Running terraform plan on test-stack..."
  $COMPOSE run --rm terraform sh -c 'cd /work/docker/test-stack && terraform plan'
}

cmd_shell() {
  cmd_build_all
  start_gateway
  info "Dropping into shell (provider built, terraform available)..."
  $COMPOSE run --rm shell
}

cmd_down() {
  info "Tearing down..."
  $COMPOSE down -v --remove-orphans 2>/dev/null || true
  info "Done."
}

cmd_logs() {
  $COMPOSE logs -f gateway
}

# ── Main ────────────────────────────────────────────────────────
COMMAND="${1:-test}"

case "$COMMAND" in
  test)     cmd_test     ;;
  test-ws)  cmd_test_ws  ;;
  test-all) cmd_test_all ;;
  apply)    cmd_apply    ;;
  plan)     cmd_plan     ;;
  shell)    cmd_shell    ;;
  down)     cmd_down     ;;
  logs)     cmd_logs     ;;
  -h|--help) usage       ;;
  *)
    error "Unknown command: $COMMAND"
    usage
    exit 1
    ;;
esac
