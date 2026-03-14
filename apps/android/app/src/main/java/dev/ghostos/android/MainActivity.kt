package dev.ghostos.android

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.runtime.*
import androidx.lifecycle.viewmodel.compose.viewModel
import dev.ghostos.android.store.SettingsStore
import dev.ghostos.android.ui.ChatScreen
import dev.ghostos.android.ui.ConfigScreen
import dev.ghostos.android.ui.theme.GhostOSTheme
import dev.ghostos.android.viewmodel.ChatViewModel

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val store = SettingsStore(applicationContext)

        setContent {
            GhostOSTheme {
                var showConfig by remember { mutableStateOf(false) }
                val viewModel: ChatViewModel = viewModel { ChatViewModel(store) }

                if (showConfig) {
                    ConfigScreen(store = store, onBack = { showConfig = false })
                } else {
                    ChatScreen(viewModel = viewModel, onNavigateToConfig = { showConfig = true })
                }
            }
        }
    }
}
