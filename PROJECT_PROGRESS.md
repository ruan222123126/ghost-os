# Ghost-OS 项目进展

更新日期：2026-03-22  
当前阶段：MVP 骨架

## 总体结论

- 项目三层结构已经稳定成型：Execution（`drivers/native`）、Central（`core/bridge`）、Perception（`apps/web`、`apps/cli`、`apps/android`）。
- 当前最成熟的是 `core/bridge`，主链路已经可用，承担会话编排、工具调度、状态管理、协议路由与安全边界。
- `apps/web` 和 `apps/cli` 已具备可用 MVP；`apps/android` 已有基础接入，但仍处于补齐阶段。
- `drivers/native` 已具备截图、键鼠输入、脚本执行、窗口查询等原子能力，但仍未达到生产完备状态。

## 分层进度

### Central: `core/bridge`

- Agent、Session、Tool、Provider、SSE、配置持久化、`ask_human` 续跑、基础 Memory 增强链路已落地。
- Memory 主链已切换到“事件节点图驱动”：
  - `core/bridge/memorystore` 新增 `event_nodes` / `event_edges` / `event_memories` / `session_event_state` 存储层，用于承载任务/目标级事件图；旧 `learned_memories` 不再接任务型自动记忆写入，只保留全局长期偏好的实现细节。
  - `core/bridge/memoryaug` 现按 `IntentPlanner -> RecallService -> LearningService` 三段式工作：每轮 prepare 固定先跑 planner，激活范围收敛到 `1 primary + 最多 2 adjacent`，recall 只读激活子图，learning 只写当前 primary event。
  - 注入 prompt 的记忆块已从旧的 `Memory slots / Other memory context` 收口为 `Active event / Relevant event memory / Global preferences`，且仍然不写回 session history。
  - `memory_augmentation_session_scope_enabled` / `memory_augmentation_user_scope_enabled` 已分别落到 recall / learning 读写路径；`memory_augmentation_max_recall_items` 现作为事件记忆总召回上限生效，prompt formatter 不再额外施加写死的 5 条截断。
  - `memory_recall_debug` 现输出 planner 决策、激活节点与最终 prompt block；`memory_learned_list` 已改为按 `event_id / session_id / status` 查看事件记忆。
  - `memory_manage` 的提示词与 GraphQL 最小示例已补齐显式 URI 工作流：新写入使用 `create`，修改/删除前先通过 `read` / `list` / `system://index` 发现精确 URI；缺失显式 memory 时返回可恢复的操作提示而不再只给裸 `not found`。
