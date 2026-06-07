package dev.ghostos.android.viewmodel

import dev.ghostos.android.model.AgentCompletionDeltaPayload
import dev.ghostos.android.model.AgentDonePayload
import dev.ghostos.android.model.AgentErrorPayload
import dev.ghostos.android.model.AgentRunStartedPayload
import dev.ghostos.android.model.AgentStreamEvent
import dev.ghostos.android.model.AgentStreamMessagePayload
import dev.ghostos.android.model.AgentToolCallFinishedPayload
import dev.ghostos.android.model.AgentToolCallStartedPayload
import dev.ghostos.android.model.SessionPushAssistantMessagePayload
import dev.ghostos.android.model.SessionPushAwaitingHumanPayload
import dev.ghostos.android.model.SessionPushEvent
import dev.ghostos.android.network.BridgeGateway
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.decodeFromJsonElement

internal fun ChatViewModel.setActiveRun(traceId: String, sessionId: String) {
    activeRun = ActiveRun(traceId = traceId.trim(), sessionId = sessionId.trim())
    _stopPending.value = false
    updateCanStop()
}

internal fun ChatViewModel.updateActiveRunSession(traceId: String, sessionId: String) {
    val current = activeRun ?: return
    if (current.traceId != traceId.trim()) {
        return
    }
    activeRun = current.copy(sessionId = sessionId.trim())
    updateCanStop()
}

internal fun ChatViewModel.finishRun(traceId: String) {
    val normalizedTraceId = traceId.trim()
    if (activeRun?.traceId == normalizedTraceId) {
        activeRun = null
        activeRunJob = null
    }
    _stopPending.value = false
    updateCanStop()
    completeTrace(normalizedTraceId)
}

internal fun ChatViewModel.wasStopRequested(traceId: String): Boolean {
    return stopRequestedTraceIds.contains(traceId.trim())
}

internal fun ChatViewModel.updateCanStop() {
    _canStop.value = activeRun != null && !_stopPending.value
}

internal suspend fun ChatViewModel.handleAgentStreamEvent(currentClient: BridgeGateway, event: AgentStreamEvent) {
    handleRuntimeEvent(
        currentClient = currentClient,
        eventType = event.type,
        traceId = event.traceId,
        sessionId = sessionIDFromPayload(event.payload) ?: currentSessionIdForTrace(event.traceId),
        payload = event.payload,
        stepId = event.stepId,
        refreshSessionsOnTerminal = true,
    )
}

internal suspend fun ChatViewModel.handleSessionPushEvent(currentClient: BridgeGateway, event: SessionPushEvent) {
    val traceId = event.traceId?.trim().orEmpty()
    if (traceId.isNotBlank() && localTraceIds.contains(traceId)) {
        return
    }
    when (event.type) {
        SessionPushEvent.TYPE_ASSISTANT_MESSAGE -> handleAssistantPushEvent(currentClient, traceId, event)
        SessionPushEvent.TYPE_AWAITING_HUMAN,
        SessionPushEvent.TYPE_RUN_STARTED,
        SessionPushEvent.TYPE_COMPLETION_DELTA,
        SessionPushEvent.TYPE_TOOL_CALL_STARTED,
        SessionPushEvent.TYPE_TOOL_CALL_FINISHED,
        SessionPushEvent.TYPE_ERROR,
        SessionPushEvent.TYPE_DONE,
        -> handleRuntimeEvent(
            currentClient = currentClient,
            eventType = event.type,
            traceId = traceId,
            sessionId = event.sessionId,
            payload = event.payload,
            stepId = "",
            refreshSessionsOnTerminal = event.type == SessionPushEvent.TYPE_AWAITING_HUMAN
                || event.type == SessionPushEvent.TYPE_DONE
                || event.type == SessionPushEvent.TYPE_ERROR,
        )
    }
}

