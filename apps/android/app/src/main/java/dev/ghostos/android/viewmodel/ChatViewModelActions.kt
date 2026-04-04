package dev.ghostos.android.viewmodel

import android.content.Context
import android.content.Intent
import android.util.Log
import androidx.core.content.FileProvider
import androidx.lifecycle.viewModelScope
import dev.ghostos.android.model.AgentStopResponsePayload
import dev.ghostos.android.network.BridgeGateway
import dev.ghostos.android.util.readableMessage
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch

private data class PreparedSend(
    val client: BridgeGateway,
    val message: String,
    val pendingQuestion: PendingQuestion?,
    val targetSessionId: String,
    val traceId: String,
)

internal fun ChatViewModel.sendMessageAction(text: String) {
    val prepared = prepareSend(text) ?: return
    launchActiveRun(prepared)
}

private fun ChatViewModel.prepareSend(text: String): PreparedSend? {
    val message = text.trim()
    if (message.isBlank()) {
        return null
    }

    val currentClient = client ?: run {
        _state.value = ChatState.Error("未配置连接")
        return null
    }

    val pending = _pendingQuestion.value
    val traceId = createClientTraceId(if (pending == null) "android-run" else "android-answer")
    val targetSessionId = pending?.sessionId ?: _sessionId.value
    registerLocalTrace(traceId)
    stopRequestedTraceIds.remove(traceId)
    appendMessage(buildUserMessage(message, traceId, targetSessionId))
    _pendingQuestion.value = null
    _state.value = ChatState.Loading
    setActiveRun(traceId, targetSessionId)
    return PreparedSend(
        client = currentClient,
        message = message,
        pendingQuestion = pending,
        targetSessionId = targetSessionId,
        traceId = traceId,
    )
}

private fun ChatViewModel.launchActiveRun(prepared: PreparedSend) {
    activeRunJob?.cancel()
    activeRunJob = viewModelScope.launch {
        executeStream(prepared)
    }
}

private suspend fun ChatViewModel.executeStream(prepared: PreparedSend) {
    runCatching {
        val stream = buildStream(prepared)
        stream.collect { event ->
            handleAgentStreamEvent(prepared.client, event)
        }
        if (wasStopRequested(prepared.traceId)) {
            finishStoppedRun(prepared.traceId)
        }
    }.onFailure { error ->
        handleStreamFailure(prepared.traceId, error)
    }
}

private fun ChatViewModel.buildStream(prepared: PreparedSend) = if (prepared.pendingQuestion == null) {
    prepared.client.streamMessage(prepared.message, prepared.targetSessionId.ifBlank { null }, prepared.traceId)
} else {
    prepared.client.streamAnswer(
        prepared.pendingQuestion.sessionId,
        prepared.pendingQuestion.questionId,
        prepared.message,
        prepared.traceId,
    )
}

private fun ChatViewModel.finishStoppedRun(traceId: String) {
    finishRun(traceId)
    if (_state.value !is ChatState.Error) {
        _state.value = ChatState.Idle
    }
}

private fun ChatViewModel.handleStreamFailure(traceId: String, error: Throwable) {
    if (error is CancellationException && wasStopRequested(traceId)) {
        finishStoppedRun(traceId)
        return
    }

    val message = error.readableMessage()
    completeStreamingAssistant(traceId)
    _state.value = ChatState.Error(message)
    appendMessage(buildErrorMessage(message, traceId, currentSessionIdForTrace(traceId)))
    finishRun(traceId)
}

internal fun ChatViewModel.stopCurrentRunAction() {
    val currentClient = client ?: run {
        _state.value = ChatState.Error("未配置连接")
        return
    }
    val run = activeRun ?: return
    if (_stopPending.value) {
        return
    }

    _stopPending.value = true
    updateCanStop()

    viewModelScope.launch {
        currentClient.stopRun(
            sessionId = run.sessionId.ifBlank { null },
            traceId = run.traceId,
        ).fold(
            onSuccess = { response ->
                handleStopSuccess(run, response)
            },
            onFailure = { error ->
                _stopPending.value = false
                updateCanStop()
                _state.value = ChatState.Error(error.readableMessage())
            },
        )
    }
}

internal fun ChatViewModel.loadSessionsAction() {
    viewModelScope.launch {
        runCatching {
            refreshSessions(client)
        }.onFailure { error ->
            Log.e(LOG_TAG, "loadSessions failed", error)
            _sessionsError.value = error.readableMessage()
        }
    }
}

internal fun ChatViewModel.selectSessionAction(id: String) {
    val trimmedID = id.trim()
    if (trimmedID.isBlank()) {
        return
    }

    val currentClient = client ?: run {
        _state.value = ChatState.Error("未配置连接")
        return
    }

    viewModelScope.launch {
        runCatching {
            skipHistoryBootstrapForSessionId = trimmedID
            persistSessionId(trimmedID)
            loadSessionHistory(currentClient, trimmedID)
        }.onFailure { error ->
            Log.e(LOG_TAG, "selectSession failed for $trimmedID", error)
            _state.value = ChatState.Error(error.readableMessage())
        }
    }
}

internal fun ChatViewModel.createNewSessionAction() {
    viewModelScope.launch {
        runCatching {
            skipHistoryBootstrapForSessionId = null
            clearConversation()
            persistSessionId("")
        }.onFailure { error ->
            Log.e(LOG_TAG, "createNewSession failed", error)
            _state.value = ChatState.Error(error.readableMessage())
        }
    }
}

internal fun ChatViewModel.openAttachmentAction(context: Context, attachment: ChatAttachment) {
    val currentClient = client ?: run {
        _state.value = ChatState.Error("未配置连接")
        return
    }

    viewModelScope.launch {
        currentClient.downloadSessionArtifact(attachment.sessionId, attachment.artifactId).fold(
            onSuccess = { downloaded ->
                runCatching {
                    val target = writeDownloadedArtifact(context, downloaded.filename, downloaded.bytes)
                    val uri = FileProvider.getUriForFile(
                        context,
                        "${context.packageName}.fileprovider",
                        target,
                    )
                    val intent = Intent(Intent.ACTION_VIEW).apply {
                        setDataAndType(uri, downloaded.mimeType)
                        addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                    }
                    context.startActivity(intent)
                }.onFailure { error ->
                    Log.e(LOG_TAG, "openAttachment failed for ${attachment.artifactId}", error)
                    _state.value = ChatState.Error(error.readableMessage())
                }
            },
            onFailure = { error ->
                Log.e(LOG_TAG, "downloadSessionArtifact failed for ${attachment.artifactId}", error)
                _state.value = ChatState.Error(error.readableMessage())
            },
        )
    }
}

internal fun ChatViewModel.handleStopSuccess(run: ActiveRun, response: AgentStopResponsePayload) {
    stopRequestedTraceIds += run.traceId
    completeStreamingAssistant(run.traceId)
    appendMessage(buildSystemMessage(response.message))
    activeRunJob?.cancel()
    if (_state.value !is ChatState.Error) {
        _state.value = ChatState.Idle
    }
    finishRun(run.traceId)
}
