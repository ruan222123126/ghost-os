#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WEB_ROOT="$ROOT/apps/web"
NEXT_ENTRY="$WEB_ROOT/node_modules/next/dist/bin/next"
BUILD_ID_FILE="$WEB_ROOT/.next/BUILD_ID"
WEB_START_MODE="${GHOST_WEB_START_MODE:-dev}"
WEB_HOST="${GHOST_WEB_HOST:-127.0.0.1}"
WEB_PORT="${GHOST_WEB_PORT:-3000}"
GHOST_BRIDGE_URL="${GHOST_BRIDGE_URL:-http://127.0.0.1:8080}"
GHOST_CONFIG_PATH="${GHOST_CONFIG_PATH:-$HOME/.ghost-os/config.toml}"

export GHOST_BRIDGE_URL GHOST_CONFIG_PATH

resolve_node_bin() {
  if [[ -n "${NODE_BIN_PATH:-}" && -x "${NODE_BIN_PATH:-}" ]]; then
    echo "$NODE_BIN_PATH"
    return 0
  fi
  if command -v node >/dev/null 2>&1; then
    command -v node
    return 0
  fi
  local candidate
  shopt -s nullglob
  for candidate in "$HOME/.nvm/versions/node"/*/bin/node /usr/local/bin/node /usr/bin/node; do
    if [[ -x "$candidate" ]]; then
      echo "$candidate"
      shopt -u nullglob
      return 0
    fi
  done
  shopt -u nullglob
  return 1
}

if [[ ! -f "$NEXT_ENTRY" ]]; then
  echo "missing web runtime: $NEXT_ENTRY; run pnpm --dir apps/web install --frozen-lockfile" >&2
  exit 1
fi

NODE_BIN="$(resolve_node_bin || true)"
if [[ -z "$NODE_BIN" ]]; then
  echo "node binary not found; set NODE_BIN_PATH or install node in PATH/system-wide" >&2
  exit 1
fi

cd "$WEB_ROOT"

if [[ "$WEB_START_MODE" == "dev" ]]; then
  rm -rf .next-dev
  export NODE_ENV=development
  exec "$NODE_BIN" "$NEXT_ENTRY" dev --hostname "$WEB_HOST" --port "$WEB_PORT"
fi

if [[ "$WEB_START_MODE" != "prod" ]]; then
  echo "invalid GHOST_WEB_START_MODE: $WEB_START_MODE; expected dev or prod" >&2
  exit 1
fi

if [[ ! -f "$BUILD_ID_FILE" ]]; then
  echo "missing web build output: $BUILD_ID_FILE; run pnpm --dir apps/web build or set GHOST_WEB_START_MODE=dev" >&2
  exit 1
fi

export NODE_ENV=production
exec "$NODE_BIN" "$NEXT_ENTRY" start --hostname "$WEB_HOST" --port "$WEB_PORT"
