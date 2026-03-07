from __future__ import annotations

import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
SCHEMA_PATH = ROOT / "core" / "shared" / "schema.json"
_SCHEMA_CACHE: dict | None = None

GO_OUTPUT = ROOT / "core" / "bridge" / "app" / "envelope_generated.go"
RUST_OUTPUT = ROOT / "apps" / "cli" / "src" / "envelope_generated.rs"
TS_OUTPUT = ROOT / "apps" / "web" / "lib" / "envelope.generated.ts"
KOTLIN_OUTPUT = ROOT / "apps" / "android" / "app" / "src" / "main" / "java" / "dev" / "ghostos" / "android" / "model" / "ApiModels.kt"

GO_OBJECT_TYPES = {
    "assistantSessionEndSignal": "assistantSessionEndSignalPayload",
    "agentRequest": "agentRequest",
    "agentResponsePayload": "agentResponse",
    "agentAwaitingHumanPayload": "askHumanAwaitingResponse",
    "agentStopResponsePayload": "agentStopResponse",
    "humanResponseRequest": "humanResponseParams",
    "humanResponseAck": "humanResponseAck",
    "memoryDecisionQueryRequest": "memoryDecisionQueryRequest",
    "memoryDecisionStatsRequest": "memoryDecisionStatsRequest",
    "memoryDecisionRebuildRequest": "memoryDecisionRebuildRequest",
    "memoryDecisionQueryPayload": "memoryDecisionQueryPayload",
    "memoryDecisionStatsPayload": "memoryDecisionStatsPayload",
    "memoryDecisionRebuildPayload": "memoryDecisionRebuildPayload",
    "sessionImageContent": "sessionImageContent",
    "sessionContentPart": "sessionContentPart",
    "sessionToolCall": "sessionToolCall",
    "sessionToolResult": "sessionToolResult",
    "sessionHumanInteraction": "sessionHumanInteraction",
    "sessionMessage": "sessionMessage",
    "sessionMetadata": "sessionMetadata",
    "sessionDetail": "sessionDetail",
    "bridgeConfig": "configResponse",
    "configUpdate": "configUpdateRequest",
    "providerConfig": "providerConfigResponse",
    "providerConfigInput": "providerConfigInput",
    "providerListResponse": "providerListResponse",
    "setActiveProviderRequest": "setActiveProviderRequest",
}
TS_OBJECT_TYPES = {
    "assistantSessionEndSignal": "AssistantSessionEndSignal",
    "agentRequest": "AgentRequest",
    "agentResponsePayload": "AgentSendSuccessResponse",
    "agentAwaitingHumanPayload": "AgentSendAwaitingHumanResponse",
    "agentStopResponsePayload": "AgentStopResponsePayload",
    "humanResponseRequest": "HumanResponseRequest",
    "humanResponseAck": "HumanResponseAck",
    "memoryDecisionQueryRequest": "MemoryDecisionQueryRequest",
    "memoryDecisionStatsRequest": "MemoryDecisionStatsRequest",
    "memoryDecisionRebuildRequest": "MemoryDecisionRebuildRequest",
    "memoryDecisionQueryPayload": "MemoryDecisionQueryPayload",
    "memoryDecisionStatsPayload": "MemoryDecisionStatsPayload",
    "memoryDecisionRebuildPayload": "MemoryDecisionRebuildPayload",
    "sessionImageContent": "SessionImageContent",
    "sessionContentPart": "SessionContentPart",
    "sessionToolCall": "SessionToolCall",
    "sessionToolResult": "SessionToolResult",
    "sessionHumanInteraction": "SessionHumanInteraction",
    "sessionMessage": "SessionMessage",
    "sessionMetadata": "SessionMetadata",
    "sessionDetail": "SessionDetail",
    "bridgeConfig": "BridgeConfig",
    "configUpdate": "ConfigUpdate",
    "providerConfig": "ProviderConfig",
    "providerConfigInput": "ProviderConfigInput",
    "providerListResponse": "ProviderListResponse",
    "setActiveProviderRequest": "SetActiveProviderRequest",
}
TS_UNION_TYPES = {
    "agentSendResultPayload": "AgentSendResponse",
}
RUST_OBJECT_TYPES = {
    "assistantSessionEndSignal": "AssistantSessionEndSignal",
    "agentRequest": "AgentRequest",
    "agentResponsePayload": "AgentSendSuccessResponse",
    "agentAwaitingHumanPayload": "AgentSendAwaitingHumanResponse",
    "agentStopResponsePayload": "AgentStopResponsePayload",
    "humanResponseRequest": "HumanResponseRequest",
    "humanResponseAck": "HumanResponseAck",
    "memoryDecisionQueryRequest": "MemoryDecisionQueryRequest",
    "memoryDecisionStatsRequest": "MemoryDecisionStatsRequest",
    "memoryDecisionRebuildRequest": "MemoryDecisionRebuildRequest",
    "memoryDecisionQueryPayload": "MemoryDecisionQueryPayload",
    "memoryDecisionStatsPayload": "MemoryDecisionStatsPayload",
    "memoryDecisionRebuildPayload": "MemoryDecisionRebuildPayload",
    "sessionImageContent": "SessionImageContent",
    "sessionContentPart": "SessionContentPart",
    "sessionToolCall": "SessionToolCall",
    "sessionToolResult": "SessionToolResult",
    "sessionHumanInteraction": "SessionHumanInteraction",
    "sessionMessage": "SessionMessage",
    "sessionMetadata": "SessionMetadata",
    "sessionDetail": "SessionDetail",
    "bridgeConfig": "BridgeConfig",
    "configUpdate": "ConfigUpdate",
    "providerConfig": "ProviderConfig",
    "providerConfigInput": "ProviderConfigInput",
    "providerListResponse": "ProviderListResponse",
    "setActiveProviderRequest": "SetActiveProviderRequest",
}
RUST_UNION_TYPES = {
    "agentSendResultPayload": "AgentPayload",
}
RUST_UNION_VARIANTS = {
    "agentSendResultPayload": {
        "agentAwaitingHumanPayload": "AwaitingHuman",
        "agentResponsePayload": "Success",
    }
}
RUST_RESERVED_FIELDS = {"type"}
KOTLIN_OBJECT_TYPES = {
    "assistantSessionEndSignal": "AssistantSessionEndSignal",
    "agentRequest": "AgentRequest",
    "agentResponsePayload": "AgentSendSuccessResponse",
    "agentAwaitingHumanPayload": "AgentAwaitingHumanResponse",
    "agentStopResponsePayload": "AgentStopResponsePayload",
    "humanResponseRequest": "HumanResponseRequest",
    "humanResponseAck": "HumanResponseAck",
    "memoryDecisionQueryRequest": "MemoryDecisionQueryRequest",
    "memoryDecisionStatsRequest": "MemoryDecisionStatsRequest",
    "memoryDecisionRebuildRequest": "MemoryDecisionRebuildRequest",
    "memoryDecisionQueryPayload": "MemoryDecisionQueryPayload",
    "memoryDecisionStatsPayload": "MemoryDecisionStatsPayload",
    "memoryDecisionRebuildPayload": "MemoryDecisionRebuildPayload",
    "sessionImageContent": "SessionImageContent",
    "sessionContentPart": "SessionContentPart",
    "sessionToolCall": "SessionToolCall",
    "sessionToolResult": "SessionToolResult",
    "sessionHumanInteraction": "SessionHumanInteraction",
    "sessionMessage": "SessionMessage",
    "sessionMetadata": "SessionMetadata",
    "sessionDetail": "SessionDetail",
    "bridgeConfig": "BridgeConfig",
    "configUpdate": "ConfigUpdate",
    "providerConfig": "ProviderConfig",
    "providerConfigInput": "ProviderConfigInput",
    "providerListResponse": "ProviderListResponse",
    "setActiveProviderRequest": "SetActiveProviderRequest",
}
KOTLIN_UNION_TYPES = {
    "agentSendResultPayload": "AgentSendResponse",
}
KOTLIN_UNION_IMPLS = {
    "agentResponsePayload": "AgentSendResponse",
    "agentAwaitingHumanPayload": "AgentSendResponse",
}
GO_INITIALISMS = {
    "api": "API",
    "id": "ID",
    "url": "URL",
    "sha256": "SHA256",
}
SHARED_OBJECT_DEF_ORDER = [
    "assistantSessionEndSignal",
    "agentRequest",
    "agentResponsePayload",
    "agentAwaitingHumanPayload",
    "agentStopResponsePayload",
    "humanResponseRequest",
    "humanResponseAck",
    "memoryDecisionQueryRequest",
    "memoryDecisionStatsRequest",
    "memoryDecisionRebuildRequest",
    "memoryDecisionQueryPayload",
    "memoryDecisionStatsPayload",
    "memoryDecisionRebuildPayload",
    "sessionImageContent",
    "sessionContentPart",
    "sessionToolCall",
    "sessionToolResult",
    "sessionHumanInteraction",
    "sessionMessage",
    "sessionMetadata",
    "sessionDetail",
    "bridgeConfig",
    "configUpdate",
    "providerConfig",
    "providerConfigInput",
    "providerListResponse",
    "setActiveProviderRequest",
]
SHARED_UNION_DEF_ORDER = ["agentSendResultPayload"]


