package dev.ghostos.android.network

import dev.ghostos.android.model.*
import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.channels.ProducerScope
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.withContext
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.net.URLEncoder
import java.util.concurrent.TimeUnit

class BridgeClient(
    private val baseUrl: String,
    private val token: String,
    private val ioDispatcher: CoroutineDispatcher = Dispatchers.IO,
) : BridgeGateway {
    private val client = OkHttpClient.Builder()
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(0, TimeUnit.SECONDS)
        .build()

    private val json = Json { ignoreUnknownKeys = true }

    override suspend fun testConnection(): Result<BridgeConfig> = withContext(ioDispatcher) {
        runCatching {
            val request = Request.Builder()
                .url("$baseUrl/api/config")
                .header("Authorization", "Bearer $token")
                .get()
                .build()

            client.newCall(request).execute().use { response ->
                val body = response.body?.string() ?: throw Exception("Empty response")
                if (!response.isSuccessful) {
                    throw Exception("HTTP ${response.code}: $body")
                }
                decodeSuccessEnvelope(body)
            }
        }
    }

    override suspend fun sendMessage(message: String, sessionId: String?): Result<AgentSendResponse> = withContext(ioDispatcher) {
        runCatching {
            val payload = AgentRequest(
                message = message,
                sessionId = sessionId,
            )
            val body = json.encodeToString(payload).toRequestBody("application/json".toMediaType())

            val request = Request.Builder()
                .url("$baseUrl/api/agent")
                .header("Authorization", "Bearer $token")
                .post(body)
                .build()

            client.newCall(request).execute().use { response ->
                val responseBody = response.body?.string() ?: throw Exception("Empty response")
                if (!response.isSuccessful) {
                    throw Exception("HTTP ${response.code}: $responseBody")
                }
                decodeAgentEnvelope(responseBody)
            }
        }
    }

    override suspend fun answerQuestion(sessionId: String, questionId: String, answer: String): Result<AgentSendResponse> = withContext(ioDispatcher) {
        runCatching {
            val payload = HumanResponseRequest(sessionId, questionId, answer)
            val body = json.encodeToString(payload).toRequestBody("application/json".toMediaType())

            val request = Request.Builder()
                .url("$baseUrl/api/questions/answer")
                .header("Authorization", "Bearer $token")
                .post(body)
                .build()

            client.newCall(request).execute().use { response ->
                val responseBody = response.body?.string() ?: throw Exception("Empty response")
                if (!response.isSuccessful) {
                    throw Exception("HTTP ${response.code}: $responseBody")
                }
                decodeAgentEnvelope(responseBody)
            }
        }
    }

    override suspend fun stopRun(sessionId: String?, traceId: String?): Result<AgentStopResponsePayload> = withContext(ioDispatcher) {
        runCatching {
            val normalizedSessionId = sessionId?.trim().orEmpty()
            val normalizedTraceId = traceId?.trim().orEmpty()
            if (normalizedSessionId.isBlank() && normalizedTraceId.isBlank()) {
                throw Exception("session_id or trace_id is required")
            }

            val body = json.encodeToString(
                ApiRequest(
                    action = "AGENT_STOP",
                    params = mapOf(
                        "session_id" to normalizedSessionId.ifBlank { null },
                        "trace_id" to normalizedTraceId.ifBlank { null },
                    ),
                    traceId = createClientTraceId("android-stop"),
                ),
            ).toRequestBody("application/json".toMediaType())

            val request = Request.Builder()
                .url("$baseUrl/api/bus")
                .header("Authorization", "Bearer $token")
                .post(body)
                .build()

            client.newCall(request).execute().use { response ->
                val responseBody = response.body?.string() ?: throw Exception("Empty response")
                if (!response.isSuccessful) {
                    throw Exception("HTTP ${response.code}: $responseBody")
                }
                decodeSuccessEnvelope(responseBody)
            }
        }
    }

    override fun streamMessage(message: String, sessionId: String?, traceId: String): Flow<AgentStreamEvent> {
        val payload = AgentRequest(
            message = message,
            sessionId = sessionId,
            traceId = traceId,
        )
        val body = json.encodeToString(payload).toRequestBody("application/json".toMediaType())
        val request = Request.Builder()
            .url("$baseUrl/api/agent/stream")
            .header("Authorization", "Bearer $token")
            .post(body)
            .build()
        return observeEventStream(request, ::decodeAgentStreamEvent)
    }

    override fun streamAnswer(sessionId: String, questionId: String, answer: String, traceId: String): Flow<AgentStreamEvent> {
        val payload = HumanResponseRequest(sessionId, questionId, answer)
        val body = json.encodeToString(payload).toRequestBody("application/json".toMediaType())
        val request = Request.Builder()
            .url("$baseUrl/api/questions/answer/stream")
            .header("Authorization", "Bearer $token")
            .header("X-Trace-ID", traceId)
            .post(body)
            .build()
        return observeEventStream(request, ::decodeAgentStreamEvent)
    }

    override suspend fun listSessions(): Result<List<SessionMetadata>> = withContext(ioDispatcher) {
        runCatching {
            val request = Request.Builder()
                .url("$baseUrl/api/sessions")
                .header("Authorization", "Bearer $token")
                .get()
                .build()

            client.newCall(request).execute().use { response ->
                val body = response.body?.string() ?: throw Exception("Empty response")
                if (!response.isSuccessful) {
                    throw Exception("HTTP ${response.code}: $body")
                }
                decodeSuccessEnvelope(body)
            }
        }
    }

    override suspend fun getSession(sessionId: String): Result<SessionDetail> = withContext(ioDispatcher) {
        runCatching {
            val encodedID = URLEncoder.encode(sessionId, Charsets.UTF_8.name())
            val request = Request.Builder()
                .url("$baseUrl/api/sessions/$encodedID")
                .header("Authorization", "Bearer $token")
                .get()
                .build()

            client.newCall(request).execute().use { response ->
                val body = response.body?.string() ?: throw Exception("Empty response")
                if (!response.isSuccessful) {
                    throw Exception("HTTP ${response.code}: $body")
                }
                decodeSuccessEnvelope(body)
            }
        }
    }

    override suspend fun downloadSessionArtifact(sessionId: String, artifactId: String): Result<DownloadedArtifact> = withContext(ioDispatcher) {
        runCatching {
            val encodedSessionID = URLEncoder.encode(sessionId, Charsets.UTF_8.name())
            val encodedArtifactID = URLEncoder.encode(artifactId, Charsets.UTF_8.name())
            val request = Request.Builder()
                .url("$baseUrl/api/sessions/$encodedSessionID/artifacts/$encodedArtifactID")
                .header("Authorization", "Bearer $token")
                .get()
                .build()

            client.newCall(request).execute().use { response ->
                if (!response.isSuccessful) {
                    val responseBody = response.body?.string().orEmpty()
                    throw Exception("HTTP ${response.code}: $responseBody")
                }
                val body = response.body ?: throw Exception("Empty response")
                val mimeType = response.header("Content-Type")?.trim().orEmpty().ifBlank { "application/octet-stream" }
                val filename = parseFilename(
                    response.header("Content-Disposition"),
                    artifactId,
                    mimeType,
                )
                DownloadedArtifact(
                    filename = filename,
                    mimeType = mimeType,
                    bytes = body.bytes(),
                )
            }
        }
    }

    override fun observeSessionEvents(sessionId: String): Flow<SessionPushEvent> {
        val encodedID = URLEncoder.encode(sessionId, Charsets.UTF_8.name())
        val request = Request.Builder()
            .url("$baseUrl/api/sessions/$encodedID/events")
            .header("Authorization", "Bearer $token")
            .get()
            .build()
        return observeEventStream(request, ::decodeSessionPushEvent)
    }

    private fun <T> observeEventStream(request: Request, decoder: (String, String?) -> T): Flow<T> = callbackFlow {
        val call = client.newCall(request)

        val readerJob = launch(ioDispatcher) {
            runCatching {
                call.execute().use { response ->
                    val body = response.body ?: throw Exception("Empty response")
                    if (!response.isSuccessful) {
                        throw Exception("HTTP ${response.code}: ${body.string()}")
                    }

                    val source = body.source()
                    var eventName: String? = null
                    val dataLines = mutableListOf<String>()

                    while (!source.exhausted()) {
                        val line = source.readUtf8Line() ?: break
                        if (line.isEmpty()) {
                            this@callbackFlow.emitSseEvent(dataLines, eventName, decoder)
                            eventName = null
                            dataLines.clear()
                            continue
                        }

                        parseSseField(line)?.let { field ->
                            when (field.name) {
                                "event" -> eventName = field.value
                                "data" -> dataLines += field.value
                            }
                        }
                    }

                    this@callbackFlow.emitSseEvent(dataLines, eventName, decoder)
                }
            }.onFailure { error ->
                close(error)
            }.onSuccess {
                close()
            }
        }

        awaitClose {
            call.cancel()
            readerJob.cancel()
        }
    }

    private suspend fun <T> ProducerScope<T>.emitSseEvent(
        dataLines: List<String>,
        eventName: String?,
        decoder: (String, String?) -> T,
    ) {
        if (dataLines.isEmpty()) return
        send(decoder(dataLines.joinToString("\n"), eventName))
    }

    private inline fun <reified T> decodeSuccessEnvelope(responseBody: String): T {
        val envelope = json.decodeFromString<ApiEnvelope<T>>(responseBody)
        if (envelope.status == "error") {
            throw Exception(envelope.error.ifBlank { "Bridge returned an unknown error" })
        }
        return envelope.payload ?: throw Exception("No payload")
    }

    private fun decodeAgentEnvelope(responseBody: String): AgentSendResponse {
        val envelope = json.decodeFromString<ApiEnvelope<JsonElement>>(responseBody)
        if (envelope.status == "error") {
            throw Exception(envelope.error.ifBlank { "Bridge returned an unknown error" })
        }

        val payload = envelope.payload ?: throw Exception("No payload")
        val payloadStatus = payload.jsonObject["status"]?.jsonPrimitive?.contentOrNull
        return if (payloadStatus == "awaiting_human") {
            json.decodeFromJsonElement<AgentAwaitingHumanResponse>(payload)
        } else {
            json.decodeFromJsonElement<AgentSendSuccessResponse>(payload)
        }
    }

    private fun decodeSessionPushEvent(data: String, eventName: String?): SessionPushEvent {
        val event = json.decodeFromString<SessionPushEvent>(data)
        if (!eventName.isNullOrBlank() && event.type != eventName) {
            throw Exception("Unexpected event type: ${event.type} (header=$eventName)")
        }
        return event
    }

    private fun decodeAgentStreamEvent(data: String, eventName: String?): AgentStreamEvent {
        val event = json.decodeFromString<AgentStreamEvent>(data)
        if (!eventName.isNullOrBlank() && event.type != eventName) {
            throw Exception("Unexpected event type: ${event.type} (header=$eventName)")
        }
        return event
    }
}

