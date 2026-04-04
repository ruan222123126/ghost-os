# Ghost-OS 项目进展

更新日期：2026-04-04  
当前阶段：MVP 稳定化（边界收口中）

## 总体结论

- 项目三层结构已经稳定成型：Execution（`drivers/native`）、Central（`core/bridge`）、Perception（`apps/web`、`apps/cli`、`apps/android`）。
- 当前最成熟的是 `core/bridge`，主链路已经可用，承担会话编排、工具调度、状态管理、协议路由与安全边界。
- `apps/web` 和 `apps/cli` 已具备可用 MVP；`apps/android` 已有基础接入，但仍处于补齐阶段。
- `drivers/native` 已具备截图、键鼠输入、脚本执行、窗口查询等原子能力，但仍未达到生产完备状态。
- Assistant 文本工具协议已收口到 Tool-Tag：后端与前端主链均以 `<t:ID>JSON</t>` + `[TOOL_TAG_RESULT]` 为基线，旧 `mutation/query` 文本调用仅作为显式协议错误处理路径。

## 分层进度

### Central: `core/bridge`

- Agent、Session、Tool、Provider、SSE、配置持久化、`ask_human` 续跑、基础 Memory 增强链路已落地。
- Bridge 启动入口的 `serve` 子命令判定已收口到 `app.IsServeSubcommand` 单点实现，`core/bridge/main.go` 与 `app.Run` 不再重复维护同构逻辑；同时清理了 `core/bridge/main.go.tmp.k29Khx` 临时文件，并移除 `core/bridge/app/agent.go` 中仅测试使用的注入缝隙层（`newAgentTurnRunner`、`runAgentWithConfigStore`）。
- context/prompt 链路完成一轮“显式失败优先”收口：删除仅测试引用的 `BuildRequest` 死路径与 `NewPromptManager` 薄封装入口；`runtime/system_prompt` 不再在加载失败时静默回退默认 prompt；`context/prompt.go` 拆分为 `prompt.go + prompt_loader.go`（各自低于 300 行）并补齐“不再静默兜底”的回归测试。
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
- Agent 工具执行与消息投影链路完成一轮可维护性收口：清理未引用测试辅助（含 `fakeGraphQLTextExecutor` 与空转 `streaming_test_helpers_test.go`）、移除仅测试使用的 `RunStream`/`RunStreamWithTraceID` 对外入口并统一走 `RunMessageStreamWithTraceID`、将 `repairProjectedGraphQLTextTurn` 与 `validateAssistantTextResult` 拆为小函数以降低圈复杂度、同时把 `tool_executor_execute.go` 按“解析/执行”职责拆分为 `tool_executor_execute.go` + `tool_executor_resolution.go`，避免单文件超 300 行。
- 对话 completion 链路已新增一次性瞬时错误重试：在 `completion_runner` 中对网络错误、HTTP 429、HTTP 5xx 提供最多 1 次重试；流式场景仅在“尚未发出任何 delta”时允许重试，已产出增量后失败不会重放，避免重复输出。
- 共享消息契约、`trace_id`、跨端 DTO 与 `core/shared/schema.json` 已基本统一。
- 任务更新路径已补回显式回滚：在 `Unregister` 前移后，若 `SaveTask` 或后续 `Upsert` 失败，会恢复旧注册并在需要时把旧任务重新写回磁盘，避免留下“磁盘仍有任务、内存已不再调度”的漂移状态；共享 task schema 也已收口为 kind-specific 契约，`system_action` 的 `action/action_params` 与 `workflow/agent_message` 的必填约束现可被 schema 正确表达。
- 任务调度器的 registration 生命周期已补上显式 retired 状态：`Stop` / `Unregister` / `register` 替换旧实例后，旧 `taskRegistration` 不会再在锁外被 `RunNow` 或定时触发重新 `beginRun`，从而封住 stale registration 复活和 `waitIdle()` 与 `runWG.Add(1)` 并发交错的风险。
- 任务调度器的 Start/Stop 边界已进一步收紧为显式门闩：`Upsert` / `RunNow` 现要求 scheduler 处于 running 生命周期内，`Stop` 后不会再被并发 API 调用重新注册或手动触发；任务删除路径也已补齐显式回滚，非法 `*.json` 任务文件名会进入 tolerant load issues，而不再被静默跳过。
- task kind 归一化已去掉“非法值静默回落到 `agent_message`”的 fallback：`task_kind` 为空时仍默认视为 `agent_message`，但未知值现在会在校验阶段显式报 `unsupported task_kind`，执行器默认分支也不会再把坏输入当作 agent task 运行。
- GraphQL 文本工具调用运行时、GUI executor / `computer_use`、任务调度、RSS、配置系统都已建立主线能力。
- RSS report 生成链路已去掉静默 fallback：agent 报告空回或失败时不再落回模板化“机会点 / 风险与约束 / 接下来可能会怎样”段落，而是显式记录 `report_error`；report prompt 也已收口到更精简的章节契约，避免重复凑段。
- GraphQL 文本工具调用运行时的协议失败已改为“可修复的结构化反馈”：解析/校验错误会写入 `[TOOL_TAG_RESULT]` 风格的 `status=error`、`kind`、`expected/received`、`hint/example` 等字段，并在同次 agent run 的下一轮 completion 中作为显式失败反馈供模型自修正。
- 文本工具调用协议已在后端硬切换为 `<t:ID>JSON</t>`：可见工具按每轮 ID 映射，执行链路使用字符级状态机串行解析 `<t:...>` 标签；旧 `mutation/query` 文本调用会直接返回结构化协议错误，不再兼容。
- Tag 文本协议的内部回执前缀已统一改名为 `[TOOL_TAG_RESULT]`，并同步到后端解析清洗、Codex 续跑边界判断、前端 internal note 过滤与相关测试，避免继续暴露 GraphQL 语义残留。
- Codex 续跑路径已同步跟进 Tag 文本协议：当增量窗口只剩文本协议 `tool_call_output` 时会触发无状态重放，避免 `input is empty` 类续跑错误。
- GraphQL tool runtime 继续以 `ToolDef.Semantics` 作为读写语义源：schema、示例与执行期校验重新按真实工具语义区分 `query` / `mutation`，不再把所有工具强行压成 mutation-only。
- GraphQL prompt 已补齐“最小可用示例”层：除了 schema/签名外，还会为当前可见工具输出最小成功 GraphQL 示例，重点覆盖 `tfind(action="load")` 的“同一用户 turn 的下一次 completion 可用”、`ask_human` 的 `options` 结构，以及 `script_exec` 的最简 mutation。
- GraphQL tool runtime 的 prompt/example 已与真实 schema 对齐：复杂参数通过命名 `input` / `enum` 暴露结构，示例里的枚举字段也改为 GraphQL enum literal，避免 `browser_control`、`computer_use`、`task_manage` 一类工具继续被模型按 JSON 字符串硬拼。
- GraphQL tool runtime 的空能力面提示已去掉 `_empty` 这类可误判为真实能力的占位字段：当当前 turn 没有可用 GraphQL 工具时，prompt 会直接输出显式说明，避免模型把占位字段当成可调用能力。
- GraphQL tool runtime 现支持“同回合 load+use”：`tfind(action="load")` 写入的动态工具会在当前用户 turn 内即时可见，bridge 会在每次 completion 前刷新 system prompt / GraphQL schema / Dynamic Tool State，因此模型无需额外追加一条用户消息，就能在下一次 completion 里直接调用新工具。
- GraphQL 文本工具调用协议已支持“单文档多次调用”：同一 assistant 文本可包含多个顺序 `mutation` operation（每个 operation 仍限制为单顶层字段），执行链路会为每一步生成独立 `tool_call_id` 与流式事件；普通工具错误会继续执行后续 operation，`awaiting_human` / 迭代交接仍会中断后续步骤。provider 请求投影也已扩到多调用修复，避免出现批量场景下的 `unknown tool_call_id` 配对失败。
- Session history 主链已切到“热窗口常驻 + 冷历史分页”：
  - `core/bridge/session` 已从单文件整段 JSON 持久化切到 SQLite；内存里只保留最近热窗口，旧消息落到 `session_messages`。
  - `/api/sessions/:id` 默认只返回最新一页，并支持 `limit` / `before` 分页窗口；响应里补上 `message_count` 和 `page` 游标信息。
  - legacy `session-id.json` 会在首次读取时自动导入 SQLite 并删除旧文件。
