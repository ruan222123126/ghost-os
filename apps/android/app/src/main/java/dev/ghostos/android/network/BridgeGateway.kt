package dev.ghostos.android.network

import dev.ghostos.android.model.AgentSendResponse
import dev.ghostos.android.model.AgentStopResponsePayload
import dev.ghostos.android.model.AgentStreamEvent
import dev.ghostos.android.model.BridgeConfig
import dev.ghostos.android.model.DownloadedArtifact
import dev.ghostos.android.model.SessionDetail
import dev.ghostos.android.model.SessionMetadata
import dev.ghostos.android.model.SessionPushEvent
import kotlinx.coroutines.flow.Flow

interface BridgeGateway {
    suspend fun testConnection(): Result<BridgeConfig>
    suspend fun sendMessage(message: String, sessionId: String?): Result<AgentSendResponse>
    suspend fun answerQuestion(sessionId: String, questionId: String, answer: String): Result<AgentSendResponse>
    suspend fun stopRun(sessionId: String?, traceId: String?): Result<AgentStopResponsePayload>
    fun streamMessage(message: String, sessionId: String?, traceId: String): Flow<AgentStreamEvent>
    fun streamAnswer(sessionId: String, questionId: String, answer: String, traceId: String): Flow<AgentStreamEvent>
    suspend fun listSessions(): Result<List<SessionMetadata>>
    suspend fun getSession(sessionId: String): Result<SessionDetail>
    suspend fun downloadSessionArtifact(sessionId: String, artifactId: String): Result<DownloadedArtifact>
    fun observeSessionEvents(sessionId: String): Flow<SessionPushEvent>
}
