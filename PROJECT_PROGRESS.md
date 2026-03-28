# Ghost-OS 项目进展

更新日期：2026-03-28  
当前阶段：MVP 骨架

## 总体结论

- 项目三层结构已经稳定成型：Execution（`drivers/native`）、Central（`core/bridge`）、Perception（`apps/web`、`apps/cli`、`apps/android`）。
- 当前最成熟的是 `core/bridge`，主链路已经可用，承担会话编排、工具调度、状态管理、协议路由与安全边界。
- `apps/web` 和 `apps/cli` 已具备可用 MVP；`apps/android` 已有基础接入，但仍处于补齐阶段。
- `drivers/native` 已具备截图、键鼠输入、脚本执行、窗口查询等原子能力，但仍未达到生产完备状态。

## 分层进度

### Central: `core/bridge`

- Agent、Session、Tool、Provider、SSE、配置持久化、`ask_human` 续跑、基础 Memory 增强链路已落地。
- 普通 Agent 请求现已补上图片入参链路：`/api/agent` / `AGENT_SEND` 支持 `images[]`，图片可用本地路径、远程 URL 或 data URL 表达；Central 会把用户图片写入 session history，并在 provider 投影阶段对 OpenAI / Anthropic / Codex 统一转成对应多模态输入，不再只支持 tool-result 图片。
- Memory 主链已切换到“事件节点图驱动”：
  - `core/bridge/memorystore` 新增 `event_nodes` / `event_edges` / `event_memories` / `session_event_state` 存储层，用于承载任务/目标级事件图；旧 `learned_memories` 不再接任务型自动记忆写入，只保留全局长期偏好的实现细节。
  - `core/bridge/memoryaug` 现按 `IntentPlanner -> RecallService -> LearningService` 三段式工作：每轮 prepare 固定先跑 planner，激活范围收敛到 `1 primary + 最多 2 adjacent`，recall 只读激活子图，learning 只写当前 primary event。
  - 注入 prompt 的记忆块已从旧的 `Memory slots / Other memory context` 收口为 `Active event / Relevant event memory / Global preferences`，且仍然不写回 session history。
  - `memory_augmentation_session_scope_enabled` / `memory_augmentation_user_scope_enabled` 已分别落到 recall / learning 读写路径；`memory_augmentation_max_recall_items` 现作为事件记忆总召回上限生效，prompt formatter 不再额外施加写死的 5 条截断。
  - `memory_recall_debug` 现输出 planner 决策、激活节点与最终 prompt block；`memory_learned_list` 已改为按 `event_id / session_id / status` 查看事件记忆。
  - `memory_manage` 的提示词与 GraphQL 最小示例已补齐显式 URI 工作流：新写入使用 `create`，修改/删除前先通过 `read` / `list` / `system://index` 发现精确 URI；缺失显式 memory 时返回可恢复的操作提示而不再只给裸 `not found`。
  - 新会话里的“纯全局偏好”消息现不再强制进入事件图 planner：像“偏好中文回答”这类长期偏好会跳过 event recall 预处理，但仍保留后续 global preference learning，避免在空事件上下文里因 planner 引用不存在 event 而直接失败成 `memory not found`。
  - 新会话里的低信息开场消息也不再进入事件图 planner：像“你好”这类问候/寒暄会直接跳过 event recall，仅保留全局偏好上下文，避免 planner 在无候选事件时生成不存在的 event id 并最终炸成 `memory not found`。
  - 事件图 planner 的 `recall_plan.event_ids` 现会收口到“已解析出的真实 active event ids”：模型输出的临时别名或占位 id（如 `evt_ai_current_landscape_search`）会在 recall 前被剔除，并在为空时回退到真实 primary/adjacent ids，避免搜索类新任务在 recall 阶段把别名当成 adjacent event 再次炸成 `memory not found`。