private suspend fun ChatViewModel.handleAssistantPushEvent(
    currentClient: BridgeGateway,
    traceId: String,
    event: SessionPushEvent,
) {
    val payload = json.decodeFromJsonElement<SessionPushAssistantMessagePayload>(event.payload)
    appendMessage(buildAssistantMessage(payload.message, traceId, event.sessionId))
    _pendingQuestion.value = null
    _state.value = ChatState.Idle
    refreshSessions(currentClient)
}

internal suspend fun ChatViewModel.handleRuntimeEvent(
    currentClient: BridgeGateway,
    eventType: String,
    traceId: String,
    sessionId: String?,
    payload: JsonObject,
    stepId: String,
    refreshSessionsOnTerminal: Boolean,
) {
    val normalizedTraceId = traceId.trim()
    val normalizedSessionId = sessionId?.trim().orEmpty()
    syncTraceSession(normalizedTraceId, normalizedSessionId)

    when (eventType) {
        AgentStreamEvent.TYPE_RUN_STARTED -> onRunStarted(payload, normalizedSessionId, normalizedTraceId)
        AgentStreamEvent.TYPE_COMPLETION_DELTA -> onCompletionDelta(payload, normalizedSessionId, normalizedTraceId)
        AgentStreamEvent.TYPE_TOOL_CALL_STARTED -> onToolCallStarted(payload, normalizedSessionId, normalizedTraceId, stepId)
        AgentStreamEvent.TYPE_TOOL_CALL_FINISHED -> onToolCallFinished(payload, normalizedSessionId, normalizedTraceId, stepId)
        AgentStreamEvent.TYPE_AWAITING_HUMAN -> onAwaitingHuman(
            currentClient,
            payload,
            normalizedSessionId,
            normalizedTraceId,
            refreshSessionsOnTerminal,
        )

        AgentStreamEvent.TYPE_MESSAGE -> onMessage(payload, normalizedSessionId, normalizedTraceId)
        AgentStreamEvent.TYPE_ERROR -> onRuntimeError(
            currentClient,
            payload,
            normalizedSessionId,
            normalizedTraceId,
            refreshSessionsOnTerminal,
        )

        AgentStreamEvent.TYPE_DONE -> onRuntimeDone(
            currentClient,
            payload,
            normalizedSessionId,
            normalizedTraceId,
            refreshSessionsOnTerminal,
        )
    }
}

private suspend fun ChatViewModel.syncTraceSession(traceId: String, sessionId: String) {
    if (sessionId.isBlank() || traceId.isBlank()) {
        return
    }
    knownSessionIdsByTrace[traceId] = sessionId
    bindRuntimeSession(sessionId)
    updateActiveRunSession(traceId, sessionId)
}

private suspend fun ChatViewModel.onRunStarted(payload: JsonObject, sessionId: String, traceId: String) {
    val decoded = json.decodeFromJsonElement<AgentRunStartedPayload>(payload)
    val resolvedSessionId = decoded.sessionId?.trim().orEmpty().ifBlank { sessionId }
    syncTraceSession(traceId, resolvedSessionId)
    ensureStreamingAssistant(traceId, resolvedSessionId)
    _state.value = ChatState.Loading
}

private fun ChatViewModel.onCompletionDelta(payload: JsonObject, sessionId: String, traceId: String) {
    val decoded = json.decodeFromJsonElement<AgentCompletionDeltaPayload>(payload)
    if (decoded.kind == AgentCompletionDeltaPayload.KIND_TEXT && !decoded.text.isNullOrEmpty()) {
        appendStreamingText(traceId, sessionId, decoded.text)
    }
    _state.value = ChatState.Loading
}

private fun ChatViewModel.onToolCallStarted(payload: JsonObject, sessionId: String, traceId: String, stepId: String) {
    val decoded = json.decodeFromJsonElement<AgentToolCallStartedPayload>(payload)
    upsertToolMessage(
        traceId = traceId,
        sessionId = sessionId,
        stepId = stepId,
        toolName = decoded.tool,
        toolCallId = decoded.toolCallId,
        status = TOOL_STATUS_RUNNING,
        text = "使用工具：${displayToolName(decoded.tool)}",
    )
    _state.value = ChatState.Loading
}

