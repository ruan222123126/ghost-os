// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

package dev.ghostos.android.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class OrchestrationNode(
    val id: String,
    val type: String,
    val group: OrchestrationGroupNode? = null,
    val agent: OrchestrationAgentNode? = null
)

@Serializable
data class OrchestrationGroupNode(
    val title: String,
    @SerialName("shared_context")
    val sharedContext: String,
    @SerialName("speaking_mode")
    val speakingMode: String,
    @SerialName("owner_agent_id")
    val ownerAgentId: String? = null,
    @SerialName("max_rounds")
    val maxRounds: Int
)

@Serializable
data class OrchestrationAgentNode(
    val title: String,
    val message: String,
    @SerialName("runtime_overrides")
    val runtimeOverrides: TaskRuntimeOverrides? = null
)

@Serializable
data class OrchestrationEdge(
    @SerialName("from_node_id")
    val fromNodeId: String,
    @SerialName("to_node_id")
    val toNodeId: String,
    val kind: String
)

@Serializable
data class OrchestrationDefinition(
    val nodes: List<OrchestrationNode>,
    val edges: List<OrchestrationEdge>
)

@Serializable
data class OrchestrationTaskCreateRequest(
    @SerialName("task_kind")
    val taskKind: String,
    val name: String,
    val orchestration: OrchestrationDefinition,
    @SerialName("interval_seconds")
    val intervalSeconds: Int? = null,
    @SerialName("cron_expr")
    val cronExpr: String? = null,
    @SerialName("trace_id")
    val traceId: String? = null
) : TaskCreateRequest

@Serializable
data class OrchestrationTaskPayload(
    val id: String,
    val name: String,
    @SerialName("task_kind")
    val taskKind: String,
    val orchestration: OrchestrationDefinition,
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
