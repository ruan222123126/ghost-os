# Ghost-OS 当前项目概述

更新日期：2026-03-21  
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

### 2026-03-21

- 收口 Agent 主循环对 GraphQL 文本协议的直接耦合：
  - `core/bridge/agent` 新增通用 `AssistantTextHandler` 扩展点，`Agent` 结构不再直接持有 GraphQL-specific executor 状态。
  - `loop_finish.go` 已移除专门的 GraphQL 文本分支与 GraphQL 常量引用，统一改走 assistant-text handler 分发；GraphQL 文本执行被下沉到独立适配文件。
  - `orchestration` 现通过 `agent.NewGraphQLTextTurnHandler(...)` 注册 GraphQL 文本协议，等待人工事件与执行反馈保持原有行为，但不再污染通用回合驱动器。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./agent ./orchestration -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./... -timeout 60s`

- 修复 GraphQL 文本执行反馈污染会话语义的问题：
  - `core/bridge/agent` 现在把 GraphQL 执行结果以 `internal` 历史角色持久化，不再伪装成 `user` 消息。
  - `core/bridge/llm` 新增 completion 前 role 投影：内部消息只在发给 provider 时映射为 provider-safe assistant 角色，存档、transcript、memory recall 与 turn learning 保持内部边界。
  - `core/bridge/memoryaug` 收紧学习角色过滤，未知/内部角色不再默认归一为 `user`。
  - Web / Android 会话历史解析已支持 `internal`，统一按系统注记展示，不再混入对话气泡。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./agent ./orchestration ./memoryaug ./llm -timeout 60s`
  - `timeout 60s pnpm -C apps/web test -- --runTestsByPath lib/chatMessages.test.ts lib/api/sessions/parser.test.ts`
  - `timeout 60s pnpm -C apps/web exec tsc --noEmit --pretty false`
  - Android 定向单测已尝试：`timeout 300s env GRADLE_USER_HOME=/tmp/gradle-home ./gradlew :app:testDebugUnitTest --tests dev.ghostos.android.viewmodel.ChatViewModelTest`，但当前环境仍停在 Gradle 8.2 wrapper 下载，超时未拿到测试结果。

- 修复 GraphQL text mutation 自动提交打穿 turn 事务边界的问题：
  - `core/bridge/agent` 在 GraphQL 文本写操作成功并生成 execution feedback 后，现会立即 `commitTurn`，不再把该子回合留在未提交历史里等待下一次 completion。
  - `core/bridge/orchestration` 的 `sessionTurnState.complete()` 现会在普通错误返回前，检测并持久化 agent 已显式提交的增量消息，避免出现“GraphQL mutation intent 已 checkpoint.Save，但对话历史没落盘”的分叉。
  - 新增 `agent` / `orchestration` 回归测试，覆盖“GraphQL 文本成功执行后下一次 completion 失败”场景，确认 session mutation 状态与消息历史保持一致。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./agent ./orchestration -timeout 60s`

- 落地“纯文本 GraphQL 执行 + 高风险审批”主链路：
  - Agent 新增 `GraphQLTextExecutor` 执行分支；当模型输出单条 GraphQL `query/mutation` 文本时直接执行，不再依赖 tool-call schema。
  - GraphQL 模式下开启 strict 协议：`finish_reason=tool_calls` 直接报错；并在会话 prompt 注入 `GraphQL Text Protocol`，要求模型输出单条 GraphQL 文档。
  - 新增 `core/bridge/tools/graphql_text_executor*.go`：`query` 走 source/domain 推断后直执行；`mutation` 按 policy 执行（低风险自动 commit，高风险挂起人工审批）。
  - `session_turn_preparer` 在 resume 前会自动扫描 `graphql_text_mutation` 已批准 intent，并自动触发 `commit/retry_commit`，不再依赖模型二次发起提交。
  - human-answer 回放链路支持 `graphql_text_mutation`，GraphQL 审批问答可被正确写回与消费。
  - 修复 GraphQL 文本协议对主 system prompt 的短路：协议块现在只会在默认运行时 prompt / fallback prompt 组装完成后追加，不再替换 `Core Job`、动态工具状态与运行上下文；已补 `session_turn_graphql_prompt_test.go` 回归。

- 新增 mutation policy 风险标记并打通跨端契约：
  - `graphql_mutation_policies` 新增 `approval_required`，已贯通 `config`、`runtime`、`tools/internal/graphqlschema`、schema lookup payload 与 orchestration shim。
  - 更新 `core/shared/schema/defs/config_runtime.json` 并执行 `python3 task.py gen-contracts`，同步 Go/TS/Rust/Kotlin 生成类型。
  - Web config parser 已支持 `approval_required` 字段解析。

- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./... -timeout 60s`
  - `pnpm -C apps/web test`
  - `pnpm -C apps/web exec tsc --noEmit --pretty false`
  - `timeout 60 cargo test --manifest-path apps/cli/Cargo.toml`

