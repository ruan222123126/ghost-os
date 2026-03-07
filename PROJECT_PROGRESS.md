# Ghost-OS Progress Snapshot

更新日期：2026-03-07
分支：main（与 `origin/main` 同步）
阶段：MVP 骨架（主链路可用，核心能力持续补齐）

## 1) 三层状态

| 层 | 当前状态 | 完成度 |
|---|---|---|
| Execution (`drivers/native`) | 原子能力可用，跨平台与浏览器能力仍在补齐 | 20% |
| Central (`core/bridge`) | Agent / Session / Tool 主流程稳定，记忆与流式能力已有 MVP 基线 | 72% |
| Perception (`apps/web`, `apps/cli`, `apps/android`) | Web Console、CLI、Android 客户端可用，交互体验仍待增强 | 45% |

## 2) 已完成（摘要）

- 三层边界、消息总线契约、`trace_id` 追踪链路已稳定。
- Bridge 主流程已打通：Provider 调用、Tool 路由、会话持久化、`ask_human` 续跑闭环。
- 会话终止语义已结构化：完整结束使用 `END_SESSION` 信号，对外通过 `session_ended/session_end` 表达。
- Native 执行层完成一轮职责拆分，主入口、action router、脚本沙箱与原子工具面已收口。
- 文件原子工具已补齐：`list_files`、`read_file`、`search_files`、`apply_diff`、`bash_exec` 可贯通 Bridge ↔ Native。
- Native persistent mode 与 Bridge persistent client 已落地，支持 framed JSON、超时杀进程重建与 one-shot fallback。
- Native `browser_query` 已按职责拆分为编排入口、参数解析、CDP 探测、窗口发现与标题匹配子模块，便于后续跨平台扩展与单测。
- Agent SSE 已落地粗粒度事件流，LLM 层已补充 token delta 流式基础能力。
- stop / inflight 保护已具备基础能力：同 session 串行保护、按 `session_id` / `trace_id` 取消运行。
- Memory MVP 已形成：L1/L2/L3 基础读写、自动召回、TTL、Markdown 节点、Dreaming 聚合沉淀已接入。
- Memory 模块已进一步拆分为 façade + services，降低会话热态串味与后台 ticker 生命周期风险。
- `read_and_summarize` Worker 工具已落地，可用于大仓库粗筛与分层摘要。
- Bridge 已接入轻量 LLM Tool Selector 基线：基于工具元数据与近期上下文可在 shadow / real mode 下选择最小工具子集，默认关闭，异常与低置信度统一回退全量工具，并始终保留 `ask_human`。
- Web / CLI 已接入 `POST /api/questions/answer`，不再需要“双请求续跑”编排。
- Bus 单一契约源已进一步收口：schema 现生成 AGENT_SEND 成功/等待人工响应类型到 Go/Web/CLI，Bridge 改用生成的 action 常量，客户端已接住 `session_ended` / `session_end`。
- CLI 公共调用已统一到高层路由：消息走 `/api/agent`，配置走 `/api/config`，`ask_human` 回答走 `/api/questions/answer`；`/api/bus` 仅保留低层兼容定位。
- Web Console 已收口 Bridge API 代理与 textarea 提交逻辑，减少模板重复。
- Android 客户端已完成 MVP 骨架：配置管理、连接测试、消息发送、本地持久化可用。
- Android 客户端已补齐最小资源基线（launcher icon + app theme），`apps/android` 可成功执行 `./gradlew assembleDebug` 产出 Debug APK。
- Android 客户端已允许明文 HTTP（便于 Tailscale `100.x` 直连场景），并验证可重新构建安装包。
- Android 客户端已补齐网络异常可读错误提示，配置页/聊天页不再显示 `null`，并已重新构建 release 安装包。
- Android 客户端网络请求已切到 `Dispatchers.IO`，修复 `NetworkOnMainThreadException`，并已重新构建 release 安装包。
- Bridge 配置主入口已切到文件优先：默认读取 `~/.ghost-os/config.toml`（支持 `GHOST_CONFIG_PATH` 覆盖）并回退环境变量；`/api/config` 更新会持久化到该文件，`bind_addr / api_token / cors_origins` 也可由同一文件统一驱动，并补充了 `docs/config.example.toml` 模板；Web 已增加配置轻量自动刷新，CLI 已补齐 `provider / api_key / base_url / model / chat_path` 运行态配置命令，形成文件与前端/终端双向同步基线。
- Bridge Agent sync/stream 回合编排已收口共享骨架：统一请求校验、错误分类、`session_end` 后处理，以及 runner 侧的会话装配 / 持久化 / `awaiting_human` 提交路径，降低 stop / retry / token 流后续演进时双分支漏改风险。

## 3) 主要短板

- `drivers/native` 仍未达到生产可用，`BROWSER_QUERY` 在非 CDP 场景能力不足。
- Web / CLI / Android 的 `ask_human` 与流式交互体验仍偏基础。
- 记忆层已有自动召回与演化基线，但向量检索、关系图谱、精细化推理尚未落地。
- 安全隔离、资源治理、压测与生产级稳定性仍有明显差距。

## 4) 下一步

- 补齐 Native 跨平台与浏览器查询能力。
- 推进记忆层向量检索、关系图谱与演化策略精细化。
- 强化安全策略、隔离能力、资源治理与稳态压测。
- 提升 Web / CLI / Android 的 SSE、stop、`ask_human` 交互体验。
- 持续观察 `GHOST_NATIVE_PERSISTENT` 灰度表现，再决定是否默认开启。