- Agent 收尾路径进一步收口：`loop_finish` 已合并 stop/length 文本完成分支的公共 finalize 流程，并移除 `toolCallExecutor.execute` 中当前调用图不可达的空 `calls` 防御分支，补充了 length 收尾与 assistant-text 分发回归测试。
- 共享消息契约、`trace_id`、跨端 DTO 与 `core/shared/schema.json` 已基本统一。
- GraphQL 文本工具调用运行时、GUI executor / `computer_use`、任务调度、RSS、配置系统都已建立主线能力。
- 任务更新路径已补回显式回滚：在 `Unregister` 前移后，若 `SaveTask` 或后续 `Upsert` 失败，会恢复旧注册并在需要时把旧任务重新写回磁盘，避免留下“磁盘仍有任务、内存已不再调度”的漂移状态；共享 task schema 也已收口为 kind-specific 契约，`system_action` 的 `action/action_params` 与 `workflow/agent_message` 的必填约束现可被 schema 正确表达。
- GraphQL 文本工具调用运行时的协议失败已改为“可修复的结构化反馈”：解析/校验错误会写入 `[GRAPHQL_TOOL_RESULT]` 风格的 `status=error`、`kind`、`expected/received` 等字段，并在同次 agent run 的下一轮 completion 中作为显式失败反馈供模型自修正。
- GraphQL tool runtime 的 query / mutation 判定已从运行时硬编码名单收口到 `ToolDef.Semantics`：工具通过统一语义元数据声明 `read_only` / `side_effect`，schema 生成、示例输出与执行期校验都复用同一份定义，未知工具仍显式按 mutation 处理。
- GraphQL prompt 已补齐“最小可用示例”层：除了 schema/签名外，还会为当前可见工具输出最小成功 GraphQL 示例，重点覆盖 `tfind(action="load")` 的“下一轮才可用”、`ask_human` 的 `options` 结构，以及 `script_exec` 的最简 mutation。
- GraphQL tool runtime 的 prompt/example 已与真实 schema 对齐：复杂参数通过命名 `input` / `enum` 暴露结构，示例里的枚举字段也改为 GraphQL enum literal，避免 `browser_control`、`computer_use`、`task_manage` 一类工具继续被模型按 JSON 字符串硬拼。
- GraphQL 模式的系统提示词已补回工具使用指导：`hidden catalog` 继续隐藏原生 `tool_defs`，但会为 prompt 保留 `ask_human`、`tfind`、`screen_action`、`computer_use` 等可见工具的“何时使用/有哪些约束”提示。
- Bridge 已新增只读 `web_rooter` 高层工具：通过固定 HTTP 契约接入独立运行的 `web-rooter` 服务，当前仅开放 `internet_search` / `research` / `academic_search` / `site_search` / `fetch` / `extract` 六个 stateless action，并在桥内统一输出 `provider/action/payload/citations/references_text/trace_id` 稳定壳；未引入上游 CLI、MCP、jobs、skills、safe mode、knowledge/visited 等双编排能力。运行时现通过 `web_rooter_enabled/base_url/api_token/timeout_ms` 显式控制接入，桥层不会默认启用或做隐式降级。
- `web_rooter` 桥接现已对上游 HTTP 响应做显式契约校验：`success/content/data/urls/error/metadata` 缺失或类型不合法、HTTP 500、200 + `success=false`、非法 JSON、超时都会直接作为错误上抛，不再被桥层静默包装成成功结果。
- prompt guidance 已按协议模式分流：普通 native `tool_calls` prompt 不再泄漏 `mutation { ... }`、`tfind(action: ...)` 一类 GraphQL 示例，GraphQL 专用样例只保留在 hidden catalog / GraphQL runtime prompt 路径中。
- `assistant-text` invocation 与显式工具调用事件闭环已补齐，通用 handler 不再被 GraphQL 反馈格式硬编码污染。
- 已移除与项目无关的旧业务 GraphQL 工具：`graphql_query`、`graphql_schema_lookup`、`graphql_mutation`；保留 GraphQL 文本协议模式供模型调用普通 Bridge 工具。
- Bridge 仍是当前主要开发中心，近期工作以收口边界、减少脆弱耦合、提升可测试性为主。
- `web_rooter` 实现已按边界重排：顶层工具只保留显式参数校验、action 路由、client 调用与稳定 envelope 输出；HTTP 传输、版本钉死校验、响应归一化下沉到 `tools/internal/webrooter`。对外 public runtime 快照也已收口为 `web_rooter_enabled` / `web_rooter_api_token_set` 布尔态，不再暴露 `base_url` / `timeout_ms`。
- `config/web_rooter_settings.go` 的运行期配置解析已补回缺失的 `resolveWebRooterBaseURL` 路径，`serve`/HTTP 入口可重新完整编译并参与 live 调试。
- `web_rooter` 的 sidecar 边界已补成显式契约并有回归测试锁定：Ghost-OS 只认外置 `base_url`，不负责拉起或管理 upstream Python 进程；桥层继续只开放六个 stateless HTTP action，不接 `knowledge` / `visited` / context snapshot；版本探测与 action 请求都会透传 `X-Trace-ID`；过大响应会返回显式超限错误，不做静默裁切。

### Perception: `apps/web` / `apps/cli` / `apps/android`

- Web Console MVP 可用，已支持基础聊天、配置读取与主要交互链路。
- CLI 基线可用，已支持基础会话与桥接操作。
- Android 已接入部分会话与展示能力，但整体成熟度低于 Web 与 CLI。

### Execution: `drivers/native`

- Native 层已支持截图、输入模拟、脚本执行、窗口/浏览器查询等原子动作。
- GUI executor 所需的双击、右键、滚动、拖拽、组合键、活动窗口信息等能力已补齐一轮基线。
- 该层仍遵守“只做原子执行，不承载业务决策”的边界，新增需求应优先走脚本/API，再考虑视觉路径。

## 已完成的主线里程碑

- 三层边界已经明确，跨层直连被持续收口到统一消息总线和共享契约。
- Bridge 主链路已可运行：模型调用、工具执行、流式返回、会话持久化、人工介入与恢复均已打通。
- Web、CLI、Bridge、Native 的共享配置与生成契约已建立统一基线。
- Memory augmentation、任务调度、RSS、浏览器控制、脚本执行、GUI executor 等能力都已有可用 MVP。
- 多个大文件和巨型模块已完成按职责拆分，仓库整体结构比早期版本清晰很多。

## 当前可用能力

- 会话管理：创建、持久化、流式输出、恢复、基础裁剪。
- 工具链：`script_exec`、`read_and_summarize`、`web_search`、`rss_fetch`、`browser_control`、`memory_manage`、`computer_use` 等。
- 配置系统：Provider、模型、运行时配置的读写、解析与校验。
- 任务系统：基础调度、立即执行、日志查询。
- 任务系统已补上 `workflow` 类型地基：继续复用现有调度器、任务存储、`/api/tasks` 与 `TASK_*` action，在不新增独立服务/存储的前提下支持 `start -> end` 最小 workflow 任务创建、查询、手动执行与运行摘要。
- Web Console / CLI：基础对话和桥接操作可用。

## 当前约束

- `drivers/native` 仍是不完整环节，不能默认视为生产级执行层。
- Memory、GraphQL runtime、GUI executor 虽已可用，但仍属于快速演进区域。
- 项目整体仍处于 MVP 骨架阶段，当前优先级是主链路稳定和边界清晰，而不是功能扩张。
- 本文件只保留项目阶段性摘要，不再记录逐日细粒度变更；详细历史请查看 `git log`。
