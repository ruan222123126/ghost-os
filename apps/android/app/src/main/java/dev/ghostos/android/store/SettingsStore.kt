package dev.ghostos.android.store

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map

private val Context.dataStore by preferencesDataStore("settings")

class SettingsStore(private val context: Context) : ChatSettingsStore {
    private val BASE_URL = stringPreferencesKey("base_url")
    private val TOKEN = stringPreferencesKey("token")
    private val SESSION_ID = stringPreferencesKey("session_id")

    override val baseUrl: Flow<String> = context.dataStore.data.map { it[BASE_URL] ?: "" }
    override val token: Flow<String> = context.dataStore.data.map { it[TOKEN] ?: "" }
    override val sessionId: Flow<String> = context.dataStore.data.map { it[SESSION_ID] ?: "" }

    suspend fun saveBaseUrl(url: String) {
        context.dataStore.edit { it[BASE_URL] = url }
    }

    suspend fun saveToken(token: String) {
        context.dataStore.edit { it[TOKEN] = token }
    }

    override suspend fun saveSessionId(id: String) {
        context.dataStore.edit { it[SESSION_ID] = id }
    }
}
