#!/usr/bin/env bash
set -euo pipefail

vars=(DISPLAY WAYLAND_DISPLAY XAUTHORITY XDG_RUNTIME_DIR DBUS_SESSION_BUS_ADDRESS)
present=()

for name in "${vars[@]}"; do
  if [[ -n "${!name:-}" ]]; then
    present+=("$name")
  fi
done

if [[ ${#present[@]} -eq 0 ]]; then
  exit 0
fi

systemctl --user import-environment "${present[@]}"
if command -v dbus-update-activation-environment >/dev/null 2>&1; then
  dbus-update-activation-environment --systemd "${present[@]}"
fi

systemctl --user start ghost-os.target
systemctl --user try-restart ghost-os-bridge.service ghost-os-web.service
