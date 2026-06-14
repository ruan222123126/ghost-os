from __future__ import annotations

from contract_codegen.catalog import collect_definitions, dereference_schema
from contract_codegen.common import (
    non_null_one_of_candidates,
    object_additional_properties_schema,
    object_has_declared_properties,
    object_is_open,
    schema_ref_name,
)

GO_INITIALISMS = {
    "api": "API",
    "id": "ID",
    "url": "URL",
    "sha256": "SHA256",
}


def go_field_name(name: str) -> str:
    rendered: list[str] = []
    for part in name.split("_"):
        lower = part.lower()
        if lower in GO_INITIALISMS:
            rendered.append(GO_INITIALISMS[lower])
            continue
        rendered.append(part[:1].upper() + part[1:])
    return "".join(rendered)


def _object_type(schema: dict, target_names: dict[str, str], prop_schema: dict) -> str:
    if object_has_declared_properties(prop_schema):
        raise ValueError(f"inline structured Go object must be promoted to $defs: {prop_schema}")

    additional = object_additional_properties_schema(prop_schema)
    if additional is not None:
        return f'map[string]{_type_for_schema(schema, target_names, additional)}'
    if object_is_open(prop_schema):
        return "map[string]any"
    raise ValueError(f"unsupported closed Go object schema: {prop_schema}")


def _inner_type(schema: dict, target_names: dict[str, str], prop_schema: dict) -> str:
    if not prop_schema:
        return "any"

    ref_value = prop_schema.get("$ref")
    if ref_value:
        ref_name = schema_ref_name(ref_value)
        if ref_name in target_names:
            return target_names[ref_name]
        return _inner_type(schema, target_names, dereference_schema(schema, prop_schema))

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
        item_schema = prop_schema.get("items", {})
        return f'[]{_inner_type(schema, target_names, item_schema)}'
    if schema_type == "object":
        return _object_type(schema, target_names, prop_schema)
    raise ValueError(f"unsupported Go schema: {prop_schema}")


def _type_for_schema(schema: dict, target_names: dict[str, str], prop_schema: dict) -> str:
    if prop_schema.get("oneOf"):
        non_null = non_null_one_of_candidates(prop_schema)
        if len(non_null) != 1:
            raise ValueError(f"unsupported Go oneOf schema: {prop_schema}")
        inner = _inner_type(schema, target_names, non_null[0])
        return inner if inner.startswith("*") else f"*{inner}"

    inner = _inner_type(schema, target_names, prop_schema)
    if prop_schema.get("x-go-pointer") and inner in {"string", "bool", "int"}:
        return f"*{inner}"
    return inner


def _render_object(schema: dict, spec, target_names: dict[str, str]) -> str:
    required = set(spec.definition.get("required", []))
    lines = [
        f"// {spec.target_name} 对齐 core/shared/schema.json 的 {spec.name}。",
        f"type {spec.target_name} struct {{",
    ]
    for prop_name, prop_schema in spec.definition.get("properties", {}).items():
        tag_suffix = "" if prop_name in required else ",omitempty"
        field_type = _type_for_schema(schema, target_names, prop_schema)
        lines.append(f'\t{go_field_name(prop_name)} {field_type} `json:"{prop_name}{tag_suffix}"`')
    lines.append("}")
    return "\n".join(lines)


def _render_actions(actions: list[str]) -> str:
    return "\n".join(f'\tbusAction{action.title().replace("_", "")} = "{action}"' for action in actions)


def _render_statuses(statuses: list[str]) -> str:
    return "\n".join(f'\tbusStatus{status.title()} = "{status}"' for status in statuses)


def _exported_type_name(name: str) -> str:
    return name[:1].upper() + name[1:]


def _render_action_aliases(actions: list[str]) -> str:
    lines = []
    for action in actions:
        suffix = action.title().replace("_", "")
        lines.append(f"\tbusAction{suffix} = bus.Action{suffix}")
    return "\n".join(lines)


def _render_status_aliases(statuses: list[str]) -> str:
    lines = []
    for status in statuses:
        suffix = status.title()
        lines.append(f"\tbusStatus{suffix} = bus.Status{suffix}")
    return "\n".join(lines)


def _render_contract_aliases(schema: dict) -> str:
    return "\n".join(
        f"type {spec.target_name} = api.{_exported_type_name(spec.target_name)}"
        for spec in collect_definitions(schema, "go", kind="object")
    )


