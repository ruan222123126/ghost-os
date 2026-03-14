# Ghost-OS 当前项目概述

更新日期：2026-03-14  
分支：main（与 `origin/main` 同步）  
阶段：MVP 骨架（主链路可用，核心能力持续补齐）

## 项目快照

- `Ghost-OS` 已形成清晰的三层结构：Execution（`drivers/native`）、Central（`core/bridge`）、Perception（`apps/web`、`apps/cli`、`apps/android`）。
- `core/bridge` 是当前最成熟的部分，Agent、Session、Tool、Provider、SSE、配置持久化、`ask_human` 续跑与基础 Memory 增强链路已可用。
- `apps/web`、`apps/cli`、`apps/android` 均已有可用 MVP；Web Console 和 CLI 已能完成基础对话与联动。
- `drivers/native` 已具备截图、输入模拟、脚本执行、文件工具、浏览器查询等原子能力，但仍处于持续补齐阶段，不应默认视为生产完备。
- 三层边界、消息总线、`trace_id`、跨端 DTO 与 `core/shared/schema.json` 契约已基本统一。

## 当前重点结论

- Bridge 主链路可用，`core/bridge` 在当前环境下已通过 `go test ./...`。
- 浏览器发现测试与 RSS 抓取测试已移除对 `httptest` 本地监听的依赖，修复了 sandbox 中 `tcp6 [::1]:0` 监听失败导致的测试不稳定。
- Native 执行层正在向“原子动作 + 无业务决策”收口；Bridge 持续承担编排、状态与协议职责。
- Web Console 已完成一轮 API 分层与界面收口，但当前仍以 MVP 可用性为优先，不以视觉细节扩张为目标。

## 近期关键进展

### 2026-03-14

- 收口 RSS 源管理工具暴露面：
  - 将 `feed_subscribe`、`feed_list`、`feed_update`、`feed_unsubscribe` 合并为单个 `feed_manage`。
  - `rss_fetch` 继续独立保留，避免“管理订阅”和“读取 feed 内容”语义混杂。
  - 已同步更新 runtime 注册、selector metadata、默认 prompt 与相关测试。
- 收口 RSS 领域边界并压缩报告职责：
  - 将 `FeedStore`、`FeedSubscription`、normalize/sort/ID 生成逻辑迁出 `core/bridge/tools`，落到 `core/bridge/rss/subscriptions`。
  - `feed_manage` 工具现仅保留参数解析、订阅探测与领域 store 调用，不再承载订阅持久化规则。
  - 将 `core/bridge/rss/rss_report.go` 拆为 `report_service.go`、`report_prompt.go`、`report_markdown.go`、`report_dossier.go`，并补独立 `report_sources.go` 以满足文件尺寸约束。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./rss/subscriptions -timeout 60s` 已通过；其余 RSS 相关跨包测试受现存 `core/bridge/llm` 重复定义阻塞。
- 继续收口 `RSSInboxService`，拆成“薄 service + 纯领域流程”：
  - `core/bridge/rss/rss_inbox_service.go` 现仅保留依赖装配、默认注入和薄入口，不再承载轮询抓取 / 去重 / 分类细节。
  - RSS poll、briefing、report 协调分别下沉到独立 workflow，service 方法只做委派，领域流程只依赖显式接口而不是整块 service。
  - `rss_inbox_service.go`、`rss_briefing.go`、`report_service.go` 已按职责拆到多个小文件，当前相关文件均收敛到 300 行以内。
  - 当前环境下 `go test -C core/bridge ./... -timeout 60s` 已通过。
- 拆分 `core/bridge/llm` provider 适配层，但只下沉共享小原语：
  - 保留 `client.go` 总调度入口；新增共享 headers 构造、tool-call stream state 与 JSON/schema 小工具。
  - `openai`、`anthropic`、`codex` 现按请求映射、响应反解、stream 解析拆到独立文件，避免修改 Codex 时连带碰到其他 provider 的流式状态机。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./... -timeout 60s` 已通过。
