# Ghost-OS Progress Snapshot

更新日期：2026-03-06  
分支：main（与 origin/main 同步）  
阶段：MVP 骨架（主链路可用，核心能力持续补齐）

## 1) 三层状态（Trinity）

| 层 | 当前状态 | 完成度 |
|---|---|---|
| Execution (`drivers/native`) | 原子链路可用，浏览器与系统能力仍在补齐 | 20% |
| Central (`core/bridge`) | Agent/Session/Tool 主流程稳定，记忆能力已有 MVP 基线 | 72% |
| Perception (`apps/web`, `apps/cli`) | Web Console 与 CLI 可用，基础交互已贯通 | 40% |

## 2) 已完成（概要）

- 三层边界、消息契约与 `trace_id` 可追踪链路已稳定。
- Bridge 主流程已打通：Provider 调用、Tool 路由、会话持久化。
- 核心工具链路可用：`web_search`、`browser_action`、`ask_human`（含续跑闭环）。
- 会话终止语义已结构化：仅完整会话结束时使用 `{"signal":"END_SESSION","message":"..."}`，普通回合结束仍允许自然文本回复；契约已下沉到 `core/shared/schema.json` 并通过 `AGENT_SEND` 的 `session_ended/session_end` 字段对外表达。
- Native 执行层完成一轮职责拆分与语义修正，降低耦合并减少误判风险。
- 记忆系统完成 MVP 形态：L1/L2/L3 基础读写、查询与归档链路已接入。
- 记忆系统已完成增强迭代：L2 新增 `importance/expires_at` 与可配置 TTL，支持自动召回上下文注入 Agent 回合。
- L3 已新增 Markdown 节点（`ColdBaseDir/markdown/nodes`）与 YAML frontmatter 元数据，支持统一查询入口按需纳入 Markdown 命中。
- 已落地后台演化基础链路（Dreaming）：L2 低重要度旧条目可聚合沉淀为 Markdown 节点并记录结构化演化日志/指标。
- 截图与多模态输入链路已贯通，避免大体积 payload 直接进入文本历史。
- 核心模块可读性与回归保障已增强（注释、结构整理、基础回归脚本）。
- 已清理 Memory 模块当前无调用函数（manager/cold/warm/markdown），并为占位字段补充 `TODO/Deprecated` 说明，降低误导性。
- 已打通“回合新增消息 -> L2 warm”写入链路：会话持久化成功后同步写入 warm（带稳定 `session_id`/消息序号 ID 元数据），自动召回可直接利用近期真实会话内容。
- 已完成 Agent Phase 1 SSE 粗粒度事件流：新增 `/api/agent/stream`，贯通 `run_started / tool_call_started / tool_call_finished / awaiting_human / message / done / error` 事件；现有 `/api/agent` 同步接口保持不变。
- 已完成 Agent Phase 2 token 级流式基础能力：LLM 层新增 `CompleteStream` / `LLMDelta` 抽象，OpenAI 与 Anthropic 已支持文本与 tool-call delta 流式解析，并通过 `completion_delta` 事件桥接到 Agent SSE。
- 已完成 Agent Phase 3 stop 与并发保护基础能力：app 层新增 inflight run registry，已有会话支持单 session 串行执行保护，并可通过 `AGENT_STOP` 按 `session_id` 或 `trace_id` 取消正在运行的任务。
- 已修复共享 `MemoryManager` 的会话热态串味风险：自动召回与 L1 查询改为显式传入 session scope，不再依赖全局可变的 `hotSessionID` / hot history。
- 已补齐 Dreaming 后台演化生命周期：`MemoryManager` 新增优雅停止能力，bridge 进程在 `SIGINT` / `SIGTERM` 下会停止后台 ticker 并等待协程退出。
- 已补回 `core/bridge/context.Builder.BuildRequest`，恢复消息/工具定义深拷贝组装逻辑，修复 `context` 包测试编译失败导致的 `trinity-check` Bridge (Go) 红灯。

## 3) 主要短板（概要）

- `drivers/native` 仍非生产就绪，`BROWSER_QUERY` 在非 CDP 场景能力不足。
- Web/CLI 的 `ask_human` 交互体验仍偏基础。
- 记忆层已具备自动召回与基础演化，但语义向量检索与图谱化关系推理仍未落地（当前为扩展预留位）。
- 安全隔离、资源治理与生产级稳定性仍有差距。

## 4) 下一步（概要）

- 补齐 Native 跨平台与浏览器查询能力。
- 继续完善三层记忆（向量语义检索、关系图谱、演化策略精细化）。
- 强化安全策略、隔离能力与稳态压测。
- 提升 Web/CLI 交互体验，推动 MVP 向可交付版本演进。
- 在 Web Console / CLI 接入 SSE 事件消费，并继续推进 Phase 2 token 级流式输出。
- 视前端实际渲染压力评估是否需要对 `completion_delta` 做批处理/节流优化。
- 继续完善 stop 体验（前端按钮、取消态提示）并评估是否需要对新建会话的首轮执行增加更细粒度的运行态展示。