private fun ChatViewModel.onToolCallFinished(payload: JsonObject, sessionId: String, traceId: String, stepId: String) {
    val decoded = json.decodeFromJsonElement<AgentToolCallFinishedPayload>(payload)
    val isError = !decoded.error.isNullOrBlank() || decoded.status == TOOL_STATUS_ERROR
    upsertToolMessage(
        traceId = traceId,
        sessionId = sessionId,
        stepId = stepId,
        toolName = decoded.tool,
        toolCallId = decoded.toolCallId,
        status = if (isError) TOOL_STATUS_ERROR else TOOL_STATUS_SUCCESS,
        text = if (isError) {
            "工具报错：${displayToolName(decoded.tool)}\n${decoded.error.orEmpty().trim()}".trim()
        } else {
            "工具完成：${displayToolName(decoded.tool)}"
        },
    )
    _state.value = ChatState.Loading
}

private suspend fun ChatViewModel.onAwaitingHuman(
    currentClient: BridgeGateway,
    payload: JsonObject,
    sessionId: String,
    traceId: String,
    refreshSessionsOnTerminal: Boolean,
) {
    val decoded = json.decodeFromJsonElement<SessionPushAwaitingHumanPayload>(payload)
    completeStreamingAssistant(traceId)
    val resolvedSessionId = sessionId.ifBlank { currentSessionIdForTrace(traceId) }
    _pendingQuestion.value = PendingQuestion(
        sessionId = resolvedSessionId,
        questionId = decoded.questionId,
        prompt = decoded.prompt,
        selectionMode = decoded.selectionMode,
        options = decoded.options,
    )

    if (_messages.value.none { it.kind == ChatMessageKind.PendingQuestion && it.questionId == decoded.questionId }) {
        appendMessage(buildPendingQuestionMessage(decoded, traceId, resolvedSessionId))
    }
    _state.value = ChatState.Idle
    finishRun(traceId)
    if (refreshSessionsOnTerminal) {
        refreshSessions(currentClient)
    }
}

private fun ChatViewModel.onMessage(payload: JsonObject, sessionId: String, traceId: String) {
    val decoded = json.decodeFromJsonElement<AgentStreamMessagePayload>(payload)
    val resolvedSessionId = decoded.sessionId?.trim().orEmpty().ifBlank { sessionId }
    finalizeStreamingAssistant(traceId, resolvedSessionId, decoded.text)
}

private suspend fun ChatViewModel.onRuntimeError(
    currentClient: BridgeGateway,
    payload: JsonObject,
    sessionId: String,
    traceId: String,
    refreshSessionsOnTerminal: Boolean,
) {
    val decoded = json.decodeFromJsonElement<AgentErrorPayload>(payload)
    completeStreamingAssistant(traceId)
    appendMessage(buildErrorMessage(decoded.message, traceId, sessionId))
    _state.value = ChatState.Error(decoded.message)
    finishRun(traceId)
    if (refreshSessionsOnTerminal) {
        refreshSessions(currentClient)
    }
}

private suspend fun ChatViewModel.onRuntimeDone(
    currentClient: BridgeGateway,
    payload: JsonObject,
    sessionId: String,
    traceId: String,
    refreshSessionsOnTerminal: Boolean,
) {
    val decoded = json.decodeFromJsonElement<AgentDonePayload>(payload)
    val resolvedSessionId = decoded.sessionId?.trim().orEmpty().ifBlank { sessionId }
    completeStreamingAssistant(traceId)

    if (_state.value !is ChatState.Error) {
        _state.value = ChatState.Idle
    }
    if (resolvedSessionId.isNotBlank() && traceId.isNotBlank()) {
        knownSessionIdsByTrace[traceId] = resolvedSessionId
    }
    if (refreshSessionsOnTerminal) {
        refreshSessions(currentClient)
    }
    if (resolvedSessionId.isNotBlank() && _sessionId.value == resolvedSessionId) {
        loadSessionHistory(currentClient, resolvedSessionId, updateErrorState = false)
    }
    finishRun(traceId)
}
