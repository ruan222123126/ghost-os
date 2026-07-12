#!/usr/bin/env python3
"""Remind Codex about uncommitted changes created during the current turn."""

from __future__ import annotations

import hashlib
import json
import os
import stat
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[2]
STATE_DIR = Path(tempfile.gettempdir()) / "ghost-os-codex-git-hook"
REMINDER = (
    "This turn changed the repository and left uncommitted changes. Review "
    "`git status` and commit the changes that belong to this task before the "
    "final response. Do not commit unrelated user changes; if changes should "
    "remain uncommitted, say so explicitly in the final response."
)


def run_git(repo_root: Path, args: list[str], *, check: bool = True) -> bytes:
    result = subprocess.run(
        ["git", "-C", str(repo_root), *args],
        check=check,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return result.stdout


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


def is_for_this_repo(event: dict[str, Any], repo_root: Path) -> bool:
    cwd_value = event.get("cwd")
    if not isinstance(cwd_value, str) or not cwd_value:
        return False

    try:
        cwd = Path(cwd_value).resolve()
        resolved_root = repo_root.resolve()
        return cwd == resolved_root or resolved_root in cwd.parents
    except OSError:
        return False


def has_changes(repo_root: Path) -> bool:
    return bool(run_git(repo_root, ["status", "--porcelain=v1", "-z"]).strip())


def update_digest(digest: Any, label: bytes, content: bytes) -> None:
    digest.update(len(label).to_bytes(8, "big"))
    digest.update(label)
    digest.update(len(content).to_bytes(8, "big"))
    digest.update(content)


def hash_untracked_file(repo_root: Path, relative_path: bytes) -> bytes:
    path = repo_root / os.fsdecode(relative_path)
    metadata = path.lstat()
    digest = hashlib.sha256()
    digest.update(stat.S_IFMT(metadata.st_mode).to_bytes(8, "big"))

    if stat.S_ISLNK(metadata.st_mode):
        digest.update(os.fsencode(os.readlink(path)))
    elif stat.S_ISREG(metadata.st_mode):
        with path.open("rb") as handle:
            for chunk in iter(lambda: handle.read(1024 * 1024), b""):
                digest.update(chunk)
    else:
        digest.update(str(metadata.st_size).encode("ascii"))
    return digest.digest()


def workspace_fingerprint(repo_root: Path) -> str:
    digest = hashlib.sha256()
    head = run_git(repo_root, ["rev-parse", "--verify", "HEAD"], check=False)
    staged = run_git(repo_root, ["diff", "--binary", "--no-ext-diff", "--cached"])
    unstaged = run_git(repo_root, ["diff", "--binary", "--no-ext-diff"])
    untracked = run_git(
        repo_root,
        ["ls-files", "--others", "--exclude-standard", "-z"],
    )

    update_digest(digest, b"head", head)
    update_digest(digest, b"staged", staged)
    update_digest(digest, b"unstaged", unstaged)
    for relative_path in sorted(filter(None, untracked.split(b"\0"))):
        update_digest(digest, b"untracked-path", relative_path)
        update_digest(
            digest,
            b"untracked-content",
            hash_untracked_file(repo_root, relative_path),
        )
    return digest.hexdigest()


def event_identity(event: dict[str, Any]) -> str:
    session_id = event.get("session_id")
    turn_id = event.get("turn_id")
    if not isinstance(session_id, str) or not session_id:
        raise ValueError("hook input is missing session_id")
    if not isinstance(turn_id, str) or not turn_id:
        raise ValueError("hook input is missing turn_id")
    return hashlib.sha256(f"{session_id}\0{turn_id}".encode()).hexdigest()


def state_path(event: dict[str, Any], state_dir: Path) -> Path:
    return state_dir / f"{event_identity(event)}.json"


def save_baseline(event: dict[str, Any], repo_root: Path, state_dir: Path) -> None:
    state_dir.mkdir(parents=True, exist_ok=True)
    target = state_path(event, state_dir)
    payload = json.dumps({"fingerprint": workspace_fingerprint(repo_root)})

    with tempfile.NamedTemporaryFile(
        mode="w",
        encoding="utf-8",
        dir=state_dir,
        prefix=f".{target.name}.",
        delete=False,
    ) as handle:
        handle.write(payload)
        temporary_path = Path(handle.name)
    temporary_path.replace(target)


def load_baseline(event: dict[str, Any], state_dir: Path) -> str | None:
    target = state_path(event, state_dir)
    try:
        value = json.loads(target.read_text(encoding="utf-8"))
    except FileNotFoundError:
        return None
    if not isinstance(value, dict) or not isinstance(value.get("fingerprint"), str):
        raise ValueError(f"invalid hook state file: {target}")
    return value["fingerprint"]


def clear_baseline(event: dict[str, Any], state_dir: Path) -> None:
    state_path(event, state_dir).unlink(missing_ok=True)


def handle_user_prompt(
    event: dict[str, Any], repo_root: Path, state_dir: Path
) -> dict[str, Any]:
    save_baseline(event, repo_root, state_dir)
    return {"continue": True}


def handle_stop(
    event: dict[str, Any], repo_root: Path, state_dir: Path
) -> dict[str, Any]:
    baseline = load_baseline(event, state_dir)
    if baseline is None:
        return {
            "continue": True,
            "systemMessage": "Codex commit reminder skipped: turn baseline is missing.",
        }

    changed_this_turn = workspace_fingerprint(repo_root) != baseline
    if not has_changes(repo_root) or not changed_this_turn:
        clear_baseline(event, state_dir)
        return {"continue": True}

    if event.get("stop_hook_active") is True:
        clear_baseline(event, state_dir)
        return {"continue": True, "systemMessage": REMINDER}

    return {"decision": "block", "reason": REMINDER}


def process_event(
    event: dict[str, Any], repo_root: Path = REPO_ROOT, state_dir: Path = STATE_DIR
) -> dict[str, Any]:
    if not is_for_this_repo(event, repo_root):
        return {"continue": True}

    event_name = event.get("hook_event_name")
    if event_name == "UserPromptSubmit":
        return handle_user_prompt(event, repo_root, state_dir)
    if event_name == "Stop":
        return handle_stop(event, repo_root, state_dir)
    return {"continue": True}


def main() -> int:
    try:
        emit(process_event(read_event()))
        return 0
    except (OSError, subprocess.CalledProcessError, ValueError) as exc:
        detail = str(exc)
        if isinstance(exc, subprocess.CalledProcessError):
            stderr = exc.stderr.decode(errors="replace") if exc.stderr else ""
            stdout = exc.stdout.decode(errors="replace") if exc.stdout else ""
            detail = (stderr or stdout or str(exc)).strip()
        emit({"continue": True, "systemMessage": f"Codex commit reminder failed: {detail}"})
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
