package dev.ghostos.android.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import dev.ghostos.android.viewmodel.ChatAttachment
import dev.ghostos.android.viewmodel.ChatMessage
import dev.ghostos.android.viewmodel.ChatMessageKind

@Composable
internal fun MessageBubble(message: ChatMessage, onOpenAttachment: (ChatAttachment) -> Unit) {
    val isUser = message.kind == ChatMessageKind.User
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (isUser) Arrangement.End else Arrangement.Start,
    ) {
        Column(
            modifier = Modifier.widthIn(max = 300.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp),
            horizontalAlignment = if (isUser) Alignment.End else Alignment.Start,
        ) {
            val badge = messageBadge(message)
            if (badge != null) {
                Text(
                    text = badge,
                    style = MaterialTheme.typography.labelSmall,
                    color = badgeColor(message),
                    fontWeight = FontWeight.SemiBold,
                )
            }

            Box(
                modifier = Modifier
                    .background(
                        color = bubbleColor(message),
                        shape = RoundedCornerShape(14.dp),
                    )
                    .padding(12.dp),
            ) {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(
                        text = bubbleText(message),
                        color = textColor(message),
                    )
                    if (message.attachments.isNotEmpty()) {
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            message.attachments.forEach { attachment ->
                                AttachmentCard(attachment, onOpen = { onOpenAttachment(attachment) })
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun AttachmentCard(attachment: ChatAttachment, onOpen: () -> Unit) {
    val colorScheme = MaterialTheme.colorScheme

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onOpen),
        colors = CardDefaults.cardColors(
            containerColor = colorScheme.surface.copy(alpha = 0.85f),
            contentColor = colorScheme.onSurface,
        ),
        shape = RoundedCornerShape(12.dp),
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 12.dp, vertical = 10.dp),
            verticalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            Text(text = attachment.name, fontWeight = FontWeight.SemiBold)
            Text(
                text = listOfNotNull(attachment.mimeType, formatAttachmentSize(attachment.bytes)).joinToString(" · ").ifBlank { "附件" },
                style = MaterialTheme.typography.bodySmall,
                color = colorScheme.onSurfaceVariant,
            )
            if (!attachment.note.isNullOrBlank()) {
                Text(
                    text = attachment.note.orEmpty(),
                    style = MaterialTheme.typography.bodySmall,
                    color = colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}

@Composable
private fun bubbleColor(message: ChatMessage): Color {
    return when (message.kind) {
        ChatMessageKind.User -> MaterialTheme.colorScheme.primary
        ChatMessageKind.Tool -> MaterialTheme.colorScheme.secondaryContainer
        ChatMessageKind.Error -> MaterialTheme.colorScheme.errorContainer
        ChatMessageKind.System -> MaterialTheme.colorScheme.surfaceVariant
        ChatMessageKind.PendingQuestion -> MaterialTheme.colorScheme.tertiaryContainer
        ChatMessageKind.Assistant -> MaterialTheme.colorScheme.surfaceVariant
    }
}

@Composable
private fun textColor(message: ChatMessage): Color {
    return when (message.kind) {
        ChatMessageKind.User -> MaterialTheme.colorScheme.onPrimary
        ChatMessageKind.Tool -> MaterialTheme.colorScheme.onSecondaryContainer
        ChatMessageKind.Error -> MaterialTheme.colorScheme.onErrorContainer
        ChatMessageKind.System -> MaterialTheme.colorScheme.onSurfaceVariant
        ChatMessageKind.PendingQuestion -> MaterialTheme.colorScheme.onTertiaryContainer
        ChatMessageKind.Assistant -> MaterialTheme.colorScheme.onSurface
    }
}

private fun messageBadge(message: ChatMessage): String? {
    return when (message.kind) {
        ChatMessageKind.User -> "你"
        ChatMessageKind.Assistant -> if (message.isStreaming) "正在思考" else "AI"
        ChatMessageKind.System -> "系统"
        ChatMessageKind.Tool -> listOfNotNull(
            message.toolName?.takeIf { it.isNotBlank() }?.let { "工具 · $it" },
            message.toolStatus?.uppercase(),
        ).joinToString("  ")
        ChatMessageKind.Error -> "错误"
        ChatMessageKind.PendingQuestion -> "等待回答"
    }.ifBlank { null }
}

@Composable
private fun badgeColor(message: ChatMessage): Color {
    return when (message.kind) {
        ChatMessageKind.Error -> MaterialTheme.colorScheme.error
        ChatMessageKind.Tool -> when (message.toolStatus) {
            "error" -> MaterialTheme.colorScheme.error
            "success" -> MaterialTheme.colorScheme.primary
            else -> MaterialTheme.colorScheme.secondary
        }
        ChatMessageKind.PendingQuestion -> MaterialTheme.colorScheme.tertiary
        else -> MaterialTheme.colorScheme.onSurfaceVariant
    }
}

private fun bubbleText(message: ChatMessage): String {
    if (message.kind == ChatMessageKind.Assistant && message.isStreaming && message.text.isBlank()) {
        return "正在思考…"
    }
    return message.text
}

private fun formatAttachmentSize(bytes: Int?): String {
    val size = bytes ?: return ""
    if (size <= 0) {
        return ""
    }
    return when {
        size < 1024 -> "$size B"
        size < 1024 * 1024 -> String.format("%.1f KB", size / 1024.0)
        size < 1024 * 1024 * 1024 -> String.format("%.1f MB", size / 1024.0 / 1024.0)
        else -> String.format("%.1f GB", size / 1024.0 / 1024.0 / 1024.0)
    }
}