- 新增 GraphQL 统一调用文档并收口外部表述：
  - 新增 `docs/graphql-mutation-unified-api.md`，明确“统一接口”发生在业务层（GraphQL Query/Mutation），而非调度层（tool-call 仍保留）。
  - 文档同步收口“写操作语义”：`graphql_mutation` 采用 `prepare -> commit/retry` 两阶段，包含 `status/discard/list_pending`，不再把写链路表述为“裸 mutation 直发”。
  - 补充三类挑战说明：先后依赖、多轮审批、幂等交付与不确定状态追踪，避免把 schema 类型校验误写成端到端执行保障。

- 按当前需求移除 Agent 侧 GraphQL 工具可用性：
  - `core/bridge/runtime/agent_runtime_factory.go` 不再调用 `registerOptionalGraphQLTools`，Bridge 运行时不再注册 `graphql_query` / `graphql_schema_lookup` / `graphql_mutation`。
  - 保留 GraphQL 配置结构与底层实现代码，避免对配置 DTO、持久化和既有模块造成跨层破坏；仅关闭模型可用工具面与执行入口。
  - `core/bridge/runtime/agent_runtime_factory_test.go` 已同步为“GraphQL 配置存在时仍禁用工具”的断言，并覆盖无效 schema/policy 不再影响 runtime Build 的行为。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./runtime ./tools ./config -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./orchestration -timeout 60s`

- 收口工具指引的按需显示条件：
  - `core/bridge/tools/prompt_guidance.go` 的 workspace 指引改为“仅当 `read_and_summarize` 与 `script_exec` 同时在本轮结构化工具 schema 中可见时才注入”，不再在只加载单工具时显示对应文案。
  - 补充 `core/bridge/tools/prompt_guidance_test.go` 回归用例，覆盖“仅单工具可见时不显示 workspace 指引”。
  - 同步更新 `core/bridge/runtime/tool_selector_factory_test.go` 断言，避免继续依赖旧的“单 `script_exec` 也应注入 workspace 指引”语义。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./tools -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./runtime -timeout 60s`

- 精简主 system prompt 默认模板：
  - `core/bridge/prompts.yaml` 已删除 `## Runtime Constraints` 与 `## Response Rules` 两个区块及对应默认文案，默认提示词仅保留 `core_job`、`tool_guidance`、`dynamic_tool_state` 与运行上下文。
  - `core/bridge/context/prompt_default_generated.go` 已通过 `go generate -C core/bridge/context` 同步更新，确保内置 fallback 模板与 `prompts.yaml` 一致。
  - `core/bridge/context/context_test.go` 已更新断言，改为校验上述两区块默认不再出现在 prompt 文本中。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./context ./runtime -timeout 60s`

### 2026-03-15

- 修复 OpenAI-compatible 非流式 tool-call finish reason 漂移：
  - 当上游错误返回 `message.tool_calls` 且 `finish_reason="stop"` 时，Bridge 现在会按 `tool_calls` 处理，不再把空 assistant 文本误判成最终回复并抛出 `agent response message is empty`。
  - Agent 回合分发新增一层显式保护：当前响应只要携带 `tool_calls`，就优先进入工具执行分支，避免再被 provider 的 finish reason 误导。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./llm ./agent ./orchestration ./transport -timeout 60s`
  - 本机 live 回归：`POST http://127.0.0.1:8080/api/agent` 使用工具请求已返回正常文本，不再报 `agent response message is empty`

