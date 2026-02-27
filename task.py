from __future__ import annotations

import os
import shlex
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent


# resolve_go_bin 在常见安装位置中定位 go 可执行文件。
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


# run 统一执行子命令，并在找不到 go 时回退到 bash -lc。
def run(cmd: list[str], cwd: Path | None = None) -> int:
    location = cwd if cwd else ROOT
    try:
        result = subprocess.run(cmd, cwd=location)
        return result.returncode
    except FileNotFoundError:
        if cmd and Path(cmd[0]).name == "go":
            # 兼容仅在 shell 初始化后才可见 Go 路径的环境。
            shell_cmd = " ".join(shlex.quote(arg) for arg in cmd)
            result = subprocess.run(["bash", "-lc", shell_cmd], cwd=location)
            return result.returncode
        raise


# build_rust 编译 native Rust 二进制（release）。
def build_rust() -> int:
    print("build rust native...")
    return run(["cargo", "build", "--release"], ROOT / "drivers/native")


# build_go 编译 bridge Go 二进制到 bin 目录。
def build_go() -> int:
    print("build go bridge...")
    (ROOT / "bin").mkdir(exist_ok=True)
    return run([resolve_go_bin(), "build", "-o", "../../bin/ghost-bridge"], ROOT / "core/bridge")


# ping 执行最小联通性验证：编译 native(debug) 后运行 bridge ping。
def ping() -> int:
    print("build rust debug...")
    if run(["cargo", "build"], ROOT / "drivers/native") != 0:
        return 1
    print("run bridge ping...")
    return run([resolve_go_bin(), "run", ".", "ping"], ROOT / "core/bridge")


# agent 运行 bridge agent 子命令，消息默认走最小问候语。
def agent() -> int:
    msg = " ".join(sys.argv[2:]) if len(sys.argv) > 2 else "Hello, what can you do?"
    print("run bridge agent...")
    return run([resolve_go_bin(), "run", ".", "agent", msg], ROOT / "core/bridge")


# init_web 初始化 Next.js Web 控制台脚手架。
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


# main 负责 task.py 命令分发。
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
