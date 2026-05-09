from __future__ import annotations

import os
import shlex
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
NATIVE_BINARY_NAME = "native.exe" if os.name == "nt" else "native"
NATIVE_REQUIRED_FEATURE = "python-sandbox"


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
def run(cmd: list[str], cwd: Path | None = None, env: dict[str, str] | None = None) -> int:
    location = cwd if cwd else ROOT
    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)
    try:
        result = subprocess.run(cmd, cwd=location, env=merged_env)
        return result.returncode
    except FileNotFoundError:
        if cmd and Path(cmd[0]).name == "go":
            # 兼容仅在 shell 初始化后才可见 Go 路径的环境。
            shell_cmd = " ".join(shlex.quote(arg) for arg in cmd)
            result = subprocess.run(["bash", "-lc", shell_cmd], cwd=location, env=merged_env)
            return result.returncode
        raise


# build_rust 编译 native Rust 二进制（release）。
def build_rust() -> int:
    print("build rust native...")
    return run(
        ["cargo", "build", "--release", "--features", NATIVE_REQUIRED_FEATURE],
        ROOT / "drivers/native",
    )


def build_rust_debug() -> int:
    print("build rust debug...")
    return run(
        ["cargo", "build", "--features", NATIVE_REQUIRED_FEATURE],
        ROOT / "drivers/native",
    )


# build_go 编译 bridge Go 二进制到 bin 目录。
def build_go() -> int:
    print("build go bridge...")
    (ROOT / "bin").mkdir(exist_ok=True)
    return run([resolve_go_bin(), "build", "-o", "../../bin/ghost-bridge"], ROOT / "core/bridge")


def native_debug_binary() -> Path:
    return ROOT / "drivers/native/target/debug" / NATIVE_BINARY_NAME


def native_release_binary() -> Path:
    return ROOT / "drivers/native/target/release" / NATIVE_BINARY_NAME


def stage_native_binary(source: Path) -> int:
    if not source.is_file():
        print(f"missing native binary: {source}", file=sys.stderr)
        return 1
    target = ROOT / "bin" / NATIVE_BINARY_NAME
    shutil.copy2(source, target)
    return 0


def bridge_native_env(native_path: Path) -> dict[str, str]:
    if (
        os.environ.get("GHOST_NATIVE_BINARY_PATH_OVERRIDE")
        or os.environ.get("GHOST_NATIVE_BIN_OVERRIDE")
        or os.environ.get("GHOST_NATIVE_BINARY_PATH")
        or os.environ.get("GHOST_NATIVE_BIN")
    ):
        return {}
    return {"GHOST_NATIVE_BINARY_PATH_OVERRIDE": str(native_path)}


def run_bridge(command: list[str], require_native_debug: bool = False) -> int:
    env: dict[str, str] | None = None
    if require_native_debug:
        if build_rust_debug() != 0:
            return 1
        env = bridge_native_env(native_debug_binary())
    return run([resolve_go_bin(), "run", "."] + command, ROOT / "core/bridge", env=env)


def build() -> int:
    if build_rust() != 0:
        return 1
    if build_go() != 0:
        return 1
    return stage_native_binary(native_release_binary())


# ping 执行最小联通性验证：编译 native(debug) 后运行 bridge ping。
def ping() -> int:
    print("run bridge ping...")
    return run_bridge(["ping"], require_native_debug=True)


# agent 运行 bridge agent 子命令，消息默认走最小问候语。
def agent() -> int:
    msg = " ".join(sys.argv[2:]) if len(sys.argv) > 2 else "Hello, what can you do?"
    print("run bridge agent...")
    return run_bridge(["agent", msg], require_native_debug=True)


# serve 启动 bridge HTTP 服务，默认监听 8080。
def serve() -> int:
    port = sys.argv[2] if len(sys.argv) > 2 else "8080"
    print(f"run bridge http server on :{port}...")
    return run_bridge(["serve", port], require_native_debug=True)


# web_dev 启动 Next.js Web 开发服务。
def web_dev() -> int:
    print("run web dev server...")
    return run(["pnpm", "--dir", "apps/web", "dev"], ROOT)


# web_build 构建 Next.js Web 应用（production）。
def web_build() -> int:
    print("build web app...")
    return run(["pnpm", "--dir", "apps/web", "build"], ROOT)


# web_test 执行 Web 测试。
def web_test() -> int:
    print("test web app...")
    return run(["pnpm", "--dir", "apps/web", "test"], ROOT)


# web_lint 执行 Web lint。
def web_lint() -> int:
    print("lint web app...")
    return run(["pnpm", "--dir", "apps/web", "lint"], ROOT)


# repo_hygiene 检查被追踪文件中是否混入本地产物或本地配置。
def repo_hygiene() -> int:
    print("check repo hygiene...")
    return run(["bash", "scripts/check_repo_hygiene.sh"], ROOT)


# gen_contracts 从 core/shared/schema.json 生成 Go/TS/Rust/Kotlin 契约类型。
def gen_contracts() -> int:
    print("generate shared contract types...")
    return run(["python3", "core/shared/generate_envelope_types.py"], ROOT)


# build_cli 编译 Rust CLI 二进制（release）。
def build_cli() -> int:
    print("build rust cli...")
    return run(["cargo", "build", "--release"], ROOT / "apps/cli")


# run_cli 启动 Rust CLI（默认交互模式）。
def run_cli() -> int:
    print("run rust cli...")
    return run(["cargo", "run"], ROOT / "apps/cli")


# check_cli 执行 Rust CLI 最小编译检查。
def check_cli() -> int:
    print("check rust cli...")
    return run(["cargo", "check"], ROOT / "apps/cli")


# install_cli 安装 Rust CLI 到 cargo bin。
def install_cli() -> int:
    print("install rust cli...")
    return run(["cargo", "install", "--path", "apps/cli"], ROOT)


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
        return build()
    if action == "ping":
        return ping()
    if action == "agent":
        return agent()
    if action == "serve":
        return serve()
    if action == "web-dev":
        return web_dev()
    if action == "web-build":
        return web_build()
    if action == "web-test":
        return web_test()
    if action == "web-lint":
        return web_lint()
    if action == "repo-hygiene":
        return repo_hygiene()
    if action == "gen-contracts":
        return gen_contracts()
    if action == "build-cli":
        return build_cli()
    if action == "run-cli":
        return run_cli()
    if action == "check-cli":
        return check_cli()
    if action == "install-cli":
        return install_cli()
    if action == "init-web":
        return init_web()

    print(
        "usage: python task.py [build | ping | agent | serve | web-dev | web-build | web-test | web-lint | repo-hygiene | gen-contracts | build-cli | run-cli | check-cli | install-cli | init-web]"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
