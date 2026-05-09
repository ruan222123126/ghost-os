from __future__ import annotations

import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from contract_codegen.catalog import collect_definitions, kotlin_union_implementers
from contract_codegen.schema_loader import load_schema


class SchemaLoaderTest(unittest.TestCase):
    def test_load_schema_resolves_fragmented_defs(self) -> None:
        schema = load_schema(ROOT / "schema.json")

        self.assertIn("agentResponsePayload", schema["$defs"])
        self.assertIn("agentStreamEvent", schema["$defs"])
        self.assertIn("sessionPushEvent", schema["$defs"])
        self.assertEqual(
            "./agent_human.json#/$defs/agentAwaitingHumanPayload",
            schema["$defs"]["agentSendResultPayload"]["oneOf"][1]["$ref"],
        )
        self.assertEqual(
            "./base.json#/$defs/providerType",
            schema["$defs"]["bridgeConfig"]["properties"]["provider_type"]["$ref"],
        )
        self.assertEqual(
            "AgentStreamEvent",
            schema["$defs"]["agentStreamEvent"]["x-codegen"]["kotlin"]["name"],
        )

    def test_collect_definitions_reads_codegen_metadata(self) -> None:
        schema = load_schema(ROOT / "schema.json")

        ts_objects = collect_definitions(schema, "ts", kind="object")
        ts_unions = collect_definitions(schema, "ts", kind="union")

        self.assertEqual("assistantSessionEndSignal", ts_objects[0].name)
        self.assertEqual("BridgeConfig", next(spec.target_name for spec in ts_objects if spec.name == "bridgeConfig"))
        self.assertEqual("AgentStreamEvent", next(spec.target_name for spec in ts_objects if spec.name == "agentStreamEvent"))
        self.assertEqual(["AgentSendResponse"], kotlin_union_implementers(schema)["agentResponsePayload"])
        self.assertEqual(
            ["agentSendResultPayload", "taskCreateRequest", "taskPayload"],
            [spec.name for spec in ts_unions],
        )


if __name__ == "__main__":
    unittest.main()
