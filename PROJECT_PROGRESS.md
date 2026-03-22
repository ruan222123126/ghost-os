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
  - `memory_recall_debug` 现输出 planner 决策、激活节点与最终 prompt block；`memory_learned_list` 已改为按 `event_id / session_id / status` 查看事件记忆。
- Agent 收尾路径进一步收口：`loop_finish` 已合并 stop/length 文本完成分支的公共 finalize 流程，并移除 `toolCallExecutor.execute` 中当前调用图不可达的空 `calls` 防御分支，补充了 length 收尾与 assistant-text 分发回归测试。
- 共享消息契约、`trace_id`、跨端 DTO 与 `core/shared/schema.json` 已基本统一。
- GraphQL 文本工具调用运行时、GUI executor / `computer_use`、任务调度、RSS、配置系统都已建立主线能力。
- GraphQL 模式的系统提示词已补回工具使用指导：`hidden catalog` 继续隐藏原生 `tool_defs`，但会为 prompt 保留 `ask_human`、`tfind`、`screen_action`、`computer_use` 等可见工具的“何时使用/有哪些约束”提示。
- `assistant-text` invocation 与显式工具调用事件闭环已补齐，通用 handler 不再被 GraphQL 反馈格式硬编码污染。
- 已移除与项目无关的旧业务 GraphQL 工具：`graphql_query`、`graphql_schema_lookup`、`graphql_mutation`；保留 GraphQL 文本协议模式供模型调用普通 Bridge 工具。
- Bridge 仍是当前主要开发中心，近期工作以收口边界、减少脆弱耦合、提升可测试性为主。

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
- Web Console / CLI：基础对话和桥接操作可用。

## 当前约束

- `drivers/native` 仍是不完整环节，不能默认视为生产级执行层。
- Memory、GraphQL runtime、GUI executor 虽已可用，但仍属于快速演进区域。
- 项目整体仍处于 MVP 骨架阶段，当前优先级是主链路稳定和边界清晰，而不是功能扩张。
- 本文件只保留项目阶段性摘要，不再记录逐日细粒度变更；详细历史请查看 `git log`。