- Agent 收尾路径进一步收口：`loop_finish` 已合并 stop/length 文本完成分支的公共 finalize 流程，并移除 `toolCallExecutor.execute` 中当前调用图不可达的空 `calls` 防御分支，补充了 length 收尾与 assistant-text 分发回归测试。
- 共享消息契约、`trace_id`、跨端 DTO 与 `core/shared/schema.json` 已基本统一。
- 任务更新路径已补回显式回滚：在 `Unregister` 前移后，若 `SaveTask` 或后续 `Upsert` 失败，会恢复旧注册并在需要时把旧任务重新写回磁盘，避免留下“磁盘仍有任务、内存已不再调度”的漂移状态；共享 task schema 也已收口为 kind-specific 契约，`system_action` 的 `action/action_params` 与 `workflow/agent_message` 的必填约束现可被 schema 正确表达。
- 任务调度器的 registration 生命周期已补上显式 retired 状态：`Stop` / `Unregister` / `register` 替换旧实例后，旧 `taskRegistration` 不会再在锁外被 `RunNow` 或定时触发重新 `beginRun`，从而封住 stale registration 复活和 `waitIdle()` 与 `runWG.Add(1)` 并发交错的风险。
- 任务调度器的 Start/Stop 边界已进一步收紧为显式门闩：`Upsert` / `RunNow` 现要求 scheduler 处于 running 生命周期内，`Stop` 后不会再被并发 API 调用重新注册或手动触发；任务删除路径也已补齐显式回滚，非法 `*.json` 任务文件名会进入 tolerant load issues，而不再被静默跳过。
- task kind 归一化已去掉“非法值静默回落到 `agent_message`”的 fallback：`task_kind` 为空时仍默认视为 `agent_message`，但未知值现在会在校验阶段显式报 `unsupported task_kind`，执行器默认分支也不会再把坏输入当作 agent task 运行。
- GraphQL 文本工具调用运行时、GUI executor / `computer_use`、任务调度、RSS、配置系统都已建立主线能力。
- RSS report 生成链路已去掉静默 fallback：agent 报告空回或失败时不再落回模板化“机会点 / 风险与约束 / 接下来可能会怎样”段落，而是显式记录 `report_error`；report prompt 也已收口到更精简的章节契约，避免重复凑段。
- GraphQL 文本工具调用运行时的协议失败已改为“可修复的结构化反馈”：解析/校验错误会写入 `[GRAPHQL_TOOL_RESULT]` 风格的 `status=error`、`kind`、`expected/received`、`hint/example` 等字段，并在同次 agent run 的下一轮 completion 中作为显式失败反馈供模型自修正。
- GraphQL tool runtime 继续以 `ToolDef.Semantics` 作为读写语义源：schema、示例与执行期校验重新按真实工具语义区分 `query` / `mutation`，不再把所有工具强行压成 mutation-only。
- GraphQL prompt 已补齐“最小可用示例”层：除了 schema/签名外，还会为当前可见工具输出最小成功 GraphQL 示例，重点覆盖 `tfind(action="load")` 的“同一用户 turn 的下一次 completion 可用”、`ask_human` 的 `options` 结构，以及 `script_exec` 的最简 mutation。
- GraphQL tool runtime 的 prompt/example 已与真实 schema 对齐：复杂参数通过命名 `input` / `enum` 暴露结构，示例里的枚举字段也改为 GraphQL enum literal，避免 `browser_control`、`computer_use`、`task_manage` 一类工具继续被模型按 JSON 字符串硬拼。
- GraphQL tool runtime 的空能力面提示已去掉 `_empty` 这类可误判为真实能力的占位字段：当当前 turn 没有可用 GraphQL 工具时，prompt 会直接输出显式说明，避免模型把占位字段当成可调用能力。
- GraphQL tool runtime 现支持“同回合 load+use”：`tfind(action="load")` 写入的动态工具会在当前用户 turn 内即时可见，bridge 会在每次 completion 前刷新 system prompt / GraphQL schema / Dynamic Tool State，因此模型无需额外追加一条用户消息，就能在下一次 completion 里直接调用新工具。
- Session history 主链已切到“热窗口常驻 + 冷历史分页”：
  - `core/bridge/session` 已从单文件整段 JSON 持久化切到 SQLite；内存里只保留最近热窗口，旧消息落到 `session_messages`。
  - `/api/sessions/:id` 默认只返回最新一页，并支持 `limit` / `before` 分页窗口；响应里补上 `message_count` 和 `page` 游标信息。
  - legacy `session-id.json` 会在首次读取时自动导入 SQLite 并删除旧文件。