- 修复 `/api/config` 的 GraphQL 配置响应契约：
  - 当未配置 GraphQL source / mutation policy 时，Bridge 现在稳定返回 `graphql_sources: []` 与 `graphql_mutation_policies: []`，不再序列化成 `null`，避免 Web Console 因 schema mismatch 抛出 `Invalid bridge config.graphql_sources: expected array`。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./config ./transport -timeout 60s`

- 移除 Bridge 配置的 legacy TOML / GraphQL 兼容层：
  - `core/bridge/config` 不再接受顶层 `provider` / `api_key` / `base_url` / `model_provider` / `model_providers` 与旧 `graphql_*` TOML 字段；读取旧配置时改为显式报错，不再自动迁移。
  - GraphQL 配置解析现只保留 `graphql_default_source` / `graphql_sources` / `graphql_mutation_policies` 新布局；旧 `GHOST_GRAPHQL_*` 环境变量也会显式报错。
  - `core/bridge/transport` 已删除启动时的自动配置迁移路径与相关日志。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./config ./transport -timeout 60s`

- 收口 RSS report prompt 的职责边界：
  - `core/bridge/rss/report_prompt.go` 已移除对 `ghost-os/bridge/tools` 的依赖，prompt 文件只保留“输入 -> 提示词字符串”的模板拼装。
  - scoped `ToolCatalog` 到提示词文案的适配已迁回 `core/bridge/rss/report_service.go`，避免 RSS prompt 层反向依赖工具目录实现。
- 清理一组低成本 warning 噪声与冗余导入：
  - `apps/web/lib/api/sessions/parser.ts` 删除未使用的 `parseOptionalStringArray` 导入，避免继续保留可被静态检查直接识别的死代码。
  - `apps/cli` 对 `envelope_generated` 模块局部收口 `dead_code` warning，避免共享契约全量生成的未使用 DTO 持续污染 CLI 的告警基线；同时把 `ConfigUpdate` 的空更新构造改成显式 `ConfigUpdate::empty()`，移除额外的 `derivable_impls` 噪声。
  - `drivers/native` 已折叠 `input/window_guard.rs` 与 `codex_cli/start.rs` 中被 Clippy 点名的嵌套 `if` / `needless return`。
- 本轮验证：
  - `cargo clippy --manifest-path apps/cli/Cargo.toml -- -W dead_code`
  - `timeout 60 cargo test --manifest-path apps/cli/Cargo.toml`
  - `cargo clippy --manifest-path drivers/native/Cargo.toml -- -W clippy::needless_return -W clippy::collapsible_if`
  - `pnpm -C apps/web exec tsc --noEmit --pretty false`

- 清理 `config` / `memoryaug` / `orchestration` / `tools` 中零散死 helper：
  - 删除仅剩定义、无实际调用的 `cloneStringPointer`、`normalizeCandidateMemoryType`、`requireTaskID`、`normalizeScheduledTask`、`newTaskRunID`、`sourceHasMutationPolicies`、`cloneParams`，避免继续保留误导性的包内包装与未使用小工具。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./config ./memoryaug ./orchestration ./tools -timeout 60s`

- 清理 orchestration 与 RSS 间已漂移的重复文本工具实现：
  - 删除 `core/bridge/orchestration/textutil.go` 中未被调用的 `effectiveWorkerModel` 与 `stripJSONCodeFence`，避免与 `core/bridge/rss/export.go` 中仍在实际使用的实现继续双份维护。
  - orchestration 侧仅保留仍有调用点的 `truncateRunes`，RSS 侧继续复用自身导出的文本辅助函数。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./rss ./orchestration -timeout 60s`

- 删除 `core/bridge/orchestration/config_shim_env.go` 遗留空包装：
  - `rss_inbox_system_tasks.go` 与 `service_router.go` 已改为直接调用 `bridgeconfig.Load()` / `bridgeconfig.LoadWithRuntime()` / `bridgeconfig.TasksPathFromEnv()`，`orchestration` 不再维护仅转发 `core/bridge/config` 的 env shim。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./orchestration -timeout 60s`

- 清理 `core/bridge/runtime/export.go` 未调用私有包装：
  - 删除 `parseStringCSV`、`getenvDefault`、`toolCatalogNames` 三个仅做转发且无调用点的私有 helper，保留现有公开导出面不变，避免 runtime 导出层继续保留无效包装。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./runtime -timeout 60s`

