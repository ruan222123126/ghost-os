package dev.ghostos.android.ui

import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Divider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import dev.ghostos.android.model.SessionMetadata

@Composable
internal fun SessionHistoryDrawer(
    sessionId: String,
    sessions: List<SessionMetadata>,
    loading: Boolean,
    error: String,
    onReload: () -> Unit,
    onNewChat: () -> Unit,
    onSelect: (String) -> Unit,
) {
    val colorScheme = MaterialTheme.colorScheme

    Column(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp, vertical = 18.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                text = "会话历史",
                style = MaterialTheme.typography.titleLarge,
                color = colorScheme.onSurface,
            )
            Text(
                text = "当前：${currentSessionLabel(sessionId)}",
                style = MaterialTheme.typography.bodyMedium,
                color = colorScheme.onSurfaceVariant,
            )
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Button(onClick = onNewChat) {
                    Text("新聊天")
                }
                TextButton(onClick = onReload) {
                    Text("刷新")
                }
            }
        }

        Divider(color = colorScheme.outline.copy(alpha = 0.35f))

        when {
            loading && sessions.isEmpty() -> {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = "正在加载会话…",
                        color = colorScheme.primary,
                    )
                }
            }

            sessions.isEmpty() -> {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(20.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    Text(
                        text = if (error.isBlank()) "暂无历史会话" else error,
                        color = if (error.isBlank()) colorScheme.onSurfaceVariant else colorScheme.error,
                    )
                }
            }

            else -> {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    if (error.isNotBlank()) {
                        item(key = "sessions-error") {
                            Text(
                                text = error,
                                color = colorScheme.error,
                                modifier = Modifier.padding(horizontal = 4.dp),
                            )
                        }
                    }
                    items(sessions, key = { it.id }) { session ->
                        SessionHistoryItem(
                            session = session,
                            selected = session.id == sessionId,
                            onClick = { onSelect(session.id) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SessionHistoryItem(
    session: SessionMetadata,
    selected: Boolean,
    onClick: () -> Unit,
) {
    val colorScheme = MaterialTheme.colorScheme
    val cardColor = if (selected) colorScheme.primaryContainer else colorScheme.surfaceVariant
    val borderColor = if (selected) colorScheme.primary else Color.Transparent

    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .border(1.dp, borderColor, RoundedCornerShape(16.dp)),
        colors = CardDefaults.cardColors(containerColor = cardColor),
        shape = RoundedCornerShape(16.dp),
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 14.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Text(
                text = "Session ${shortSessionId(session.id)}",
                style = MaterialTheme.typography.titleMedium,
                color = if (selected) colorScheme.onPrimaryContainer else colorScheme.onSurface,
            )
            Text(
                text = "创建 ${formatRelativeTime(session.createdAt)} · 更新 ${formatRelativeTime(session.updatedAt)}",
                style = MaterialTheme.typography.bodySmall,
                color = if (selected) colorScheme.onPrimaryContainer.copy(alpha = 0.8f) else colorScheme.onSurfaceVariant,
            )
        }
    }
}
