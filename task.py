import os, sys


def build_rust():
    print("🦀 编译 Rust 底层驱动...")
    os.system("cd drivers/native && cargo build --release")


def build_go():
    print("🐹 编译 Go 逻辑中枢...")
    os.system("cd core/bridge && go build -o ../../bin/ghost-bridge")


def init_web():
    print("🌐 初始化 Web 控制台 (Next.js)...")
    os.system("npx create-next-app@latest apps/web --typescript --tailwind --eslint")


if __name__ == "__main__":
    action = sys.argv[1] if len(sys.argv) > 1 else "help"
    if action == "build":
        build_rust()
        build_go()
    elif action == "init-web":
        init_web()
    else:
        print("用法: python task.py [build | init-web]")
