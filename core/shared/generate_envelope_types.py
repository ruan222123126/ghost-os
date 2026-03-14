from __future__ import annotations

from pathlib import Path

from contract_codegen.emitters.go import render as render_go
from contract_codegen.emitters.kotlin import render as render_kotlin
from contract_codegen.emitters.rust import render as render_rust
from contract_codegen.emitters.ts import render as render_ts
from contract_codegen.schema_loader import load_schema

ROOT = Path(__file__).resolve().parents[2]
SCHEMA_PATH = ROOT / "core" / "shared" / "schema.json"

GO_OUTPUT = ROOT / "core" / "bridge" / "orchestration" / "envelope_generated.go"
GO_PACKAGE = GO_OUTPUT.parent.name
RUST_OUTPUT = ROOT / "apps" / "cli" / "src" / "envelope_generated.rs"
TS_OUTPUT = ROOT / "apps" / "web" / "lib" / "envelope.generated.ts"
KOTLIN_OUTPUT = (
    ROOT
    / "apps"
    / "android"
    / "app"
    / "src"
    / "main"
    / "java"
    / "dev"
    / "ghostos"
    / "android"
    / "model"
    / "ApiModels.kt"
)


def _write_file(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def generate() -> None:
    schema = load_schema(SCHEMA_PATH)
    outputs = [
        (GO_OUTPUT, render_go(schema, GO_PACKAGE)),
        (RUST_OUTPUT, render_rust(schema)),
        (TS_OUTPUT, render_ts(schema)),
        (KOTLIN_OUTPUT, render_kotlin(schema)),
    ]
    for path, content in outputs:
        _write_file(path, content)


def main() -> int:
    generate()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
