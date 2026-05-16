// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

package dev.ghostos.android.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

@Serializable
data class SessionImageContent(
    val path: String? = null,
    val url: String? = null,
    @SerialName("mime_type")
    val mimeType: String? = null,
    val width: Int? = null,
    val height: Int? = null,
    val sha256: String? = null,
    val bytes: Int? = null
)

@Serializable
data class SessionFileContent(
    @SerialName("artifact_id")
    val artifactId: String,
    val name: String,
    @SerialName("mime_type")
    val mimeType: String? = null,
    val bytes: Int? = null,
    val sha256: String? = null,
    @SerialName("download_url")
    val downloadUrl: String,
    @SerialName("source_path")
    val sourcePath: String? = null,
    val note: String? = null
)

@Serializable
data class SessionContentPart(
    val type: String,
    val text: String? = null,
    val image: SessionImageContent? = null,
    val file: SessionFileContent? = null
)

@Serializable
data class SessionToolCall(
    val id: String,
    val name: String,
    val arguments: JsonObject
)

@Serializable
data class SessionToolResult(
    val status: String,
    val tool: String,
    @SerialName("trace_id")
    val traceId: String? = null,
    val output: String? = null,
    val error: String? = null
)

@Serializable
data class SessionHumanInteraction(
    @SerialName("question_id")
    val questionId: String,
    val prompt: String,
    @SerialName("selection_mode")
    val selectionMode: String? = null,
    val options: List<AskHumanOption>? = null,
    val answer: String? = null
)

@Serializable
data class SessionMessage(
    val index: Int,
    val role: String,
    val text: String? = null,
    val content: List<SessionContentPart>? = null,
    @SerialName("tool_calls")
    val toolCalls: List<SessionToolCall>? = null,
    @SerialName("tool_result")
    val toolResult: SessionToolResult? = null,
    @SerialName("human_interaction")
    val humanInteraction: SessionHumanInteraction? = null,
    @SerialName("tool_call_id")
    val toolCallId: String? = null,
    @SerialName("in_progress")
    val inProgress: Boolean? = null,
    val thinking: String? = null
)

@Serializable
data class SessionMetadata(
    val id: String,
    val title: String,
    @SerialName("created_at")
    val createdAt: String,
    @SerialName("updated_at")
    val updatedAt: String,
    @SerialName("message_count")
    val messageCount: Int,
    @SerialName("token_count")
    val tokenCount: Int
)

@Serializable
data class SessionSidebarPartition(
    val id: String,
    val name: String
)

@Serializable
data class SessionSidebarPartitionState(
    val version: Int,
    val partitions: List<SessionSidebarPartition>,
    val assignments: Map<String, String>
)

@Serializable
data class SessionSidebarPartitionPutRequest(
    val version: Int,
    val partitions: List<SessionSidebarPartition>,
    val assignments: Map<String, String>,
    @SerialName("trace_id")
    val traceId: String? = null
)

@Serializable
data class SessionMessagePage(
    val limit: Int,
    val before: Int? = null,
    @SerialName("start_index")
    val startIndex: Int? = null,
    @SerialName("end_index")
    val endIndex: Int? = null,
    @SerialName("has_more_before")
    val hasMoreBefore: Boolean,
    @SerialName("next_before")
    val nextBefore: Int? = null
)

@Serializable
data class SessionTurnDraftSegment(
    val id: String,
    val content: String
)

@Serializable
data class SessionTurnDraftTool(
    val id: String,
    val content: String,
    @SerialName("tool_input")
    val toolInput: String? = null,
    @SerialName("tool_name")
    val toolName: String? = null,
    @SerialName("tool_status")
    val toolStatus: String? = null,
    @SerialName("tool_call_id")
    val toolCallId: String? = null,
    @SerialName("trace_id")
    val traceId: String? = null
)

@Serializable
data class SessionTurnDraft(
    @SerialName("trace_id")
    val traceId: String,
    val turn: Int,
    @SerialName("assistant_segments")
    val assistantSegments: List<SessionTurnDraftSegment>,
    @SerialName("thinking_segments")
    val thinkingSegments: List<SessionTurnDraftSegment>,
    val tools: List<SessionTurnDraftTool>,
    @SerialName("item_order")
    val itemOrder: List<String>
)

@Serializable
data class SessionDetail(
    val id: String,
    val title: String,
    val messages: List<SessionMessage>,
    @SerialName("created_at")
    val createdAt: String,
    @SerialName("updated_at")
    val updatedAt: String,
    @SerialName("message_count")
    val messageCount: Int,
    val page: SessionMessagePage,
    @SerialName("token_count")
    val tokenCount: Int,
    @SerialName("turn_draft")
    val turnDraft: SessionTurnDraft? = null
)