def load_schema() -> dict:
    global _SCHEMA_CACHE
    if _SCHEMA_CACHE is None:
        with SCHEMA_PATH.open("r", encoding="utf-8") as fp:
            _SCHEMA_CACHE = json.load(fp)
    return _SCHEMA_CACHE


def schema_ref_name(ref_value: str) -> str:
    return ref_value.rsplit("/", 1)[-1]


def dereference_schema(prop_schema: dict) -> dict:
    if "$ref" not in prop_schema:
        return prop_schema
    return load_schema()["$defs"][schema_ref_name(prop_schema["$ref"])]


def non_null_one_of_candidates(prop_schema: dict) -> list[dict]:
    return [candidate for candidate in prop_schema.get("oneOf", []) if candidate.get("type") != "null"]


def go_field_name(name: str) -> str:
    parts = name.split("_")
    rendered: list[str] = []
    for part in parts:
        lower = part.lower()
        if lower in GO_INITIALISMS:
            rendered.append(GO_INITIALISMS[lower])
            continue
        rendered.append(part[:1].upper() + part[1:])
    return "".join(rendered)


def kotlin_field_name(name: str) -> str:
    parts = name.split("_")
    if not parts:
        return name
    head, *tail = parts
    return head[:1].lower() + head[1:] + "".join(part[:1].upper() + part[1:] for part in tail)


