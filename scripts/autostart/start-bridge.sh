#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BRIDGE_BIN="$ROOT/bin/ghost-bridge"
NATIVE_BIN="$ROOT/bin/native"
CONFIG_PATH="${GHOST_CONFIG_PATH:-$HOME/.ghost-os/config.toml}"
BRIDGE_PORT="${GHOST_BRIDGE_PORT:-8080}"

if [[ ! -x "$BRIDGE_BIN" ]]; then
  echo "missing bridge binary: $BRIDGE_BIN; run python3 task.py build" >&2
  exit 1
fi

if [[ ! -x "$NATIVE_BIN" ]]; then
  echo "missing native binary: $NATIVE_BIN; run python3 task.py build" >&2
  exit 1
fi

if [[ ! -f "$CONFIG_PATH" ]]; then
  echo "missing bridge config: $CONFIG_PATH" >&2
  exit 1
fi

if [[ -z "${GHOST_NATIVE_BINARY_PATH_OVERRIDE:-}" ]]; then
  export GHOST_NATIVE_BINARY_PATH_OVERRIDE="$NATIVE_BIN"
fi

exec "$BRIDGE_BIN" serve "$BRIDGE_PORT"
