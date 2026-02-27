from __future__ import annotations

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent


def run(cmd: list[str], cwd: Path | None = None) -> int:
    location = cwd if cwd else ROOT
    result = subprocess.run(cmd, cwd=location)
    return result.returncode


def build_rust() -> int:
    print("build rust native...")
    return run(["cargo", "build", "--release"], ROOT / "drivers/native")


def build_go() -> int:
    print("build go bridge...")
    (ROOT / "bin").mkdir(exist_ok=True)
    return run(["go", "build", "-o", "../../bin/ghost-bridge"], ROOT / "core/bridge")


def ping() -> int:
    print("build rust debug...")
    if run(["cargo", "build"], ROOT / "drivers/native") != 0:
        return 1
    print("run bridge ping...")
    return run(["go", "run", "."], ROOT / "core/bridge")


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
    if action == "init-web":
        return init_web()

    print("usage: python task.py [build | ping | init-web]")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