def go_inner_type_for_schema(prop_schema: dict) -> str:
    if "$ref" in prop_schema:
        ref_name = schema_ref_name(prop_schema["$ref"])
        if ref_name in GO_OBJECT_TYPES:
            return GO_OBJECT_TYPES[ref_name]
        return go_inner_type_for_schema(dereference_schema(prop_schema))

    schema_type = prop_schema.get("type")
    if schema_type == "string":
        return "string"
    if schema_type == "boolean":
        return "bool"
    if schema_type == "integer":
        return "int"
    if schema_type == "number":
        return "float64"
    if schema_type == "array":
        return f'[]{go_inner_type_for_schema(prop_schema.get("items", {}))}'
    if schema_type == "object":
        return "map[string]any"
    raise ValueError(f"unsupported Go schema: {prop_schema}")


def go_type_for_schema(prop_schema: dict) -> str:
    one_of = prop_schema.get("oneOf")
    if one_of:
        non_null = non_null_one_of_candidates(prop_schema)
        if len(non_null) == 1:
            inner = go_inner_type_for_schema(non_null[0])
            return inner if inner.startswith("*") else f"*{inner}"
        raise ValueError(f"unsupported Go oneOf schema: {prop_schema}")

    inner = go_inner_type_for_schema(prop_schema)
    if prop_schema.get("x-go-pointer") and inner in {"string", "bool", "int"}:
        return f"*{inner}"
    return inner


def render_go_object(def_name: str, definition: dict) -> str:
    type_name = GO_OBJECT_TYPES[def_name]
    required = set(definition.get("required", []))
    lines = [
        f"// {type_name} 对齐 core/shared/schema.json 的 {def_name}。",
        f"type {type_name} struct {{",
    ]
    for prop_name, prop_schema in definition.get("properties", {}).items():
        tag_suffix = "" if prop_name in required else ",omitempty"
        lines.append(
            f'\t{go_field_name(prop_name)} {go_type_for_schema(prop_schema)} `json:"{prop_name}{tag_suffix}"`'
        )
    lines.append("}")
    return "\n".join(lines)


