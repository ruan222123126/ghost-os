// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

package dev.ghostos.android.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

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