- GraphQL 文本标准化路径已补上“定向 sanitize + 显式开关”：默认开启 `graphql_text_sanitize_enabled`，只清理首尾空白、代码围栏与误拼接的 `[GRAPHQL_TOOL_RESULT]` 后缀；每次命中都会打带 `trace_id` / `kind` 的结构化日志，关闭开关后回到现有严格解析行为。
- GraphQL 模式的系统提示词已补回工具使用指导：`hidden catalog` 继续隐藏原生 `tool_defs`，但会为 prompt 保留 `ask_human`、`tfind`、`screen_action`、`computer_use` 等可见工具的“何时使用/有哪些约束”提示。
- GraphQL 文本工具调用的 provider 请求投影已补齐 assistant/tool 协议配对：持久化 transcript 仍保留原始 assistant 文本 + tool result + internal feedback，但在发给 OpenAI/Anthropic/Codex 前会为已执行的 GraphQL 文本 turn 按原文重建合法的 assistant `tool_calls`，修复下一轮 completion 因 `tool message references unknown tool_call_id` 直接失败的问题。
- Codex 续跑请求构造已补上空增量防护：当 `previous_response_id` 模式下增量窗口只剩 GraphQL internal feedback 或最终变成空 `input` 时，会跳过该 feedback 作为增量边界并在必要时显式回退到无 `previous_response_id` 的非空输入，避免继续向上游发送缺失 `input` 的请求并触发 `input is required`。
- Codex / Responses 参数层已补齐首批标准化透传：`prompt_cache_key`、`prompt_cache_retention`、`safety_identifier`、`metadata`、`store` 可从 config/env 注入并进入 provider 请求；`GHOST_RESPONSE_METADATA_*` 前缀键支持映射 metadata。旧的 Codex 4xx 自动无状态回退改为默认关闭，仅在显式 `codex_stateless_retry_enabled` 打开时才会触发，失败路径默认直出，避免隐式降级。
- Bridge 已新增只读 `web_rooter` 高层工具：通过固定 HTTP 契约接入独立运行的 `web-rooter` 服务，当前仅开放 `internet_search` / `research` / `academic_search` / `site_search` / `fetch` / `extract` 六个 stateless action，并在桥内统一输出 `provider/action/payload/citations/references_text/trace_id` 稳定壳；未引入上游 CLI、MCP、jobs、skills、safe mode、knowledge/visited 等双编排能力。运行时现通过 `web_rooter_enabled/base_url/api_token/timeout_ms` 显式控制接入，桥层不会默认启用或做隐式降级。
- `web_rooter` 桥接现已对上游 HTTP 响应做显式契约校验：`success/content/data/urls/error/metadata` 缺失或类型不合法、HTTP 500、200 + `success=false`、非法 JSON、超时都会直接作为错误上抛，不再被桥层静默包装成成功结果。
- `web_rooter` 实现已按边界重排：顶层工具只保留显式参数校验、action 路由、client 调用与稳定 envelope 输出；HTTP 传输、版本钉死校验、响应归一化下沉到 `tools/internal/webrooter`。对外 public runtime 快照也已收口为 `web_rooter_enabled` / `web_rooter_api_token_set` 布尔态，不再暴露 `base_url` / `timeout_ms`。
- `config/web_rooter_settings.go` 的运行期配置解析已补回缺失的 `resolveWebRooterBaseURL` 路径，`serve`/HTTP 入口可重新完整编译并参与 live 调试。
- `web_rooter` 的 sidecar 边界已补成显式契约并有回归测试锁定：Ghost-OS 只认外置 `base_url`，不负责拉起或管理 upstream Python 进程；桥层继续只开放六个 stateless HTTP action，不接 `knowledge` / `visited` / context snapshot；版本探测与 action 请求都会透传 `X-Trace-ID`；过大响应会返回显式超限错误，不做静默裁切。
- prompt guidance 已按协议模式分流：普通 native `tool_calls` prompt 不再泄漏 `mutation { ... }`、`tfind(action: ...)` 一类 GraphQL 示例，GraphQL 专用样例只保留在 hidden catalog / GraphQL runtime prompt 路径中。
- `web_rooter` 的 prompt guidance 已补齐联网分流规则：需要引用、出处、多源交叉验证、学术资料或深度研究时优先走 `web_rooter`；普通即时网页搜继续走 `web_search`，避免模型把所有联网任务都打到同一层搜索能力。
- `web_search` 的 Tavily / Exa provider 现支持显式自定义 endpoint：运行时配置可分别填写 `web_search_tavily_url` / `web_search_exa_url`，留空时继续走官方接口，填写后请求会直接命中自定义 URL，原有 API key 语义保持不变。
- `web_search` 在同时配置 Tavily 和 Exa API key 时，工具参数 schema 与 GraphQL 最小示例现会把 `provider` 明确提升为必填，并在工具描述中显式说明原因，避免模型继续按 `web_search(query: ...)` 生成错误调用后表现成“什么都没搜到”。
- `tfind(action="search")` 的候选工具匹配已从“整句 substring”改为规范化自然语言词匹配：会统一处理空格/下划线/标点，并优先匹配工具名与标签，避免像 `website search tool availability; web_search, browser_control, internet retrieval, web browser` 这类查询继续把 `web_search` / `browser_control` 搜成空结果。
- GraphQL 文本 sanitize 现可在显式 sanitize 模式下提取并校验嵌入在同一 assistant 文本里的合法 GraphQL 文档：像 `mutation { ... }你好` 这类“工具调用 + 额外文字”不再一律直接 parse error；已知的 `[GRAPHQL_TOOL_RESULT]` 后缀剥离语义保持不变，关闭 `graphql_text_sanitize_enabled` 后仍回到严格纯文档模式。
- 工具可见性语义已拆分为“常驻 allowlist”与“严格 allowlist-only”两层：`tool_allowlist` 现在只定义当前 turn 的 resident 工具；当 `tool_allowlist_only = true` 时，selector 与静态工具面才会一起收紧到 allowlist。非 strict 模式下，selector 仍可为主模型挑选其他未被 `tool_blocklist` 屏蔽的静态工具。
- `assistant-text` invocation 与显式工具调用事件闭环已补齐，通用 handler 不再被 GraphQL 反馈格式硬编码污染。
- 已移除与项目无关的旧业务 GraphQL 工具：`graphql_query`、`graphql_schema_lookup`、`graphql_mutation`；保留 GraphQL 文本协议模式供模型调用普通 Bridge 工具。
- 已完成一次后端 Agent 工具能力全量实测，并沉淀到 `docs/backend-agent-tool-capability-2026-03-28.md`：在临时测试配置（`max_turns=1`、memory 关闭、全工具 allowlist）下 15 个工具均完成至少一次真实调用；其中 `send_file`、`computer_use` 归类为需调试，`codex_cli`、`browser_control` 受前置配置/会话约束。
- `screen_action` 截图链路已切到文件引用：`SCREEN_CAPTURE` 改为返回 `image_path`，Bridge 侧截图 artifact 改为基于文件流复制与流式哈希，不再经过 `image_base64 -> decode -> 写文件` 这条高内存路径；`OCR_IMAGE` / `TEMPLATE_MATCH_IMAGE` 的入参也已改为传 `image_path`。
- Bridge 仍是当前主要开发中心，近期工作以收口边界、减少脆弱耦合、提升可测试性为主。

