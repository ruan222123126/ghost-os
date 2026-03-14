package dev.ghostos.android.viewmodel

import android.content.Context
import android.content.Intent
import android.util.Log
import androidx.core.content.FileProvider
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dev.ghostos.android.model.AgentCompletionDeltaPayload
import dev.ghostos.android.model.AgentDonePayload
import dev.ghostos.android.model.AgentErrorPayload
import dev.ghostos.android.model.AgentRunStartedPayload
import dev.ghostos.android.model.AgentStreamEvent
import dev.ghostos.android.model.AgentStreamMessagePayload
import dev.ghostos.android.model.AgentStopResponsePayload
import dev.ghostos.android.model.AgentToolCallFinishedPayload
import dev.ghostos.android.model.AgentToolCallStartedPayload
import dev.ghostos.android.model.AskHumanOption
import dev.ghostos.android.model.SessionDetail
import dev.ghostos.android.model.SessionFileContent
import dev.ghostos.android.model.SessionHumanInteraction
import dev.ghostos.android.model.SessionMessage
import dev.ghostos.android.model.SessionMetadata
import dev.ghostos.android.model.SessionPushAssistantMessagePayload
import dev.ghostos.android.model.SessionPushAwaitingHumanPayload
import dev.ghostos.android.model.SessionPushEvent
import dev.ghostos.android.model.SessionToolResult
import dev.ghostos.android.network.BridgeClient
import dev.ghostos.android.network.BridgeGateway
import dev.ghostos.android.store.ChatSettingsStore
import dev.ghostos.android.util.readableMessage
import java.io.File
import java.util.UUID
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.collect
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.flow.combine
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.decodeFromJsonElement

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

private data class SessionHistorySnapshot(
    val messages: List<ChatMessage>,
    val pendingQuestion: PendingQuestion?,
)

private data class ActiveRun(
    val traceId: String,
    val sessionId: String,
)

private const val EVENT_RUN_STARTED = "run_started"
private const val EVENT_COMPLETION_DELTA = "completion_delta"
private const val EVENT_TOOL_CALL_STARTED = "tool_call_started"
private const val EVENT_TOOL_CALL_FINISHED = "tool_call_finished"
private const val EVENT_AWAITING_HUMAN = "awaiting_human"
private const val EVENT_MESSAGE = "message"
private const val EVENT_DONE = "done"
private const val EVENT_ERROR = "error"
private const val EVENT_ASSISTANT_MESSAGE = "assistant_message"
private const val COMPLETION_DELTA_TEXT = "text"
private const val TOOL_STATUS_RUNNING = "running"
private const val TOOL_STATUS_SUCCESS = "success"
private const val TOOL_STATUS_ERROR = "error"
private const val LOG_TAG = "GhostOS-ChatViewModel"

