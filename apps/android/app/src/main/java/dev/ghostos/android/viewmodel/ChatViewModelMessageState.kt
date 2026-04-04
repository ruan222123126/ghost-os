package dev.ghostos.android.viewmodel

internal fun ChatViewModel.ensureStreamingAssistant(traceId: String, sessionId: String): String {
    val normalizedTraceId = traceId.trim()
    val existingId = streamingMessageIdsByTrace[normalizedTraceId]
    if (existingId != null) {
        return existingId
    }

    val message = ChatMessage(
        kind = ChatMessageKind.Assistant,
        text = "",
        traceId = normalizedTraceId.ifBlank { null },
        sessionId = sessionId.ifBlank { null },
        isStreaming = true,
    )
    appendMessage(message)
    if (normalizedTraceId.isNotBlank()) {
        streamingMessageIdsByTrace[normalizedTraceId] = message.id
    }
    return message.id
}

internal fun ChatViewModel.appendStreamingText(traceId: String, sessionId: String, text: String) {
    val messageId = ensureStreamingAssistant(traceId, sessionId)
    updateMessage(messageId) { current ->
        current.copy(
            text = current.text + text,
            sessionId = current.sessionId ?: sessionId.ifBlank { null },
            isStreaming = true,
        )
    }
}

internal fun ChatViewModel.finalizeStreamingAssistant(traceId: String, sessionId: String, finalText: String) {
    val existingId = streamingMessageIdsByTrace[traceId]
    if (existingId == null) {
        appendMessage(buildAssistantMessage(finalText, traceId, sessionId))
        return
    }

    updateMessage(existingId) { current ->
        current.copy(
            text = finalText,
            sessionId = current.sessionId ?: sessionId.ifBlank { null },
            isStreaming = false,
        )
    }
    streamingMessageIdsByTrace.remove(traceId)
}

internal fun ChatViewModel.completeStreamingAssistant(traceId: String) {
    val messageId = streamingMessageIdsByTrace.remove(traceId) ?: return
    updateMessage(messageId) { current ->
        current.copy(isStreaming = false)
    }
}

internal fun ChatViewModel.upsertToolMessage(
    traceId: String,
    sessionId: String,
    stepId: String,
    toolName: String?,
    toolCallId: String?,
    status: String,
    text: String,
) {
    val key = toolMessageKey(traceId, toolCallId, toolName, stepId)
    val existingId = toolMessageIdsByKey[key]
    if (existingId == null) {
        val message = ChatMessage(
            kind = ChatMessageKind.Tool,
            text = text,
            traceId = traceId.ifBlank { null },
            sessionId = sessionId.ifBlank { null },
            toolName = toolName?.trim()?.ifBlank { null },
            toolStatus = status,
            toolCallId = toolCallId?.trim()?.ifBlank { null },
        )
        appendMessage(message)
        toolMessageIdsByKey[key] = message.id
        return
    }

    updateMessage(existingId) { current ->
        current.copy(
            text = text,
            sessionId = current.sessionId ?: sessionId.ifBlank { null },
            toolName = current.toolName ?: toolName?.trim()?.ifBlank { null },
            toolStatus = status,
            toolCallId = current.toolCallId ?: toolCallId?.trim()?.ifBlank { null },
        )
    }
}

internal fun ChatViewModel.currentSessionIdForTrace(traceId: String): String {
    return knownSessionIdsByTrace[traceId]?.trim().orEmpty()
}

internal fun ChatViewModel.registerLocalTrace(traceId: String) {
    val normalizedTraceId = traceId.trim()
    if (normalizedTraceId.isBlank()) {
        return
    }

    localTraceIds += normalizedTraceId
    while (localTraceIds.size > 32) {
        val oldest = localTraceIds.firstOrNull() ?: break
        localTraceIds.remove(oldest)
    }
}

internal fun ChatViewModel.completeTrace(traceId: String) {
    streamingMessageIdsByTrace.remove(traceId)
    knownSessionIdsByTrace.remove(traceId)
    toolMessageIdsByKey.keys.removeAll { it.startsWith("$traceId|") }
}

internal fun ChatViewModel.appendMessage(message: ChatMessage) {
    _messages.value = _messages.value + message
}

internal fun ChatViewModel.updateMessage(messageId: String, transform: (ChatMessage) -> ChatMessage) {
    _messages.value = _messages.value.map { message ->
        if (message.id == messageId) transform(message) else message
    }
}

internal fun ChatViewModel.clearConversation() {
    activeRunJob?.cancel()
    activeRunJob = null
    activeRun = null
    _stopPending.value = false
    updateCanStop()
    clearRuntimeTracking()
    _messages.value = emptyList()
    _pendingQuestion.value = null
    _state.value = ChatState.Idle
}

internal fun ChatViewModel.clearRuntimeTracking() {
    streamingMessageIdsByTrace.clear()
    toolMessageIdsByKey.clear()
    knownSessionIdsByTrace.clear()
    stopRequestedTraceIds.clear()
}
