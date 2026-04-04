package dev.ghostos.android.viewmodel

import android.content.Context
import androidx.lifecycle.ViewModel
import dev.ghostos.android.model.SessionMetadata
import dev.ghostos.android.network.BridgeClient
import dev.ghostos.android.network.BridgeGateway
import dev.ghostos.android.store.ChatSettingsStore
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.serialization.json.Json

class ChatViewModel(
    internal val store: ChatSettingsStore,
    internal val clientFactory: (String, String) -> BridgeGateway = { url, token -> BridgeClient(url, token) },
) : ViewModel() {
    internal val _messages = MutableStateFlow<List<ChatMessage>>(emptyList())
    val messages: StateFlow<List<ChatMessage>> = _messages.asStateFlow()

    internal val _state = MutableStateFlow<ChatState>(ChatState.Idle)
    val state: StateFlow<ChatState> = _state.asStateFlow()

    internal val _sessionId = MutableStateFlow("")
    val sessionId: StateFlow<String> = _sessionId.asStateFlow()

    internal val _pendingQuestion = MutableStateFlow<PendingQuestion?>(null)
    val pendingQuestion: StateFlow<PendingQuestion?> = _pendingQuestion.asStateFlow()

    internal val _sessions = MutableStateFlow<List<SessionMetadata>>(emptyList())
    val sessions: StateFlow<List<SessionMetadata>> = _sessions.asStateFlow()

    internal val _sessionsLoading = MutableStateFlow(false)
    val sessionsLoading: StateFlow<Boolean> = _sessionsLoading.asStateFlow()

    internal val _sessionsError = MutableStateFlow("")
    val sessionsError: StateFlow<String> = _sessionsError.asStateFlow()

    internal val _historyLoading = MutableStateFlow(false)
    val historyLoading: StateFlow<Boolean> = _historyLoading.asStateFlow()

    internal val _canStop = MutableStateFlow(false)
    val canStop: StateFlow<Boolean> = _canStop.asStateFlow()

    internal val _stopPending = MutableStateFlow(false)
    val stopPending: StateFlow<Boolean> = _stopPending.asStateFlow()

    internal var client: BridgeGateway? = null
    internal var sessionEventsJob: Job? = null
    internal var activeRunJob: Job? = null
    internal var skipHistoryBootstrapForSessionId: String? = null
    internal val json = Json { ignoreUnknownKeys = true }

    internal val streamingMessageIdsByTrace = linkedMapOf<String, String>()
    internal val toolMessageIdsByKey = linkedMapOf<String, String>()
    internal val knownSessionIdsByTrace = linkedMapOf<String, String>()
    internal val localTraceIds = linkedSetOf<String>()
    internal val stopRequestedTraceIds = linkedSetOf<String>()
    internal var activeRun: ActiveRun? = null

    init {
        observeBridgeConnectionLifecycle()
        observeSessionBindingLifecycle()
    }

    fun sendMessage(text: String) {
        sendMessageAction(text)
    }

    fun stopCurrentRun() {
        stopCurrentRunAction()
    }

    fun loadSessions() {
        loadSessionsAction()
    }

    fun selectSession(id: String) {
        selectSessionAction(id)
    }

    fun createNewSession() {
        createNewSessionAction()
    }

    fun clearError() {
        if (_state.value is ChatState.Error) {
            _state.value = ChatState.Idle
        }
    }

    fun openAttachment(context: Context, attachment: ChatAttachment) {
        openAttachmentAction(context, attachment)
    }
}
