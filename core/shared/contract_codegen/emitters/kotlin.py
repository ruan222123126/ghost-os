from __future__ import annotations

from dataclasses import dataclass

from contract_codegen.catalog import (
    DefinitionSpec,
    collect_definitions,
    dereference_schema,
    kotlin_union_implementers,
    target_name_map,
)
from contract_codegen.common import (
    non_null_one_of_candidates,
    object_additional_properties_schema,
    object_has_declared_properties,
    object_is_open,
    schema_ref_name,
)

GENERATED_HEADER = "// CODE GENERATED. DO NOT EDIT. Source: core/shared/schema.json"
KOTLIN_PACKAGE = "dev.ghostos.android.model"


@dataclass(frozen=True)
class KotlinOutputGroup:
    filename: str
    definitions: tuple[str, ...]
    include_api_types: bool = False


OUTPUT_GROUPS = (
    KotlinOutputGroup("ApiModels.kt", (), include_api_types=True),
    KotlinOutputGroup(
        "AgentModels.kt",
        (
            "assistantSessionEndSignal",
            "agentRequest",
            "agentIterationSummaryItem",
            "askHumanOption",
            "agentResponsePayload",
            "agentAwaitingHumanPayload",
            "agentStopResponsePayload",
            "humanResponseRequest",
            "humanResponseAck",
            "agentSendResultPayload",
        ),
    ),
    KotlinOutputGroup(
        "SessionModels.kt",
        (
            "sessionImageContent",
            "sessionFileContent",
            "sessionContentPart",
            "sessionToolCall",
            "sessionToolResult",
            "sessionHumanInteraction",
            "sessionMessage",
            "sessionMetadata",
            "sessionSidebarPartition",
            "sessionSidebarPartitionState",
            "sessionSidebarPartitionPutRequest",
            "sessionMessagePage",
            "sessionTurnDraftSegment",
            "sessionTurnDraftTool",
            "sessionTurnDraft",
            "sessionDetail",
        ),
    ),
    KotlinOutputGroup(
        "StreamingModels.kt",
        (
            "agentStreamEvent",
            "sessionPushEvent",
            "agentRunStartedPayload",
            "agentCompletionDeltaPayload",
            "agentToolCallStartedPayload",
            "agentToolCallFinishedPayload",
            "agentStreamMessagePayload",
            "agentDonePayload",
            "agentErrorPayload",
            "sessionPushAssistantMessagePayload",
            "sessionPushAwaitingHumanPayload",
        ),
    ),
    KotlinOutputGroup(
        "ConfigModels.kt",
        (
            "bridgeConfig",
            "configUpdate",
            "providerConfig",
            "providerConfigInput",
            "providerListResponse",
            "setActiveProviderRequest",
        ),
    ),
    KotlinOutputGroup(
        "TaskModels.kt",
        (
            "taskRuntimeOverrides",
            "taskRelayConfig",
            "agentMessageTaskCreateRequest",
            "taskCreateRequest",
            "agentMessageTaskPayload",
            "taskUpdateRequest",
            "taskPayload",
        ),
    ),
    KotlinOutputGroup(
        "WorkflowModels.kt",
        (
            "workflowNode",
            "workflowEdge",
            "workflowDefinition",
            "workflowToolNode",
            "workflowLLMNode",
            "workflowAgentNode",
            "workflowIfNode",
            "workflowTaskCreateRequest",
            "workflowLoopNode",
            "workflowTaskPayload",
            "workflowStartNode",
            "workflowInputVariable",
        ),
    ),
    KotlinOutputGroup(
        "OrchestrationModels.kt",
        (
            "orchestrationNode",
            "orchestrationGroupNode",
            "orchestrationAgentNode",
            "orchestrationEdge",
            "orchestrationDefinition",
            "orchestrationTaskCreateRequest",
            "orchestrationTaskPayload",
        ),
    ),
)


def _field_name(name: str) -> str:
    head, *tail = name.split("_")
    return head[:1].lower() + head[1:] + "".join(part[:1].upper() + part[1:] for part in tail)


def _object_type(schema: dict, target_names: dict[str, str], prop_schema: dict) -> str:
    if object_has_declared_properties(prop_schema):
        raise ValueError(f"inline structured Kotlin object must be promoted to $defs: {prop_schema}")

    additional = object_additional_properties_schema(prop_schema)
    if additional is not None:
        return f"Map<String, {_type_for_schema(schema, target_names, additional, required=True)}>"
    if object_is_open(prop_schema):
        return "JsonObject"
    raise ValueError(f"unsupported closed Kotlin object schema: {prop_schema}")


