# Ghost-OS 当前项目概述

更新日期：2026-03-07  
分支：main（与 `origin/main` 同步）  
阶段：MVP 骨架（主链路可用，核心能力持续补齐）

## 项目状态

- `Ghost-OS` 当前已形成清晰的三层架构：Execution、Central、Perception，整体目标是构建一个可通过 AI 驱动的数字分身执行层。
- **Execution Layer**（`drivers/native`）已具备基础原子能力与脚本执行能力，可承接截图、输入模拟、文件处理、脚本执行等底层动作。
- **Central Layer**（`core/bridge`）是当前最成熟的部分，已打通 Agent、Session、Tool、Provider、配置管理、会话持久化、流式响应、`ask_human` 续跑与 Memory 基线。
- **Perception Layer**（`apps/web`、`apps/cli`、`apps/android`）均已有可用 MVP，Web Console、CLI 与 Android 客户端能够完成基础交互与联动。

## 当前基础能力

- 三层边界、消息总线契约、`trace_id` 追踪链路、会话结束语义与跨端 DTO 已基本统一。
- Native 执行链路已具备文件工具、脚本沙箱、persistent mode 与浏览器查询相关基础模块，可与 Bridge 形成稳定调用闭环。
- Bridge 主流程已经可用，支持 Provider 调用、Tool 路由、Session 管理、停止控制、SSE 事件流与配置文件持久化。
- Memory 已从基础读写扩展到自动召回、Graph Memory、Decision Memory 与 Truth Schema shadow write，为后续上下文增强与经验沉淀提供基础。

## 当前工程状态

