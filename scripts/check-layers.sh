#!/usr/bin/env bash
set -euo pipefail

check_no_matches() {
  local pattern="$1"
  shift
  local paths=()
  local path

  for path in "$@"; do
    if [ -e "$path" ]; then
      paths+=("$path")
    fi
  done

  if [ "${#paths[@]}" -eq 0 ]; then
    return 0
  fi

  if grep -rEn "$pattern" "${paths[@]}"; then
    return 1
  fi

  local rc=$?
  if [ "$rc" -eq 1 ]; then
    return 0
  fi
  return "$rc"
}

# apps/* 不得 import drivers/*
check_no_matches 'drivers/native|drivers/native/src' apps/cli/src apps/android/src apps/web/app apps/web/lib apps/web/components

(
  cd core/bridge
  go test ./transport -run 'TestTransportProduction(Import|Scope)Guard' -timeout 60s
  go test ./orchestration/internal -run 'TestOrchestrationM2StructureBudget|TestOrchestrationTopLevelFileAllowlist|TestDomainConcreteImportFreeze' -timeout 60s
)

echo "layer check ok"
