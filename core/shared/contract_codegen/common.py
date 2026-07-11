from __future__ import annotations

from typing import Any


def schema_ref_name(ref_value: str) -> str:
    return ref_value.rsplit("/", 1)[-1]


def non_null_one_of_candidates(prop_schema: dict) -> list[dict]:
    return [candidate for candidate in prop_schema.get("oneOf", []) if candidate.get("type") != "null"]


def object_additional_properties_schema(prop_schema: dict[str, Any]) -> dict[str, Any] | None:
    additional = prop_schema.get("additionalProperties")
    return additional if isinstance(additional, dict) else None


def object_has_declared_properties(prop_schema: dict[str, Any]) -> bool:
    properties = prop_schema.get("properties")
    return isinstance(properties, dict) and len(properties) > 0


def object_is_open(prop_schema: dict[str, Any]) -> bool:
    additional = prop_schema.get("additionalProperties")
    return additional is None or additional is True
