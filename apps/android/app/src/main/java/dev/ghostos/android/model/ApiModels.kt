// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json (https://ghost-os.dev/schemas/bus-envelope.schema.json)

package dev.ghostos.android.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class ApiRequest<TParams>(
    val action: String,
    val params: TParams,
    @SerialName("trace_id") val traceId: String,
    @SerialName("request_id") val requestId: String? = null
)

@Serializable
data class ApiEnvelope<TPayload>(
    val status: String,
    val payload: TPayload? = null,
    val error: String = ""
)
