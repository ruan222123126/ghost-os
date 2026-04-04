package dev.ghostos.android.viewmodel

import dev.ghostos.android.model.AskHumanOption

enum class ChatMessageKind {
    User,
    Assistant,
    System,
    Tool,
    Error,
    PendingQuestion,
}

data class ChatMessage(
    val id: String = nextChatMessageId(),
    val kind: ChatMessageKind,
    val text: String,
    val attachments: List<ChatAttachment> = emptyList(),
    val timestamp: Long = System.currentTimeMillis(),
    val traceId: String? = null,
    val sessionId: String? = null,
    val toolName: String? = null,
    val toolStatus: String? = null,
    val toolCallId: String? = null,
    val isStreaming: Boolean = false,
    val questionId: String? = null,
) {
    val isUser: Boolean
        get() = kind == ChatMessageKind.User
}

data class ChatAttachment(
    val sessionId: String,
    val artifactId: String,
    val name: String,
    val downloadUrl: String,
    val mimeType: String? = null,
    val bytes: Int? = null,
    val sha256: String? = null,
    val sourcePath: String? = null,
    val note: String? = null,
)

data class PendingQuestion(
    val sessionId: String,
    val questionId: String,
    val prompt: String,
    val selectionMode: String? = null,
    val options: List<AskHumanOption>? = null,
)

sealed class ChatState {
    object Idle : ChatState()
    object Loading : ChatState()
    data class Error(val message: String) : ChatState()
}

internal data class SessionHistorySnapshot(
    val messages: List<ChatMessage>,
    val pendingQuestion: PendingQuestion?,
)

internal data class ActiveRun(
    val traceId: String,
    val sessionId: String,
)

internal const val EVENT_RUN_STARTED = "run_started"
internal const val EVENT_COMPLETION_DELTA = "completion_delta"
internal const val EVENT_TOOL_CALL_STARTED = "tool_call_started"
internal const val EVENT_TOOL_CALL_FINISHED = "tool_call_finished"
internal const val EVENT_AWAITING_HUMAN = "awaiting_human"
internal const val EVENT_MESSAGE = "message"
internal const val EVENT_DONE = "done"
internal const val EVENT_ERROR = "error"
internal const val EVENT_ASSISTANT_MESSAGE = "assistant_message"
internal const val COMPLETION_DELTA_TEXT = "text"
internal const val TOOL_STATUS_RUNNING = "running"
internal const val TOOL_STATUS_SUCCESS = "success"
internal const val TOOL_STATUS_ERROR = "error"
internal const val LOG_TAG = "GhostOS-ChatViewModel"