def ts_inner_type_for_schema(prop_schema: dict) -> str:
    if "$ref" in prop_schema:
        ref_name = schema_ref_name(prop_schema["$ref"])
        if ref_name in TS_OBJECT_TYPES:
            return TS_OBJECT_TYPES[ref_name]
        return ts_inner_type_for_schema(dereference_schema(prop_schema))

    schema_type = prop_schema.get("type")
    if schema_type == "string":
        if "const" in prop_schema:
            return f"'{prop_schema['const']}'"
        if "enum" in prop_schema:
            return " | ".join(f"'{value}'" for value in prop_schema["enum"])
        return "string"
    if schema_type == "boolean":
        return "boolean"
    if schema_type == "integer":
        return "number"
    if schema_type == "number":
        return "number"
    if schema_type == "array":
        return f'{ts_type_for_schema(prop_schema.get("items", {}))}[]'
    if schema_type == "object":
        return "Record<string, unknown>"
    raise ValueError(f"unsupported TS schema: {prop_schema}")


def ts_type_for_schema(prop_schema: dict) -> str:
    one_of = prop_schema.get("oneOf")
    if one_of:
        rendered: list[str] = []
        for candidate in one_of:
            if candidate.get("type") == "null":
                rendered.append("null")
                continue
            rendered.append(ts_inner_type_for_schema(candidate))
        return " | ".join(rendered)

    return ts_inner_type_for_schema(prop_schema)


def render_ts_object(def_name: str, definition: dict) -> str:
    type_name = TS_OBJECT_TYPES[def_name]
    required = set(definition.get("required", []))
    lines = [f"export interface {type_name} {{"]
    for prop_name, prop_schema in definition.get("properties", {}).items():
        optional = "" if prop_name in required else "?"
        lines.append(f"  {prop_name}{optional}: {ts_type_for_schema(prop_schema)};")
    lines.append("}")
    return "\n".join(lines)


def render_ts_union(def_name: str, definition: dict) -> str:
    refs = [TS_OBJECT_TYPES[schema_ref_name(candidate["$ref"])] for candidate in definition.get("oneOf", [])]
    return f"export type {TS_UNION_TYPES[def_name]} = {' | '.join(refs)};"


def rust_inner_type_for_schema(prop_schema: dict) -> str:
    if "$ref" in prop_schema:
        ref_name = schema_ref_name(prop_schema["$ref"])
        if ref_name in RUST_OBJECT_TYPES:
            return RUST_OBJECT_TYPES[ref_name]
        return rust_inner_type_for_schema(dereference_schema(prop_schema))

    schema_type = prop_schema.get("type")
    if schema_type == "string":
        return "String"
    if schema_type == "boolean":
        return "bool"
    if schema_type == "integer":
        return "i64"
    if schema_type == "number":
        return "f64"
    if schema_type == "array":
        return f'Vec<{rust_type_for_schema(prop_schema.get("items", {}), required=True)}>'
    if schema_type == "object":
        return "Value"
    raise ValueError(f"unsupported Rust schema: {prop_schema}")


def rust_type_for_schema(prop_schema: dict, *, required: bool) -> str:
    one_of = prop_schema.get("oneOf")
    if one_of:
        non_null = non_null_one_of_candidates(prop_schema)
        if len(non_null) == 1:
            return f'Option<{rust_inner_type_for_schema(non_null[0])}>'
        raise ValueError(f"unsupported Rust oneOf schema: {prop_schema}")

    inner = rust_inner_type_for_schema(prop_schema)
    if not required:
        return f"Option<{inner}>"
    return inner


def rust_field_name(name: str) -> str:
    if name in RUST_RESERVED_FIELDS:
        return f"r#{name}"
    return name


def render_rust_object(def_name: str, definition: dict) -> str:
    type_name = RUST_OBJECT_TYPES[def_name]
    required = set(definition.get("required", []))
    lines = ["#[derive(Debug, Deserialize, Serialize, Clone, PartialEq)]", f"pub struct {type_name} {{"]
    for prop_name, prop_schema in definition.get("properties", {}).items():
        field_name = rust_field_name(prop_name)
        if prop_name not in required:
            lines.append("    #[serde(default)]")
        if field_name != prop_name:
            lines.append(f'    #[serde(rename = "{prop_name}")]')
        lines.append(
            f"    pub {field_name}: {rust_type_for_schema(prop_schema, required=prop_name in required)},"
        )
    lines.append("}")
    return "\n".join(lines)


def render_rust_union(def_name: str, definition: dict) -> str:
    variants = RUST_UNION_VARIANTS[def_name]
    lines = ["#[derive(Debug, Deserialize, Clone, PartialEq)]", "#[serde(untagged)]", f"pub enum {RUST_UNION_TYPES[def_name]} {{"]
    for candidate in definition.get("oneOf", []):
        ref_name = schema_ref_name(candidate["$ref"])
        lines.append(f"    {variants[ref_name]}({RUST_OBJECT_TYPES[ref_name]}),")
    lines.append("}")
    return "\n".join(lines)


