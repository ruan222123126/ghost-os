package dev.ghostos.android.store

import kotlinx.coroutines.flow.Flow

interface ChatSettingsStore {
    val baseUrl: Flow<String>
    val token: Flow<String>
    val sessionId: Flow<String>

    suspend fun saveSessionId(id: String)
}