- 清理 transport SSE 死代码：
  - `core/bridge/transport/transport_sse.go` 已删除未被任何 transport 路径调用的 `eventTurnTracker`、`finalAssistantTurn`、`emitStreamErrorEvent` 及其连带 helper，避免 SSE 入口继续保留误导性分支。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./transport -timeout 60s`

- 收口 CLI config DTO 分叉：
  - `apps/cli` 已删除手写 `ConfigResponse` / `ConfigUpdate`，改为直接复用共享生成的 `BridgeConfig` / `ConfigUpdate`。
  - `client`、命令执行链与终端渲染测试现统一消费共享契约；后续 config schema 扩字段时，CLI 不再保留独立 DTO 漂移入口。
  - CLI 侧仅补了一个本地 `ConfigUpdate::default()` 实现用于命令构造；如果共享契约继续增字段，这里会在编译期显式暴露未覆盖项。
- 本轮验证：
  - `timeout 60 cargo test --manifest-path apps/cli/Cargo.toml`

- 收口流式协议的共享契约来源：
  - `core/shared/schema.json` 与新增 `core/shared/schema/defs/streaming_events.json` 现纳入 `agent stream` / `session push` 的 envelope 与核心 payload 定义，不再只有 request/response DTO 在共享 schema 内。
  - 重新生成 Go / TS / Rust / Kotlin 契约后，Android 已删除手写 `SessionPushModels.kt`，直接使用共享生成的 `AgentStreamEvent`、`SessionPushEvent` 及其 payload 模型。
  - `core/bridge/orchestration` 已移除手写 `assistantMessagePushPayload` / `awaitingHumanPushPayload`，并新增 `stream_contract_test.go` 校验手写实时事件 envelope 仍与共享生成契约兼容。
- 本轮验证：
  - `python3 -m unittest core/shared/tests/test_schema_loader.py core/shared/tests/test_emitters.py`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./orchestration ./transport -timeout 60s`
  - `timeout 180s ./gradlew testDebugUnitTest --tests dev.ghostos.android.network.BridgeClientTest --tests dev.ghostos.android.viewmodel.ChatViewModelTest`

- 收口共享契约 codegen 的 object 语义漂移：
  - `core/shared/contract_codegen` 现区分三类 object：具名结构体、`additionalProperties` typed map、以及真正 free-form object；TS/Go/Rust/Kotlin 不再把 `headers`、provider token override 这类 typed map 统一降成 `unknown/any/Value/JsonObject`。
  - `core/shared/schema/defs/agent_core.json` 新增具名 `agentIterationSummaryItem`，补上此前被 inline object 吃掉的迭代摘要结构；各端生成产物现直接暴露强类型迭代摘要。
  - `core/bridge/orchestration` 已移除围绕 GraphQL headers / provider override 的 `any` 中转与手工再解码，直接使用生成后的 `map[string]string` / `map[string]int`；Web parser 也同步改为显式校验 string/number record。
- 本轮验证：
  - `python3 -m unittest discover -s core/shared/tests`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./orchestration ./config ./tools -timeout 60s`
  - `timeout 60 cargo test --manifest-path apps/cli/Cargo.toml`
  - `pnpm -C apps/web test -- --runTestsByPath lib/api/config/api.test.ts`
  - `pnpm -C apps/web exec tsc --noEmit`

- 修复 GraphQL 配置更新链路的 header 类型漂移：
  - `core/shared/contract_codegen/emitters/go.py` 现按 `additionalProperties` 生成强类型 map；`graphqlSourceInput.headers` / `graphqlSourceResponse.headers` 已回到 `map[string]string`，无效 header value 会在 orchestration 参数解码阶段显式失败。
  - `core/bridge/orchestration/config_shim.go` 已移除 `any -> string -> nil` 的静默降级路径，GraphQL source header 不再因非法值被整块清空。
  - `core/bridge/orchestration/service_config_runtime_test.go` 新增回归，覆盖“非法非字符串 header 返回 400，且既有 header 保持不变”。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./orchestration ./config -timeout 60s` 已通过。