def kotlin_inner_type_for_schema(prop_schema: dict) -> str:
    if "$ref" in prop_schema:
        ref_name = schema_ref_name(prop_schema["$ref"])
        if ref_name in KOTLIN_OBJECT_TYPES:
            return KOTLIN_OBJECT_TYPES[ref_name]
        return kotlin_inner_type_for_schema(dereference_schema(prop_schema))

    schema_type = prop_schema.get("type")
    if schema_type == "string":
        return "String"
    if schema_type == "boolean":
        return "Boolean"
    if schema_type == "integer":
        return "Int"
    if schema_type == "number":
        return "Double"
    if schema_type == "array":
        return f'List<{kotlin_type_for_schema(prop_schema.get("items", {}), required=True)}>'
    if schema_type == "object":
        return "JsonObject"
    raise ValueError(f"unsupported Kotlin schema: {prop_schema}")


def kotlin_type_for_schema(prop_schema: dict, *, required: bool) -> str:
    one_of = prop_schema.get("oneOf")
    if one_of:
        non_null = non_null_one_of_candidates(prop_schema)
        if len(non_null) == 1:
            return f'{kotlin_inner_type_for_schema(non_null[0])}?'
        raise ValueError(f"unsupported Kotlin oneOf schema: {prop_schema}")

    inner = kotlin_inner_type_for_schema(prop_schema)
    if not required:
        return f"{inner}?"
    return inner


def render_kotlin_object(def_name: str, definition: dict) -> str:
    type_name = KOTLIN_OBJECT_TYPES[def_name]
    required = set(definition.get("required", []))
    properties = list(definition.get("properties", {}).items())
    lines = ["@Serializable", f"data class {type_name}("]
    for index, (prop_name, prop_schema) in enumerate(properties):
        field_name = kotlin_field_name(prop_name)
        if field_name != prop_name:
            lines.append(f'    @SerialName("{prop_name}")')
        field_type = kotlin_type_for_schema(prop_schema, required=prop_name in required)
        default = "" if prop_name in required else " = null"
        suffix = "," if index < len(properties) - 1 else ""
        lines.append(f"    val {field_name}: {field_type}{default}{suffix}")
    impl_suffix = f" : {KOTLIN_UNION_IMPLS[def_name]}" if def_name in KOTLIN_UNION_IMPLS else ""
    lines.append(f"){impl_suffix}")
    return "\n".join(lines)


def render_kotlin_union(def_name: str, definition: dict) -> str:
    union_name = KOTLIN_UNION_TYPES[def_name]
    if union_name == "AgentSendResponse":
        return "sealed interface AgentSendResponse"
    raise ValueError(f"unsupported Kotlin union schema: {definition}")


def render_go(schema: dict) -> str:
    actions = schema["$defs"]["action"]["enum"]
    statuses = schema["$defs"]["status"]["enum"]
    assistant_session_end_signal = schema["$defs"]["assistantSessionEndSignal"]["properties"]["signal"]["const"]
    schema_id = schema.get("$id", "")

    go_actions = "\n".join(
        f'\tbusAction{action.title().replace("_", "")} = "{action}"' for action in actions
    )
    go_statuses = "\n".join(
        f'\tbusStatus{status.title()} = "{status}"' for status in statuses
    )
    go_objects = "\n\n".join(
        render_go_object(def_name, schema["$defs"][def_name]) for def_name in SHARED_OBJECT_DEF_ORDER
    )

    return f'''// Code generated by core/shared/generate_envelope_types.py; DO NOT EDIT.
// Source: core/shared/schema.json ({schema_id})

package app

import "encoding/json"

const (
{go_actions}
)

const (
{go_statuses}
)

const busAssistantSessionEndSignal = "{assistant_session_end_signal}"

{go_objects}

// apiRequest 对齐 core/shared/schema.json 的 requestEnvelope。
type apiRequest struct {{
\tAction    string          `json:"action"`
\tParams    json.RawMessage `json:"params"`
\tTraceID   string          `json:"trace_id"`
\tRequestID string          `json:"request_id,omitempty"`
}}

// apiResponse 对齐 core/shared/schema.json 的 responseEnvelope。
type apiResponse struct {{
\tStatus    string `json:"status"`
\tPayload   any    `json:"payload"`
\tError     string `json:"error"`
\tRequestID string `json:"request_id,omitempty"`
}}
'''


