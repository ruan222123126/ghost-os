from __future__ import annotations

import os
import shlex
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent


def resolve_go_bin() -> str:
    configured = Path(os.environ["GO_BIN"]) if "GO_BIN" in os.environ else None
    if configured and configured.is_file():
        return str(configured)

    found = shutil.which("go")
    if found:
        return found

    candidates = [
        Path.home() / ".local/go/bin/go",
        Path.home() / "go/bin/go",
        Path("/usr/local/go/bin/go"),
        Path("/usr/lib/go/bin/go"),
        Path("/opt/homebrew/bin/go"),
    ]
    for candidate in candidates:
        if candidate.is_file():
            return str(candidate)

    return "go"


def run(cmd: list[str], cwd: Path | None = None) -> int:
    location = cwd if cwd else ROOT
    try:
        result = subprocess.run(cmd, cwd=location)
        return result.returncode
    except FileNotFoundError:
        if cmd and Path(cmd[0]).name == "go":
            # Fallback for environments where Go is only available in shell init PATH.
            shell_cmd = " ".join(shlex.quote(arg) for arg in cmd)
            result = subprocess.run(["bash", "-lc", shell_cmd], cwd=location)
            return result.returncode
        raise


def build_rust() -> int:
    print("build rust native...")
    return run(["cargo", "build", "--release"], ROOT / "drivers/native")


def build_go() -> int:
    print("build go bridge...")
    (ROOT / "bin").mkdir(exist_ok=True)
    return run([resolve_go_bin(), "build", "-o", "../../bin/ghost-bridge"], ROOT / "core/bridge")


def ping() -> int:
    print("build rust debug...")
    if run(["cargo", "build"], ROOT / "drivers/native") != 0:
        return 1
    print("run bridge ping...")
    return run([resolve_go_bin(), "run", ".", "ping"], ROOT / "core/bridge")


def agent() -> int:
    msg = " ".join(sys.argv[2:]) if len(sys.argv) > 2 else "Hello, what can you do?"
    print("run bridge agent...")
    return run([resolve_go_bin(), "run", ".", "agent", msg], ROOT / "core/bridge")


def init_web() -> int:
    print("init web console...")
    return run(
        [
            "npx",
            "create-next-app@latest",
            "apps/web",
            "--typescript",
            "--tailwind",
            "--eslint",
        ]
    )


def main() -> int:
    action = sys.argv[1] if len(sys.argv) > 1 else "help"
    if action == "build":
        if build_rust() != 0:
            return 1
        return build_go()
    if action == "ping":
        return ping()
    if action == "agent":
        return agent()
    if action == "init-web":
        return init_web()

    print("usage: python task.py [build | ping | agent | init-web]")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
