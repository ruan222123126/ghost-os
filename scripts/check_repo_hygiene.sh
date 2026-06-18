#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

is_allowed_env_example() {
  case "$1" in
    */.env.example|.env.example|*/.env.sample|.env.sample|*/.env.template|.env.template)
      return 0
      ;;
  esac
  return 1
}

is_forbidden_tracked_path() {
  local path="$1"

  case "$path" in
    .DS_Store|*/.DS_Store|Thumbs.db|*/Thumbs.db|*.iml|*/local.properties)
      return 0
      ;;
    *.swp|*.swo|*.tmp.*|.tmp_*|*/.tmp_*|*.tsbuildinfo)
      return 0
      ;;
    .codex|*/.codex|.codex/*|*/.codex/*)
      return 0
      ;;
    data/logs/*|.pnpm-store/*|*/.pnpm-store/*|target/*|*/target/*|dist/*|*/dist/*|build/*|*/build/*|coverage/*|*/coverage/*)
      return 0
      ;;
    node_modules/*|*/node_modules/*|.next/*|*/.next/*|.next-dev/*|*/.next-dev/*|.next.bak*/*|*/.next.bak*/*|.gradle/*|*/.gradle/*|.idea/*|*/.idea/*|.vscode/*|*/.vscode/*)
      return 0
      ;;
    .env|*/.env|.env.*|*/.env.*)
      is_allowed_env_example "$path" && return 1
      return 0
      ;;
  esac

  return 1
}

main() {
  local -a offenders=()
  local path

  cd "$ROOT_DIR"
  while IFS= read -r path; do
    if [ ! -e "$path" ]; then
      continue
    fi
    if is_forbidden_tracked_path "$path"; then
      offenders+=("$path")
    fi
  done < <(git ls-files)

  if ((${#offenders[@]} > 0)); then
    printf 'tracked repo hygiene violations:\n' >&2
    printf ' - %s\n' "${offenders[@]}" >&2
    return 1
  fi

  printf 'repo hygiene check passed.\n'
}

main "$@"
