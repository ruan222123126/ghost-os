package dev.ghostos.android.viewmodel

import dev.ghostos.android.model.AgentStreamEvent
import dev.ghostos.android.model.AgentSendResponse
import dev.ghostos.android.model.AskHumanOption
import dev.ghostos.android.model.BridgeConfig
import dev.ghostos.android.model.DownloadedArtifact
import dev.ghostos.android.model.SessionDetail
import dev.ghostos.android.model.SessionHumanInteraction
import dev.ghostos.android.model.SessionMessage
import dev.ghostos.android.model.SessionMetadata
import dev.ghostos.android.model.SessionPushEvent
import dev.ghostos.android.model.SessionToolResult
import dev.ghostos.android.network.BridgeGateway
import dev.ghostos.android.store.ChatSettingsStore
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.awaitCancellation
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.emptyFlow
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class ChatViewModelTest {
    @get:Rule
    val mainDispatcherRule = MainDispatcherRule()

    @Test
    fun `loadSessions populates history list on success`() = runTest(mainDispatcherRule.dispatcher.scheduler) {
        val sessions = listOf(
            SessionMetadata(
                id = "session-1",
                createdAt = "2026-03-07T10:00:00Z",
                updatedAt = "2026-03-07T11:00:00Z",
                messageCount = 2,
                tokenCount = 42,
            ),
        )
        val bridge = FakeBridgeGateway(listSessionsResult = Result.success(sessions))
        val viewModel = createViewModel(bridge)

        advanceUntilIdle()

        assertEquals(sessions, viewModel.sessions.value)
        assertFalse(viewModel.sessionsLoading.value)
        assertEquals("", viewModel.sessionsError.value)
    }

    @Test
    fun `selectSession loads detail and replaces chat messages`() = runTest(mainDispatcherRule.dispatcher.scheduler) {
        val detail = SessionDetail(
            id = "session-1",
            messages = listOf(
                SessionMessage(role = "user", text = "旧问题"),
                SessionMessage(role = "internal", text = "[GRAPHQL_EXECUTION_RESULT]\n{\"data\":{\"viewer\":{\"id\":\"1\"}}}"),
                SessionMessage(role = "assistant", text = "旧回答"),
            ),
            createdAt = "2026-03-07T10:00:00Z",
            updatedAt = "2026-03-07T11:00:00Z",
            tokenCount = 99,
        )
        val bridge = FakeBridgeGateway(
            listSessionsResult = Result.success(emptyList()),
            sessionDetails = mutableMapOf("session-1" to Result.success(detail)),
        )
        val store = FakeChatSettingsStore(baseUrl = "http://bridge.test", token = "token")
        val viewModel = ChatViewModel(store) { _, _ -> bridge }

        advanceUntilIdle()
        viewModel.selectSession("session-1")
        advanceUntilIdle()

        assertEquals("session-1", viewModel.sessionId.value)
        assertEquals(
            listOf("旧问题", "[GRAPHQL_EXECUTION_RESULT]\n{\"data\":{\"viewer\":{\"id\":\"1\"}}}", "旧回答"),
            viewModel.messages.value.map { it.text }
        )
        assertEquals(
            listOf(ChatMessageKind.User, ChatMessageKind.System, ChatMessageKind.Assistant),
            viewModel.messages.value.map { it.kind }
        )
        assertEquals(listOf("session-1"), store.savedSessionIds)
    }

    @Test
    fun `selectSession restores pending question from history detail`() = runTest(mainDispatcherRule.dispatcher.scheduler) {
        val detail = SessionDetail(
            id = "session-pending",
            messages = listOf(
                SessionMessage(role = "user", text = "继续前需要确认"),
                SessionMessage(
                    role = "tool",
                    humanInteraction = SessionHumanInteraction(
                        questionId = "q-1",
                        prompt = "是否继续执行？",
                        selectionMode = "single",
                        options = listOf(AskHumanOption(label = "继续")),
                        answer = null,
                    ),
                ),
            ),
            createdAt = "2026-03-07T10:00:00Z",
            updatedAt = "2026-03-07T11:00:00Z",
            tokenCount = 100,
        )
        val bridge = FakeBridgeGateway(
            listSessionsResult = Result.success(emptyList()),
            sessionDetails = mutableMapOf("session-pending" to Result.success(detail)),
        )
        val viewModel = createViewModel(bridge)

        advanceUntilIdle()
        viewModel.selectSession("session-pending")
        advanceUntilIdle()

        val pending = viewModel.pendingQuestion.value
        assertNotNull(pending)
        assertEquals("session-pending", pending?.sessionId)
        assertEquals("q-1", pending?.questionId)
        assertEquals("是否继续执行？", pending?.prompt)
        assertEquals(listOf("继续前需要确认", "是否继续执行？"), viewModel.messages.value.map { it.text })
    }

    @Test
    fun `createNewSession clears current binding without dropping history list`() = runTest(mainDispatcherRule.dispatcher.scheduler) {
        val sessions = listOf(
            SessionMetadata(
                id = "session-1",
                createdAt = "2026-03-07T10:00:00Z",
                updatedAt = "2026-03-07T11:00:00Z",
                messageCount = 2,
                tokenCount = 42,
            ),
        )
        val detail = SessionDetail(
            id = "session-1",
            messages = listOf(SessionMessage(role = "assistant", text = "hello")),
            createdAt = "2026-03-07T10:00:00Z",
            updatedAt = "2026-03-07T11:00:00Z",
            tokenCount = 42,
        )
        val bridge = FakeBridgeGateway(
            listSessionsResult = Result.success(sessions),
            sessionDetails = mutableMapOf("session-1" to Result.success(detail)),
        )
        val store = FakeChatSettingsStore(baseUrl = "http://bridge.test", token = "token")
        val viewModel = ChatViewModel(store) { _, _ -> bridge }

        advanceUntilIdle()
        viewModel.selectSession("session-1")
        advanceUntilIdle()
        viewModel.createNewSession()
        advanceUntilIdle()

        assertEquals("", viewModel.sessionId.value)
        assertTrue(viewModel.messages.value.isEmpty())
        assertNull(viewModel.pendingQuestion.value)
        assertEquals(sessions, viewModel.sessions.value)
        assertEquals(listOf("session-1", ""), store.savedSessionIds)
    }

    @Test
    fun `sendMessage streams assistant output and finalizes one assistant bubble`() = runTest(mainDispatcherRule.dispatcher.scheduler) {
        val bridge = FakeBridgeGateway(
            sessionDetails = mutableMapOf(
                "session-stream" to Result.success(
                    SessionDetail(
                        id = "session-stream",
                        messages = listOf(
                            SessionMessage(role = "user", text = "你好"),
                            SessionMessage(role = "assistant", text = "正在回复"),
                        ),
                        createdAt = "2026-03-15T10:00:00Z",
                        updatedAt = "2026-03-15T10:01:00Z",
                        tokenCount = 12,
                    ),
                ),
            ),
            streamMessageFlow = flowOf(
                streamEvent("trace-1", "run_started", payload("session_id" to "session-stream")),
                streamEvent("trace-1", "completion_delta", payload("kind" to "text", "text" to "正在")),
                streamEvent("trace-1", "completion_delta", payload("kind" to "text", "text" to "回复")),
                streamEvent("trace-1", "message", payload("text" to "正在回复", "session_id" to "session-stream")),
                streamEvent("trace-1", "done", payload("session_id" to "session-stream", "session_ended" to false)),
            ),
        )
        val viewModel = createViewModel(bridge)

        advanceUntilIdle()
        viewModel.sendMessage("你好")
        advanceUntilIdle()

        assertEquals(listOf("你好", "正在回复"), viewModel.messages.value.map { it.text })
        assertEquals(listOf(ChatMessageKind.User, ChatMessageKind.Assistant), viewModel.messages.value.map { it.kind })
        assertFalse(viewModel.messages.value.last().isStreaming)
        assertEquals(ChatState.Idle, viewModel.state.value)
        assertEquals("session-stream", viewModel.sessionId.value)
    }

    @Test
    fun `sendMessage shows tool activity and marks status`() = runTest(mainDispatcherRule.dispatcher.scheduler) {
        val bridge = FakeBridgeGateway(
            sessionDetails = mutableMapOf(
                "session-tools" to Result.success(
                    SessionDetail(
                        id = "session-tools",
                        messages = listOf(
                            SessionMessage(role = "user", text = "列一下文件"),
                            SessionMessage(
                                role = "tool",
                                text = "工具完成：list_files",
                                toolCallId = "call-1",
                                toolResult = SessionToolResult(
                                    status = "success",
                                    tool = "list_files",
                                    output = "工具完成：list_files",
                                ),
                            ),
                            SessionMessage(role = "assistant", text = "已完成"),
                        ),
                        createdAt = "2026-03-15T10:00:00Z",
                        updatedAt = "2026-03-15T10:01:00Z",
                        tokenCount = 18,
                    ),
                ),
            ),
            streamMessageFlow = flowOf(
                streamEvent("trace-2", "run_started", payload("session_id" to "session-tools")),
                streamEvent("trace-2", "tool_call_started", payload("tool" to "list_files", "tool_call_id" to "call-1")),
                streamEvent("trace-2", "tool_call_finished", payload("tool" to "list_files", "tool_call_id" to "call-1", "status" to "success")),
                streamEvent("trace-2", "message", payload("text" to "已完成", "session_id" to "session-tools")),
                streamEvent("trace-2", "done", payload("session_id" to "session-tools", "session_ended" to false)),
            ),
        )
        val viewModel = createViewModel(bridge)

        advanceUntilIdle()
        viewModel.sendMessage("列一下文件")
        advanceUntilIdle()

        val toolMessage = viewModel.messages.value.firstOrNull { it.kind == ChatMessageKind.Tool }
        assertNotNull(toolMessage)
        assertEquals("success", toolMessage?.toolStatus)
        assertTrue(toolMessage?.text?.contains("工具完成") == true)
    }

    @Test
    fun `sendMessage restores pending question from stream awaiting human`() = runTest(mainDispatcherRule.dispatcher.scheduler) {
        val bridge = FakeBridgeGateway(
            streamMessageFlow = flowOf(
                streamEvent("trace-3", "run_started", payload("session_id" to "session-await")),
                streamEvent(
                    "trace-3",
                    "awaiting_human",
                    payload(
                        "question_id" to "q-2",
                        "prompt" to "要继续吗？",
                        "selection_mode" to "single",
                    ),
                ),
            ),
        )
        val viewModel = createViewModel(bridge)

        advanceUntilIdle()
        viewModel.sendMessage("继续")
        advanceUntilIdle()

        assertEquals("q-2", viewModel.pendingQuestion.value?.questionId)
        assertEquals("要继续吗？", viewModel.pendingQuestion.value?.prompt)
        assertEquals(ChatState.Idle, viewModel.state.value)
        assertEquals(ChatMessageKind.PendingQuestion, viewModel.messages.value.last().kind)
    }

    @Test
    fun `stopCurrentRun stops active stream and returns idle`() = runTest(mainDispatcherRule.dispatcher.scheduler) {
        val bridge = FakeBridgeGateway(
            streamMessageFactory = { _, _, traceId -> flow {
                emit(streamEvent(traceId, "run_started", payload("session_id" to "session-stop")))
                awaitCancellation()
            } },
        )
        val viewModel = createViewModel(bridge)

        advanceUntilIdle()
        viewModel.sendMessage("请开始执行")
        advanceUntilIdle()

        assertTrue(viewModel.canStop.value)
        assertEquals(ChatState.Loading, viewModel.state.value)

        viewModel.stopCurrentRun()
        advanceUntilIdle()

        assertEquals(1, bridge.stopCalls.size)
        assertEquals("session-stop", bridge.stopCalls.first().sessionId)
        assertEquals(bridge.lastStreamTraceId, bridge.stopCalls.first().traceId)
        assertFalse(viewModel.canStop.value)
        assertFalse(viewModel.stopPending.value)
        assertEquals(ChatState.Idle, viewModel.state.value)
        assertTrue(viewModel.messages.value.any { it.kind == ChatMessageKind.System && it.text.contains("cancelled successfully") })
    }

    private fun createViewModel(bridge: FakeBridgeGateway): ChatViewModel {
        val store = FakeChatSettingsStore(baseUrl = "http://bridge.test", token = "token")
        return ChatViewModel(store) { _, _ -> bridge }
    }
}

