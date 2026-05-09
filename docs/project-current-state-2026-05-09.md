# Ghost-OS 当前项目现状导出

导出日期：2026-05-09  
当前分支：`feat/workflow-tool-card-runtime-vars`  
当前提交：`cd65886`  
工作区状态：导出前已有 `116` 个已修改文件、`17` 个未跟踪文件。

## 生成方式

本机未安装 `tree`，执行 `tree -a -L 4 -I ...` 时返回 `command not found`。本报告使用等价的 `find` 命令生成结构，并排除了常见本地产物。

```bash
find . \
  \( -name .git -o -name node_modules -o -name __pycache__ \
  -o -name .next -o -name .next-dev -o -name .next.bak.1772240201 \
  -o -name .pnpm-store -o -name target -o -name build \
  -o -name .gradle -o -name coverage -o -name dist \) \
  -prune -o -maxdepth 4 -print | sort
```

依赖文件读取命令：

```bash
rg --files -g 'package.json' -g 'pnpm-lock.yaml' -g 'go.mod' \
  -g 'go.sum' -g 'Cargo.toml' -g 'Cargo.lock' \
  -g 'build.gradle.kts' -g 'settings.gradle.kts' -g 'gradle.properties'
```

## 顶层结构

```text
.
├── .github/workflows/        # CI: android/cli/complexity/repo-hygiene/trinity
├── apps/
│   ├── android/              # Kotlin + Jetpack Compose Android 客户端
│   │   ├── app/src/main/java/dev/ghostos/android/
│   │   │   ├── model/
│   │   │   ├── network/
│   │   │   ├── store/
│   │   │   ├── ui/
│   │   │   ├── util/
│   │   │   └── viewmodel/
│   │   └── gradle/
│   ├── cli/                  # Rust 终端客户端
│   │   ├── src/
│   │   │   ├── commands/
│   │   │   └── repl/
│   │   └── tests/
│   └── web/                  # Next.js Web Console
│       ├── app/              # 页面与 API proxy routes
│       ├── components/       # chat/config/workflow/orchestration UI
│       ├── hooks/
│       └── lib/              # API parser、runtime、store、contract types
├── bin/                      # 本地构建产物，已在 .gitignore 中忽略
├── core/
│   ├── bridge/               # Go Central Layer
│   │   ├── agent/
│   │   ├── app/
│   │   ├── artifacts/
│   │   ├── config/
│   │   ├── context/
│   │   ├── execution/
│   │   ├── guiagent/
│   │   ├── llm/
│   │   ├── mode/
│   │   ├── orchestration/
│   │   ├── rss/
│   │   ├── runtime/
│   │   ├── session/
│   │   ├── skills/
│   │   ├── streaming/
│   │   ├── tasks/
│   │   ├── tools/
│   │   └── transport/
│   └── shared/
│       ├── contract_codegen/
│       ├── schema/
│       └── tests/
├── data/logs/                # 运行日志，内容已忽略
├── deploy/                   # systemd/autostart 模板
├── docs/
├── drivers/native/           # Rust Execution Layer
│   └── src/
│       ├── browser_query/
│       ├── codex_cli/
│       ├── file_actions/
│       ├── input/
│       ├── sandbox/
│       ├── screen/
│       └── script_exec/
├── scripts/
│   ├── autostart/
│   └── e2e/
├── AGENTS.md
├── PROJECT_PROGRESS.md
└── task.py
```

## 当前功能基线

- 项目定位：AI 驱动的 digital twin 执行层，不是传统远程桌面。
- 架构边界：Execution=`drivers/native`，Central=`core/bridge`，Perception=`apps/web`、`apps/cli`、`apps/android`。
- 当前阶段：MVP 稳定化，主开发中心是 `core/bridge`。
- Web/CLI：主链路可用，流式输出和用户图片输入已接通。
- Android：已接入会话和基础展示，但成熟度低于 Web/CLI。
- Native：截图、输入、脚本、窗口查询等原子能力可用，但仍非生产完备。
- Assistant 工具调用协议：`<t:ID>JSON</t>` + `[TOOL_TAG_RESULT]`。
- 共享契约：`core/shared/schema.json` 与 `core/shared/schema/defs/*`。

## 代码规模快照

按模块文件数：

```text
core/bridge      693
apps/web         436
drivers/native    79
apps/android      50
core/shared       28
apps/cli          24
```

按主要文件类型：

```text
go       474
go_test  215
ts       215
ts_test   98
rs        95
tsx       88
kt        28
py        16
sh         7
md         5
kt_test    3
```

超过 300 行的主要生产文件示例：

```text
core/bridge/orchestration/task_orchestration_runner.go      1047
apps/android/app/src/main/java/dev/ghostos/android/model/ApiModels.kt 858
core/bridge/orchestration/task_workflow_runner_nodes.go      585
apps/web/lib/i18n/messages/settings.ts                      500
apps/web/lib/i18n/messages/workflow.ts                      458
drivers/native/src/screen/match_engine.rs                   424
apps/web/lib/types.ts                                       421
core/bridge/orchestration/session_turn_draft_projector.go    391
core/bridge/tasks/types.go                                  376
apps/web/lib/workflow-editor/draftExpand.ts                 373
core/bridge/orchestration/task_orchestration_validation.go   341
core/bridge/orchestration/session_microcompact.go            334
core/bridge/orchestration/session_turn_preparer.go           330
core/bridge/config/config_runtime_resolve.go                 319
apps/web/components/workflow/workflowCanvasState.ts          308
apps/web/components/message/toolDetailCommon.ts              308
apps/web/components/config/OrchestrationSettingsSection.tsx  307
```

