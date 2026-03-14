from __future__ import annotations

import copy
import json
from functools import lru_cache
from pathlib import Path


@lru_cache(maxsize=None)
def _load_json_file(path: str) -> dict:
    with Path(path).open("r", encoding="utf-8") as handle:
        return json.load(handle)


def _split_ref(ref_value: str) -> tuple[str, str | None]:
    if "#" not in ref_value:
        return ref_value, None
    path_part, pointer = ref_value.split("#", 1)
    return path_part, pointer or None


def _unescape_pointer_token(token: str) -> str:
    return token.replace("~1", "/").replace("~0", "~")


def _resolve_pointer(document: dict, pointer: str | None) -> dict:
    if pointer in {None, ""}:
        return document
    if not pointer.startswith("/"):
        raise ValueError(f"unsupported JSON pointer: {pointer}")

    current: object = document
    for raw_token in pointer.lstrip("/").split("/"):
        token = _unescape_pointer_token(raw_token)
        if not isinstance(current, dict) or token not in current:
            raise KeyError(f"json pointer segment not found: {pointer}")
        current = current[token]

    if not isinstance(current, dict):
        raise ValueError(f"expected object at pointer {pointer}")
    return current


def _resolve_external_ref(schema_path: Path, ref_value: str) -> dict:
    path_part, pointer = _split_ref(ref_value)
    target_path = (schema_path.parent / path_part).resolve() if path_part else schema_path.resolve()
    document = _load_json_file(str(target_path))
    return _resolve_pointer(document, pointer)


def load_schema(schema_path: Path) -> dict:
    schema = copy.deepcopy(_load_json_file(str(schema_path.resolve())))
    defs = schema.get("$defs", {})
    resolved_defs: dict[str, dict] = {}
    for name, definition in defs.items():
        if isinstance(definition, dict) and set(definition) == {"$ref"}:
            resolved_defs[name] = copy.deepcopy(_resolve_external_ref(schema_path, definition["$ref"]))
            continue
        resolved_defs[name] = copy.deepcopy(definition)
    schema["$defs"] = resolved_defs
    return schema

