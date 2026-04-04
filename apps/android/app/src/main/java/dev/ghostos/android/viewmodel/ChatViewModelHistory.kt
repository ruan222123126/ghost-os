package dev.ghostos.android.viewmodel

import dev.ghostos.android.model.SessionDetail
import dev.ghostos.android.model.SessionMessage

internal fun ChatViewModel.mapSessionDetail(detail: SessionDetail): SessionHistorySnapshot {
    val messages = mutableListOf<ChatMessage>()
    var pendingQuestion: PendingQuestion? = null

    detail.messages.forEach { message ->
        when (message.role) {
            "user" -> messages += buildUserMessage(message.text.orEmpty())
            "assistant" -> messages += buildAssistantMessage(message.text.orEmpty())
            "system" -> messages += buildSystemMessage(message.text?.ifBlank { "[system]" } ?: "[system]")
            "internal" -> messages += buildSystemMessage(message.text?.ifBlank { "[internal]" } ?: "[internal]")
            "tool" -> {
                val interaction = message.humanInteraction
                if (interaction != null) {
                    messages += buildPendingQuestionMessage(interaction)
                    if (interaction.answer == null) {
                        pendingQuestion = PendingQuestion(
                            sessionId = detail.id,
                            questionId = interaction.questionId,
                            prompt = interaction.prompt,
                            selectionMode = interaction.selectionMode,
                            options = interaction.options,
                        )
                    } else {
                        messages += buildUserMessage(interaction.answer)
                    }
                } else {
                    messages += buildToolHistoryMessage(message, detail.id)
                }
            }
        }
    }

    return SessionHistorySnapshot(messages = messages, pendingQuestion = pendingQuestion)
}

private fun buildToolHistoryMessage(message: SessionMessage, sessionId: String): ChatMessage {
    val toolResult = message.toolResult
    return ChatMessage(
        kind = ChatMessageKind.Tool,
        text = formatToolHistoryText(message.text.orEmpty(), toolResult),
        attachments = attachmentsFromContent(message.content, sessionId),
        traceId = toolResult?.traceId,
        toolName = toolResult?.tool,
        toolStatus = toolResult?.status,
        toolCallId = message.toolCallId,
    )
}
