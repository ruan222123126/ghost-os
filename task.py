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
CONTRACT_PATHS = (
    "core/shared/schema.json",
    "core/bridge/orchestration/envelope_generated.go",
    "apps/web/lib/envelope.generated.ts",
    "apps/cli/src/envelope_generated.rs",
    "apps/android/app/src/main/java/dev/ghostos/android/model/AgentModels.kt",
    "apps/android/app/src/main/java/dev/ghostos/android/model/ApiModels.kt",
    "apps/android/app/src/main/java/dev/ghostos/android/model/ConfigModels.kt",
    "apps/android/app/src/main/java/dev/ghostos/android/model/OrchestrationModels.kt",
    "apps/android/app/src/main/java/dev/ghostos/android/model/SessionModels.kt",
    "apps/android/app/src/main/java/dev/ghostos/android/model/StreamingModels.kt",
    "apps/android/app/src/main/java/dev/ghostos/android/model/TaskModels.kt",
    "apps/android/app/src/main/java/dev/ghostos/android/model/WorkflowModels.kt",
)


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


def resolve_node_bin() -> str:
    configured = Path(os.environ["NODE_BIN_PATH"]) if "NODE_BIN_PATH" in os.environ else None
    if configured and configured.is_file():
        return str(configured)

    found = shutil.which("node")
    if found:
        return found

    candidates = sorted((Path.home() / ".nvm/versions/node").glob("*/bin/node"), reverse=True)
    candidates.extend([Path("/usr/local/bin/node"), Path("/usr/bin/node")])
    for candidate in candidates:
        if candidate.is_file():
            return str(candidate)

    return "node"


def resolve_tool_bin(env_name: str, names: list[str], candidates: list[Path]) -> str:
    configured = Path(os.environ[env_name]) if env_name in os.environ else None
    if configured and configured.is_file():
        return str(configured)

    for name in names:
        found = shutil.which(name)
        if found:
            return found

    for candidate in candidates:
        if candidate.is_file():
            return str(candidate)

    return names[0]


def resolve_golangci_lint_bin() -> str:
    return resolve_tool_bin(
        "GOLANGCI_LINT_BIN",
        ["golangci-lint"],
        [Path.home() / "go/bin/golangci-lint"],
    )


def clear_web_dev_dist() -> None:
    shutil.rmtree(ROOT / "apps/web/.next-dev", ignore_errors=True)


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
    next_entry = ROOT / "apps/web/node_modules/next/dist/bin/next"
    if not next_entry.is_file():
        print(f"missing web runtime: {next_entry}; run pnpm --dir apps/web install --frozen-lockfile", file=sys.stderr)
        return 1
    clear_web_dev_dist()
    host = os.environ.get("GHOST_WEB_HOST", "127.0.0.1").strip() or "127.0.0.1"
    port = os.environ.get("GHOST_WEB_PORT", "3000").strip() or "3000"
    node_bin = resolve_node_bin()
    if node_bin == "node":
        print("node binary not found; set NODE_BIN_PATH or install node in PATH/system-wide", file=sys.stderr)
        return 1
    env = {"NODE_ENV": "development"}
    return run(
        [node_bin, str(next_entry), "dev", "--hostname", host, "--port", port],
        ROOT / "apps/web",
        env=env,
    )


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


def web_knip() -> int:
    print("scan unused web files and exports...")
    return run(["pnpm", "--dir", "apps/web", "knip"], ROOT)


def go_tidy() -> int:
    print("tidy go bridge module...")
    return run([resolve_go_bin(), "mod", "tidy"], ROOT / "core/bridge")


def go_lint() -> int:
    print("lint go bridge...")
    return run([resolve_golangci_lint_bin(), "run"], ROOT / "core/bridge")


def rust_native_clippy() -> int:
    print("clippy rust native...")
    return run(
        [
            "cargo",
            "clippy",
            "--all-targets",
            "--features",
            NATIVE_REQUIRED_FEATURE,
            "--",
            "-D",
            "warnings",
        ],
        ROOT / "drivers/native",
    )


def rust_cli_clippy() -> int:
    print("clippy rust cli...")
    return run(["cargo", "clippy", "--all-targets", "--", "-D", "warnings"], ROOT / "apps/cli")


def rust_clippy() -> int:
    if rust_native_clippy() != 0:
        return 1
    return rust_cli_clippy()


def rust_native_udeps() -> int:
    print("scan unused native rust dependencies...")
    return run(
        ["cargo", "+nightly", "udeps", "--all-targets", "--features", NATIVE_REQUIRED_FEATURE],
        ROOT / "drivers/native",
    )


def rust_cli_udeps() -> int:
    print("scan unused cli rust dependencies...")
    return run(["cargo", "+nightly", "udeps", "--all-targets"], ROOT / "apps/cli")


def rust_udeps() -> int:
    if rust_native_udeps() != 0:
        return 1
    return rust_cli_udeps()


def scavenge() -> int:
    for step in (go_tidy, go_lint, rust_clippy, rust_udeps, web_knip):
        rc = step()
        if rc != 0:
            return rc
    return 0


# repo_hygiene 检查被追踪文件中是否混入本地产物或本地配置。
def repo_hygiene() -> int:
    print("check repo hygiene...")
    return run(["bash", "scripts/check_repo_hygiene.sh"], ROOT)


def check_layers() -> int:
    print("check layer boundaries...", flush=True)
    return run(["bash", "scripts/check-layers.sh"], ROOT)


# gen_contracts 从 core/shared/schema.json 生成 Go/TS/Rust/Kotlin 契约类型。
def gen_contracts() -> int:
    print("generate shared contract types...", flush=True)
    return run(["python3", "core/shared/generate_envelope_types.py"], ROOT)


def verify_contracts() -> int:
    rc = gen_contracts()
    if rc != 0:
        return rc
    return run(["git", "diff", "--exit-code", "--", *CONTRACT_PATHS], ROOT)


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
    if action == "web-knip":
        return web_knip()
    if action == "go-tidy":
        return go_tidy()
    if action == "go-lint":
        return go_lint()
    if action == "rust-clippy":
        return rust_clippy()
    if action == "rust-udeps":
        return rust_udeps()
    if action == "scavenge":
        return scavenge()
    if action == "repo-hygiene":
        return repo_hygiene()
    if action == "check-layers":
        return check_layers()
    if action == "gen-contracts":
        return gen_contracts()
    if action == "verify-contracts":
        return verify_contracts()
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
        "usage: python task.py [build | ping | agent | serve | web-dev | web-build | web-test | web-lint | web-knip | go-tidy | go-lint | rust-clippy | rust-udeps | scavenge | repo-hygiene | check-layers | gen-contracts | verify-contracts | build-cli | run-cli | check-cli | install-cli | init-web]"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