def render_rust(schema: dict) -> str:
    schema_id = schema.get("$id", "")
    rust_objects = "\n\n".join(
        render_rust_object(def_name, schema["$defs"][def_name]) for def_name in SHARED_OBJECT_DEF_ORDER
    )
    rust_unions = "\n\n".join(
        render_rust_union(def_name, schema["$defs"][def_name]) for def_name in SHARED_UNION_DEF_ORDER
    )

    return f'''// Code generated by core/shared/generate_envelope_types.py; DO NOT EDIT.
// Source: core/shared/schema.json ({schema_id})

use anyhow::{{Result, anyhow}};
use serde::{{Deserialize, Serialize}};
use serde_json::Value;

#[derive(Debug, Serialize)]
pub struct ApiRequest<TParams> {{
    pub action: &'static str,
    pub params: TParams,
    pub trace_id: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub request_id: Option<String>,
}}

#[derive(Debug, Deserialize)]
pub struct ApiResponse<TPayload> {{
    pub status: String,
    pub payload: Option<TPayload>,
    #[serde(default)]
    pub error: String,
}}

impl<TPayload> ApiResponse<TPayload> {{
    pub fn into_result(self) -> Result<TPayload> {{
        if self.status == "success" {{
            return self
                .payload
                .ok_or_else(|| anyhow!("bridge response payload is missing"));
        }}
        let message = self.error.trim();
        if message.is_empty() {{
            return Err(anyhow!("bridge returned an unknown error"));
        }}
        Err(anyhow!(message.to_string()))
    }}
}}

{rust_objects}

{rust_unions}
'''


def render_ts(schema: dict) -> str:
    actions = schema["$defs"]["action"]["enum"]
    statuses = schema["$defs"]["status"]["enum"]
    schema_id = schema.get("$id", "")

    action_union = " | ".join(f"'{action}'" for action in actions)
    status_union = " | ".join(f"'{status}'" for status in statuses)
    ts_objects = "\n\n".join(
        render_ts_object(def_name, schema["$defs"][def_name]) for def_name in SHARED_OBJECT_DEF_ORDER
    )
    ts_unions = "\n\n".join(
        render_ts_union(def_name, schema["$defs"][def_name]) for def_name in SHARED_UNION_DEF_ORDER
    )

    return f'''// Code generated by core/shared/generate_envelope_types.py; DO NOT EDIT.
// Source: core/shared/schema.json ({schema_id})

export type BusAction = {action_union};
export type BusStatus = {status_union};

export interface ApiRequest<TParams extends Record<string, unknown>> {{
  action: BusAction;
  params: TParams;
  trace_id: string;
  request_id?: string;
}}

export interface ApiSuccessEnvelope<TPayload> {{
  status: 'success';
  payload: TPayload;
  error: string;
  request_id?: string;
}}

export interface ApiErrorEnvelope {{
  status: 'error';
  payload: Record<string, unknown>;
  error: string;
  request_id?: string;
}}

export type ApiEnvelope<TPayload> = ApiSuccessEnvelope<TPayload> | ApiErrorEnvelope;

{ts_objects}

{ts_unions}
'''


def render_kotlin(schema: dict) -> str:
    schema_id = schema.get("$id", "")
    kotlin_unions = "\n\n".join(
        render_kotlin_union(def_name, schema["$defs"][def_name]) for def_name in SHARED_UNION_DEF_ORDER
    )
    kotlin_objects = "\n\n".join(
        render_kotlin_object(def_name, schema["$defs"][def_name]) for def_name in SHARED_OBJECT_DEF_ORDER
    )

    return f'''// Code generated by core/shared/generate_envelope_types.py; DO NOT EDIT.
// Source: core/shared/schema.json ({schema_id})

package dev.ghostos.android.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonObject

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
    val error: String = "",
    @SerialName("request_id") val requestId: String? = null
)

{kotlin_unions}

{kotlin_objects}
'''


def write_output(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def main() -> int:
    schema = load_schema()

    write_output(GO_OUTPUT, render_go(schema))
    write_output(RUST_OUTPUT, render_rust(schema))
    write_output(TS_OUTPUT, render_ts(schema))
    write_output(KOTLIN_OUTPUT, render_kotlin(schema))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