## 核心依赖

### Web Console

路径：`apps/web/package.json`，锁文件：`apps/web/pnpm-lock.yaml`，包管理器：`pnpm@10.30.3`。

- Runtime：`next@14.2.5`、`react@18.3.1`、`react-dom@18.3.1`
- UI/编辑器：`@tanstack/react-virtual@^3.13.23`、`@xyflow/react@^12.10.2`
- Markdown：`react-markdown@^10.1.0`、`remark-gfm@^4.0.1`
- Tooling：`typescript@5.5.4`、`tailwindcss@3.4.7`、`jest@29.7.0`、`ts-jest@29.2.5`、`eslint-config-next@14.2.5`
- Scripts：`dev`、`build`、`start`、`lint`、`test`

### Bridge

路径：`core/bridge/go.mod`，校验文件：`core/bridge/go.sum`。

- Go：`1.25.0`
- 直接依赖：`BurntSushi/toml@v1.5.0`、`robfig/cron/v3@v3.0.1`、`yaml.v3@v3.0.1`、`modernc.org/sqlite@v1.33.1`
- 间接依赖中包含：`google/uuid`、`gobwas/ws`、`gqlparser/v2`、`golang.org/x/sys`、`modernc.org/libc`

### Native Driver

路径：`drivers/native/Cargo.toml`，锁文件：`drivers/native/Cargo.lock`。

- Rust edition：`2024`
- Feature：`python-sandbox = ["dep:pyo3", "dep:reqwest"]`
- 主要依赖：`serde`、`serde_json`、`base64`、`xcap`、`dbus`、`pyo3`、`libc`、`glob`、`regex`、`html2text`、`reqwest`、`sha2`、`rayon`
- macOS/Windows 输入依赖：`enigo@0.1.2`

### CLI

路径：`apps/cli/Cargo.toml`，锁文件：`apps/cli/Cargo.lock`。

- Rust edition：`2024`
- 主要依赖：`anyhow`、`clap`、`colored`、`libc`、`reqwest`、`rustyline`、`serde`、`serde_json`

### Android

路径：`apps/android/build.gradle.kts`、`apps/android/app/build.gradle.kts`、`apps/android/settings.gradle.kts`。

- Android Gradle Plugin：`8.2.0`
- Kotlin：`1.9.20`
- compileSdk/targetSdk：`34`
- minSdk：`26`
- Java/Kotlin target：`17`
- Compose compiler：`1.5.4`
- 主要依赖：AndroidX Core、Lifecycle、Activity Compose、Compose BOM `2024.01.00`、Material3、Navigation Compose、OkHttp、Kotlinx Serialization、DataStore Preferences
- 测试依赖：JUnit、Kotlinx Coroutines Test、MockWebServer

## 共享契约与生成物

契约源：

```text
core/shared/schema.json
core/shared/schema/defs/agent_core.json
core/shared/schema/defs/agent_human.json
core/shared/schema/defs/base.json
core/shared/schema/defs/config_provider.json
core/shared/schema/defs/config_runtime.json
core/shared/schema/defs/execution.json
core/shared/schema/defs/session_content.json
core/shared/schema/defs/session_history.json
core/shared/schema/defs/streaming_events.json
core/shared/schema/defs/tasks.json
```

生成物：

```text
apps/cli/src/envelope_generated.rs
apps/web/lib/envelope.generated.ts
core/bridge/orchestration/envelope_generated.go
```

生成入口：`python3 task.py gen_contracts`。

## 常用工程入口

`task.py` 统一封装了主要命令：

```text
build, ping, agent, serve,
web_dev, web_build, web_test, web_lint,
repo_hygiene, gen_contracts,
build_cli, run_cli, check_cli, install_cli
```

Native 构建默认使用 `python-sandbox` feature；`ping/agent/serve` 会优先编译 debug native，并通过 `GHOST_NATIVE_BINARY_PATH_OVERRIDE` 传给 bridge。

## 当前痛点

1. `core/bridge` 体量最大，`orchestration`、`config`、`tools` 三个目录合计文件数很高，Central Layer 的协调成本明显高于其他层。
2. `core/bridge/orchestration` 同时承载 service、task、session、runtime adapter、workflow、orchestration runner、tool schema 等职责，职责边界容易继续膨胀。
3. 多个生产文件超过仓库规则中的 300 行文件限制，其中 orchestration 和 Web workflow/config 相关文件最集中。
4. 共享契约已经有 schema 和 Go/TS/Rust 生成物，但 Android 侧仍有较大的 `ApiModels.kt`，跨端契约一致性需要特别留意。
5. Web 侧状态分散在 `components`、`hooks`、`lib`，workflow/orchestration/chat streaming/tool preview 的 UI 状态与解析逻辑交织较多。
6. 根目录没有 `README.md`，入门信息主要分散在 `AGENTS.md`、`PROJECT_PROGRESS.md`、`docs/*`、`task.py`；`PROJECT_PROGRESS.md` 已经明显超过其自身声明的摘要行数。
7. 当前工作区不是干净基线，已有大量未提交变更；继续修改前需要区分既有改动和本次改动。
8. 本地产物目录实际存在较多，例如 `.next*`、`target`、`.gradle`、`.pnpm-store`、`bin`、`__pycache__`，虽然已被 `.gitignore` 覆盖，但导出和分析时必须排除。
9. Native 与 Android 在项目进度文档中都明确低于 Web/CLI 成熟度，涉及 GUI 执行、截图链路、移动端体验时风险更高。
10. Go、Rust、pnpm、Gradle 分别管理依赖，没有根级统一 workspace manifest；跨层验证主要依赖 `task.py` 和 CI workflow。
