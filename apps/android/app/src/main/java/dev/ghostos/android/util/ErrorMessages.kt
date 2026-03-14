package dev.ghostos.android.util

import java.io.IOException
import java.net.ConnectException
import java.net.SocketTimeoutException
import java.net.UnknownHostException
import javax.net.ssl.SSLException

private fun Throwable.rootCause(): Throwable {
    var current: Throwable = this
    while (current.cause != null && current.cause !== current) {
        current = current.cause!!
    }
    return current
}

private fun Throwable.firstMessage(): String? {
    var current: Throwable? = this
    while (current != null) {
        val message = current.message?.trim()
        if (!message.isNullOrEmpty()) {
            return message
        }
        current = current.cause
    }
    return null
}

fun Throwable.readableMessage(): String {
    val root = rootCause()
    val detail = firstMessage()

    return when (root) {
        is UnknownHostException -> "无法解析服务器地址，请检查 URL 是否正确"
        is ConnectException -> detail ?: "无法连接到服务器，请确认 Bridge 已启动且地址与端口可访问"
        is SocketTimeoutException -> "连接超时，请检查网络后重试"
        is SSLException -> detail ?: "TLS/SSL 连接失败，请检查是否应使用 https:// 地址"
        is IOException -> detail ?: "网络请求失败，请检查网络连接"
        else -> detail ?: (root::class.java.simpleName.takeIf { it.isNotBlank() } ?: "未知错误")
    }
}