- GraphQL 文本标准化路径已补上“定向 sanitize + 显式开关”：默认开启 `graphql_text_sanitize_enabled`，只清理首尾空白、代码围栏与误拼接的 `[TOOL_TAG_RESULT]` 后缀；每次命中都会打带 `trace_id` / `kind` 的结构化日志，关闭开关后回到现有严格解析行为。
- GraphQL 模式的系统提示词已补回工具使用指导：`hidden catalog` 继续隐藏原生 `tool_defs`，但会为 prompt 保留 `ask_human`、`tfind`、`screen_action`、`computer_use` 等可见工具的“何时使用/有哪些约束”提示。
- GraphQL 文本工具调用的 provider 请求投影已补齐 assistant/tool 协议配对：持久化 transcript 仍保留原始 assistant 文本 + tool result + internal feedback，但在发给 OpenAI/Anthropic/Codex 前会为已执行的 GraphQL 文本 turn 按原文重建合法的 assistant `tool_calls`，修复下一轮 completion 因 `tool message references unknown tool_call_id` 直接失败的问题。
- assistant-text 工具调用提交路径已改为“同回合同步结构化提交”：当 assistant 文本（含 GraphQL `mutation`）被识别为可执行调用时，Bridge 会在执行工具前就把该条 assistant 消息按 `tool_calls` 结构写入 history，并在同轮立即执行，不再只以纯文本落库后等待后续 provider 请求阶段再做投影修复。
- Codex 续跑请求构造已补上空增量防护：当 `previous_response_id` 模式下增量窗口只剩 GraphQL internal feedback 或最终变成空 `input` 时，会跳过该 feedback 作为增量边界并在必要时显式回退到无 `previous_response_id` 的非空输入，避免继续向上游发送缺失 `input` 的请求并触发 `input is required`。
- Codex / Responses 参数层已补齐首批标准化透传：`prompt_cache_key`、`prompt_cache_retention`、`safety_identifier`、`metadata`、`store` 可从 config/env 注入并进入 provider 请求；`GHOST_RESPONSE_METADATA_*` 前缀键支持映射 metadata。旧的 Codex 4xx 自动无状态回退改为默认关闭，仅在显式 `codex_stateless_retry_enabled` 打开时才会触发，失败路径默认直出，避免隐式降级。
- Codex continuation 的 4xx 回退判定已补齐 `No tool call found for function_call_output/function call output with call_id ...` 场景：在显式开启 `codex_stateless_retry_enabled` 时，这类“只有工具结果、上游丢失对应 function_call 上下文”的错误会自动触发一次无状态重放，并已补齐同步/流式回归测试。
- Codex continuation 在显式开启 `codex_stateless_retry_enabled` 且消息里存在 `function_call + function_call_output` 时，现改为首发请求就走无状态重放（不再先依赖 `previous_response_id` 等上游报错后再降级）；同步/流式链路均已补齐回归测试，普通纯文本 follow-up 续跑策略保持不变。
- Codex 工具 schema sanitize 已修复 `properties` 容器污染：`sanitizeCodexToolSchema` 不再把 `properties`/`$defs` 等“schema map 容器”误判为 schema 节点并注入伪 `type:"string"` 字段，修复 `browser_control` 在普通 tool-calling 下触发的 `Invalid schema for function ... tools[0].parameters`（`string` 不是 `object|boolean`）400 错误；同时补充了回归测试锁定该路径。
- Bridge 已新增只读 `web_rooter` 高层工具：通过固定 HTTP 契约接入独立运行的 `web-rooter` 服务，当前仅开放 `internet_search` / `research` / `academic_search` / `site_search` / `fetch` / `extract` 六个 stateless action，并在桥内统一输出 `provider/action/payload/citations/references_text/trace_id` 稳定壳；未引入上游 CLI、MCP、jobs、skills、safe mode、knowledge/visited 等双编排能力。运行时现通过 `web_rooter_enabled/base_url/api_token/timeout_ms` 显式控制接入，桥层不会默认启用或做隐式降级。
- `web_rooter` 桥接现已对上游 HTTP 响应做显式契约校验：`success/content/data/urls/error/metadata` 缺失或类型不合法、HTTP 500、200 + `success=false`、非法 JSON、超时都会直接作为错误上抛，不再被桥层静默包装成成功结果。
- `web_rooter` 实现已按边界重排：顶层工具只保留显式参数校验、action 路由、client 调用与稳定 envelope 输出；HTTP 传输、版本钉死校验、响应归一化下沉到 `tools/internal/webrooter`。对外 public runtime 快照也已收口为 `web_rooter_enabled` / `web_rooter_api_token_set` 布尔态，不再暴露 `base_url` / `timeout_ms`。
- `config/web_rooter_settings.go` 的运行期配置解析已补回缺失的 `resolveWebRooterBaseURL` 路径，`serve`/HTTP 入口可重新完整编译并参与 live 调试。
- `web_rooter` 的 sidecar 边界已补成显式契约并有回归测试锁定：Ghost-OS 只认外置 `base_url`，不负责拉起或管理 upstream Python 进程；桥层继续只开放六个 stateless HTTP action，不接 `knowledge` / `visited` / context snapshot；版本探测与 action 请求都会透传 `X-Trace-ID`；过大响应会返回显式超限错误，不做静默裁切。
- prompt guidance 已按协议模式分流：普通 native `tool_calls` prompt 不再泄漏 `mutation { ... }`、`tfind(action: ...)` 一类 GraphQL 示例，GraphQL 专用样例只保留在 hidden catalog / GraphQL runtime prompt 路径中。
- `browser_control` 的 prompt guidance 已补上显式动作约束：系统提示现在会直接列出合法 `action`（`connect|launch|goto|click|type|press|evaluate|content|screenshot|info|close`），并明确 `goto`/`wait`/`content` 用法，减少模型继续误用 `navigate`、独立 `wait`、`extract` 的概率。
- `screen_action` / `task_manage` / `feed_manage` / `codex_cli` 的 prompt guidance 也已补上显式合法操作约束：分别给出 `action/operation/op` 枚举与关键必填字段；同时修正 `codex_cli` 工具描述中的旧文案 `exec` 为真实枚举值 `start`，避免模型生成非法 `op`。
- `web_rooter` 的 prompt guidance 已补齐联网分流规则：需要引用、出处、多源交叉验证、学术资料或深度研究时优先走 `web_rooter`；普通即时网页搜继续走 `web_search`，避免模型把所有联网任务都打到同一层搜索能力。
- `web_search` 的 Tavily / Exa provider 现支持显式自定义 endpoint：运行时配置可分别填写 `web_search_tavily_url` / `web_search_exa_url`，留空时继续走官方接口，填写后请求会直接命中自定义 URL，原有 API key 语义保持不变。
- `web_search` 在同时配置 Tavily 和 Exa API key 时，工具参数 schema 与 GraphQL 最小示例现会把 `provider` 明确提升为必填，并在工具描述中显式说明原因，避免模型继续按 `web_search(query: ...)` 生成错误调用后表现成“什么都没搜到”。
- `tfind(action="search")` 的候选工具匹配已从“整句 substring”改为规范化自然语言词匹配：会统一处理空格/下划线/标点，并优先匹配工具名与标签，避免像 `website search tool availability; web_search, browser_control, internet retrieval, web browser` 这类查询继续把 `web_search` / `browser_control` 搜成空结果。
- GraphQL 文本 sanitize 现可在显式 sanitize 模式下提取并校验嵌入在同一 assistant 文本里的合法 GraphQL 文档：像 `mutation { ... }你好` 这类“工具调用 + 额外文字”不再一律直接 parse error；已知的 `[TOOL_TAG_RESULT]` 后缀剥离语义保持不变，关闭 `graphql_text_sanitize_enabled` 后仍回到严格纯文档模式。
- 工具可见性语义已拆分为“常驻 allowlist”与“严格 allowlist-only”两层：`tool_allowlist` 现在只定义当前 turn 的 resident 工具；当 `tool_allowlist_only = true` 时，selector 与静态工具面才会一起收紧到 allowlist。非 strict 模式下，selector 仍可为主模型挑选其他未被 `tool_blocklist` 屏蔽的静态工具。
- `assistant-text` invocation 与显式工具调用事件闭环已补齐，通用 handler 不再被 GraphQL 反馈格式硬编码污染。
- 已移除与项目无关的旧业务 GraphQL 工具：`graphql_query`、`graphql_schema_lookup`、`graphql_mutation`；保留 GraphQL 文本协议模式供模型调用普通 Bridge 工具。
- GraphQL 文本协议的遗留死代码已完成一轮清理：删除未接线的文档预算校验模块、schema render 辅助模块，以及一组未引用的协议错误构造器/工具 ID 辅助函数，`core/bridge/tools` 的 staticcheck(U1000) 不再报告这批不可达路径。
- 已完成一次后端 Agent 工具能力全量实测，并沉淀到 `docs/backend-agent-tool-capability-2026-03-28.md`：在临时测试配置（`max_turns=1`、memory 关闭、全工具 allowlist）下 15 个工具均完成至少一次真实调用；其中 `send_file`、`computer_use` 归类为需调试，`codex_cli`、`browser_control` 受前置配置/会话约束。
- `screen_action` 截图链路已切到文件引用：`SCREEN_CAPTURE` 改为返回 `image_path`，Bridge 侧截图 artifact 改为基于文件流复制与流式哈希，不再经过 `image_base64 -> decode -> 写文件` 这条高内存路径；`OCR_IMAGE` / `TEMPLATE_MATCH_IMAGE` 的入参也已改为传 `image_path`。
- native binary 解析顺序已收口为“优先仓库内 `drivers/native/target/*` 构建产物，再尝试裸名 `native`”：避免误命中过期二进制导致 `SCREEN_CAPTURE` payload 与 Bridge 契约漂移；同时 `screen_action` / `computer_use` 对截图 payload 增加了显式契约校验，在缺失 `image_path` 或命中旧 `image_base64` 字段时会直接报结构化错误，不再只给 `empty image_path`。
- Bridge 启动层已补齐专用回归测试：`core/bridge/app/startup_router_test.go` 与 `startup_error_test.go` 覆盖了 `Run` 的 `serve`/非 `serve` 路由、startup checkpoint 日志、`usageError` 透传以及非 usage 错误包装（`serve dispatch failed`）路径。
- artifacts 存储读取接口已做一次边界收口：`ResolveStoredPath` / `OpenStoredFile` 移除未使用 options 并固定启用 symlink 逃逸校验，`normalizeIdentifier` 删除重复的路径分隔符分支，`SessionFileArtifact` 不再写入未被消费的 `CreatedAt` 元数据字段。
- `core/bridge/config` 已完成一轮死代码与复杂度收口：删除未接线私有 env 包装函数与重复 GraphQL env 解析路径（含整文件 `config_graphql_env.go`），并将 runtime 配置构建按职责拆分为 `config_runtime_resolve_helpers.go`、`config_runtime_sections_rss.go`、`config_runtime_sections_tools.go`；`config_runtime_resolve.go` 已降到 300 行以内，相关热点函数均拆到 50 行以内。
- Bridge 仍是当前主要开发中心，近期工作以收口边界、减少脆弱耦合、提升可测试性为主。

