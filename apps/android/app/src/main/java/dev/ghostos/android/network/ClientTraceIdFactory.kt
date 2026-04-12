package dev.ghostos.android.network

import java.util.UUID

internal interface ClientTraceIdFactory {
    fun create(prefix: String): String
}

internal object UuidTraceIdFactory : ClientTraceIdFactory {
    override fun create(prefix: String): String {
        return "$prefix-${UUID.randomUUID()}"
    }
}
