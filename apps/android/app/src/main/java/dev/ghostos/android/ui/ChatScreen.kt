package dev.ghostos.android.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Divider
import androidx.compose.material3.DrawerValue
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalDrawerSheet
import androidx.compose.material3.ModalNavigationDrawer
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberDrawerState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.platform.LocalContext
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import dev.ghostos.android.model.SessionMetadata
import dev.ghostos.android.viewmodel.ChatMessage
import dev.ghostos.android.viewmodel.ChatAttachment
import dev.ghostos.android.viewmodel.ChatMessageKind
import dev.ghostos.android.viewmodel.ChatState
import dev.ghostos.android.viewmodel.ChatViewModel
import java.time.Duration
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatScreen(viewModel: ChatViewModel, onNavigateToConfig: () -> Unit) {
    val messages by viewModel.messages.collectAsState()
    val state by viewModel.state.collectAsState()
    val pendingQuestion by viewModel.pendingQuestion.collectAsState()
    val canStop by viewModel.canStop.collectAsState()
    val stopPending by viewModel.stopPending.collectAsState()
    val sessionId by viewModel.sessionId.collectAsState()
    val sessions by viewModel.sessions.collectAsState()
    val sessionsLoading by viewModel.sessionsLoading.collectAsState()
    val sessionsError by viewModel.sessionsError.collectAsState()
    val historyLoading by viewModel.historyLoading.collectAsState()
    val colorScheme = MaterialTheme.colorScheme
    val context = LocalContext.current
    var input by remember { mutableStateOf("") }
    val listState = rememberLazyListState()
    val drawerState = rememberDrawerState(initialValue = DrawerValue.Closed)
    val coroutineScope = rememberCoroutineScope()

    LaunchedEffect(messages.size) {
        if (messages.isNotEmpty()) {
            listState.animateScrollToItem(messages.size - 1)
        }
    }

    ModalNavigationDrawer(
        drawerState = drawerState,
        drawerContent = {
            ModalDrawerSheet(
                drawerContainerColor = colorScheme.surface,
                drawerContentColor = colorScheme.onSurface,
            ) {
                SessionHistoryDrawer(
                    sessionId = sessionId,
                    sessions = sessions,
                    loading = sessionsLoading,
                    error = sessionsError,
                    onReload = viewModel::loadSessions,
                    onNewChat = {
                        coroutineScope.launch { drawerState.close() }
                        viewModel.createNewSession()
                    },
                    onSelect = { selectedId ->
                        coroutineScope.launch { drawerState.close() }
                        viewModel.selectSession(selectedId)
                    },
                )
            }
        },
    ) {
        Scaffold(
            topBar = {
                TopAppBar(
                    title = { Text("Ghost-OS") },
                    navigationIcon = {
                        TextButton(
                            onClick = {
                                viewModel.loadSessions()
                                coroutineScope.launch { drawerState.open() }
                            },
                        ) {
                            Text("历史")
                        }
                    },
                    actions = {
                        TextButton(onClick = onNavigateToConfig) {
                            Text("配置")
                        }
                    },
                    colors = TopAppBarDefaults.topAppBarColors(
                        containerColor = colorScheme.surface,
                        titleContentColor = colorScheme.onSurface,
                        navigationIconContentColor = colorScheme.primary,
                        actionIconContentColor = colorScheme.primary,
                    ),
                )
            },
            containerColor = colorScheme.background,
        ) { padding ->
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding),
            ) {
                if (historyLoading || state is ChatState.Loading) {
                    Text(
                        text = if (historyLoading) "正在加载会话…" else "正在处理中…",
                        modifier = Modifier.fillMaxWidth(),
                        color = colorScheme.primary,
                    )
                }

                LazyColumn(
                    state = listState,
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxWidth(),
                    contentPadding = PaddingValues(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    items(items = messages, key = { it.id }) { msg ->
                        MessageBubble(msg, onOpenAttachment = { attachment ->
                            viewModel.openAttachment(context, attachment)
                        })
                    }
                }

                Text(
                    text = "当前会话：${currentSessionLabel(sessionId)}",
                    color = colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
                )

                if (state is ChatState.Error) {
                    Text(
                        text = (state as ChatState.Error).message,
                        color = colorScheme.error,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                    )
                }

                if (pendingQuestion != null) {
                    Text(
                        text = "等待回答中，请直接在下方输入。",
                        color = colorScheme.tertiary,
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp),
                    )
                }

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    TextField(
                        value = input,
                        onValueChange = { input = it },
                        modifier = Modifier.weight(1f),
                        placeholder = { Text(if (pendingQuestion == null) "输入消息..." else "输入你的回答...") },
                        enabled = state !is ChatState.Loading && !historyLoading,
                        colors = TextFieldDefaults.colors(
                            focusedContainerColor = colorScheme.surface,
                            unfocusedContainerColor = colorScheme.surface,
                            disabledContainerColor = colorScheme.surfaceVariant,
                            focusedTextColor = colorScheme.onSurface,
                            unfocusedTextColor = colorScheme.onSurface,
                            focusedPlaceholderColor = colorScheme.onSurfaceVariant,
                            unfocusedPlaceholderColor = colorScheme.onSurfaceVariant,
                            focusedIndicatorColor = colorScheme.primary,
                            unfocusedIndicatorColor = colorScheme.outline,
                            cursorColor = colorScheme.primary,
                        ),
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(
                        onClick = {
                            if (canStop || stopPending) {
                                viewModel.stopCurrentRun()
                            } else if (input.isNotBlank()) {
                                viewModel.sendMessage(input)
                                input = ""
                            }
                        },
                        enabled = when {
                            stopPending -> false
                            canStop -> true
                            else -> state !is ChatState.Loading && !historyLoading && input.isNotBlank()
                        },
                    ) {
                        Text(
                            when {
                                stopPending -> "停止中"
                                canStop -> "停止"
                                pendingQuestion == null -> "发送"
                                else -> "回答"
                            },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SessionHistoryDrawer(
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

@Composable
fun MessageBubble(message: ChatMessage, onOpenAttachment: (ChatAttachment) -> Unit) {
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
        ChatMessageKind.Tool -> listOfNotNull(message.toolName?.takeIf { it.isNotBlank() }?.let { "工具 · $it" }, message.toolStatus?.uppercase()).joinToString("  ")
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

private fun currentSessionLabel(sessionId: String): String {
    val trimmedID = sessionId.trim()
    return if (trimmedID.isBlank()) "新聊天" else shortSessionId(trimmedID)
}

private fun shortSessionId(sessionId: String): String {
    val trimmedID = sessionId.trim()
    return if (trimmedID.length <= 8) trimmedID else trimmedID.take(8)
}

private fun formatRelativeTime(value: String): String {
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