- 2026-03-07: 将 `read_and_summarize` 内部拆为 runner / chunk reader / worker client，收紧并发编排、READ_FILE 分块读取与 worker prompt/综合职责边界；补充分块截断与 chunk 聚合单测。
- 2026-03-07: 继续将 `read_and_summarize` 结果输出收口为独立 formatter，tool 入口进一步收窄为参数校验 + runner/formatter 装配。
- 2026-03-07: Web `useBridgeChat` 已拆出独立消息/时间线映射层，历史消息保留 `system` / `tool` 结构；`ask_human` 已在前端回放中重建为 Question + User 时间线，为后续工具卡片与更细粒度流式渲染留出扩展位。
- 2026-03-07: 收口会话历史 tool 对外契约：Bridge 读取内部 tool result envelope 后投影稳定的 `tool_result` / `human_interaction` 字段，`/api/sessions/:id` 不再要求 Web 反解 `message.text` JSON 或理解 `ask_human` 内部输出结构。

- 2026-03-07: Bridge 配置切到 `~/.ghost-os/config.toml` + 多 Provider 列表模型；新增 YAML→TOML 自动迁移、`/api/config/providers` CRUD 与 Web Console Provider 管理面板。
- 2026-03-07: Bridge 配置进一步收口为纯 TOML：Provider 从 `[[model_providers]]` 改为 `[providers.<id>]` 命名表，显式新增 `type = openai|anthropic|custom`，保留旧 TOML provider 布局一次性自动重写到新格式；Web Provider 面板与跨端 DTO 已同步补齐 `provider_type/type`。
- 2026-03-07: Android 客户端已对齐 `AGENT_SEND` / `/api/questions/answer` 的联合响应契约，补齐 `awaiting_human` 解析、待回答问题续跑与输入框回答态，避免 `ask_human` 命中时因按单一成功 payload 反序列化而直接不兼容。
- 2026-03-07: 将 `BridgeConfig` / `SessionDetail` / `ProviderListResponse` / `HumanResponseRequest` 等跨端业务 DTO 并入 `core/shared/schema.json`，扩展生成链统一产出 Go/Web/CLI/Android 契约；Bridge 会话详情改为稳定输出 snake_case `sessionMessage`，Android 移除漂移的 `native_driver_ready` 展示并改为读取真实配置字段。
- 2026-03-07: 记忆层完成 PR1 基础增强：`MemoryEntry` / Markdown node 新增 summary、anchors、confidence、source_ids 等结构化字段；L1/L2/L3/Markdown 改为统一 temporal + anchor rerank；dreaming 已接上 worker summarizer 与规则/可选 LLM anchor 提炼，并补齐旧 warm JSON / 旧 markdown frontmatter 兼容与配置/排序回归测试。
- 2026-03-07: Web 会话历史读取已在 API 边界严格校验 `SessionDetail` / `sessionMessage` 蛇形契约，显式拒绝 `Role/Text/ToolCalls` 这类漂移字段；`chatMessages` 映射层改为直接消费稳定字段，避免 UI 继续猜测服务端实际输出。
- 2026-03-07: Web `getSession` 运行时解析已补齐 `tool_result` / `human_interaction` 校验与测试，确保 Bridge 投影后的稳定 tool DTO 不会再被前端误判为非法 payload。
- 2026-03-07: `SessionAgentRunner` 已将回合装配下沉到 `sessionTurnPreparer`，把 runtime build、会话加载、run 注册、tool selector 与 memory auto recall 从执行/stream facade 中拆出；保留对外 `SessionTurnRunner` 契约不变，并补齐 preparer 级选择器回归测试。
- 2026-03-07: `core/bridge/agent` 对话循环已继续按职责拆分：`loop.go` 仅保留 turn 控制与退出策略，模型调用/流式桥接、tool 执行与 stderr 记录、event sink 封装、tool result envelope 已分别下沉到独立协作者，减少后续 stop / awaiting_human / event payload 演进时的耦合改动面。
- 2026-03-07: `browser_action` 已收口为语义 action→native action 映射；截图 base64 解码、artifact 持久化与 vision 图生成移到独立 post-processor / artifact store，Agent 工具执行链新增统一 result post-process 钩子，保留截图多模态输入适配与对外输出契约不变。
- 2026-03-07: 记忆层完成 PR2 Graph Memory / GraphRAG MVP：新增文件持久化 graph sidecar（nodes / edges / aliases）、规则+可选 worker 的实体关系抽取、实体归一化与别名解析、单值关系 supersede / conflict 处理、1~2 hop graph recall 接入自动召回与 `MEMORY_QUERY`、graph rebuild/backfill、配置项与核心单测/Bus 集成测试。
- 2026-03-07: 清理 `core/bridge/memory` 高置信度死代码：删除未被引用的 `MemoryLayer` 接口与未生效的 `DecayFactor / Priority / UseTimeDecay / MinPriority` 残留字段，移除自动召回中的无效写入与查询侧无效过滤，并补充 warm memory 兼容旧字段载荷的回归测试。
- 2026-03-07: Web Console 收口会话/消息与 Bridge API 代理的重复骨架：`useSessions` 合并重复 setter，`useBridgeChat` 内联一次性包装函数，Provider CRUD 提炼共享异步 helper，API 代理路由改为统一 handler 工厂；同时清理 `drivers/native` 若干 clippy 冗余并确认 `pymethods` 宏展开告警仅做模块级抑制。
- 2026-03-07: 记忆层启动 PR3 基础设施，新增 decision memo/recipe schema、文件化 store 与配置壳；capture/query/selector 接线待后续 commit。
- 2026-03-07: PR3 第二步已接入 decision turn capture；回合完成后会把工具路径、人工阻塞点、answered questions、环境指纹与结构化经验写入 decision memo，仍未接 recall/selector/recipe distill。