private class FakeChatSettingsStore(
    baseUrl: String = "",
    token: String = "",
    sessionId: String = "",
) : ChatSettingsStore {
    override val baseUrl = MutableStateFlow(baseUrl)
    override val token = MutableStateFlow(token)
    override val sessionId = MutableStateFlow(sessionId)
    val savedSessionIds = mutableListOf<String>()

    override suspend fun saveSessionId(id: String) {
        savedSessionIds += id
        sessionId.value = id
    }
}

private class FakeBridgeGateway(
    private var listSessionsResult: Result<List<SessionMetadata>> = Result.success(emptyList()),
    private val sessionDetails: MutableMap<String, Result<SessionDetail>> = mutableMapOf(),
    private val streamMessageFlow: Flow<AgentStreamEvent> = emptyFlow(),
    private val streamAnswerFlow: Flow<AgentStreamEvent> = emptyFlow(),
    private val streamMessageFactory: ((String, String?, String) -> Flow<AgentStreamEvent>)? = null,
) : BridgeGateway {
    val stopCalls = mutableListOf<StopCall>()
    var lastStreamTraceId: String = ""

    override suspend fun testConnection(): Result<BridgeConfig> = Result.failure(UnsupportedOperationException())

    override suspend fun sendMessage(message: String, sessionId: String?): Result<AgentSendResponse> = Result.failure(UnsupportedOperationException())

    override suspend fun answerQuestion(sessionId: String, questionId: String, answer: String): Result<AgentSendResponse> = Result.failure(UnsupportedOperationException())

    override suspend fun stopRun(sessionId: String?, traceId: String?): Result<dev.ghostos.android.model.AgentStopResponsePayload> {
        stopCalls += StopCall(sessionId = sessionId.orEmpty(), traceId = traceId.orEmpty())
        return Result.success(dev.ghostos.android.model.AgentStopResponsePayload(status = "stopped", message = "agent run cancelled successfully"))
    }

    override fun streamMessage(message: String, sessionId: String?, traceId: String): Flow<AgentStreamEvent> {
        lastStreamTraceId = traceId
        return streamMessageFactory?.invoke(message, sessionId, traceId) ?: streamMessageFlow
    }

    override fun streamAnswer(sessionId: String, questionId: String, answer: String, traceId: String): Flow<AgentStreamEvent> = streamAnswerFlow

    override suspend fun listSessions(): Result<List<SessionMetadata>> = listSessionsResult

    override suspend fun getSession(sessionId: String): Result<SessionDetail> {
        return sessionDetails[sessionId] ?: Result.failure(IllegalArgumentException("missing session detail: $sessionId"))
    }

    override suspend fun downloadSessionArtifact(sessionId: String, artifactId: String): Result<DownloadedArtifact> {
        return Result.failure(UnsupportedOperationException())
    }

    override fun observeSessionEvents(sessionId: String): Flow<SessionPushEvent> = flow {
        awaitCancellation()
    }
}

private data class StopCall(
    val sessionId: String,
    val traceId: String,
)

private fun streamEvent(traceId: String, type: String, payload: JsonObject): AgentStreamEvent {
    return AgentStreamEvent(
        id = "$traceId-$type",
        stepId = "",
        traceId = traceId,
        turn = 0,
        type = type,
        payload = payload,
    )
}

private fun payload(vararg entries: Pair<String, Any?>): JsonObject {
    return buildJsonObject {
        entries.forEach { (key, value) ->
            when (value) {
                null -> Unit
                is String -> put(key, value)
                is Boolean -> put(key, value)
                is Int -> put(key, value)
                else -> error("unsupported payload type: ${value::class}")
            }
        }
    }
}
