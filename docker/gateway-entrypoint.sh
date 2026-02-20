#!/bin/sh
# Entrypoint for the OpenClaw gateway container.
# Seeds ~/.openclaw/openclaw.json with the auth token so that
# non-loopback connections (e.g. from Docker bridge) can authenticate.
set -e

CONFIG_DIR="${HOME}/.openclaw"
CONFIG_FILE="${CONFIG_DIR}/openclaw.json"

mkdir -p "$CONFIG_DIR"

if [ -n "$OPENCLAW_GATEWAY_TOKEN" ]; then
  cat > "$CONFIG_FILE" <<EOF
{
  "gateway": {
    "auth": {
      "token": "$OPENCLAW_GATEWAY_TOKEN"
    }
  }
}
EOF
  echo "Seeded $CONFIG_FILE with auth token."
else
  # Start with empty config if no token provided
  echo "{}" > "$CONFIG_FILE"
  echo "Seeded $CONFIG_FILE with empty config (no auth token)."
fi

exec "$@"