### Perception: `apps/web` / `apps/cli` / `apps/android`

- Web Console MVP 可用，已支持基础聊天、配置读取与主要交互链路。
- Web Console 现已补上左下角设置入口：侧边栏底部新增 `Settings` 按钮，可直接打开现有运行时配置弹窗，不再需要依赖隐式入口或额外页面跳转。
- Web Console 的 Runtime Settings 现已补上 Tavily / Exa 自定义 URL 输入框，可直接查看、保存或清空搜索 endpoint；未填写时仍默认使用官方地址。
- Web Console 设置弹窗现已完成一轮整体视觉重构：改为左侧导航 + 右侧内容区的白底配置面板，Provider 区切到“列表态 / 编辑态”单视图切换，Runtime 区与 Provider 表单统一为同一套卡片式输入样式，同时保留现有真实配置读写链路。
- Web Console 设置弹窗已对齐新设计稿：左侧导航扩展为 `General / Provider / Appearance / Data & Memory / Notifications / Security` 六个分组项，`Provider` 与 `General` 继续接真实配置读写，其余分组先提供占位页并保持同一视觉框架。
- Web Console 会话主链已切到流式：前端现直接消费 Bridge SSE 的 `run_started / completion_delta / tool_call_started / tool_call_finished / awaiting_human / message / done / error` 事件，回复文本和工具状态可在回合进行中实时落屏；回合结束后仍会回填一次 session history 以收口最终持久化内容、工具输出与附件。
- Web Console 前端的用户侧图片发送链路现已接通：输入区加号按钮可选择多张图片，浏览器会将图片转成 data URL 通过现有 `/api/agent/stream` `images[]` contract 发给 Bridge；前端同时补上发送前预览、纯图片提交、图文混发，以及 session history 里的用户图片回显。
- Web Console 的聊天消息去重已补上 GraphQL 工具文本抑制：当 assistant 的纯 `query/mutation { ... }` 文本已被解析并呈现为 tool card 时，前端不会再额外渲染同一段原始 GraphQL 工具调用文本；历史回放与流式工具事件两条路径都已收口。
- Web Console 的聊天前端已把流式热路径从“整段 `messages[]` 重建”改成“`committedMessages + streamingAssistantText + streamingTools + pendingQuestions`”分层状态：`completion_delta` 不再复制长历史数组，terminal 后只同步最近一页 session history 做 merge；消息列表同时接通现有 older-history 分页并改为 `@tanstack/react-virtual` 虚拟渲染，显著降低长会话下的内存 churn 和整表重渲染放大。
- Web Console 会话详情已完成窗口化 hydrate：首次进入只加载最近一页，旧消息通过顶部补页按页回拉；补页时保持滚动位置，切换会话时重置到当前会话尾部，本地 `local:` / `stream-*` 临时消息会在尾页同步时和持久化消息做稳定 ID 合并。
- Web Console 聊天输入框初始高度已下调一档：输入区初始行数由 `4` 调整为 `3`，在不影响自动增高的前提下减少默认占用空间。
- Web Console 聊天输入提交交互已改为“先清空再发送”：发送后输入框会立即刷新为空；若发送链路抛错，则自动回填草稿与待发图片，避免内容丢失。
- CLI 基线可用，已支持基础会话与桥接操作。
- Android 已接入部分会话与展示能力，但整体成熟度低于 Web 与 CLI。