class ChatViewModel(
    private val store: ChatSettingsStore,
    private val clientFactory: (String, String) -> BridgeGateway = { url, token -> BridgeClient(url, token) },
) : ViewModel() {
    private val _messages = MutableStateFlow<List<ChatMessage>>(emptyList())
    val messages: StateFlow<List<ChatMessage>> = _messages.asStateFlow()

    private val _state = MutableStateFlow<ChatState>(ChatState.Idle)
    val state: StateFlow<ChatState> = _state.asStateFlow()

    private val _sessionId = MutableStateFlow("")
    val sessionId: StateFlow<String> = _sessionId.asStateFlow()

    private val _pendingQuestion = MutableStateFlow<PendingQuestion?>(null)
    val pendingQuestion: StateFlow<PendingQuestion?> = _pendingQuestion.asStateFlow()

    private val _sessions = MutableStateFlow<List<SessionMetadata>>(emptyList())
    val sessions: StateFlow<List<SessionMetadata>> = _sessions.asStateFlow()

    private val _sessionsLoading = MutableStateFlow(false)
    val sessionsLoading: StateFlow<Boolean> = _sessionsLoading.asStateFlow()

    private val _sessionsError = MutableStateFlow("")
    val sessionsError: StateFlow<String> = _sessionsError.asStateFlow()

    private val _historyLoading = MutableStateFlow(false)
    val historyLoading: StateFlow<Boolean> = _historyLoading.asStateFlow()

    private val _canStop = MutableStateFlow(false)
    val canStop: StateFlow<Boolean> = _canStop.asStateFlow()

    private val _stopPending = MutableStateFlow(false)
    val stopPending: StateFlow<Boolean> = _stopPending.asStateFlow()

    private var client: BridgeGateway? = null
    private var sessionEventsJob: Job? = null
    private var activeRunJob: Job? = null
    private var skipHistoryBootstrapForSessionId: String? = null
    private val json = Json { ignoreUnknownKeys = true }

    private val streamingMessageIdsByTrace = linkedMapOf<String, String>()
    private val toolMessageIdsByKey = linkedMapOf<String, String>()
    private val knownSessionIdsByTrace = linkedMapOf<String, String>()
    private val localTraceIds = linkedSetOf<String>()
    private val stopRequestedTraceIds = linkedSetOf<String>()
    private var activeRun: ActiveRun? = null

    init {
        observeBridgeConnection()
        observeSessionBinding()
    }

    fun sendMessage(text: String) {
        val message = text.trim()
        if (message.isBlank()) {
            return
        }

        val currentClient = client ?: run {
            _state.value = ChatState.Error("未配置连接")
            return
        }

        val pending = _pendingQuestion.value
        val traceId = createClientTraceId(if (pending == null) "android-run" else "android-answer")
        registerLocalTrace(traceId)
        stopRequestedTraceIds.remove(traceId)
        appendMessage(buildUserMessage(message, traceId, pending?.sessionId ?: _sessionId.value))
        _pendingQuestion.value = null
        _state.value = ChatState.Loading
        setActiveRun(traceId, pending?.sessionId ?: _sessionId.value)

        activeRunJob?.cancel()
        activeRunJob = viewModelScope.launch {
            runCatching {
                val stream = if (pending == null) {
                    currentClient.streamMessage(message, _sessionId.value.ifBlank { null }, traceId)
                } else {
                    currentClient.streamAnswer(pending.sessionId, pending.questionId, message, traceId)
                }
                stream.collect { event ->
                    handleAgentStreamEvent(currentClient, event)
                }
                if (wasStopRequested(traceId)) {
                    finishRun(traceId)
                    if (_state.value !is ChatState.Error) {
                        _state.value = ChatState.Idle
                    }
                }
            }.onFailure { error ->
                if (error is CancellationException && wasStopRequested(traceId)) {
                    finishRun(traceId)
                    if (_state.value !is ChatState.Error) {
                        _state.value = ChatState.Idle
                    }
                    return@onFailure
                }
                completeStreamingAssistant(traceId)
                _state.value = ChatState.Error(error.readableMessage())
                appendMessage(buildErrorMessage(error.readableMessage(), traceId, currentSessionIdForTrace(traceId)))
                finishRun(traceId)
            }
        }
    }

    fun stopCurrentRun() {
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

    fun loadSessions() {
        viewModelScope.launch {
            runCatching {
                refreshSessions(client)
            }.onFailure { error ->
                Log.e(LOG_TAG, "loadSessions failed", error)
                _sessionsError.value = error.readableMessage()
            }
        }
    }

    fun selectSession(id: String) {
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

    fun createNewSession() {
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

    fun clearError() {
        if (_state.value is ChatState.Error) {
            _state.value = ChatState.Idle
        }
    }

    fun openAttachment(context: Context, attachment: ChatAttachment) {
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

    private fun observeBridgeConnection() {
        viewModelScope.launch {
            combine(store.baseUrl, store.token) { url, token ->
                url.trim() to token.trim()
            }.collectLatest { (url, token) ->
                runCatching {
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
                }.onFailure { error ->
                    Log.e(LOG_TAG, "observeBridgeConnection update failed", error)
                    _state.value = ChatState.Error(error.readableMessage())
                }
            }
        }
    }

    private fun observeSessionBinding() {
        viewModelScope.launch {
            var previousSessionId = ""
            store.sessionId.collectLatest { storedID ->
                runCatching {
                    val trimmedID = storedID.trim()
                    val changed = previousSessionId != trimmedID
                    previousSessionId = trimmedID

                    if (_pendingQuestion.value?.sessionId != trimmedID) {
                        _pendingQuestion.value = null
                    }
                    _sessionId.value = trimmedID
                    bindSessionEvents(client, trimmedID)

                    if (!changed) {
                        return@collectLatest
                    }
                    if (trimmedID.isBlank()) {
                        clearConversation()
                        return@collectLatest
                    }

                    val skippedID = skipHistoryBootstrapForSessionId
                    if (skippedID != null && skippedID == trimmedID) {
                        skipHistoryBootstrapForSessionId = null
                        return@collectLatest
                    }

                    val currentClient = client ?: return@collectLatest
                    loadSessionHistory(currentClient, trimmedID, updateErrorState = false)
                }.onFailure { error ->
                    Log.e(LOG_TAG, "observeSessionBinding update failed", error)
                    _state.value = ChatState.Error(error.readableMessage())
                }
            }
        }
    }

    private suspend fun refreshSessions(currentClient: BridgeGateway?) {
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

    private suspend fun loadSessionHistory(
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

    private suspend fun persistSessionId(sessionId: String) {
        val trimmedID = sessionId.trim()
        if (_sessionId.value == trimmedID) {
            return
        }
        _sessionId.value = trimmedID
        store.saveSessionId(trimmedID)
    }

    private suspend fun bindRuntimeSession(sessionId: String) {
        val trimmedID = sessionId.trim()
        if (trimmedID.isBlank()) {
            return
        }
        skipHistoryBootstrapForSessionId = trimmedID
        persistSessionId(trimmedID)
    }

    private fun setActiveRun(traceId: String, sessionId: String) {
        activeRun = ActiveRun(traceId = traceId.trim(), sessionId = sessionId.trim())
        _stopPending.value = false
        updateCanStop()
    }

    private fun updateActiveRunSession(traceId: String, sessionId: String) {
        val current = activeRun ?: return
        if (current.traceId != traceId.trim()) {
            return
        }
        activeRun = current.copy(sessionId = sessionId.trim())
        updateCanStop()
    }

    private fun finishRun(traceId: String) {
        val normalizedTraceId = traceId.trim()
        if (activeRun?.traceId == normalizedTraceId) {
            activeRun = null
            activeRunJob = null
        }
        _stopPending.value = false
        updateCanStop()
        completeTrace(normalizedTraceId)
    }

    private fun wasStopRequested(traceId: String): Boolean {
        return stopRequestedTraceIds.contains(traceId.trim())
    }

    private fun updateCanStop() {
        _canStop.value = activeRun != null && !_stopPending.value
    }

    private fun handleStopSuccess(run: ActiveRun, response: AgentStopResponsePayload) {
        stopRequestedTraceIds += run.traceId
        completeStreamingAssistant(run.traceId)
        appendMessage(buildSystemMessage(response.message))
        activeRunJob?.cancel()
        if (_state.value !is ChatState.Error) {
            _state.value = ChatState.Idle
        }
        finishRun(run.traceId)
    }

    private fun bindSessionEvents(currentClient: BridgeGateway?, sessionId: String) {
        sessionEventsJob?.cancel()
        sessionEventsJob = null

        if (currentClient == null || sessionId.isBlank()) {
            return
        }

        sessionEventsJob = viewModelScope.launch {
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
    }

    private suspend fun handleAgentStreamEvent(currentClient: BridgeGateway, event: AgentStreamEvent) {
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

    private suspend fun handleSessionPushEvent(currentClient: BridgeGateway, event: SessionPushEvent) {
        val traceId = event.traceId?.trim().orEmpty()
        if (traceId.isNotBlank() && localTraceIds.contains(traceId)) {
            return
        }
        when (event.type) {
            EVENT_ASSISTANT_MESSAGE -> {
                val payload = json.decodeFromJsonElement<SessionPushAssistantMessagePayload>(event.payload)
                appendMessage(buildAssistantMessage(payload.message, traceId, event.sessionId))
                _pendingQuestion.value = null
                _state.value = ChatState.Idle
                refreshSessions(currentClient)
            }

            EVENT_AWAITING_HUMAN,
            EVENT_RUN_STARTED,
            EVENT_COMPLETION_DELTA,
            EVENT_TOOL_CALL_STARTED,
            EVENT_TOOL_CALL_FINISHED,
            EVENT_ERROR,
            EVENT_DONE,
            -> handleRuntimeEvent(
                currentClient = currentClient,
                eventType = event.type,
                traceId = traceId,
                sessionId = event.sessionId,
                payload = event.payload,
                stepId = "",
                refreshSessionsOnTerminal = event.type == EVENT_AWAITING_HUMAN || event.type == EVENT_DONE || event.type == EVENT_ERROR,
            )
        }
    }

    private suspend fun handleRuntimeEvent(
        currentClient: BridgeGateway,
        eventType: String,
        traceId: String,
        sessionId: String?,
        payload: kotlinx.serialization.json.JsonObject,
        stepId: String,
        refreshSessionsOnTerminal: Boolean,
    ) {
        val normalizedTraceId = traceId.trim()
        val normalizedSessionId = sessionId?.trim().orEmpty()
        if (normalizedSessionId.isNotBlank() && normalizedTraceId.isNotBlank()) {
            knownSessionIdsByTrace[normalizedTraceId] = normalizedSessionId
            bindRuntimeSession(normalizedSessionId)
            updateActiveRunSession(normalizedTraceId, normalizedSessionId)
        }

        when (eventType) {
            EVENT_RUN_STARTED -> {
                val decoded = json.decodeFromJsonElement<AgentRunStartedPayload>(payload)
                val resolvedSessionId = decoded.sessionId?.trim().orEmpty().ifBlank { normalizedSessionId }
                if (resolvedSessionId.isNotBlank() && normalizedTraceId.isNotBlank()) {
                    knownSessionIdsByTrace[normalizedTraceId] = resolvedSessionId
                    bindRuntimeSession(resolvedSessionId)
                    updateActiveRunSession(normalizedTraceId, resolvedSessionId)
                }
                ensureStreamingAssistant(normalizedTraceId, resolvedSessionId)
                _state.value = ChatState.Loading
            }

            EVENT_COMPLETION_DELTA -> {
                val decoded = json.decodeFromJsonElement<AgentCompletionDeltaPayload>(payload)
                if (decoded.kind == COMPLETION_DELTA_TEXT && !decoded.text.isNullOrEmpty()) {
                    appendStreamingText(normalizedTraceId, normalizedSessionId, decoded.text)
                }
                _state.value = ChatState.Loading
            }

            EVENT_TOOL_CALL_STARTED -> {
                val decoded = json.decodeFromJsonElement<AgentToolCallStartedPayload>(payload)
                upsertToolMessage(
                    traceId = normalizedTraceId,
                    sessionId = normalizedSessionId,
                    stepId = stepId,
                    toolName = decoded.tool,
                    toolCallId = decoded.toolCallId,
                    status = TOOL_STATUS_RUNNING,
                    text = "使用工具：${displayToolName(decoded.tool)}",
                )
                _state.value = ChatState.Loading
            }

            EVENT_TOOL_CALL_FINISHED -> {
                val decoded = json.decodeFromJsonElement<AgentToolCallFinishedPayload>(payload)
                val isError = !decoded.error.isNullOrBlank() || decoded.status == TOOL_STATUS_ERROR
                upsertToolMessage(
                    traceId = normalizedTraceId,
                    sessionId = normalizedSessionId,
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

            EVENT_AWAITING_HUMAN -> {
                val decoded = json.decodeFromJsonElement<SessionPushAwaitingHumanPayload>(payload)
                completeStreamingAssistant(normalizedTraceId)
                val resolvedSessionId = normalizedSessionId.ifBlank { currentSessionIdForTrace(normalizedTraceId) }
                _pendingQuestion.value = PendingQuestion(
                    sessionId = resolvedSessionId,
                    questionId = decoded.questionId,
                    prompt = decoded.prompt,
                    selectionMode = decoded.selectionMode,
                    options = decoded.options,
                )
                if (_messages.value.none { it.kind == ChatMessageKind.PendingQuestion && it.questionId == decoded.questionId }) {
                    appendMessage(buildPendingQuestionMessage(decoded, normalizedTraceId, resolvedSessionId))
                }
                _state.value = ChatState.Idle
                finishRun(normalizedTraceId)
                if (refreshSessionsOnTerminal) {
                    refreshSessions(currentClient)
                }
            }

            EVENT_MESSAGE -> {
                val decoded = json.decodeFromJsonElement<AgentStreamMessagePayload>(payload)
                val resolvedSessionId = decoded.sessionId?.trim().orEmpty().ifBlank { normalizedSessionId }
                finalizeStreamingAssistant(normalizedTraceId, resolvedSessionId, decoded.text)
            }

            EVENT_ERROR -> {
                val decoded = json.decodeFromJsonElement<AgentErrorPayload>(payload)
                completeStreamingAssistant(normalizedTraceId)
                appendMessage(buildErrorMessage(decoded.message, normalizedTraceId, normalizedSessionId))
                _state.value = ChatState.Error(decoded.message)
                finishRun(normalizedTraceId)
                if (refreshSessionsOnTerminal) {
                    refreshSessions(currentClient)
                }
            }

            EVENT_DONE -> {
                val decoded = json.decodeFromJsonElement<AgentDonePayload>(payload)
                val resolvedSessionId = decoded.sessionId?.trim().orEmpty().ifBlank { normalizedSessionId }
                completeStreamingAssistant(normalizedTraceId)
                if (_state.value !is ChatState.Error) {
                    _state.value = ChatState.Idle
                }
                if (resolvedSessionId.isNotBlank() && normalizedTraceId.isNotBlank()) {
                    knownSessionIdsByTrace[normalizedTraceId] = resolvedSessionId
                }
                if (refreshSessionsOnTerminal) {
                    refreshSessions(currentClient)
                }
                if (resolvedSessionId.isNotBlank() && _sessionId.value == resolvedSessionId) {
                    loadSessionHistory(currentClient, resolvedSessionId, updateErrorState = false)
                }
                finishRun(normalizedTraceId)
            }
        }
    }

    private fun ensureStreamingAssistant(traceId: String, sessionId: String): String {
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

    private fun appendStreamingText(traceId: String, sessionId: String, text: String) {
        val messageId = ensureStreamingAssistant(traceId, sessionId)
        updateMessage(messageId) { current ->
            current.copy(
                text = current.text + text,
                sessionId = current.sessionId ?: sessionId.ifBlank { null },
                isStreaming = true,
            )
        }
    }

    private fun finalizeStreamingAssistant(traceId: String, sessionId: String, finalText: String) {
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

    private fun completeStreamingAssistant(traceId: String) {
        val messageId = streamingMessageIdsByTrace.remove(traceId) ?: return
        updateMessage(messageId) { current ->
            current.copy(isStreaming = false)
        }
    }

    private fun upsertToolMessage(
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

    private fun currentSessionIdForTrace(traceId: String): String {
        return knownSessionIdsByTrace[traceId]?.trim().orEmpty()
    }

    private fun registerLocalTrace(traceId: String) {
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

    private fun completeTrace(traceId: String) {
        streamingMessageIdsByTrace.remove(traceId)
        knownSessionIdsByTrace.remove(traceId)
        toolMessageIdsByKey.keys.removeAll { it.startsWith("$traceId|") }
    }

    private fun appendMessage(message: ChatMessage) {
        _messages.value = _messages.value + message
    }

    private fun updateMessage(messageId: String, transform: (ChatMessage) -> ChatMessage) {
        _messages.value = _messages.value.map { message ->
            if (message.id == messageId) transform(message) else message
        }
    }

    private fun clearConversation() {
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

    private fun clearRuntimeTracking() {
        streamingMessageIdsByTrace.clear()
        toolMessageIdsByKey.clear()
        knownSessionIdsByTrace.clear()
        stopRequestedTraceIds.clear()
    }

    private fun mapSessionDetail(detail: SessionDetail): SessionHistorySnapshot {
        val messages = mutableListOf<ChatMessage>()
        var pendingQuestion: PendingQuestion? = null

        detail.messages.forEach { message ->
            when (message.role) {
                "user" -> messages += buildUserMessage(message.text.orEmpty())
                "assistant" -> messages += buildAssistantMessage(message.text.orEmpty())
                "system" -> messages += buildSystemMessage(message.text?.ifBlank { "[system]" } ?: "[system]")
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

    private fun buildPendingQuestionMessage(interaction: SessionHumanInteraction): ChatMessage {
        return ChatMessage(
            kind = ChatMessageKind.PendingQuestion,
            text = interaction.prompt,
            questionId = interaction.questionId,
        )
    }
}

private fun buildUserMessage(text: String, traceId: String? = null, sessionId: String? = null): ChatMessage {
    return ChatMessage(
        kind = ChatMessageKind.User,
        text = text,
        traceId = traceId,
        sessionId = sessionId,
    )
}

private fun buildAssistantMessage(text: String, traceId: String? = null, sessionId: String? = null): ChatMessage {
    return ChatMessage(
        kind = ChatMessageKind.Assistant,
        text = text,
        traceId = traceId,
        sessionId = sessionId,
    )
}

private fun buildSystemMessage(text: String): ChatMessage {
    return ChatMessage(kind = ChatMessageKind.System, text = text)
}

private fun buildErrorMessage(text: String, traceId: String? = null, sessionId: String? = null): ChatMessage {
    return ChatMessage(
        kind = ChatMessageKind.Error,
        text = text,
        traceId = traceId,
        sessionId = sessionId,
    )
}

private fun buildPendingQuestionMessage(
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

private fun formatToolHistoryText(fallbackText: String, toolResult: SessionToolResult?): String {
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

private fun toolMessageKey(traceId: String, toolCallId: String?, toolName: String?, stepId: String): String {
    val normalizedToolCallId = toolCallId?.trim().orEmpty()
    if (normalizedToolCallId.isNotBlank()) {
        return "$traceId|$normalizedToolCallId"
    }
    val normalizedToolName = toolName?.trim().orEmpty()
    return "$traceId|$stepId|$normalizedToolName"
}

private fun sessionIDFromPayload(payload: kotlinx.serialization.json.JsonObject): String? {
    return payload["session_id"]?.toString()?.trim('"')
}

private fun attachmentsFromContent(
    content: List<dev.ghostos.android.model.SessionContentPart>?,
    sessionId: String,
): List<ChatAttachment> {
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

private fun SessionFileContent.toChatAttachment(sessionId: String): ChatAttachment {
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

private fun writeDownloadedArtifact(context: Context, filename: String, bytes: ByteArray): File {
    val dir = File(context.cacheDir, "attachments")
    if (!dir.exists() && !dir.mkdirs()) {
        throw IllegalStateException("无法创建附件缓存目录")
    }
    val target = File(dir, filename)
    target.writeBytes(bytes)
    return target
}

private fun displayToolName(toolName: String?): String {
    return toolName?.trim().orEmpty().ifBlank { "unknown" }
}

private fun createClientTraceId(prefix: String): String {
    return "$prefix-${UUID.randomUUID()}"
}

private fun nextChatMessageId(): String = UUID.randomUUID().toString()
