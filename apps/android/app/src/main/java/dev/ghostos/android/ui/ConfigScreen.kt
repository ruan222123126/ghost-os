package dev.ghostos.android.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import dev.ghostos.android.network.BridgeClient
import dev.ghostos.android.store.SettingsStore
import dev.ghostos.android.util.readableMessage
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ConfigScreen(store: SettingsStore, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val colorScheme = MaterialTheme.colorScheme
    var baseUrl by remember { mutableStateOf("") }
    var token by remember { mutableStateOf("") }
    var sessionId by remember { mutableStateOf("") }
    var testResult by remember { mutableStateOf<String?>(null) }
    var testing by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        store.baseUrl.collect { baseUrl = it }
    }
    LaunchedEffect(Unit) {
        store.token.collect { token = it }
    }
    LaunchedEffect(Unit) {
        store.sessionId.collect { sessionId = it }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("配置") },
                navigationIcon = {
                    TextButton(onClick = onBack) {
                        Text("返回")
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = colorScheme.surface,
                    titleContentColor = colorScheme.onSurface,
                    navigationIconContentColor = colorScheme.primary,
                ),
            )
        },
        containerColor = colorScheme.background,
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            OutlinedTextField(
                value = baseUrl,
                onValueChange = { baseUrl = it },
                label = { Text("Bridge URL") },
                placeholder = { Text("https://your-tunnel.example.com") },
                modifier = Modifier.fillMaxWidth(),
                colors = TextFieldDefaults.colors(
                    focusedContainerColor = colorScheme.surface,
                    unfocusedContainerColor = colorScheme.surface,
                    focusedTextColor = colorScheme.onSurface,
                    unfocusedTextColor = colorScheme.onSurface,
                    focusedLabelColor = colorScheme.primary,
                    unfocusedLabelColor = colorScheme.onSurfaceVariant,
                    focusedPlaceholderColor = colorScheme.onSurfaceVariant,
                    unfocusedPlaceholderColor = colorScheme.onSurfaceVariant,
                    focusedIndicatorColor = colorScheme.primary,
                    unfocusedIndicatorColor = colorScheme.outline,
                    cursorColor = colorScheme.primary,
                ),
            )

            OutlinedTextField(
                value = token,
                onValueChange = { token = it },
                label = { Text("API Token") },
                modifier = Modifier.fillMaxWidth(),
                colors = TextFieldDefaults.colors(
                    focusedContainerColor = colorScheme.surface,
                    unfocusedContainerColor = colorScheme.surface,
                    focusedTextColor = colorScheme.onSurface,
                    unfocusedTextColor = colorScheme.onSurface,
                    focusedLabelColor = colorScheme.primary,
                    unfocusedLabelColor = colorScheme.onSurfaceVariant,
                    focusedIndicatorColor = colorScheme.primary,
                    unfocusedIndicatorColor = colorScheme.outline,
                    cursorColor = colorScheme.primary,
                ),
            )

            OutlinedTextField(
                value = sessionId,
                onValueChange = { sessionId = it },
                label = { Text("Session ID（可选）") },
                placeholder = { Text("绑定已有会话后可接收跨端推送") },
                modifier = Modifier.fillMaxWidth(),
                colors = TextFieldDefaults.colors(
                    focusedContainerColor = colorScheme.surface,
                    unfocusedContainerColor = colorScheme.surface,
                    focusedTextColor = colorScheme.onSurface,
                    unfocusedTextColor = colorScheme.onSurface,
                    focusedLabelColor = colorScheme.primary,
                    unfocusedLabelColor = colorScheme.onSurfaceVariant,
                    focusedIndicatorColor = colorScheme.primary,
                    unfocusedIndicatorColor = colorScheme.outline,
                    cursorColor = colorScheme.primary,
                ),
            )

            Button(
                onClick = {
                    scope.launch {
                        store.saveBaseUrl(baseUrl.trim())
                        store.saveToken(token.trim())
                        store.saveSessionId(sessionId.trim())
                    }
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("保存")
            }

            Button(
                onClick = {
                    testing = true
                    testResult = null
                    scope.launch {
                        val client = BridgeClient(baseUrl.trim(), token.trim())
                        client.testConnection().fold(
                            onSuccess = { config ->
                                testResult = "连接成功\nProvider: ${config.provider}\nModel: ${config.model}\nAPI Key: ${if (config.apiKeySet) "已设置" else "未设置"}"
                            },
                            onFailure = { error ->
                                testResult = "连接失败: ${error.readableMessage()}"
                            },
                        )
                        testing = false
                    }
                },
                modifier = Modifier.fillMaxWidth(),
                enabled = !testing && baseUrl.isNotBlank() && token.isNotBlank(),
            ) {
                Text(if (testing) "测试中..." else "测试连接")
            }

            testResult?.let {
                Text(
                    text = it,
                    color = if (it.startsWith("连接成功")) Color(0xFF138A5C) else colorScheme.error,
                    modifier = Modifier.padding(top = 8.dp),
                )
            }
        }
    }
}