- 统一 Web 客户端响应解析的前向兼容语义：
  - `apps/web/lib/api/shared.ts` 新增“只提取已知字段”的 parser helper，`agent/config/rss/sessions` 解析器不再因 bridge 新增字段直接报错，改为忽略未知字段并继续校验已知字段类型与必填项。
  - `apps/web` 已补 config/rss/sessions 定向回归，覆盖“新增字段不破坏旧客户端”与“缺少 canonical 必填字段仍显式报错”两类边界，和文档及 Android `ignoreUnknownKeys = true` 语义对齐。
- 本轮验证：
  - `timeout 60s pnpm -C apps/web test -- --runTestsByPath lib/api/agent/api.test.ts lib/api/config/api.test.ts lib/api/config/parser.test.ts lib/api/rss/api.test.ts lib/api/rss/parser.test.ts lib/api/sessions/api.test.ts lib/api/sessions/parser.test.ts`

- 收口 `tfind` 的模型侧工具面管理语义：
  - `tools.FormatPromptGuidanceForCatalog` 新增 `tfind` 专用 workflow guidance，明确 `search -> load -> next turn use -> list/unload` 的使用顺序与同轮限制。
  - `core/bridge/orchestration` 新增仅作用于模型输入的 history projection；会话存储仍保留完整 `tfind` assistant/tool 原始 JSON，下一轮喂给模型时只把 `load` / `list` / `unload` span 压成简短 assistant 摘要。
  - `core/bridge/runtime` 新增 `Dynamic Tool State` prompt 区块，按 session 当前 `dynamic_tool_loads` 注入 pending/active/expired 状态；基础回合与 selector subset 回合现都按当前 session catalog 重建 prompt。
  - `core/bridge/context` 默认 prompt 渲染已补齐 `dynamic_tool_state` 缺省值，并重新生成内置 `prompts.yaml` 默认模板。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go generate -C core/bridge/context`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./context -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./tools -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./runtime -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./orchestration -timeout 60s`

- 收口主 prompt 模板并消除双份维护：
  - `core/bridge/prompts.yaml` 现只保留最小骨架，`runtime_constraints` 与 `response_rules` 改为独立 section；默认模板不再常驻写入 RSS pipeline、`screen_action` 细分说明或 `END_SESSION` 协议。
  - `core/bridge/context` 改为从单一 `prompts.yaml` 真源生成内置默认配置，`prompt.go` 不再手写第二份同构长模板。
  - `PromptLoadOptions` 与 runtime/config 已新增 `prompts_runtime_constraint_files`、`prompts_response_rule_files`，真正支持把重块 prompt 片段外置拼装。
  - `tools.FormatPromptGuidanceForCatalog` 现按当前 visible catalog 动态注入 RSS、`ask_human`、`screen_action` 等低频规则，不再把这些说明塞进 every-turn always-on prompt。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./context -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./config ./context ./tools ./runtime -timeout 60s` 仍受当前工作区现有 GraphQL 编译错误阻塞：`tools/graphql_mutation_commit_execute.go:56 source.Headers undefined`

- 收口 memory recall prompt 注入块：
  - `memoryaug.FormatPromptBlock` 现保留 `Memory slots` 与最多 2 条 `Other memory context`，非 slot 记忆只注入 `summary`，每条限制 80 字，不再向 system prompt 追加 `content=`。
  - 已补 formatter 与 session turn 的定向回归，覆盖“只保留短块、不落会话历史、超额条目丢弃、长 summary 截断”。
- 本轮验证：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./memoryaug -run 'TestFormatPromptBlock' -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./orchestration -run 'TestSessionRunner(InjectsRecallOnlyIntoPrompt|InjectsCompactRecallPromptBlock|RecallDoesNotBreakAskHumanContinuation|BuildsRecallQueryFromRecentContext)' -timeout 60s` 仍受当前工作区现有 GraphQL 编译错误阻塞：`tools/graphql_mutation_commit_execute.go:56 source.Headers undefined`

