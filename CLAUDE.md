# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 当前项目状态

参见 [`PROJECT_PROGRESS.md`](PROJECT_PROGRESS.md) 了解最新进展。当前 MVP 骨架：
- Bridge 主流程可用（`core/bridge`）
- Web 控制台 MVP 可用（`apps/web`）
- CLI 基线可用（`apps/cli`）
- `drivers/native` 原子能力仍在完善中，尚未生产就绪

## 使命

Ghost-OS 不是传统远程桌面工具，而是 AI 驱动的数字孪生执行层。用户应能通过 Web/CLI 命令 AI，如同操作自己的双手控制远程机器。

## 核心理念

1. **极简主义**：优先使用原生、轻量、高性能的 Rust/Go 实现
2. **Bash 优先**：默认使用脚本化操作（Bash/PowerShell/Python），仅当 GUI 无法脚本化时才使用视觉回退
3. **无缝生态**：浏览器、CLI、后端是协调统一的系统
4. **简洁美观**：代码保持简洁和明确；UX 默认采用高对比度深色风格

## 包管理

1. Web 应用使用 **pnpm**（`apps/web`）
2. 使用 `pnpm install` 和 `pnpm run <script>` 管理 Web 依赖
3. 提交 `pnpm-lock.yaml`，不要引入 `package-lock.json` 或 `yarn.lock`

## 三层架构（严格边界）

1. **执行层**（`drivers/native`，Rust）
   - 无状态、原子操作
   - 处理截图、输入模拟、窗口树查询
   - 不得包含业务决策

2. **中心层**（`core/bridge`，Go）
   - 管理状态、协议路由、AI 编排、安全检查
   - 不得实现具体的 OS 系统调用

3. **感知层**（`apps/web`、`apps/cli`）
   - 交互与反馈
   - Web 渲染、浏览器结构访问、沉浸式 CLI 控制

## 核心/Bridge 内部结构（`core/bridge`）

- `main.go` – 入口，启动 HTTP 服务或 CLI 模式
- `server/` – HTTP 路由、中间件、SSE 流
- `session/` – 会话管理、持久化、历史记录
- `tools/` – 工具定义和执行分发
- `orchestration/` – 工作流引擎、Agent 循环、任务调度
- `runtime/` – 运行时执行客户端
- `transport/` – 消息总线和协议编解码
- `config/` – 配置加载和验证

测试：`cd core/bridge && go test ./...`，添加 `-v` 获取详细输出。运行单个测试：`go test -run TestName ./path/to/package`。

## Native Driver 结构（`drivers/native`）

- `src/screen/` – 截图捕获（X11、Wayland via PipeWire）
- `src/input/` – 键盘/鼠标模拟
- `src/window/` – 窗口列表和焦点管理

构建：`cargo build --release`（在 `drivers/native` 目录）
测试：`cargo test`

## Web 控制台结构（`apps/web`）

- Next.js 14 + App Router（`app/` 目录）
- `app/api/` – API 路由（tools、prompts、sessions）
- `components/` – React 组件（聊天、工作流编辑器、设置）
- `lib/` – 共享工具、API 客户端、i18n
- `hooks/` – 自定义 React Hooks

测试：`pnpm test`（Jest）。添加 `--watch` 进入交互模式。运行单个测试：`pnpm test -- -t "test description"`。

## 共享契约（`core/shared/schema.json`）

- 定义请求/响应的信封格式
- 类型生成：`python3 task.py gen-contracts` 从 schema 重新生成 TypeScript、Go、Rust、Kotlin 类型
- 修改 `schema.json` 后运行此命令，保持各层类型同步

## 决策优先级

始终按以下顺序选择实现路径：
1. **脚本优先**：优先使用本地脚本解决
2. **API/原生**：使用系统/浏览器原生 API
3. **视觉兜底**：截图-识别-点击作为最后手段

## 通信与契约规则

1. 组件通过标准化消息总线通信
2. 禁止跨层直接耦合
3. 每个操作必须可追踪（带 `trace_id` 的 trace 日志）
4. 跨进程负载必须严格遵循 `core/shared/schema.json`：
   - 请求：`{ "action": "string", "params": "object", "trace_id": "string", "request_id": "string?" }`
   - 响应：`{ "status": "success|error", "payload": "object", "error": "string", "request_id": "string?" }`

## 持久化 Native 子进程模式

设置 `GHOST_NATIVE_PERSISTENT=true` 可将 native driver 作为长生命周期的 framed 子进程运行，而非一次性执行。这能减少连续操作（如截图 → 点击 → 截图）的启动开销。`SCRIPT_EXEC` 工具仍使用隔离的 `--sandbox-worker` 模式。

## 关键环境变量

- `GHOST_NATIVE_PERSISTENT=true` – 启用持久化 native 子进程
- `GHOST_NATIVE_BINARY_PATH` – 覆盖 native 二进制路径（未设置时自动检测）
- `GO_BIN` – Go 可执行文件路径（回退到 `which go` 或常见安装位置）

## 构建与开发命令

主要入口：`python3 task.py`


## 仓库卫生

- 提交示例配置文件（如 `apps/web/.env.example`），不要提交真实的 `.env*` 本地文件
- 生成的协议产物和锁文件应被追踪；本地构建输出如 `.next/`、`target/`、`build/`、`coverage/`、`*.tsbuildinfo` 不提交

## 工程美学

1. 只写必要的代码
2. 明确优先于隐式；避免过度抽象
3. 保持极简、函数式风格
4. Web/终端输出默认深色模式
5. 和用户交流永远用中文
