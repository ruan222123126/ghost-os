package dev.ghostos.android.network

import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.channels.ProducerScope
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import okhttp3.Request

internal class SseEventStreamObserver(
    private val client: OkHttpClient,
    private val ioDispatcher: CoroutineDispatcher,
) {
    fun <T> observe(request: Request, decoder: (String, String?) -> T): Flow<T> = callbackFlow {
        val call = client.newCall(request)
        val readerJob = launch(ioDispatcher) {
            runCatching {
                call.execute().use { response ->
                    val body = response.body ?: throw Exception("Empty response")
                    if (!response.isSuccessful) {
                        throw Exception("HTTP ${response.code}: ${body.string()}")
                    }
                    parseEventStream(this@callbackFlow, body.source(), decoder)
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

    private suspend fun <T> parseEventStream(
        scope: ProducerScope<T>,
        source: okio.BufferedSource,
        decoder: (String, String?) -> T,
    ) {
        var eventName: String? = null
        val dataLines = mutableListOf<String>()

        while (!source.exhausted()) {
            val line = source.readUtf8Line() ?: break
            if (line.isEmpty()) {
                scope.emitSseEvent(dataLines, eventName, decoder)
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
        scope.emitSseEvent(dataLines, eventName, decoder)
    }
}

private suspend fun <T> ProducerScope<T>.emitSseEvent(
    dataLines: List<String>,
    eventName: String?,
    decoder: (String, String?) -> T,
) {
    if (dataLines.isEmpty()) {
        return
    }
    send(decoder(dataLines.joinToString("\n"), eventName))
}

private data class SseField(val name: String, val value: String)

private fun parseSseField(line: String): SseField? {
    if (line.startsWith(":")) {
        return null
    }

    val separator = line.indexOf(':')
    if (separator < 0) {
        return SseField(line, "")
    }

    val name = line.substring(0, separator)
    val rawValue = line.substring(separator + 1)
    val value = if (rawValue.startsWith(" ")) rawValue.substring(1) else rawValue
    return SseField(name, value)
}