### Perception: `apps/web` / `apps/cli` / `apps/android`

- Web Console MVP 可用，已支持基础聊天、配置读取与主要交互链路。
- Web Console 现已补上左下角设置入口：侧边栏底部新增 `Settings` 按钮，可直接打开现有运行时配置弹窗，不再需要依赖隐式入口或额外页面跳转。
- Web Console 的 Runtime Settings 现已补上 Tavily / Exa 自定义 URL 输入框，可直接查看、保存或清空搜索 endpoint；未填写时仍默认使用官方地址。
- Web Console 设置弹窗现已完成一轮整体视觉重构：改为左侧导航 + 右侧内容区的白底配置面板，Provider 区切到“列表态 / 编辑态”单视图切换，Runtime 区与 Provider 表单统一为同一套卡片式输入样式，同时保留现有真实配置读写链路。
- Web Console 设置弹窗已对齐新设计稿：左侧导航扩展为 `General / Provider / Appearance / Data & Memory / Notifications / Security` 六个分组项，`Provider` 与 `General` 继续接真实配置读写，其余分组先提供占位页并保持同一视觉框架。
- Web Console 会话主链已切到流式：前端现直接消费 Bridge SSE 的 `run_started / completion_delta / tool_call_started / tool_call_finished / awaiting_human / message / done / error` 事件，回复文本和工具状态可在回合进行中实时落屏；回合结束后仍会回填一次 session history 以收口最终持久化内容、工具输出与附件。
- Web Console 前端的用户侧图片发送链路现已接通：输入区加号按钮可选择多张图片，浏览器会将图片转成 data URL 通过现有 `/api/agent/stream` `images[]` contract 发给 Bridge；前端同时补上发送前预览、纯图片提交、图文混发，以及 session history 里的用户图片回显。
- Web Console 的聊天消息渲染已切到 Tool-Tag 协议：流式链路通过 `<t:ID>JSON</t>` 状态机实时分流普通文本与工具参数，历史映射会剥离标签仅展示可见文本；纯标签 assistant 消息不会重复渲染原始协议文本。
- Web Console 的 internal note 渲染已补上 `[TOOL_TAG_RESULT]` 的 `tfind` 结果收口：前端展示时只保留“已加载/可用”的 tool 项，未加载或已过期项不再占据消息区。
- Web Console 的聊天前端已把流式热路径从“整段 `messages[]` 重建”改成“`committedMessages + streamingAssistantText + streamingTools + pendingQuestions`”分层状态：`completion_delta` 不再复制长历史数组，terminal 后只同步最近一页 session history 做 merge；消息列表同时接通现有 older-history 分页并改为 `@tanstack/react-virtual` 虚拟渲染，显著降低长会话下的内存 churn 和整表重渲染放大。
- Web Console 流式消息列表已补上显式事件顺序轨道：前端新增 `streamingItemOrder`（assistant/tool/question）并按该顺序渲染 streaming rows，不再固定按“assistant 段 + tool 段 + question 段”分块拼接，工具卡片会随 SSE 到达顺序由上到下展开。
- Web Console 流式渲染已从“单 assistant 文本缓冲”切到“assistant 分段时间线”：流式文本会在工具/提问事件之间按真实到达顺序拆段插入，`<t:ID>...</t>` 解析也改为输出有序 text/tool 单元，避免运行中出现“工具堆在上面、文本整块压在底部，结束后才对齐”的错位观感。
- Web Console 会话详情已完成窗口化 hydrate：首次进入只加载最近一页，旧消息通过顶部补页按页回拉；补页时保持滚动位置，切换会话时重置到当前会话尾部，本地 `local:` / `stream-*` 临时消息会在尾页同步时和持久化消息做稳定 ID 合并。
- Web Console 聊天输入框初始高度已下调一档：输入区初始行数由 `4` 调整为 `3`，在不影响自动增高的前提下减少默认占用空间。
- Web Console 聊天输入提交交互已改为“先清空再发送”：发送后输入框会立即刷新为空；若发送链路抛错，则自动回填草稿与待发图片，避免内容丢失。
- CLI 基线可用，已支持基础会话与桥接操作。
- CLI 启动入口已完成阶段A路由重构：先按原始 argv 快速判定 `--version`/`-v`/`-V`，再进入 clap；`--message` 与 REPL 路径拆为独立 handler，`main` 保持薄入口。
- CLI 启动观测已完成阶段B：新增 `startup_profiler`，通过 `GHOST_CLI_PROFILE_STARTUP=1` 输出 checkpoint（`entry`、`args_parsed`、`config_loaded`、`client_ready`、`repl_started`、`oneshot_done`）与总耗时；支持 `GHOST_CLI_PROFILE_STARTUP_FILE` 写文件，写入失败显式报错。
- CLI 启动入口与 profiler 已补齐集成回归：新增 `apps/cli/tests/startup_router_test.rs`，覆盖版本快捷路由（`--version`/`-v`/`-V`）、startup profiler 文件输出、profiler 非法环境变量错误、`--message` 空文本错误路径。
- CLI REPL 已补上首轮早输入预填充：在初始化 `rustyline` 前短窗口捕获 TTY 输入并注入首轮 `readline_with_initial`，只做预填充不自动发送；非 TTY 自动禁用，Ctrl+C/Ctrl+D 主行为保持与现有路径一致。
- Android 已接入部分会话与展示能力，但整体成熟度低于 Web 与 CLI。

