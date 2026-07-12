from __future__ import annotations

import json
import os
import subprocess
from pathlib import Path

LAYER_ROOTS = ("core/bridge/", "drivers/native/", "apps/web/")
SOURCE_EXTENSIONS = {".go", ".rs", ".ts", ".tsx", ".js", ".jsx"}


def run_git(args: list[str]) -> str:
    result = subprocess.run(["git", *args], capture_output=True, text=True, check=False)
    if result.returncode == 0:
        return result.stdout
    stderr = result.stderr.strip() or "unknown git error"
    raise RuntimeError(f"git {' '.join(args)} failed: {stderr}")


def event_base_head() -> tuple[str, str] | None:
    event_path = os.environ.get("GITHUB_EVENT_PATH")
    if not event_path:
        return None
    payload = json.loads(Path(event_path).read_text(encoding="utf-8"))
    pull_request = payload.get("pull_request")
    if pull_request:
        return pull_request["base"]["sha"], pull_request["head"]["sha"]
    before = payload.get("before")
    after = payload.get("after")
    if before and after and not set(before) == {"0"}:
        return before, after
    return None


def diff_range() -> tuple[str, str]:
    override_base = os.environ.get("COMPLEXITY_DIFF_BASE", "").strip()
    if override_base:
        base = run_git(["rev-parse", override_base]).strip()
        head = run_git(["rev-parse", "HEAD"]).strip()
        return base, head

    from_event = event_base_head()
    if from_event:
        return from_event
    head = run_git(["rev-parse", "HEAD"]).strip()
    try:
        base = run_git(["rev-parse", "HEAD~1"]).strip()
        return base, head
    except RuntimeError:
        return head, head


def changed_source_files(base: str, head: str) -> list[Path]:
    diff = run_git(["diff", "--name-only", "--diff-filter=ACMRTUXB", f"{base}...{head}"])
    files: list[Path] = []
    for raw in diff.splitlines():
        path = raw.strip()
        if not path or not path.startswith(LAYER_ROOTS):
            continue
        candidate = Path(path)
        if candidate.suffix not in SOURCE_EXTENSIONS or not candidate.exists():
            continue
        files.append(candidate)
    return sorted(files)
