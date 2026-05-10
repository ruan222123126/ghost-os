# Orchestration Map

更新时间：2026-05-10

## 范围

本盘点只覆盖 `core/bridge/orchestration` 第一层 Go 文件，共 169 个文件：118 个生产文件，51 个测试文件。

不纳入主清单的相邻范围：

- `core/bridge/orchestration/internal/**`：已有初步 internal 分层，共 103 个文件，新增 `internal/app/trace`、`internal/domain/sessionturn` 与 `internal/domain/task` 承接本轮下沉。
- `apps/web/**/orchestration*`：Web 编辑器与 API facade，不属于本轮 Central orchestration 顶层拆分范围。
- `.next*` 产物、构建缓存、非 Go 文件不计入。

目标子域：

- `loop/`：Agent 循环主流程，包含 session turn、plan/relay、group orchestration runner。
- `workflow/`：静态 workflow 编排、图校验、节点执行。
- `policy/`：重试、超时、运行时配置、校验、allowlist/blocklist、生命周期策略。
- `dispatch/`：HTTP/bus/service facade、任务入口、工具与 RSS action 分派。
- `trace/`：`trace_id`、流式事件、run/session 投影、transcript 与 node result。

## 统计

| Subdomain | Files |
| --- | ---: |
| `loop/` | 67 |
| `workflow/` | 17 |
| `policy/` | 21 |
| `dispatch/` | 52 |
| `trace/` | 12 |
| Total | 169 |

## 已清理

| File | Evidence |
| --- | --- |
| `pro_mode.go` / `pro_mode_prompt.go` / `pro_mode_runner.go` | 已删除不可达 pro runner/parser；`agentturn` special mode 仅保留 plan。 |
| `pro_mode_test.go` | 已删除 pro parser/runner 死测；`pro ...` 前缀按普通消息处理的兼容测试移至 `agent_standard_mode_test.go`。 |
| `session_microcompact_summary_tfind.go` | 已重命名为 `session_microcompact_summary_sfind.go`，行为与摘要输出不变。 |

非死代码但需标注：

- RSS 相关文件仍被 Web/API/transport 使用，不标死代码。
- `service_result_legacy.go` 仍被 task/RSS adapter 使用，不标死代码。

## 文件归属