### Execution: `drivers/native`

- Native 层已支持截图、输入模拟、脚本执行、窗口/浏览器查询等原子动作。
- GUI executor 所需的双击、右键、滚动、拖拽、组合键、活动窗口信息等能力已补齐一轮基线。
- Native 截图子模块已去掉主链路 base64 载荷：`SCREEN_CAPTURE` 返回临时 PNG 文件路径，`OCR_IMAGE` 直接消费 `image_path`，`crop_image` 改为仅复制裁剪区域，避免整图 clone 后再裁切。
- Native 入口阶段D已收口：`--sandbox-worker` / `--persistent` / oneshot 由统一路由决策函数分流，入口执行结果统一为 `{handled,error}` 语义；oneshot `emit` 写出失败改为显式 stderr + 非零退出，未知参数与冲突参数会直接报错。
- Native 启动路由回归已补齐：新增 `drivers/native/src/main_startup_tests.rs`，覆盖默认 oneshot、`--sandbox-worker`、`--persistent` 分流和未知参数/冲突参数错误路径，启动入口决策具备自动化锁定。
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
- 运行时工具暴露继续走 `tool_allowlist` 常驻机制；本地 Bridge 配置已验证可直接将 `script_exec`、`browser_control`、`screen_action`、`computer_use` 与 memory 工具常驻暴露给 Agent，`tfind` 需配合 `tool_search_enabled=true` 才会实际注册。
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
- 回归门禁脚本已固定 backend 单测命令为 `timeout 60s go test ./...`（`scripts/e2e/mvp_regression.sh` 与 `.github/workflows/trinity-check.yml` 同步），避免后端测试任务无界阻塞。
