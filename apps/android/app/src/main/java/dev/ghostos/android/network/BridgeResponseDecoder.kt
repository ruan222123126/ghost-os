package dev.ghostos.android.network

import dev.ghostos.android.model.AgentAwaitingHumanResponse
import dev.ghostos.android.model.AgentSendResponse
import dev.ghostos.android.model.AgentSendSuccessResponse
import dev.ghostos.android.model.AgentStreamEvent
import dev.ghostos.android.model.ApiEnvelope
import dev.ghostos.android.model.SessionPushEvent
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.decodeFromJsonElement
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

internal class BridgeResponseDecoder(private val json: Json) {
    inline fun <reified T> decodeSuccessEnvelope(responseBody: String): T {
        val envelope = json.decodeFromString<ApiEnvelope<T>>(responseBody)
        if (envelope.status == "error") {
            throw Exception(envelope.error.ifBlank { "Bridge returned an unknown error" })
        }
        return envelope.payload ?: throw Exception("No payload")
    }

    fun decodeAgentEnvelope(responseBody: String): AgentSendResponse {
        val envelope = json.decodeFromString<ApiEnvelope<JsonElement>>(responseBody)
        if (envelope.status == "error") {
            throw Exception(envelope.error.ifBlank { "Bridge returned an unknown error" })
        }

        val payload = envelope.payload ?: throw Exception("No payload")
        val payloadStatus = payload.jsonObject["status"]?.jsonPrimitive?.contentOrNull
        return if (payloadStatus == AgentAwaitingHumanResponse.STATUS_AWAITING_HUMAN) {
            json.decodeFromJsonElement<AgentAwaitingHumanResponse>(payload)
        } else {
            json.decodeFromJsonElement<AgentSendSuccessResponse>(payload)
        }
    }

    fun decodeSessionPushEvent(data: String, eventName: String?): SessionPushEvent {
        val event = json.decodeFromString<SessionPushEvent>(data)
        validateEventType(event.type, eventName)
        return event
    }

    fun decodeAgentStreamEvent(data: String, eventName: String?): AgentStreamEvent {
        val event = json.decodeFromString<AgentStreamEvent>(data)
        validateEventType(event.type, eventName)
        return event
    }

    private fun validateEventType(type: String, eventName: String?) {
        if (!eventName.isNullOrBlank() && type != eventName) {
            throw Exception("Unexpected event type: $type (header=$eventName)")
        }
    }
}
