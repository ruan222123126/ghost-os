package dev.ghostos.android.network

import kotlinx.coroutines.CoroutineDispatcher
import kotlinx.coroutines.Dispatchers
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient
import java.util.concurrent.TimeUnit

private const val CONNECT_TIMEOUT_SECONDS = 30L
private const val NO_READ_TIMEOUT_SECONDS = 0L

internal data class BridgeClientDependencies(
    val ioDispatcher: CoroutineDispatcher = Dispatchers.IO,
    val httpClient: OkHttpClient = defaultBridgeHttpClient(),
    val json: Json = defaultBridgeJson(),
    val traceIdFactory: ClientTraceIdFactory = UuidTraceIdFactory,
    val filenameResolver: ArtifactFilenameResolver = ArtifactFilenameResolver(),
)

internal fun defaultBridgeHttpClient(): OkHttpClient {
    return OkHttpClient.Builder()
        .connectTimeout(CONNECT_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        .readTimeout(NO_READ_TIMEOUT_SECONDS, TimeUnit.SECONDS)
        .build()
}

internal fun defaultBridgeJson(): Json {
    return Json { ignoreUnknownKeys = true }
}
