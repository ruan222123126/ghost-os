package dev.ghostos.android.network

import dev.ghostos.android.model.SessionPushEvent
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.Assert.assertEquals
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class BridgeClientTest {
    @Test
    fun `observeSessionEvents streams server sent events`() = runTest {
        val server = MockWebServer()
        server.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "text/event-stream")
                .setBody(
                    """
                    event: assistant_message
                    data: {"id":"evt-1","type":"assistant_message","session_id":"session-1","payload":{"message":"hello"}}
                    
                    """.trimIndent(),
                ),
        )

        server.start()
        try {
            val client = BridgeClient(
                baseUrl = server.url("/").toString().removeSuffix("/"),
                token = "token",
            )

            val event: SessionPushEvent = client.observeSessionEvents("session-1").first()

            assertEquals("assistant_message", event.type)
            assertEquals("session-1", event.sessionId)
            assertEquals("hello", event.payload["message"]?.toString()?.trim('"'))

            val recorded = server.takeRequest()
            assertEquals("/api/sessions/session-1/events", recorded.path)
            assertEquals("Bearer token", recorded.getHeader("Authorization"))
        } finally {
            server.shutdown()
        }
    }

    @Test
    fun `observeSessionEvents joins multiline data payload`() = runTest {
        val server = MockWebServer()
        server.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setHeader("Content-Type", "text/event-stream")
                .setBody(
                    """
                    event: assistant_message
                    data: {"id":"evt-2","type":"assistant_message","session_id":"session-2","payload":{
                    data:"message":"hello"
                    data:}}
                    
                    """.trimIndent(),
                ),
        )

        server.start()
        try {
            val client = BridgeClient(
                baseUrl = server.url("/").toString().removeSuffix("/"),
                token = "token",
            )

            val event: SessionPushEvent = client.observeSessionEvents("session-2").first()

            assertEquals("assistant_message", event.type)
            assertEquals("session-2", event.sessionId)
            assertEquals("hello", event.payload["message"]?.toString()?.trim('"'))
        } finally {
            server.shutdown()
        }
    }
}
