from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from contract_codegen.common import schema_ref_name


@dataclass(frozen=True)
class DefinitionSpec:
    name: str
    definition: dict[str, Any]
    order: int
    target_name: str
    target_config: dict[str, Any]
    kind: str


def _definition_kind(definition: dict) -> str:
    if definition.get("type") == "object":
        return "object"
    if "oneOf" in definition:
        return "union"
    raise ValueError(f"unsupported generated definition: {definition}")


def _target_config(definition: dict, language: str) -> dict[str, Any] | None:
    codegen = definition.get("x-codegen", {})
    config = codegen.get(language)
    return config if isinstance(config, dict) else None


def collect_definitions(schema: dict, language: str, *, kind: str | None = None) -> list[DefinitionSpec]:
    specs: list[DefinitionSpec] = []
    for name, definition in schema["$defs"].items():
        config = _target_config(definition, language)
        if config is None:
            continue
        definition_kind = _definition_kind(definition)
        if kind and definition_kind != kind:
            continue
        order = int(definition.get("x-codegen", {}).get("order", 0))
        specs.append(
            DefinitionSpec(
                name=name,
                definition=definition,
                order=order,
                target_name=str(config["name"]),
                target_config=config,
                kind=definition_kind,
            )
        )
    return sorted(specs, key=lambda spec: (spec.order, spec.name))


def target_name_map(schema: dict, language: str) -> dict[str, str]:
    return {spec.name: spec.target_name for spec in collect_definitions(schema, language)}


def dereference_schema(schema: dict, prop_schema: dict) -> dict:
    ref_value = prop_schema.get("$ref")
    if not ref_value:
        return prop_schema
    return schema["$defs"][schema_ref_name(ref_value)]


def union_members(definition: dict) -> list[str]:
    return [schema_ref_name(candidate["$ref"]) for candidate in definition.get("oneOf", [])]


def kotlin_union_implementers(schema: dict) -> dict[str, list[str]]:
    implementers: dict[str, list[str]] = {}
    for spec in collect_definitions(schema, "kotlin", kind="union"):
        if spec.target_config.get("kind") != "sealed_interface":
            continue
        for member in union_members(spec.definition):
            implementers.setdefault(member, []).append(spec.target_name)
    return {name: sorted(names) for name, names in implementers.items()}