### Execution: `drivers/native`

- Native 层已支持截图、输入模拟、脚本执行、窗口/浏览器查询等原子动作。
- GUI executor 所需的双击、右键、滚动、拖拽、组合键、活动窗口信息等能力已补齐一轮基线。
- Native 截图子模块已去掉主链路 base64 载荷：`SCREEN_CAPTURE` 返回临时 PNG 文件路径，`OCR_IMAGE` 直接消费 `image_path`，`crop_image` 改为仅复制裁剪区域，避免整图 clone 后再裁切。
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
- `workflow` 运行时已扩到线性节点链：现支持固定参数的 `tool` / `llm` / `agent` 节点，继续复用现有 runtime factory、任务存储与调度器；节点间暂不传递输出，`agent` 节点按 fresh session 运行，`tool` 节点受独立 `workflow_tool_allowlist` 显式约束。
- Web Console / CLI：基础对话和桥接操作可用。
- Web Console 前端的用户侧图片发送已可用：支持点击加号选择多图、发送纯图片或图文混发，并能在历史消息中回显用户图片。

## 当前约束

- `drivers/native` 仍是不完整环节，不能默认视为生产级执行层。
- Memory、GraphQL runtime、GUI executor 虽已可用，但仍属于快速演进区域。
- 项目整体仍处于 MVP 骨架阶段，当前优先级是主链路稳定和边界清晰，而不是功能扩张。
- 本文件只保留项目阶段性摘要，不再记录逐日细粒度变更；详细历史请查看 `git log`。
