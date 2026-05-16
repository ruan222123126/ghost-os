// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

package dev.ghostos.android.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

@Serializable
data class AgentStreamEvent(
    val id: String,
    @SerialName("step_id")
    val stepId: String,
    @SerialName("trace_id")
    val traceId: String,
    @SerialName("session_id")
    val sessionId: String? = null,
    val turn: Int,
    val type: String,
    val payload: JsonObject,
    val at: String? = null
)

@Serializable
data class SessionPushEvent(
    val id: String,
    val type: String,
    @SerialName("trace_id")
    val traceId: String? = null,
    @SerialName("session_id")
    val sessionId: String,
    val payload: JsonObject,
    val at: String? = null
)

@Serializable
data class AgentRunStartedPayload(
    @SerialName("session_id")
    val sessionId: String? = null
)

@Serializable
data class AgentCompletionDeltaPayload(
    val kind: String,
    val text: String? = null,
    val thinking: String? = null,
    @SerialName("tool_call_index")
    val toolCallIndex: Int? = null,
    @SerialName("tool_call_id")
    val toolCallId: String? = null,
    @SerialName("tool_name")
    val toolName: String? = null,
    @SerialName("arguments_fragment")
    val argumentsFragment: String? = null
)

@Serializable
data class AgentToolCallStartedPayload(
    val tool: String? = null,
    @SerialName("tool_call_id")
    val toolCallId: String? = null,
    @SerialName("arguments_json")
    val argumentsJson: String? = null
)

@Serializable
data class AgentToolCallFinishedPayload(
    val tool: String? = null,
    @SerialName("tool_call_id")
    val toolCallId: String? = null,
    val status: String? = null,
    val error: String? = null,
    val output: String? = null
)

@Serializable
data class AgentStreamMessagePayload(
    val text: String,
    @SerialName("session_id")
    val sessionId: String? = null
)

@Serializable
data class AgentDonePayload(
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("session_ended")
    val sessionEnded: Boolean? = null
)

@Serializable
data class AgentErrorPayload(
    val message: String,
    @SerialName("session_id")
    val sessionId: String? = null,
    val code: Int? = null
)

@Serializable
data class SessionPushAssistantMessagePayload(
    val message: String,
    @SerialName("session_ended")
    val sessionEnded: Boolean,
    @SerialName("session_end")
    val sessionEnd: AssistantSessionEndSignal? = null
)

@Serializable
data class SessionPushAwaitingHumanPayload(
    @SerialName("question_id")
    val questionId: String,
    val prompt: String,
    @SerialName("selection_mode")
    val selectionMode: String? = null,
    val options: List<AskHumanOption>? = null
)
