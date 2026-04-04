package dev.ghostos.android.viewmodel

import android.content.Context
import dev.ghostos.android.model.SessionContentPart
import dev.ghostos.android.model.SessionFileContent
import dev.ghostos.android.model.SessionHumanInteraction
import dev.ghostos.android.model.SessionPushAwaitingHumanPayload
import dev.ghostos.android.model.SessionToolResult
import java.io.File
import java.util.UUID
import kotlinx.serialization.json.JsonObject

internal fun buildUserMessage(text: String, traceId: String? = null, sessionId: String? = null): ChatMessage {
    return ChatMessage(
        kind = ChatMessageKind.User,
        text = text,
        traceId = traceId,
        sessionId = sessionId,
    )
}

internal fun buildAssistantMessage(text: String, traceId: String? = null, sessionId: String? = null): ChatMessage {
    return ChatMessage(
        kind = ChatMessageKind.Assistant,
        text = text,
        traceId = traceId,
        sessionId = sessionId,
    )
}

internal fun buildSystemMessage(text: String): ChatMessage {
    return ChatMessage(kind = ChatMessageKind.System, text = text)
}

internal fun buildErrorMessage(text: String, traceId: String? = null, sessionId: String? = null): ChatMessage {
    return ChatMessage(
        kind = ChatMessageKind.Error,
        text = text,
        traceId = traceId,
        sessionId = sessionId,
    )
}

internal fun buildPendingQuestionMessage(
    payload: SessionPushAwaitingHumanPayload,
    traceId: String? = null,
    sessionId: String? = null,
): ChatMessage {
    return ChatMessage(
        kind = ChatMessageKind.PendingQuestion,
        text = payload.prompt,
        traceId = traceId,
        sessionId = sessionId,
        questionId = payload.questionId,
    )
}

internal fun buildPendingQuestionMessage(interaction: SessionHumanInteraction): ChatMessage {
    return ChatMessage(
        kind = ChatMessageKind.PendingQuestion,
        text = interaction.prompt,
        questionId = interaction.questionId,
    )
}

internal fun formatToolHistoryText(fallbackText: String, toolResult: SessionToolResult?): String {
    if (!toolResult?.error.isNullOrBlank()) {
        return toolResult?.error.orEmpty()
    }
    if (!toolResult?.output.isNullOrBlank()) {
        return toolResult?.output.orEmpty()
    }
    if (!toolResult?.tool.isNullOrBlank()) {
        return "[${toolResult?.tool}]"
    }
    return fallbackText.ifBlank { "[tool]" }
}

internal fun toolMessageKey(traceId: String, toolCallId: String?, toolName: String?, stepId: String): String {
    val normalizedToolCallId = toolCallId?.trim().orEmpty()
    if (normalizedToolCallId.isNotBlank()) {
        return "$traceId|$normalizedToolCallId"
    }
    val normalizedToolName = toolName?.trim().orEmpty()
    return "$traceId|$stepId|$normalizedToolName"
}

internal fun sessionIDFromPayload(payload: JsonObject): String? {
    return payload["session_id"]?.toString()?.trim('"')
}

internal fun attachmentsFromContent(content: List<SessionContentPart>?, sessionId: String): List<ChatAttachment> {
    if (content.isNullOrEmpty()) {
        return emptyList()
    }
    val normalizedSessionId = sessionId.trim()
    if (normalizedSessionId.isBlank()) {
        return emptyList()
    }
    return content.mapNotNull { part ->
        if (part.type != "file") {
            return@mapNotNull null
        }
        part.file?.toChatAttachment(normalizedSessionId)
    }
}

internal fun SessionFileContent.toChatAttachment(sessionId: String): ChatAttachment {
    return ChatAttachment(
        sessionId = sessionId,
        artifactId = artifactId,
        name = name,
        downloadUrl = downloadUrl,
        mimeType = mimeType,
        bytes = bytes,
        sha256 = sha256,
        sourcePath = sourcePath,
        note = note,
    )
}

internal fun writeDownloadedArtifact(context: Context, filename: String, bytes: ByteArray): File {
    val dir = File(context.cacheDir, "attachments")
    if (!dir.exists() && !dir.mkdirs()) {
        throw IllegalStateException("无法创建附件缓存目录")
    }
    val target = File(dir, filename)
    target.writeBytes(bytes)
    return target
}

internal fun displayToolName(toolName: String?): String {
    return toolName?.trim().orEmpty().ifBlank { "unknown" }
}

internal fun createClientTraceId(prefix: String): String {
    return "$prefix-${UUID.randomUUID()}"
}

internal fun nextChatMessageId(): String = UUID.randomUUID().toString()