- 收口主 system prompt 与工具提示职责：
  - 主 system prompt 不再静态写死工具名单、`tools_count` / `tool_list` 或 `read_file` / `search_files` / `apply_diff` 这类工具级限制，改为基于当前 catalog 注入最小 `tool_guidance`。
  - 基础回合与 selector subset 回合现共用同一套 system prompt 装配逻辑，避免 prompt 继续提到已被裁掉的工具。
  - RSS report 调查 prompt 改为只按当前 scoped catalog 动态提示可用工具；少数跨工具引用的 tool description 也已收口为“仅依赖当前可用能力”的表述。

- 修复 tool selector 子集回合的 system prompt 变量漏传：
  - `buildSystemPromptForCatalog` 现在会补齐 `project_root`，避免 selector 选出工具子集后重建 prompt 时残留 `{{project_root}}` 占位符。
  - 已新增 runtime 回归测试覆盖该路径。

- 完成 GraphQL 第二步基线：
  - `core/bridge/config` 从单 source 升级为 `graphql_default_source + graphql_sources[]`，保留旧单源字段只读迁移入口；旧配置读取时显式物化为 `default` source，经新接口更新后仅持久化新结构。
  - `core/bridge/runtime` 启动时为所有 GraphQL source 预加载 schema snapshot，并通过共享只读 registry 向 `graphql_query` / `graphql_schema_lookup` 提供多 source 路由能力；工具仍保持 `OnDemand: true`，默认不进入静态工具面。
  - `graphql_schema_lookup` 新增 `list_sources`、`list_domains`，并支持按 `source/domain` 裁剪 `list_root_queries`、`describe_type`、`find_field` 可见面。
  - `graphql_query` 升级为 `source/domain` aware，执行前基于 AST 做 operation 选择、domain root field 校验、深度/字段数/root field/fragment 静态预算校验，并拒绝 `__schema` / `__type` introspection 字段。
  - GraphQL 工具补齐结构化 trace 日志；成功/失败都会记录 `trace_id`、`source`、`domain`、`operation`、query stats、耗时、响应大小与错误摘要。
  - 共享契约与 Web parser 已同步到多 source 模型；snapshot 仅暴露 `api_key_set`，不返回任何 source 明文 key。
- 第二步相关验证已通过：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./config ./runtime ./tools ./orchestration -timeout 60s`
  - `pnpm -C apps/web test -- --runTestsByPath lib/api/config/api.test.ts`

- 完成 GraphQL 第三步提交链路：
  - schema snapshot 补齐 `root_mutations`，GraphQL registry 新增 mutation policy allowlist 解析与 source/domain/root mutation 路由；非法 policy 会在 registry 构建阶段显式失败。
  - `core/bridge/config` 新增 `graphql_mutation_policies[]` 契约，并同步到 runtime snapshot、transport shim、Web parser 与共享 envelope codegen，不引入“默认允许所有 mutation”的模式。
  - session 新增持久化 `PendingGraphQLMutationIntents`，human question 新增 `ToolName`，已回答问题回放不再硬编码成 `ask_human`；`graphql_mutation` 会把批准结果稳定注入为自身 tool result。
  - 新增按需加载的 `graphql_mutation` 工具，支持 `prepare` / `commit` / `discard` / `list_pending`；`prepare` 只校验与挂起审批，`commit` 只接受 `intent_id` 并执行 prepare 时冻结的 mutation 文本与变量。
  - `graphql_schema_lookup` 新增 `list_root_mutations` 与 `describe_mutation_policy`，仅暴露 allowlist 允许的 mutation root 与预算摘要；GraphQL mutation trace 已覆盖 `prepare` / `commit` / `discard`。
- 第三步相关验证已通过：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./session ./config ./runtime ./tools ./orchestration -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./tools/... -timeout 60s`
  - `pnpm -C apps/web test -- --runTestsByPath lib/api/config/api.test.ts`