private fun createClientTraceId(prefix: String): String {
    return "$prefix-${System.currentTimeMillis()}"
}

private data class SseField(val name: String, val value: String)

private fun parseSseField(line: String): SseField? {
    if (line.startsWith(":")) return null
    val separator = line.indexOf(':')
    if (separator < 0) return SseField(line, "")

    val name = line.substring(0, separator)
    var value = line.substring(separator + 1)
    if (value.startsWith(" ")) {
        value = value.substring(1)
    }
    return SseField(name, value)
}

private fun parseFilename(contentDisposition: String?, fallbackArtifactId: String, mimeType: String): String {
    val header = contentDisposition?.trim().orEmpty()
    if (header.isNotEmpty()) {
        Regex("""filename=\"([^\"]+)\"""")
            .find(header)
            ?.groupValues
            ?.getOrNull(1)
            ?.trim()
            ?.takeIf { it.isNotEmpty() }
            ?.let { return it }
    }

    val extension = when (mimeType.substringBefore(';').trim().lowercase()) {
        "text/plain" -> ".txt"
        "text/markdown" -> ".md"
        "application/json" -> ".json"
        "application/pdf" -> ".pdf"
        "image/png" -> ".png"
        "image/jpeg" -> ".jpg"
        "image/gif" -> ".gif"
        else -> ""
    }
    return if (fallbackArtifactId.endsWith(extension) || extension.isEmpty()) fallbackArtifactId else "$fallbackArtifactId$extension"
}
