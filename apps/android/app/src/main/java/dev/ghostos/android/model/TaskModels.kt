// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

package dev.ghostos.android.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
sealed interface TaskCreateRequest

@Serializable
sealed interface TaskPayload

@Serializable
data class TaskRuntimeOverrides(
    @SerialName("provider_name")
    val providerName: String? = null,
    val model: String? = null,
    @SerialName("system_prompt")
    val systemPrompt: String? = null,
    @SerialName("preset_id")
    val presetId: String? = null,
    @SerialName("tool_allowlist")
    val toolAllowlist: List<String>? = null,
    @SerialName("tool_allowlist_only")
    val toolAllowlistOnly: Boolean? = null,
    @SerialName("max_turns")
    val maxTurns: Int? = null
)

@Serializable
data class TaskRelayConfig(
    @SerialName("stop_policy")
    val stopPolicy: String,
    @SerialName("max_rounds")
    val maxRounds: Int,
    @SerialName("execution_timeout_ms")
    val executionTimeoutMs: Int? = null
)

@Serializable
data class AgentMessageTaskCreateRequest(
    val message: String,
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("runtime_overrides")
    val runtimeOverrides: TaskRuntimeOverrides? = null,
    @SerialName("agent_mode")
    val agentMode: String? = null,
    val relay: TaskRelayConfig? = null,
    @SerialName("task_kind")
    val taskKind: String? = null,
    @SerialName("interval_seconds")
    val intervalSeconds: Int? = null,
    @SerialName("cron_expr")
    val cronExpr: String? = null,
    @SerialName("trace_id")
    val traceId: String? = null
) : TaskCreateRequest

@Serializable
data class AgentMessageTaskPayload(
    val id: String,
    val message: String,
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("runtime_overrides")
    val runtimeOverrides: TaskRuntimeOverrides? = null,
    @SerialName("agent_mode")
    val agentMode: String? = null,
    val relay: TaskRelayConfig? = null,
    @SerialName("task_kind")
    val taskKind: String,
    @SerialName("schedule_type")
    val scheduleType: String,
    @SerialName("interval_seconds")
    val intervalSeconds: Int? = null,
    @SerialName("cron_expr")
    val cronExpr: String? = null,
    val enabled: Boolean,
    @SerialName("created_at")
    val createdAt: String,
    @SerialName("updated_at")
    val updatedAt: String,
    @SerialName("last_run_at")
    val lastRunAt: String? = null,
    @SerialName("next_run_at")
    val nextRunAt: String? = null,
    @SerialName("last_error")
    val lastError: String? = null
) : TaskPayload

@Serializable
data class TaskUpdateRequest(
    val id: String,
    val message: String? = null,
    val name: String? = null,
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("runtime_overrides")
    val runtimeOverrides: TaskRuntimeOverrides? = null,
    @SerialName("agent_mode")
    val agentMode: String? = null,
    val relay: TaskRelayConfig? = null,
    @SerialName("task_kind")
    val taskKind: String? = null,
    val workflow: WorkflowDefinition? = null,
    val orchestration: OrchestrationDefinition? = null,
    @SerialName("interval_seconds")
    val intervalSeconds: Int? = null,
    @SerialName("cron_expr")
    val cronExpr: String? = null,
    val enabled: Boolean? = null,
    @SerialName("trace_id")
    val traceId: String? = null
)

@Serializable
data class TaskPatchRequest(
    val message: String? = null,
    val name: String? = null,
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("runtime_overrides")
    val runtimeOverrides: TaskRuntimeOverrides? = null,
    @SerialName("agent_mode")
    val agentMode: String? = null,
    val relay: TaskRelayConfig? = null,
    @SerialName("task_kind")
    val taskKind: String? = null,
    val workflow: WorkflowDefinition? = null,
    val orchestration: OrchestrationDefinition? = null,
    @SerialName("interval_seconds")
    val intervalSeconds: Int? = null,
    @SerialName("cron_expr")
    val cronExpr: String? = null,
    val enabled: Boolean? = null,
    @SerialName("trace_id")
    val traceId: String? = null
)

@Serializable
data class TaskRunNodeResult(
    @SerialName("node_id")
    val nodeId: String,
    @SerialName("node_type")
    val nodeType: String,
    val status: String,
    @SerialName("started_at")
    val startedAt: String? = null,
    @SerialName("finished_at")
    val finishedAt: String? = null,
    @SerialName("completed_seq")
    val completedSeq: Int? = null,
    @SerialName("branch_id")
    val branchId: String? = null,
    val input: JsonElement? = null,
    val output: JsonElement? = null,
    val preview: String? = null,
    val error: String? = null
)

@Serializable
data class TaskRunCard(
    @SerialName("card_id")
    val cardId: String,
    @SerialName("run_id")
    val runId: String? = null,
    val kind: String,
    val title: String? = null,
    @SerialName("node_id")
    val nodeId: String? = null,
    @SerialName("node_type")
    val nodeType: String? = null,
    val round: Int? = null,
    val iteration: Int? = null,
    @SerialName("branch_id")
    val branchId: String? = null,
    @SerialName("source_session_id")
    val sourceSessionId: String? = null,
    @SerialName("started_at")
    val startedAt: String? = null,
    val status: String? = null,
    @SerialName("finished_at")
    val finishedAt: String? = null,
    val preview: String? = null,
    val error: String? = null,
    @SerialName("final_text")
    val finalText: String? = null,
    @SerialName("source_events")
    val sourceEvents: List<AgentStreamEvent>? = null
)

@Serializable
data class TaskRunLog(
    @SerialName("task_id")
    val taskId: String,
    @SerialName("run_id")
    val runId: String,
    @SerialName("trace_id")
    val traceId: String,
    @SerialName("task_kind")
    val taskKind: String? = null,
    @SerialName("scheduled_at")
    val scheduledAt: String,
    @SerialName("started_at")
    val startedAt: String? = null,
    @SerialName("finished_at")
    val finishedAt: String? = null,
    val status: String,
    @SerialName("session_id_input")
    val sessionIdInput: String? = null,
    @SerialName("session_id_output")
    val sessionIdOutput: String? = null,
    @SerialName("response_preview")
    val responsePreview: String? = null,
    @SerialName("node_results")
    val nodeResults: List<TaskRunNodeResult>? = null,
    @SerialName("run_cards")
    val runCards: List<TaskRunCard>? = null,
    val error: String? = null
)

@Serializable
data class TaskRunPayload(
    val task: TaskPayload,
    val run: TaskRunLog
)

@Serializable
data class TaskRunStopRequest(
    @SerialName("run_id")
    val runId: String
)

@Serializable
data class TaskRunStopResponse(
    val status: String,
    val message: String,
    @SerialName("task_id")
    val taskId: String,
    @SerialName("run_id")
    val runId: String? = null,
    val run: TaskRunLog? = null
)