def _inner_type(schema: dict, target_names: dict[str, str], prop_schema: dict) -> str:
    if not prop_schema:
        return "JsonElement"

    ref_value = prop_schema.get("$ref")
    if ref_value:
        ref_name = schema_ref_name(ref_value)
        if ref_name in target_names:
            return target_names[ref_name]
        return _inner_type(schema, target_names, dereference_schema(schema, prop_schema))

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
        inner = _type_for_schema(schema, target_names, prop_schema.get("items", {}), required=True)
        return f"List<{inner}>"
    if schema_type == "object":
        return _object_type(schema, target_names, prop_schema)
    raise ValueError(f"unsupported Kotlin schema: {prop_schema}")


def _type_for_schema(schema: dict, target_names: dict[str, str], prop_schema: dict, *, required: bool) -> str:
    if prop_schema.get("oneOf"):
        non_null = non_null_one_of_candidates(prop_schema)
        if len(non_null) != 1:
            raise ValueError(f"unsupported Kotlin oneOf schema: {prop_schema}")
        return f'{_inner_type(schema, target_names, non_null[0])}?'

    inner = _inner_type(schema, target_names, prop_schema)
    return inner if required else f"{inner}?"


def _render_object(schema: dict, spec, target_names: dict[str, str], implementers: dict[str, list[str]]) -> str:
    required = set(spec.definition.get("required", []))
    properties = list(spec.definition.get("properties", {}).items())
    lines = ["@Serializable", f"data class {spec.target_name}("]
    for index, (prop_name, prop_schema) in enumerate(properties):
        field_name = _field_name(prop_name)
        if field_name != prop_name:
            lines.append(f'    @SerialName("{prop_name}")')
        field_type = _type_for_schema(schema, target_names, prop_schema, required=prop_name in required)
        default = "" if prop_name in required else " = null"
        suffix = "," if index < len(properties) - 1 else ""
        lines.append(f"    val {field_name}: {field_type}{default}{suffix}")
    union_names = implementers.get(spec.name, [])
    impl_suffix = f" : {', '.join(union_names)}" if union_names else ""
    lines.append(f"){impl_suffix}")
    return "\n".join(lines)


def _render_union(spec) -> str:
    if spec.target_config.get("kind") != "sealed_interface":
        raise ValueError(f"unsupported Kotlin union config for {spec.name}")
    return f"@Serializable\nsealed interface {spec.target_name}"


def _render_api_types() -> str:
    return """@Serializable
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
)"""


def _imports_for_body(body: str) -> str:
    imports = []
    if "@SerialName(" in body:
        imports.append("import kotlinx.serialization.SerialName")
    if "@Serializable" in body:
        imports.append("import kotlinx.serialization.Serializable")
    if "JsonElement" in body:
        imports.append("import kotlinx.serialization.json.JsonElement")
    if "JsonObject" in body:
        imports.append("import kotlinx.serialization.json.JsonObject")
    return "\n".join(imports)


def _render_group_header(schema: dict, body: str) -> str:
    imports = _imports_for_body(body)
    return f"""{GENERATED_HEADER}
// Source: core/shared/schema.json ({schema.get("$id", "")})

package {KOTLIN_PACKAGE}

{imports}"""


def _validate_groups(spec_by_name: dict[str, DefinitionSpec]) -> None:
    assigned = {name for group in OUTPUT_GROUPS for name in group.definitions}
    missing = sorted(set(spec_by_name) - assigned)
    unknown = sorted(assigned - set(spec_by_name))
    if missing:
        raise ValueError(f"unassigned Kotlin contract definitions: {', '.join(missing)}")
    if unknown:
        raise ValueError(f"unknown Kotlin contract definitions: {', '.join(unknown)}")


def _render_group_body(schema: dict, group: KotlinOutputGroup, spec_by_name: dict[str, DefinitionSpec]) -> str:
    target_names = target_name_map(schema, "kotlin")
    implementers = kotlin_union_implementers(schema)
    specs = [spec_by_name[name] for name in group.definitions]
    blocks = [_render_api_types()] if group.include_api_types else []
    blocks.extend(_render_union(spec) for spec in specs if spec.kind == "union")
    blocks.extend(_render_object(schema, spec, target_names, implementers) for spec in specs if spec.kind == "object")
    return "\n\n".join(blocks)


def render_files(schema: dict) -> dict[str, str]:
    spec_by_name = {spec.name: spec for spec in collect_definitions(schema, "kotlin")}
    _validate_groups(spec_by_name)
    files = {}
    for group in OUTPUT_GROUPS:
        body = _render_group_body(schema, group, spec_by_name)
        files[group.filename] = f"{_render_group_header(schema, body)}\n\n{body}\n"
    return files


def render(schema: dict) -> str:
    return "\n\n".join(render_files(schema).values())
