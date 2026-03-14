from __future__ import annotations


def schema_ref_name(ref_value: str) -> str:
    return ref_value.rsplit("/", 1)[-1]


def non_null_one_of_candidates(prop_schema: dict) -> list[dict]:
    return [candidate for candidate in prop_schema.get("oneOf", []) if candidate.get("type") != "null"]

