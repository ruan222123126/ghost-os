#!/usr/bin/env python3
"""Codex Stop hook that reminds the agent to handle pending git commits."""

from __future__ import annotations

import json
import subprocess
import sys
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[2]
REMINDER = (
    "The repository has uncommitted changes. Review `git status` and commit the "
    "changes that belong to this task before the final response. Do not commit "
    "unrelated user changes; if changes should remain uncommitted, say so "
    "explicitly in the final response."
)


def run_git(args: list[str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", "-C", str(REPO_ROOT), *args],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )


def emit(payload: dict[str, Any]) -> None:
    print(json.dumps(payload, ensure_ascii=False))


def read_event() -> dict[str, Any]:
    try:
        value = json.load(sys.stdin)
    except json.JSONDecodeError as exc:
        raise ValueError(f"invalid hook JSON: {exc}") from exc
    if not isinstance(value, dict):
        raise ValueError("hook input must be a JSON object")
    return value


def is_for_this_repo(event: dict[str, Any]) -> bool:
    cwd_value = event.get("cwd")
    if not isinstance(cwd_value, str) or not cwd_value:
        return False

    try:
        cwd = Path(cwd_value).resolve()
        return cwd == REPO_ROOT or REPO_ROOT in cwd.parents
    except OSError:
        return False


def has_changes() -> bool:
    return bool(run_git(["status", "--porcelain=v1"]).stdout.strip())


def main() -> int:
    try:
        event = read_event()
        if event.get("hook_event_name") != "Stop" or not is_for_this_repo(event):
            emit({"continue": True})
            return 0

        if not has_changes():
            emit({"continue": True})
            return 0

        if event.get("stop_hook_active") is True:
            emit({"continue": True, "systemMessage": REMINDER})
            return 0

        emit({"decision": "block", "reason": REMINDER})
        return 0
    except (OSError, subprocess.CalledProcessError, ValueError) as exc:
        detail = str(exc)
        if isinstance(exc, subprocess.CalledProcessError):
            detail = (exc.stderr or exc.stdout or str(exc)).strip()
        emit({"continue": True, "systemMessage": f"Codex commit reminder failed: {detail}"})
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
