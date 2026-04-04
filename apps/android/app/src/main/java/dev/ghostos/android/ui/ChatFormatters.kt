package dev.ghostos.android.ui

import java.time.Duration
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

internal fun currentSessionLabel(sessionId: String): String {
    val trimmedID = sessionId.trim()
    return if (trimmedID.isBlank()) "新聊天" else shortSessionId(trimmedID)
}

internal fun shortSessionId(sessionId: String): String {
    val trimmedID = sessionId.trim()
    return if (trimmedID.length <= 8) trimmedID else trimmedID.take(8)
}

internal fun formatRelativeTime(value: String): String {
    val instant = runCatching { Instant.parse(value) }.getOrNull() ?: return value
    val seconds = Duration.between(instant, Instant.now()).seconds.coerceAtLeast(0)
    return when {
        seconds < 60 -> "刚刚"
        seconds < 3600 -> "${seconds / 60} 分钟前"
        seconds < 86_400 -> "${seconds / 3600} 小时前"
        seconds < 604_800 -> "${seconds / 86_400} 天前"
        else -> DateTimeFormatter.ofPattern("yyyy-MM-dd")
            .withZone(ZoneId.systemDefault())
            .format(instant)
    }
}
