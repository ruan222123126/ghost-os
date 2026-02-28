#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BRIDGE_PORT="${GHOST_E2E_BRIDGE_PORT:-}"
RUN_LIVE_SMOKE="${GHOST_E2E_LIVE_SMOKE:-0}"
NATIVE_BIN="${ROOT_DIR}/drivers/native/target/debug/native"

BRIDGE_PID=""
BRIDGE_LOG=""

log() {
  printf '\n[%s] %s\n' "$(date +%H:%M:%S)" "$*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

cleanup() {
  if [[ -n "${BRIDGE_PID}" ]] && kill -0 "${BRIDGE_PID}" >/dev/null 2>&1; then
    kill "${BRIDGE_PID}" >/dev/null 2>&1 || true
    wait "${BRIDGE_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${BRIDGE_LOG}" ]] && [[ -f "${BRIDGE_LOG}" ]]; then
    rm -f "${BRIDGE_LOG}"
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"
  if [[ "${haystack}" != *"${needle}"* ]]; then
    echo "assertion failed: ${message}" >&2
    echo "expected to find: ${needle}" >&2
    echo "actual: ${haystack}" >&2
    exit 1
  fi
}

wait_bridge_ready() {
  local url="http://127.0.0.1:${BRIDGE_PORT}/api/config"
  for _ in $(seq 1 60); do
    if curl -sS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  echo "bridge did not become ready: ${url}" >&2
  if [[ -n "${BRIDGE_LOG}" ]] && [[ -f "${BRIDGE_LOG}" ]]; then
    echo "--- bridge log ---" >&2
    cat "${BRIDGE_LOG}" >&2
    echo "------------------" >&2
  fi
  exit 1
}

select_bridge_port() {
  if [[ -n "${BRIDGE_PORT}" ]]; then
    return 0
  fi

  for _ in $(seq 1 60); do
    local candidate
    candidate="$((20000 + RANDOM % 20000))"
    if command -v ss >/dev/null 2>&1; then
      if ss -ltn 2>/dev/null | grep -q "[.:]${candidate}[[:space:]]"; then
        continue
      fi
    fi
    BRIDGE_PORT="${candidate}"
    return 0
  done

  BRIDGE_PORT="18080"
}

run_live_native_smoke() {
  log "Running optional live native smoke checks"

  if [[ -z "${DISPLAY:-}" ]]; then
    echo "skip live smoke: DISPLAY is not set" >&2
    return 0
  fi

  if ! command -v wmctrl >/dev/null 2>&1; then
    echo "skip live smoke: wmctrl is not installed" >&2
    return 0
  fi

  local window_resp
  window_resp="$(printf '{"action":"BROWSER_QUERY","params":{"query_type":"window_list"},"trace_id":"e2e-live-window-list"}\n' | "${NATIVE_BIN}")"
  assert_contains "${window_resp}" '"status":"success"' "window_list should return success in live smoke"
}

main() {
  trap cleanup EXIT

  require_cmd cargo
  require_cmd go
  require_cmd pnpm
  require_cmd curl
  select_bridge_port

  log "Running deterministic regression checks across Trinity layers"
  cargo test --manifest-path "${ROOT_DIR}/drivers/native/Cargo.toml"
  cargo test --manifest-path "${ROOT_DIR}/apps/cli/Cargo.toml"
  (cd "${ROOT_DIR}/core/bridge" && go test ./...)
  (cd "${ROOT_DIR}/apps/web" && pnpm test -- --runInBand lib/api.test.ts)
  (cd "${ROOT_DIR}/apps/web" && pnpm exec tsc --noEmit)

  log "Building native binary for contract checks"
  cargo build --manifest-path "${ROOT_DIR}/drivers/native/Cargo.toml"

  log "Checking native action contracts (headless-safe)"
  local mouse_resp
  mouse_resp="$(printf '{"action":"MOUSE_CLICK","params":{"y":200},"trace_id":"e2e-mouse-missing-x"}\n' | "${NATIVE_BIN}")"
  assert_contains "${mouse_resp}" '"status":"error"' "MOUSE_CLICK missing x should fail"
  assert_contains "${mouse_resp}" 'x is required' "MOUSE_CLICK error should mention missing x"

  local browser_resp
  browser_resp="$(printf '{"action":"BROWSER_QUERY","params":{"query_type":"active_tab"},"trace_id":"e2e-browser-active"}\n' | "${NATIVE_BIN}")"
  assert_contains "${browser_resp}" '"status":' "BROWSER_QUERY should return a structured response"
  if [[ "${browser_resp}" == *'unsupported action: BROWSER_QUERY'* ]]; then
    echo "BROWSER_QUERY route is not wired" >&2
    exit 1
  fi

  log "Starting bridge and running HTTP smoke checks"
  BRIDGE_LOG="$(mktemp -t ghost-bridge-e2e-XXXX.log)"
  (
    cd "${ROOT_DIR}/core/bridge"
    GHOST_BIND_ADDR="127.0.0.1:${BRIDGE_PORT}" \
      GHOST_NATIVE_BIN="${NATIVE_BIN}" \
      go run . serve "${BRIDGE_PORT}" >"${BRIDGE_LOG}" 2>&1
  ) &
  BRIDGE_PID=$!
  wait_bridge_ready

  local config_resp
  config_resp="$(curl -sS "http://127.0.0.1:${BRIDGE_PORT}/api/config" || true)"
  assert_contains "${config_resp}" '"status":"success"' "/api/config should return success envelope"

  local config_bus_resp
  config_bus_resp="$(curl -sS -X POST "http://127.0.0.1:${BRIDGE_PORT}/api/bus" \
    -H "Content-Type: application/json" \
    -d '{"action":"CONFIG_GET","params":{},"trace_id":"e2e-config-get"}' || true)"
  assert_contains "${config_bus_resp}" '"status":"success"' "CONFIG_GET should succeed"
  assert_contains "${config_bus_resp}" '"provider"' "CONFIG_GET payload should include provider"

  log "CLI/Web smoke checks"
  cargo run --manifest-path "${ROOT_DIR}/apps/cli/Cargo.toml" -- --version >/dev/null
  (cd "${ROOT_DIR}/apps/web" && pnpm test -- --runInBand lib/api.test.ts >/dev/null)

  if [[ "${RUN_LIVE_SMOKE}" == "1" ]]; then
    run_live_native_smoke
  fi

  log "MVP regression checks passed"
}

main "$@"
