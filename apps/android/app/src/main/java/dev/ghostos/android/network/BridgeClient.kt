package dev.ghostos.android.network

import dev.ghostos.android.model.AgentRequest
import dev.ghostos.android.model.AgentSendResponse
import dev.ghostos.android.model.AgentStopRequest
import dev.ghostos.android.model.AgentStopResponsePayload
import dev.ghostos.android.model.AgentStreamEvent
import dev.ghostos.android.model.ApiRequest
import dev.ghostos.android.model.BridgeConfig
import dev.ghostos.android.model.DownloadedArtifact
import dev.ghostos.android.model.HumanResponseRequest
import dev.ghostos.android.model.SessionDetail
import dev.ghostos.android.model.SessionMetadata
import dev.ghostos.android.model.SessionPushEvent
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.withContext
import kotlinx.serialization.encodeToString
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import java.net.URLEncoder

private const val JSON_MEDIA_TYPE = "application/json"
private const val OCTET_STREAM_MIME = "application/octet-stream"

class BridgeClient internal constructor(
    private val baseUrl: String,
    private val token: String,
    private val dependencies: BridgeClientDependencies,
) : BridgeGateway {
    constructor(baseUrl: String, token: String) : this(
        baseUrl = baseUrl,
        token = token,
        dependencies = BridgeClientDependencies(),
    )

    constructor(baseUrl: String, token: String, ioDispatcher: CoroutineDispatcher) : this(
        baseUrl = baseUrl,
        token = token,
        dependencies = BridgeClientDependencies(ioDispatcher = ioDispatcher),
    )

    private val json = dependencies.json
    private val client = dependencies.httpClient
    private val ioDispatcher = dependencies.ioDispatcher
    private val traceIdFactory = dependencies.traceIdFactory
    private val filenameResolver = dependencies.filenameResolver
    private val decoder = BridgeResponseDecoder(json)
    private val streamObserver = SseEventStreamObserver(client, ioDispatcher)

    override suspend fun testConnection(): Result<BridgeConfig> = withContext(ioDispatcher) {
        runCatching {
            val request = authorizedRequestBuilder("/api/config")
                .get()
                .build()
            decoder.decodeSuccessEnvelope(executeJsonRequest(request))
        }
    }

    override suspend fun sendMessage(message: String, sessionId: String?): Result<AgentSendResponse> = withContext(ioDispatcher) {
        runCatching {
            val payload = AgentRequest(message = message, sessionId = sessionId)
            val request = postJsonRequest("/api/agent", payload)
            decoder.decodeAgentEnvelope(executeJsonRequest(request))
        }
    }

    override suspend fun answerQuestion(sessionId: String, questionId: String, answer: String): Result<AgentSendResponse> = withContext(ioDispatcher) {
        runCatching {
            val payload = HumanResponseRequest(sessionId, questionId, answer)
            val request = postJsonRequest("/api/questions/answer", payload)
            decoder.decodeAgentEnvelope(executeJsonRequest(request))
        }
    }

    override suspend fun stopRun(sessionId: String?, traceId: String?): Result<AgentStopResponsePayload> = withContext(ioDispatcher) {
        runCatching {
            val normalizedSessionId = sessionId?.trim().orEmpty()
            val normalizedTraceId = traceId?.trim().orEmpty()
            if (normalizedSessionId.isBlank() && normalizedTraceId.isBlank()) {
                throw Exception("session_id or trace_id is required")
            }

            val payload = ApiRequest(
                action = "AGENT_STOP",
                params = AgentStopRequest(
                    sessionId = normalizedSessionId.ifBlank { null },
                    traceId = normalizedTraceId.ifBlank { null },
                ),
                traceId = traceIdFactory.create("android-stop"),
            )
            val request = postJsonRequest("/api/bus", payload)
            decoder.decodeSuccessEnvelope(executeJsonRequest(request))
        }
    }

    override fun streamMessage(message: String, sessionId: String?, traceId: String): Flow<AgentStreamEvent> {
        val payload = AgentRequest(message = message, sessionId = sessionId, traceId = traceId)
        val request = postJsonRequest("/api/agent/stream", payload)
        return streamObserver.observe(request, decoder::decodeAgentStreamEvent)
    }

    override fun streamAnswer(sessionId: String, questionId: String, answer: String, traceId: String): Flow<AgentStreamEvent> {
        val payload = HumanResponseRequest(sessionId, questionId, answer)
        val request = postJsonRequest("/api/questions/answer/stream", payload, "X-Trace-ID" to traceId)
        return streamObserver.observe(request, decoder::decodeAgentStreamEvent)
    }

    override suspend fun listSessions(): Result<List<SessionMetadata>> = withContext(ioDispatcher) {
        runCatching {
            val request = authorizedRequestBuilder("/api/sessions")
                .get()
                .build()
            decoder.decodeSuccessEnvelope(executeJsonRequest(request))
        }
    }

    override suspend fun getSession(sessionId: String): Result<SessionDetail> = withContext(ioDispatcher) {
        runCatching {
            val encodedSessionID = encodePathSegment(sessionId)
            val request = authorizedRequestBuilder("/api/sessions/$encodedSessionID")
                .get()
                .build()
            decoder.decodeSuccessEnvelope(executeJsonRequest(request))
        }
    }

    override suspend fun downloadSessionArtifact(sessionId: String, artifactId: String): Result<DownloadedArtifact> = withContext(ioDispatcher) {
        runCatching {
            val encodedSessionID = encodePathSegment(sessionId)
            val encodedArtifactID = encodePathSegment(artifactId)
            val request = authorizedRequestBuilder("/api/sessions/$encodedSessionID/artifacts/$encodedArtifactID")
                .get()
                .build()
            executeRequest(request) { response ->
                val body = response.body ?: throw Exception("Empty response")
                if (!response.isSuccessful) {
                    val responseBody = body.string()
                    throw Exception("HTTP ${response.code}: $responseBody")
                }
                val mimeType = response.header("Content-Type")?.trim().orEmpty().ifBlank { OCTET_STREAM_MIME }
                val filename = filenameResolver.resolve(
                    response.header("Content-Disposition"),
                    artifactId,
                    mimeType,
                )
                DownloadedArtifact(filename = filename, mimeType = mimeType, bytes = body.bytes())
            }
        }
    }

    override fun observeSessionEvents(sessionId: String): Flow<SessionPushEvent> {
        val encodedSessionID = encodePathSegment(sessionId)
        val request = authorizedRequestBuilder("/api/sessions/$encodedSessionID/events")
            .get()
            .build()
        return streamObserver.observe(request, decoder::decodeSessionPushEvent)
    }

    private fun authorizedRequestBuilder(path: String): Request.Builder {
        return Request.Builder()
            .url("$baseUrl$path")
            .header("Authorization", "Bearer $token")
    }

    private inline fun <reified TPayload> postJsonRequest(
        path: String,
        payload: TPayload,
        extraHeader: Pair<String, String>? = null,
    ): Request {
        val body = json.encodeToString(payload).toRequestBody(JSON_MEDIA_TYPE.toMediaType())
        val builder = authorizedRequestBuilder(path).post(body)
        extraHeader?.let { header ->
            builder.header(header.first, header.second)
        }
        return builder.build()
    }

    private suspend fun executeJsonRequest(request: Request): String {
        return executeRequest(request) { response ->
            val responseBody = response.body?.string() ?: throw Exception("Empty response")
            if (!response.isSuccessful) {
                throw Exception("HTTP ${response.code}: $responseBody")
            }
            responseBody
        }
    }

    private suspend fun <T> executeRequest(request: Request, handler: (Response) -> T): T {
        return client.newCall(request).execute().use(handler)
    }

    private fun encodePathSegment(value: String): String {
        return URLEncoder.encode(value, Charsets.UTF_8.name())
    }
}