| File | Target | Status | Notes |
| --- | --- | --- | --- |
| `agent_contract.go` | `loop/` | `active` | agent turn 输入、响应或 app adapter。 |
| `agent_contract_test.go` | `loop/` | `active` | agent turn 输入、响应或 app adapter。 |
| `agent_input.go` | `loop/` | `active` | agent turn 输入、响应或 app adapter。 |
| `agent_input_runner.go` | `loop/` | `active` | agent turn 输入、响应或 app adapter。 |
| `agent_input_test.go` | `loop/` | `active` | agent turn 输入、响应或 app adapter。 |
| `agent_prompt_test.go` | `loop/` | `active` | agent turn 输入、响应或 app adapter。 |
| `agent_runtime_test_helpers_test.go` | `loop/` | `active` | 通用 agent runtime 测试 helper。 |
| `agent_standard_mode_test.go` | `loop/` | `active` | `pro ...` 前缀按普通消息处理的兼容测试。 |
| `agent_turn_app_adapter.go` | `loop/` | `active` | agent turn 输入、响应或 app adapter。 |
| `agent_usecase_helpers.go` | `loop/` | `active` | agent turn 输入、响应或 app adapter。 |
| `api_types.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `config_public_runtime.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `config_public_runtime_test.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `config_request_mapper.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `envelope_generated.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `export_runtime.go` | `loop/` | `active` | agent loop 运行时装配或请求级运行时上下文。 |
| `export_service.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `export_service_sessions_partitions.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `export_types.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `logging.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `trace_compat.go` | `trace/` | `active` | 顶层 trace 兼容薄层，具体实现已下沉到 `internal/app/trace` 与 `internal/domain/*`。 |
| `plan_mode.go` | `loop/` | `active` | plan-only agent loop。 |
| `plan_mode_test.go` | `loop/` | `active` | plan-only agent loop。 |
| `relay_mode_catalog.go` | `loop/` | `active` | relay task agent 循环。 |
| `relay_mode_prompt.go` | `loop/` | `active` | relay task agent 循环。 |
| `relay_mode_result.go` | `loop/` | `active` | relay task agent 循环。 |
| `relay_mode_rounds.go` | `loop/` | `active` | relay task agent 循环。 |
| `relay_mode_rounds_test.go` | `loop/` | `active` | relay task agent 循环。 |
| `relay_mode_runner.go` | `loop/` | `active` | relay task agent 循环。 |
| `request_runtime_options.go` | `loop/` | `active` | agent loop 运行时装配或请求级运行时上下文。 |
| `request_runtime_options_test.go` | `loop/` | `active` | agent loop 运行时装配或请求级运行时上下文。 |
| `rss_report_builder_runtime.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `rss_runtime_adapter.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `run_registry_test.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `runtime_adapter.go` | `loop/` | `active` | agent loop 运行时装配或请求级运行时上下文。 |
| `service_action_codec.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_action_contract_test.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_action_router.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_config_providers.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `service_config_runtime.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `service_config_runtime_test.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `service_lifecycle.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `service_presets.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_prompts.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_result.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_result_adapters.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_result_legacy.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_router.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_router_actions.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_router_task_scope_test.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_rss_inbox.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_runtime_state.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `service_session_guards.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `service_session_guards_test.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `service_sessions.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_sessions_partitions.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_skills.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_skills_discovery.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_skills_security.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_skills_test.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_tasks.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_tools.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_tools_find_icon.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_tools_find_icon_preview.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_tools_find_icon_test.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_tools_mouse_position.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_tools_schema.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_tools_schema_test.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_usecase_agent.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_usecase_agent_stream.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `service_usecase_human.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `session_contract.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_contract_draft.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_contract_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_contract_thinking.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_end_signal.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_end_signal_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_history_builder.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_history_builder_deepseek_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_history_builder_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_history_projection.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_history_sanitize.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_human_tool_results.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_microcompact.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_microcompact_projection_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_microcompact_summary_helpers.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_microcompact_summary_screen.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_microcompact_summary_sfind.go` | `loop/` | `active` | sfind tool 摘要。 |
| `session_microcompact_summary_text.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_push_adapter.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `session_resume_runner.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_runner.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_runner_adapter.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_runner_image_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_runner_precreate_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_stream_checkpoint_test.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `session_title.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_title_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_committer.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_draft_projector_test.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `session_turn_preparer.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_preparer_human_resume.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_preparer_prompt.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_preparer_prompt_refresh.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_preparer_prompt_refresh_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_preparer_selector.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_runtime_overrides.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_state.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `session_turn_state_test.go` | `loop/` | `active` | session turn loop、历史构建或上下文裁剪。 |
| `skill_adapter.go` | `loop/` | `active` | agent loop 运行时装配或请求级运行时上下文。 |
| `stream_contract_test.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `stream_terminal_buffer_test.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `streaming_test_helpers_test.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `task_api_types.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_bridge.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_contract_schema_test.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_kind_validation_test.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `task_list_params.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_mutation_runner.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_mutation_runner_delete.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_mutation_runner_delete_test.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_mutation_runner_test.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_orchestration_baseline_test.go` | `loop/` | `active` | group orchestration 执行 loop。 |
| `task_orchestration_dispatch_tool.go` | `dispatch/` | `active` | 群主调度工具名桥接。 |
| `task_orchestration_member_adapter.go` | `loop/` | `active` | group orchestration 执行 loop。 |
| `task_orchestration_owner_group.go` | `loop/` | `active` | group orchestration 执行 loop。 |
| `task_orchestration_owner_runtime.go` | `loop/` | `active` | group orchestration 执行 loop。 |
| `task_orchestration_private_send_test.go` | `loop/` | `active` | group orchestration 执行 loop。 |
| `task_orchestration_rounds.go` | `loop/` | `active` | group orchestration 执行 loop。 |
| `task_orchestration_runner.go` | `loop/` | `active` | group orchestration 执行 loop。 |
| `task_orchestration_runtime_test.go` | `loop/` | `active` | group orchestration 执行 loop。 |
| `task_orchestration_standard.go` | `loop/` | `active` | group orchestration 执行 loop。 |
| `task_orchestration_validation.go` | `policy/` | `active` | orchestration 定义归一化与校验策略。 |
| `task_orchestration_validation_test.go` | `policy/` | `active` | orchestration 定义归一化与校验策略。 |
| `task_query_runner.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_relay_node_result.go` | `trace/` | `active` | trace_id、流式事件、run/session 投影或日志记录。 |
| `task_run_transcript.go` | `trace/` | `active` | run transcript 顶层保存 wrapper；纯投影已下沉 `internal/domain/task`。 |
| `task_runtime_config.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `task_runtime_config_test.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `task_runtime_overrides.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `task_runtime_overrides_test.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `task_usecase_runner.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_usecase_runner_build.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_usecase_runner_patch.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_usecase_runner_session.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `task_validation.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `task_workflow_find_icon_references.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_find_icon_templates.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_graph.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_input_validation.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_node_results.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_nodes_test.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_plan.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_runner.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_runner_nodes.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_runner_nodes_find_icon_test.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_runner_nodes_screen_control_refs_test.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_runner_nodes_screen_control_steps_test.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_runner_paths.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_runtime_test.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_screen_control_refs.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_test.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `task_workflow_validation.go` | `workflow/` | `active` | 静态 workflow 图、节点执行或测试。 |
| `test_helpers_test.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `textutil.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `tool_api_types.go` | `dispatch/` | `active` | HTTP/bus/service facade、任务调度入口或工具/RSS 分派。 |
| `tool_lists_shim.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `tool_policy_test.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
| `tool_policy_test_helper_test.go` | `policy/` | `active` | 配置、运行时覆盖、校验或生命周期策略。 |