BRIDGE_EXPORT_BLOCK = '''const defaultMaxRequestBodyBytes int64 = 1 << 20

type agentParams = api.AgentParams
type sessionIDParams = api.SessionIDParams
type sessionGetParams = api.SessionGetParams
type sessionDeleteResponse = api.SessionDeleteResponse
type taskCreateParams = api.TaskCreateParams
type taskUpdateParams = api.TaskUpdateParams
type taskListParams = api.TaskListParams
type taskIDParams = api.TaskIDParams
type taskRunNowParams = api.TaskRunNowParams
type taskStopParams = api.TaskStopParams
type taskLogsParams = api.TaskLogsParams
type taskPayload = api.TaskPayload
type taskDeleteResponse = api.TaskDeleteResponse
type taskRunPayload = api.TaskRunPayload
type taskStopResponse = api.TaskStopResponse
type taskRunLogPayload = api.TaskRunLogPayload
type toolNameParams = api.ToolNameParams
type toolUpdateRequest = api.ToolUpdateRequest
type toolPayload = api.ToolPayload
type findIconTemplateUploadRequest = api.FindIconTemplateUploadRequest
type findIconTemplateUploadPayload = api.FindIconTemplateUploadPayload
type findIconPreviewRegion = api.FindIconPreviewRegion
type findIconPreviewRequest = api.FindIconPreviewRequest
type findIconPreviewPayload = api.FindIconPreviewPayload
type mousePositionRequest = api.MousePositionRequest
type mousePositionPayload = api.MousePositionPayload

type providerCreateRequest = providerConfigInput

type providerUpdateRequest = providerConfigInput

const (
\tBusActionAgentSend           = busActionAgentSend
\tBusStatusSuccess             = busStatusSuccess
\tBusStatusError               = busStatusError
\tBusAssistantSessionEndSignal = busAssistantSessionEndSignal

\tDefaultMaxRequestBodyBytes = defaultMaxRequestBodyBytes
\tTaskListScopeUser          = taskListScopeUser
\tTaskListScopeSystem        = taskListScopeSystem
\tTaskListScopeOrchestration = taskListScopeOrchestration

\tSessionPushAssistantMessage = sessionPushAssistantMessage
\tSessionPushAwaitingHuman    = sessionPushAwaitingHuman
\tSessionPushRunStarted       = sessionPushRunStarted
\tSessionPushCompletionDelta  = sessionPushCompletionDelta
\tSessionPushToolCallStarted  = sessionPushToolCallStarted
\tSessionPushToolCallFinished = sessionPushToolCallFinished
\tSessionPushError            = sessionPushError
\tSessionPushDone             = sessionPushDone
)

type APIRequest = apiRequest
type APIResponse = apiResponse
type AgentRequest = agentRequest
type AgentParams = agentParams
type HumanResponseParams = humanResponseParams
type SessionIDParams = sessionIDParams
type SessionGetParams = sessionGetParams
type SessionSidebarPartition = sessionSidebarPartition
type SessionSidebarPartitionState = sessionSidebarPartitionState
type SessionSidebarPartitionPutRequest = sessionSidebarPartitionPutRequest
type SessionSourceAssignment = sessionSourceAssignment
type SessionSourceResolution = sessionSourceResolution
type SessionDeleteResponse = sessionDeleteResponse
type ConfigResponse = configResponse
type ConfigUpdateRequest = configUpdateRequest
type ProviderCreateRequest = providerCreateRequest
type ProviderUpdateRequest = providerUpdateRequest
type ProviderConfigResponse = providerConfigResponse
type ProviderListResponse = providerListResponse
type SetActiveProviderRequest = setActiveProviderRequest
type TaskCreateParams = taskCreateParams
type TaskUpdateParams = taskUpdateParams
type TaskListParams = taskListParams
type TaskIDParams = taskIDParams
type TaskRunNowParams = taskRunNowParams
type TaskStopParams = taskStopParams
type TaskLogsParams = taskLogsParams
type TaskPayload = taskPayload
type TaskDeleteResponse = taskDeleteResponse
type TaskRunPayload = taskRunPayload
type TaskStopResponse = taskStopResponse
type TaskRunLogPayload = taskRunLogPayload
type ToolNameParams = toolNameParams
type ToolUpdateRequest = toolUpdateRequest
type ToolPayload = toolPayload
type FindIconTemplateUploadRequest = findIconTemplateUploadRequest
type FindIconTemplateUploadPayload = findIconTemplateUploadPayload
type FindIconPreviewRequest = findIconPreviewRequest
type FindIconPreviewPayload = findIconPreviewPayload
type MousePositionRequest = mousePositionRequest
type MousePositionPayload = mousePositionPayload
type SessionPushEventType = sessionPushEventType
type SessionPushEvent = sessionPushEvent
type SessionPushHub = sessionPushHub

func ValidateBusRequest(req APIRequest) error {
\treturn validateBusRequest(req)
}

func NewSessionPushHub() *SessionPushHub {
\treturn newSessionPushHub()
}

func ResolveFindIconTemplateRootPath() (string, error) {
\treturn resolveFindIconTemplateRoot()
}'''


def _render_bridge_export_block() -> str:
    return BRIDGE_EXPORT_BLOCK


def render(schema: dict, package_name: str) -> str:
    actions = schema["$defs"]["action"]["enum"]
    statuses = schema["$defs"]["status"]["enum"]

    return f'''// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json
// Source: core/shared/schema.json ({schema.get("$id", "")})

package {package_name}

import (
\t"ghost-os/bridge/orchestration/internal/contracts/api"
\t"ghost-os/bridge/orchestration/internal/contracts/bus"
)

const (
{_render_action_aliases(actions)}
)

const (
{_render_status_aliases(statuses)}
)

const busAssistantSessionEndSignal = bus.AssistantSessionEndSignal

{_render_contract_aliases(schema)}

type apiRequest = bus.RequestEnvelope
type apiResponse = bus.ResponseEnvelope

{_render_bridge_export_block()}
'''
