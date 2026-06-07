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
) {
    companion object {
        const val TYPE_RUN_STARTED = "run_started"
        const val TYPE_COMPLETION_DELTA = "completion_delta"
        const val TYPE_TOOL_CALL_STARTED = "tool_call_started"
        const val TYPE_TOOL_CALL_FINISHED = "tool_call_finished"
        const val TYPE_AWAITING_HUMAN = "awaiting_human"
        const val TYPE_MESSAGE = "message"
        const val TYPE_DONE = "done"
        const val TYPE_ERROR = "error"
    }
}

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
) {
    companion object {
        const val TYPE_ASSISTANT_MESSAGE = "assistant_message"
        const val TYPE_AWAITING_HUMAN = "awaiting_human"
        const val TYPE_RUN_STARTED = "run_started"
        const val TYPE_COMPLETION_DELTA = "completion_delta"
        const val TYPE_TOOL_CALL_STARTED = "tool_call_started"
        const val TYPE_TOOL_CALL_FINISHED = "tool_call_finished"
        const val TYPE_ERROR = "error"
        const val TYPE_DONE = "done"
        const val TYPE_TASK_RUN_CARD_STARTED = "task_run_card_started"
        const val TYPE_TASK_RUN_CARD_EVENT = "task_run_card_event"
        const val TYPE_TASK_RUN_CARD_FINISHED = "task_run_card_finished"
    }
}

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
) {
    companion object {
        const val KIND_TEXT = "text"
        const val KIND_THINKING = "thinking"
        const val KIND_TOOL_CALL_START = "tool_call_start"
        const val KIND_TOOL_CALL_DELTA = "tool_call_delta"
        const val KIND_TOOL_CALL_END = "tool_call_end"
    }
}

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
data class AgentAwaitingHumanStreamPayload(
    val tool: String? = null,
    @SerialName("tool_call_id")
    val toolCallId: String? = null,
    @SerialName("question_id")
    val questionId: String,
    val prompt: String,
    @SerialName("selection_mode")
    val selectionMode: String? = null,
    val options: List<AskHumanOption>? = null
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

@Serializable
data class TaskRunCardStartedPayload(
    @SerialName("card_id")
    val cardId: String,
    @SerialName("run_id")
    val runId: String,
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
    val startedAt: String
)

@Serializable
data class TaskRunCardEventPayload(
    @SerialName("card_id")
    val cardId: String,
    @SerialName("source_session_id")
    val sourceSessionId: String? = null,
    @SerialName("source_event")
    val sourceEvent: AgentStreamEvent
)

@Serializable
data class TaskRunCardFinishedPayload(
    @SerialName("card_id")
    val cardId: String,
    val status: String,
    @SerialName("finished_at")
    val finishedAt: String,
    val preview: String? = null,
    val error: String? = null,
    @SerialName("final_text")
    val finalText: String? = null,
    @SerialName("source_session_id")
    val sourceSessionId: String? = null
)
