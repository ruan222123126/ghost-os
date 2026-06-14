// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

package dev.ghostos.android.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
sealed interface AgentSendResponse

@Serializable
data class AssistantSessionEndSignal(
    val signal: String,
    val message: String
)

@Serializable
data class AgentRequest(
    val mode: String? = null,
    val message: String? = null,
    val images: List<SessionImageContent>? = null,
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("project_root")
    val projectRoot: String? = null,
    @SerialName("trace_id")
    val traceId: String? = null
)

@Serializable
data class AskHumanOption(
    val label: String,
    @SerialName("allow_custom")
    val allowCustom: Boolean? = null
)

@Serializable
data class AgentSendSuccessResponse(
    val message: String,
    @SerialName("session_id")
    val sessionId: String,
    @SerialName("session_ended")
    val sessionEnded: Boolean,
    val mode: String? = null,
    @SerialName("session_end")
    val sessionEnd: AssistantSessionEndSignal? = null
) : AgentSendResponse

@Serializable
data class AgentAwaitingHumanResponse(
    val status: String,
    @SerialName("session_id")
    val sessionId: String,
    @SerialName("question_id")
    val questionId: String,
    val prompt: String,
    @SerialName("selection_mode")
    val selectionMode: String? = null,
    val options: List<AskHumanOption>? = null
) : AgentSendResponse {
    companion object {
        const val STATUS_AWAITING_HUMAN = "awaiting_human"
        const val SELECTION_MODE_SINGLE = "single"
        const val SELECTION_MODE_MULTIPLE = "multiple"
    }
}

@Serializable
data class AgentStopRequest(
    @SerialName("session_id")
    val sessionId: String? = null,
    @SerialName("trace_id")
    val traceId: String? = null
)

@Serializable
data class AgentStopResponsePayload(
    val status: String,
    val message: String,
    @SerialName("session_id")
    val sessionId: String? = null
) {
    companion object {
        const val STATUS_STOPPED = "stopped"
        const val STATUS_NOT_RUNNING = "not_running"
    }
}

@Serializable
data class HumanResponseRequest(
    @SerialName("session_id")
    val sessionId: String,
    @SerialName("question_id")
    val questionId: String,
    val answer: String,
    val cancelled: Boolean? = null
)

@Serializable
data class HumanResponseAck(
    @SerialName("session_id")
    val sessionId: String,
    @SerialName("question_id")
    val questionId: String,
    val accepted: Boolean
)
