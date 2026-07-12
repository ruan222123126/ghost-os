# Ghost-OS

Ghost-OS 是一个 AI 驱动的数字孪生执行层。它通过统一消息总线连接 Bridge、原子化本机驱动以及 Web、CLI、Android 客户端，让模型的操作可编排、可追踪、可恢复，而不是把系统简化成传统远程桌面。

## 核心架构

```text
apps/web · apps/cli · apps/android
              │
              ▼
     core/bridge（状态、路由、编排、安全）
              │
              ▼
 drivers/native（无状态原子执行）
```

- `core/bridge`：Go 编写的协议入口、会话与任务编排层。
- `drivers/native`：Rust 编写的本机原子能力，不承载业务决策。
- `apps/web`：Next.js Web 控制台。
- `apps/cli`：Rust 终端客户端。
- `apps/android`：React + Tauri 2 Android 客户端。
- `core/signaling`：移动端 WebRTC 信令服务。
- `core/shared`：跨进程契约、Schema 与类型生成器。

所有跨进程操作都使用统一 envelope，并携带 `trace_id`：

```json
{
  "action": "string",
  "params": {},
  "trace_id": "string"
}
```

响应格式：

```json
{
  "status": "success",
  "payload": {},
  "error": ""
}
```

## 环境要求

- Python 3.11+
- Go 1.25+
- Rust stable（edition 2024）
- Node.js 20+
- pnpm 10+
- Android 开发需要 Android SDK、NDK、JDK 与 Tauri 2 工具链

Linux 上构建 `drivers/native` 还需要项目所用截图、桌面总线及 Python 绑定对应的系统开发库。

## 快速开始

复制配置模板并填写 Provider、模型和 API Token：

```bash
mkdir -p ~/.ghost-os
cp docs/config.example.toml ~/.ghost-os/config.toml
```

安装 Web 依赖：

```bash
pnpm --dir apps/web install --frozen-lockfile
```

启动 Bridge：

```bash
python3 task.py serve
```

默认监听 `127.0.0.1:8080`。另开终端启动 Web：

```bash
python3 task.py web-dev
```

默认访问地址为 `http://127.0.0.1:3000`。

## 常用命令

```bash
python3 task.py build             # 构建 native 与 bridge 到 bin/
python3 task.py ping              # 最小 Bridge/Native 联通检查
python3 task.py agent "你好"      # 直接运行一次 Agent 请求
python3 task.py serve 8080        # 启动 Bridge HTTP 服务
python3 task.py web-test          # Web 测试
python3 task.py web-lint          # Web lint
python3 task.py verify-contracts  # 验证生成契约与 Schema 同步
python3 task.py check-layers      # 验证架构边界
python3 task.py repo-hygiene      # 检查仓库污染
python3 task.py check-cli         # CLI 编译检查
```

CLI 与 Android 的独立说明见：

- [CLI 使用说明](apps/cli/README.md)
- [Android 开发说明](apps/android/README.md)

## 配置与运行数据

默认主配置路径为 `~/.ghost-os/config.toml`，完整字段示例见 [`docs/config.example.toml`](docs/config.example.toml)。请勿把真实 API Key、Token 或本地设备凭据提交到仓库。

常用运行数据默认位于 `~/.ghost-os/`：

- `sessions/`：会话数据
- `tasks/`：任务定义与状态
- `mobile-devices.json`：移动设备凭据（启用移动端时）

## 文档

- [systemd 用户级自启动](docs/systemd-autostart.md)
- [移动端公网接入](docs/mobile-integration.md)
- [编排地图](docs/orchestration-map.md)
- [阶段 0 依赖图](docs/orchestration-stage0-dependency-graph.md)
- [配置公共 API](docs/config-public-api-m0.md)
- [Sandbox/Driver 审计](docs/audit-sandbox-in-driver.md)

## 验证顺序

提交前优先运行与改动相关的测试，再执行契约、架构和仓库检查：

```bash
python3 task.py verify-contracts
python3 task.py check-layers
python3 task.py repo-hygiene
```

Web 项目只使用 `pnpm`，请保留 `pnpm-lock.yaml`，不要新增 `package-lock.json` 或 `yarn.lock`。