- 完成 GraphQL 第四步交付安全：
  - mutation policy 现强制声明 `idempotency_mode`，仅支持 `header` 与 `variable_path`；配置、registry、schema lookup、runtime snapshot 与共享契约已同步幂等字段，无显式策略的写 policy 会显式失败。
  - `graphql_mutation.prepare` 现冻结 `delivery_key`、注入后的 variables 与 `request_hash`；intent 持久化补齐 `CommitState`、attempt 计数、最新错误摘要、响应 hash/bytes 与统一 `Receipts[]` 回执结构。
  - 工具上下文新增最小 `SessionCheckpoint`；`commit` / `retry_commit` 会先把 intent 落成 `committing` 再发请求，成功落 `executed`，网络超时/连接中断/响应不确定统一落 `delivery_unknown`，且所有尝试都会追加 receipt。
  - `graphql_mutation` 新增 `status` 与 `retry_commit`；retry 仅允许 `approved` 或 `delivery_unknown`，并强制复用冻结的 `delivery_key` 与 `request_hash`，已 `executed` 的 intent 仍不可重复提交。
  - GraphQL mutation trace 现补齐 `delivery_key`、`commit_state`、`attempt`、`request_hash`、`response_hash`、`http_status`；`delivery_unknown` 会单独打点，便于补偿与审计。
- 第四步相关验证已通过：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./config ./session ./tools ./runtime ./orchestration -timeout 60s`
  - `pnpm -C apps/web test -- --runTestsByPath lib/api/config/api.test.ts`
  - `python3 -m unittest discover -s core/shared/tests -p 'test_*.py'`

- 新增只读 GraphQL 查询工具：
  - `graphql_query` 仅允许 `query`，在执行前做 GraphQL 文本校验，拒绝 `mutation` / `subscription` / 混合操作。
  - 请求固定走单一 endpoint，支持配置级 headers / API key、超时与响应体大小限制，HTTP 与 GraphQL errors 均显式失败。
- 新增本地 GraphQL schema lookup：
  - `graphql_schema_lookup` 基于本地 schema snapshot 启动时加载，不访问网络，不做运行时 introspection。
  - 新增 `core/bridge/tools/internal/graphqlschema` 负责 snapshot 读取、校验与只读索引检索。
- 默认不暴露，支持按需加载：
  - GraphQL 工具在配置合法时注册到 registry，但默认不进入静态工具面。
  - `tfind` 可检索并按需加载 `graphql_query` 与 `graphql_schema_lookup`；selector 默认不看到未加载的 GraphQL 工具。
- 相关测试通过：
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./config ./tools ./runtime -timeout 60s`
  - `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./tools/... -timeout 60s`

### 2026-03-14

- 新增会话级动态工具搜索与装载基线：
  - `core/bridge` 新增默认关闭的 `tfind` 工具；启用后主 Agent 默认可见，但 `ToolSelector` 不再看到或选择它。
  - 会话新增 `turn_index` 与 `dynamic_tool_loads`，支持“本轮 load、下轮生效、连续 3 轮未调用自动卸载、Agent 可主动 unload”。
  - 工具可见性改为“静态默认暴露 + 会话动态装载”双层目录；系统提示词新增当前可用工具最简清单，不再只提供工具数量。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./config ./session ./tools ./runtime ./context ./agent -timeout 60s` 已通过；`./orchestration` 仍受仓库现存 `core/bridge/rss` 重复定义阻塞。
- 收口 `core/bridge/orchestration` 第一批入口文件：
  - 将 `service_rss_inbox.go`、`service_tasks.go`、`pro_mode.go`、`session_push.go`、`service_usecase_agent.go`、`service_usecase_human.go` 拆成“薄入口 + usecase runner + adapter”结构。
  - RSS inbox / briefing、任务 CRUD / run-now、pro/prox 迭代引擎、ask_human 续跑与 session push 广播已分别下沉到独立协作者；`bridgeService` 现主要保留 action 入口、依赖解析与状态码映射。
  - `config_shim.go` 与 `core/bridge/transport/orchestration_shim.go` 已拆分为更小的 shim 文件，避免 transport / orchestration 再承载大块转发代码。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./... -timeout 60s` 已通过。
- 收口 `core/bridge/agent` 对话循环实现：
  - 将 `loop.go` 拆为“薄入口 + 回合处理 / 运行态 / 历史提交 helper”结构，保留 `Agent` 对外接口与错误语义不变。
  - 本轮未预拆 `loop_tool_test.go`、`task_scheduler_test.go`；仅调整生产代码边界，避免把大测试文件体积误判为首要风险。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./... -timeout 60s` 已通过。
- 收口 `core/bridge/session` 会话裁剪职责：
  - 将 `pruning.go` 拆为 context limit resolver、token estimator、message pruner、message truncation 等纯策略组件。
  - 保留 `EstimateTokens`、`PruneMessages`、`GetContextLimit` 作为薄入口，避免影响 `session` 与 `orchestration` 调用面。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./session ./orchestration -timeout 60s` 已通过。
