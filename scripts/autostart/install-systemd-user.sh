#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
UNIT_SRC="$ROOT/deploy/systemd/user"
AUTOSTART_SRC="$ROOT/deploy/autostart"
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"
AUTOSTART_DIR="$HOME/.config/autostart"
RUNTIME_ENV_DIR="$HOME/.config/ghost-os"
RUNTIME_ENV_FILE="$RUNTIME_ENV_DIR/autostart.env"
SKIP_BUILD=false

if [[ "${1:-}" == "--skip-build" ]]; then
  SKIP_BUILD=true
fi

render_template() {
  local source="$1"
  local target="$2"
  sed "s|__ROOT__|$ROOT|g" "$source" >"$target"
}

prepare_runtime() {
  python3 "$ROOT/task.py" build
  pnpm --dir "$ROOT/apps/web" install --frozen-lockfile
}

write_env_file() {
  if [[ -f "$RUNTIME_ENV_FILE" ]]; then
    return
  fi
  cat >"$RUNTIME_ENV_FILE" <<EOF
GHOST_CONFIG_PATH=$HOME/.ghost-os/config.toml
GHOST_BRIDGE_URL=http://127.0.0.1:8080
GHOST_WEB_START_MODE=dev
HOSTNAME=127.0.0.1
PORT=3000
EOF
}

install_units() {
  render_template "$UNIT_SRC/ghost-os.target.template" "$SYSTEMD_USER_DIR/ghost-os.target"
  render_template "$UNIT_SRC/ghost-os-bridge.service.template" "$SYSTEMD_USER_DIR/ghost-os-bridge.service"
  render_template "$UNIT_SRC/ghost-os-web.service.template" "$SYSTEMD_USER_DIR/ghost-os-web.service"
  render_template "$AUTOSTART_SRC/ghost-os-session-env.desktop.template" "$AUTOSTART_DIR/ghost-os-session-env.desktop"
}

mkdir -p "$SYSTEMD_USER_DIR" "$AUTOSTART_DIR" "$RUNTIME_ENV_DIR"

if ! $SKIP_BUILD; then
  prepare_runtime
fi

write_env_file
install_units
systemctl --user daemon-reload
systemctl --user enable --now ghost-os.target