- 修复 `core/bridge/tools` 在受限环境中的测试可移植性问题：
  - `browser_control` 发现测试改为注入内存 `RoundTripper`。
  - `rss_fetch` 测试改为注入内存 `http.Client`，移除 `httptest.NewTLSServer`。
  - `web_search` 测试改为注入内存 `RoundTripper`，移除 `httptest.NewServer`，规避 `tcp6 [::1]:0` 监听失败。
  - `web_search` provider 解析改为执行时决议，修复构造期编译回归。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./... -timeout 60s` 已通过。
- 扩展 Web Search 配置面：
  - `core/bridge` 新增 `web_search_tavily_api_key`、`web_search_exa_api_key` 持久化与运行态快照。
  - `web_search` 工具新增 Exa provider，并支持在工具参数中显式选择 Tavily / Exa。
  - `apps/web` Runtime 设置页分别保存 Tavily / Exa key，不再提供静态 provider 切换。
- 收口 Web Console 聊天与 bridge 代理的 feature 边界：
  - `MessageList` 拆到 `apps/web/components/message/*`，将复制按钮、ToolCard、附件列表、消息分发与空状态分离。
  - `useBridgeChat` 拆到 `apps/web/hooks/chat/*`，将历史加载、回复处理、运行控制、问答续跑拆成独立 hook。
  - `bridgeProxy` 拆到 `apps/web/lib/server/bridge/*`，将 auth header 决议、JSON body 处理、route handler 工厂与请求透传分离，并迁移原有测试。
- 修复 `core/bridge/runtime` memory augmentation 关闭语义与 memory DB 路径透传回归：
  - augmentation 关闭时不再触碰 SQLite。
  - `GHOST_MEMORY_PATH` 可正确覆盖默认 memory DB 路径。
- 将共享契约生成链路收口为“薄入口 + 分片 schema + 语言 emitter”结构，降低 `schema.json` 与 codegen 维护成本。
- 将仓库卫生纳入基线：
  - 收敛 `.gitignore`。
  - 新增仓库卫生校验脚本与 CI。
  - 清理误追踪的本地产物与示例环境文件。
- 将 `drivers/native` 继续收口为原子执行层：
  - 用 `screen/*`、`input/*` 模块替代旧巨型实现。
  - 将截图、裁剪、OCR、模板匹配、坐标换算留在 Native，业务编排保留在 Bridge。
  - 将 `sandbox/file_tools.rs` 拆为 `file_tools/mod.rs` + `bindings/*_py.rs` + `read_write.rs` + `search.rs` + `export.rs`，仅做 FFI 入口与原子文件能力的边界拆分，未重写内部行为。
  - 将 `codex_cli.rs` 收口为 `codex_cli/*` 目录模块，分离 action 分发、参数解析、sandbox 路径决议、进程跟踪、输出缓冲与 session 提取，降低 persistent CLI 改动脆弱性。
- 收口 `apps/cli` 命令层边界：
  - 将 `commands.rs` 拆为 `commands/*` 目录模块，分离命令解析、配置更新映射、本地副作用执行与终端渲染，保留 `repl` 侧兼容入口。
  - 为命令解析、执行结果映射、清屏行为与输出格式新增定向测试；当前环境下 `timeout 60 cargo test --manifest-path apps/cli/Cargo.toml -- --nocapture` 已通过。
- 完成 `apps/web` API 客户端分层重构，拆出 `agent/config/sessions/rss` 等领域模块，移除巨型 `lib/api.ts`。
- 完成 `core/bridge/tools` 第一批大文件拆分，重点收口 `screen_action` 与 `browser_control`。
- 落地第一阶段自动记忆增强：
  - 新增 `memorystore` 与 `memoryaug`。
  - runtime 统一装配 `memory_manage`、`memory_learned_list`、`memory_recall_debug`。
  - session prepare / commit 已接入 recall 与 learning 基线。
- 收紧自动记忆写入并提升 recall 性价比：
  - `memoryaug` 新增稳定写入策略：自动学习先做 durable signal 闸门，一次性任务与弱信号对话不再调用 extractor。
  - learned memory 新增 `memory_key`，同一槽位优先 refresh / supersede，减少近义重复记忆持续累积。
  - 自动学习默认只沉淀高价值 `profile/preference/workflow`，`fact` 需显式“记住”意图才允许自动进入 learned memory。
  - recall query 不再只用当前用户一句话，现会拼接少量近期 user/assistant 上下文再检索。
  - 已通过 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./memoryaug ./orchestration ./runtime ./tools -timeout 60s` 的相关定向回归。
- 将 learned memory 收口为已注册槽位：
  - `memoryaug` 新增 6 个已知槽位注册表：`reply_language`、`response_style`、`approval_style`、`package_manager`、`build_command`、`test_command`。
  - 自动学习仅接受已注册槽位，写入时固定 `memory_key` 语义，并将规范值写入 `metadata.value` 与 `slot_version`。
  - recall 新增槽位直召回，prompt 注入改为优先输出 `Memory slots` 结构化短块，其余自由文本记忆落到 `Other memory context`。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./memoryaug ./orchestration -timeout 60s` 已通过。
- 完成 `core/bridge/app` 编排层拆分：
  - `config`、`runtime`、`rss`、`tasks`、`orchestration`、`transport` 已独立成包。
  - `app` 仅保留启动、装配与薄入口。
- 拆分 `core/bridge/tasks` 调度器实现：
  - 将原 `task_scheduler.go` 按职责拆为 `scheduler.go`、`schedule_plan.go`、`registration.go`、`run_executor.go`。
  - 调度入口、计划计算、并发注册状态与执行结果持久化已解耦，后续调整调度算法不再需要同时穿插锁状态和落库副作用。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./tasks -timeout 60s` 已通过。
- 收紧与清理工具面：
  - Agent 默认暴露面以 `script_exec` 为主入口。
  - `memory_manage`、`browser_control`、`script_exec`、`send_file` 等工具的输出与元数据更稳定。

### 2026-03-13

- 完成 `script_exec` 主入口收口，逐步替代散落的执行类工具暴露。
- 新增 `set_project_root` 与动态 WorkingDir 支持，便于 Agent 在项目上下文内执行。
- 新增 `browser_control` 工具，支持 connect / launch / goto / click / type / evaluate / screenshot / close。
- 新增 `memory_manage` MVP，提供 SQLite 持久化与 URI 风格 CRUD / search / list。
- 核心提示词支持按目录与文件列表拼装，提示词装配更可维护。

### 2026-03-10

- 收紧流式事件、会话裁剪、配置读取、artifact 路径校验、execution 错误返回等运行时边界，减少“看似成功但状态不一致”的问题。
- 新增工具白名单模式与更严格的模型选择控制，后端和前端行为已对齐。
- 提升 persistent / one-shot native execution 的稳定性与可观测性，补充超时、回收、错误上下文与协议诊断。
- Web Console 完成一轮主界面与聊天区收口，移除多余栏位与重复视觉噪音，保持 MVP 聚焦。

### 2026-03-08

- 持续压缩 `core/bridge/tools` 与 `core/bridge/memory` 的目录和文件体积，推动“薄入口 + 内部实现”结构。
- 为 Bridge 增加定时任务基线，支持 CRUD、手动触发、日志、调度停止与运行取消。
- 启动并推进 memory truth-first / query cutover / projection / decision lineage 等探索性重构，为后续记忆系统演进打基础。
- `apps/web`、`apps/android` 完成一轮样式与主题统一收口；`drivers/native` 默认沙箱白名单也已放宽到常用工作目录与挂载盘。

## 当前可用能力

- 会话管理：创建、持久化、流式输出、结束语义、基础裁剪。
- Provider 调用：运行时配置、模型切换、流式响应。
- 工具链：`script_exec`、`read_and_summarize`、`rss_fetch`、`web_search`、`send_file`、`text_input`、`screen_action`、`browser_control`、`memory_manage`。
- 任务系统：基础定时执行、run-now、日志查询、系统动作入口。
- Web Console：会话列表、聊天主界面、配置面板、基础模型/Provider 选择。

## 已知约束

- `drivers/native` 仍未达到“所有原子能力都稳定完备”的状态，新增功能时仍应优先走脚本与 API，再考虑视觉回退。
- Memory 子系统处于持续演进阶段，当前以可用和可验证为先，不宜假设其内部模型已经稳定冻结。
- `PROJECT_PROGRESS.md` 现改为压缩版快照，不再保留逐次微调的完整流水账；如需追溯细粒度变更，应查看 `git log`。
