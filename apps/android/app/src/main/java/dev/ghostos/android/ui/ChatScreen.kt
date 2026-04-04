package dev.ghostos.android.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.Button
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
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import dev.ghostos.android.viewmodel.ChatState
import dev.ghostos.android.viewmodel.ChatViewModel
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
