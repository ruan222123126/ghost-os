package dev.ghostos.android.viewmodel

import android.util.Log
import androidx.lifecycle.viewModelScope
import dev.ghostos.android.network.BridgeGateway
import dev.ghostos.android.util.readableMessage
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch

internal fun ChatViewModel.observeBridgeConnectionLifecycle() {
    viewModelScope.launch {
        combine(store.baseUrl, store.token) { url, token ->
            url.trim() to token.trim()
        }.collectLatest { (url, token) ->
            runCatching {
                handleBridgeCredentialChange(url, token)
            }.onFailure { error ->
                Log.e(LOG_TAG, "observeBridgeConnection update failed", error)
                _state.value = ChatState.Error(error.readableMessage())
            }
        }
    }
}

private suspend fun ChatViewModel.handleBridgeCredentialChange(url: String, token: String) {
    client = if (url.isNotBlank() && token.isNotBlank()) {
        clientFactory(url, token)
    } else {
        null
    }

    val currentClient = client
    bindSessionEvents(currentClient, _sessionId.value)
    refreshSessions(currentClient)
    if (currentClient != null && _sessionId.value.isNotBlank()) {
        loadSessionHistory(currentClient, _sessionId.value, updateErrorState = false)
    }
}

internal fun ChatViewModel.observeSessionBindingLifecycle() {
    viewModelScope.launch {
        var previousSessionId = ""
        store.sessionId.collectLatest { storedID ->
            runCatching {
                val trimmedID = storedID.trim()
                val changed = previousSessionId != trimmedID
                previousSessionId = trimmedID
                applyStoredSession(trimmedID, changed)
            }.onFailure { error ->
                Log.e(LOG_TAG, "observeSessionBinding update failed", error)
                _state.value = ChatState.Error(error.readableMessage())
            }
        }
    }
}

private suspend fun ChatViewModel.applyStoredSession(sessionId: String, changed: Boolean) {
    if (_pendingQuestion.value?.sessionId != sessionId) {
        _pendingQuestion.value = null
    }
    _sessionId.value = sessionId
    bindSessionEvents(client, sessionId)

    if (!changed) {
        return
    }
    if (sessionId.isBlank()) {
        clearConversation()
        return
    }

    if (skipHistoryBootstrapForSessionId == sessionId) {
        skipHistoryBootstrapForSessionId = null
        return
    }

    val currentClient = client ?: return
    loadSessionHistory(currentClient, sessionId, updateErrorState = false)
}

internal suspend fun ChatViewModel.refreshSessions(currentClient: BridgeGateway?) {
    if (currentClient == null) {
        _sessions.value = emptyList()
        _sessionsError.value = "未配置连接"
        _sessionsLoading.value = false
        return
    }

    _sessionsLoading.value = true
    _sessionsError.value = ""
    runCatching {
        currentClient.listSessions().fold(
            onSuccess = { loaded ->
                _sessions.value = loaded
            },
            onFailure = { error ->
                throw error
            },
        )
    }.onFailure { error ->
        Log.e(LOG_TAG, "refreshSessions failed", error)
        _sessionsError.value = error.readableMessage()
    }
    _sessionsLoading.value = false
}

internal suspend fun ChatViewModel.loadSessionHistory(
    currentClient: BridgeGateway,
    sessionId: String,
    updateErrorState: Boolean = true,
) {
    val trimmedID = sessionId.trim()
    if (trimmedID.isBlank()) {
        clearConversation()
        return
    }

    _historyLoading.value = true
    clearRuntimeTracking()
    _messages.value = emptyList()
    _pendingQuestion.value = null
    if (updateErrorState) {
        _state.value = ChatState.Idle
    }

    runCatching {
        currentClient.getSession(trimmedID).fold(
            onSuccess = { detail ->
                val snapshot = mapSessionDetail(detail)
                _messages.value = snapshot.messages
                _pendingQuestion.value = snapshot.pendingQuestion
                if (updateErrorState) {
                    _state.value = ChatState.Idle
                }
            },
            onFailure = { error ->
                throw error
            },
        )
    }.onFailure { error ->
        Log.e(LOG_TAG, "loadSessionHistory failed for $trimmedID", error)
        if (updateErrorState) {
            _state.value = ChatState.Error(error.readableMessage())
        }
    }
    _historyLoading.value = false
}

internal suspend fun ChatViewModel.persistSessionId(sessionId: String) {
    val trimmedID = sessionId.trim()
    if (_sessionId.value == trimmedID) {
        return
    }
    _sessionId.value = trimmedID
    store.saveSessionId(trimmedID)
}

internal suspend fun ChatViewModel.bindRuntimeSession(sessionId: String) {
    val trimmedID = sessionId.trim()
    if (trimmedID.isBlank()) {
        return
    }
    skipHistoryBootstrapForSessionId = trimmedID
    persistSessionId(trimmedID)
}

internal fun ChatViewModel.bindSessionEvents(currentClient: BridgeGateway?, sessionId: String) {
    sessionEventsJob?.cancel()
    sessionEventsJob = null

    if (currentClient == null || sessionId.isBlank()) {
        return
    }

    sessionEventsJob = viewModelScope.launch {
        streamSessionEvents(currentClient, sessionId)
    }
}

private suspend fun ChatViewModel.streamSessionEvents(currentClient: BridgeGateway, sessionId: String) {
    while (_sessionId.value == sessionId && client === currentClient) {
        runCatching {
            currentClient.observeSessionEvents(sessionId).collect { event ->
                handleSessionPushEvent(currentClient, event)
            }
        }.onFailure { error ->
            if (_sessionId.value == sessionId && client === currentClient) {
                Log.e(LOG_TAG, "session event stream failed for $sessionId", error)
                _sessionsError.value = error.readableMessage()
            }
        }
        delay(1500)
    }
}
