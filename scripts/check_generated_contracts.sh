#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GENERATED_FILES=(
  "core/bridge/orchestration/envelope_generated.go"
  "apps/web/lib/envelope.generated.ts"
  "apps/cli/src/envelope_generated.rs"
  "apps/android/app/src/main/java/dev/ghostos/android/model/ApiModels.kt"
)

main() {
  cd "$ROOT_DIR"
  python3 core/shared/generate_envelope_types.py

  if ! git diff --quiet -- "${GENERATED_FILES[@]}"; then
    printf 'shared contract generated files are out of sync.\n' >&2
    printf 'run: python3 core/shared/generate_envelope_types.py\n' >&2
    printf 'changed files:\n' >&2
    git diff --name-only -- "${GENERATED_FILES[@]}" >&2
    return 1
  fi

  printf 'shared contract generated files are up to date.\n'
}

main "$@"