- Web、CLI、Android 与 Bridge 的接口契约正在持续收口，跨端行为与数据结构相比早期版本已明显稳定。
- `core/bridge/app`、`core/bridge/tools`、`core/bridge/memory` 等核心模块已完成多轮拆分整理，职责边界与可维护性持续提升。
- 近期工作重点以“收口、统一、拆分、稳定”为主，说明项目已经从单点功能搭建阶段进入到系统化整理与增强阶段。
- `2026-03-07`：将 `core/bridge/agent/loop_test.go` 拆为 `loop_helpers_test.go` / `loop_run_test.go` / `loop_tool_test.go` / `loop_stream_test.go`，并将 `apps/web/lib/api.test.ts` 拆为 `api_agent.test.ts` / `api_config.test.ts` / `api_sessions.test.ts` + `api.test.helpers.ts`；已通过 `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go -C core/bridge test ./agent` 与 `pnpm --dir apps/web test -- --runTestsByPath lib/api_agent.test.ts lib/api_config.test.ts lib/api_sessions.test.ts` 验证。
- `2026-03-07`：继续收口测试结构语义：将 `core/bridge/app/transport_test_helpers_test.go`、`core/bridge/app/session_agent_runner_helpers_test.go`、`core/bridge/memory/memory_helpers_test.go` 统一迁入 `*_test.go` 语义域；把 `core/bridge/app` 的 bus / middleware / SSE 测试进一步按职责拆开；并将 Web API client 测试文件名对齐为 `apps/web/lib/api.agent.test.ts` / `apps/web/lib/api.config.test.ts` / `apps/web/lib/api.sessions.test.ts`。`pnpm --dir apps/web test -- lib/api.agent.test.ts lib/api.config.test.ts lib/api.sessions.test.ts` 已通过；`GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go -C core/bridge test ./app ./memory` 仍被当前工作树中的 `core/bridge/memory` 接口漂移阻塞。
- `2026-03-07`：继续整理 `core/bridge/agent` 测试夹具：在 `loop_helpers_test.go` 中收口 `fakeCompleter` / `fakeToolCatalog` / `fakeTool` 的共享构造器与响应 builder（如 stop/tool_call/awaiting_human），并把 `loop_run_test.go`、`loop_stream_test.go`、`loop_tool_test.go` 的重复 fake/stub 装配改为复用 helper，降低单测样板噪音；已再次通过 `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go -C core/bridge test ./agent` 验证。
- `2026-03-07`：记忆层已接入 deterministic intent planner、truth-object vector sidecar 与 shadow recall：`MemoryQuery` / `MemoryQueryResult` 新增 `intent_plan`、`vector_hits`、`shadow_report` 调试视图，truth shadow append/upsert 后会 best-effort 增量同步向量索引，并支持从 truth snapshot 全量重建；`QueryResultWithScope` 会在 live recall 完成后并行跑 planner/vector 仅做 debug/metrics，`BuildContextWindow` 与 Entries 排序保持冻结不变，`GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go -C core/bridge test ./memory ./app` 已通过。
- `2026-03-07`：为 `apps/web` 增加暗色 `--ui-*` token 层与 `ui-*` 视觉原语，保持现有 `app.*` 色板不变；`ConfigPanel`、`ModelSelector`、`ChatInput`、`QuestionInput`、`MessageList`、`SessionSidebar`、`SessionItem`、`SessionList`、`app/page.tsx`、`app/error.tsx` 已切换到复用面板 / 输入 / 按钮 / hint / scrollbar 皮肤。`pnpm --dir apps/web test` 与顺序执行的 `pnpm --dir apps/web build` 已通过；并发执行时曾短暂触发 `/api/bus` collect page data 失败，顺序重跑后稳定通过。
- `2026-03-07`：启动 Web Chat Composer 第一阶段迁移：新增 `apps/web/components/ChatComposer.tsx`，以受控 React 组件重写 NextAI 风格的发送框外壳；`apps/web/components/ChatInput.tsx` 现改为仅承接业务 hint / draft / submit 状态，并通过 `apps/web/app/globals.css` 新增 `ui-composer-*` 样式实现内嵌发送按钮、底部工具区预留与 meta 行。当前仍只接现有 `sendMessage` 流程，尚未接入 stop / toolbar 业务项 / 附件能力。
- `2026-03-07`：`ask_human` 契约升级为可选单选/多选结构化提问：Bridge schema/DTO 新增 `selection_mode` + `options`，要求最后一个选项支持自定义输入；Web Console 与 CLI 已支持选项选择、自定义补充与取消提问（取消后结束当前会话），历史投影也会保留结构化问题信息。已通过 `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go -C core/bridge test ./app ./agent ./session && go -C core/bridge test ./tools -run 'TestAskHuman'`、`cargo test --manifest-path apps/cli/Cargo.toml`、`pnpm --dir apps/web test -- --runInBand --runTestsByPath lib/api.agent.test.ts lib/api.sessions.test.ts lib/chatMessages.test.ts` 与 `pnpm --dir apps/web exec tsc --noEmit` 验证。
- `2026-03-07`：记忆层已接入 Week 3 的 truth live read + hybrid rerank：新增 `truth_query.go` / `truth_index.go` / `truth_hit.go` / `memory_candidate.go` / `memory_verify.go` / `memory_rerank.go`，`TruthWriter` sidecar 升级为 fanout，`QueryResultWithScope` 现可在灰度开关下把 hot/warm/cold/markdown/decision/graph/truth/vector 统一成 candidate，完成跨层去重、support/conflict 聚合、confidence 校准与 hybrid 排序；`MemoryEntry` / `MemoryQueryResult` 新增 `freshness` / `evidence_count` / `source_refs` / `truth_status` / `rerank_score` / `truth_hits` / `rerank_report`，`BuildContextWindow` 仅注入高置信、非冲突结果。已通过 `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go -C core/bridge test ./memory` 与 `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go -C core/bridge test ./app ./memory` 验证。
- `2026-03-07`：记忆层已接入 Week 4 recipe reuse 闭环：在 `core/bridge/memory` 新增 recipe 选中 / advisory 注入 / `recipe_run` 执行跟踪 / 回合结束反馈回写 / 历史 backfill runner / 灰度默认与 kill switch，普通 hybrid rerank 仍保留为低置信回退；并将 `core/bridge/app` / `docs/config.example.toml` 补齐 `memory_decision_recipe_*` 配置透传。已通过 `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go -C core/bridge test ./memory ./app` 验证。
- `2026-03-07`：cold memory 已完成 step 1 的 ledger 迁移骨架：`core/bridge/memory` 新增 `LegacyColdStore` + `ColdMemory` façade、append-only `ledger` event schema/segment/manifest/replay/backfill/checkpoint、兼容 `Retrieve` / `ListSessions` / `ListArchives` 的 ledger replay、shadow compare 指标，以及 `core/bridge/app/session_turn_committer.go` 的每轮 dual-write；`core/bridge/app` 已补齐 `memory_ledger_*` 配置透传。已通过 `GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go -C core/bridge test ./memory ./app` 验证；`go -C core/bridge test ./memory ./memory/graph ./memory/decision ./app` 额外尝试时，`./memory/graph` 与 `./memory/decision` 子包仍受现有工作树中的包边界漂移影响未通过，本次未顺手修复。