- 收口 `session/config/runtime` 第二批状态与策略边界：
  - `core/bridge/session` 现拆为会话实体、迭代态、ask_human 状态、ID/debug 辅助与存储编解码/路径处理，`session.go`、`storage.go` 不再承载全部细节。
  - `core/bridge/config` 现拆分文件模型、TOML 读写、写前归一化、provider 编解码、路径/字符串规整与 env/default 取值解析，避免 `config_file*.go` 继续成为全局杂糅点。
  - `core/bridge/runtime/tool_selector` 现拆为薄执行入口、构造工厂、prompt 组装与响应解析/合法性校验，工具选择策略边界更清晰。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./... -timeout 60s` 已通过。
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
  - 将 `script_exec.rs` 拆为 `script_exec/{mod,budget,worker,result,types}.rs`，分离协议入口、预算裁剪、sandbox worker 生命周期与结果反解；将 `file_actions.rs` 拆为 `file_actions/{mod,params,handlers,format,tests}.rs`，分离文件协议参数解析与返回映射。
- 收口 `apps/cli` 命令层边界：
  - 将 `commands.rs` 拆为 `commands/*` 目录模块，分离命令解析、配置更新映射、本地副作用执行与终端渲染，保留 `repl` 侧兼容入口。
  - 将 `repl.rs` 拆为 `repl/{mod,input,ask_human,view}.rs`，分离输入路由、命令调用、`ask_human` 续跑与终端输出，避免后续执行链路调整同时触碰 CLI 交互层。
  - 为命令解析、执行结果映射、清屏行为与输出格式新增定向测试；当前环境下 `timeout 60 cargo test --manifest-path apps/cli/Cargo.toml -- --nocapture` 已通过。
  - 当前环境下 `timeout 60 cargo test --manifest-path drivers/native/Cargo.toml` 与 `timeout 60 cargo test --manifest-path apps/cli/Cargo.toml` 已通过。
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
- 收口第三批有状态仓储与执行通道：
  - `core/bridge/rss/rss_inbox_store.go` 现拆为薄仓储入口 + item 归一化 + list query + snapshot 读写，避免 store 再混查询过滤与去重索引细节。
  - `core/bridge/rss/rss_inbox_aggregate.go` 现拆为聚合入口、query spec、聚合规则、结果组装，避免 aggregate 同时承载参数规整和 group materialize。
  - `core/bridge/tasks/store.go` 现拆为任务读写、任务扫描查询、run log 仓储，避免任务定义与执行日志继续共享单个巨型 store 文件。
  - `core/bridge/execution/persistent_client.go` 现拆为协调入口、协议封包交换、进程生命周期、状态轮询，persistent call/fallback/handshake 边界更清晰。
  - 当前环境下 `env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./rss ./tasks ./execution -timeout 60s` 已通过；`env GOCACHE=/tmp/go-build GOTMPDIR=/tmp/go-tmp go test -C core/bridge ./... -timeout 60s` 仍受仓库现存 `./orchestration` allowlist 测试 `TestSessionTurnPreparer_SelectToolsForTurn_AppliesAllowlistToSubset` 失败阻塞。
- 收紧与清理工具面：
  - Agent 默认暴露面以 `script_exec` 为主入口。
  - `memory_manage`、`browser_control`、`script_exec`、`send_file` 等工具的输出与元数据更稳定。
- 修复 Web 到 bridge 代理层的 traceability / 流式边界回归：
  - `apps/web/lib/server/bridge/forward.ts` 不再重建 bridge 响应，不再把普通 API 响应 `text()` 缓冲后只回写 `Content-Type`。
  - `/api/*` 与 artifact 下载代理现都直接透传 bridge `Response`，保留 `X-Trace-ID` 等响应头，并允许浏览器侧消费流式 body。
  - 已补 Web 侧回归测试，覆盖普通代理与下载代理的 header/body 透传语义。
- 本轮验证：
  - `pnpm -C apps/web test -- --runTestsByPath lib/server/bridge/index.test.ts`

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
