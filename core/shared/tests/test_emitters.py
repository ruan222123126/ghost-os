from __future__ import annotations

import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from contract_codegen.emitters.go import render as render_go
from contract_codegen.emitters.kotlin import render as render_kotlin
from contract_codegen.emitters.rust import render as render_rust
from contract_codegen.emitters.ts import render as render_ts
from contract_codegen.schema_loader import load_schema


class EmittersTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.schema = load_schema(ROOT / "schema.json")

    def test_go_renderer_uses_schema_metadata(self) -> None:
        rendered = render_go(self.schema, "orchestration")

        self.assertIn('const busAssistantSessionEndSignal = "END_SESSION"', rendered)
        self.assertIn("type askHumanOption struct {", rendered)
        self.assertLess(rendered.index("type askHumanOption struct {"), rendered.index("type agentResponse struct {"))
        self.assertIn("type agentIterationSummaryItem struct {", rendered)
        self.assertIn("type agentStreamEventContract struct {", rendered)
        self.assertIn("type assistantMessagePushPayload struct {", rendered)
        self.assertIn("IterationSummary []agentIterationSummaryItem", rendered)
        self.assertIn("Headers map[string]string", rendered)
        self.assertIn("ModelContextWindowTokens map[string]int", rendered)

    def test_ts_renderer_emits_union_and_cross_file_refs(self) -> None:
        rendered = render_ts(self.schema)

        self.assertIn("export interface SessionHumanInteraction {", rendered)
        self.assertIn("export interface AgentStreamEvent {", rendered)
        self.assertIn("options?: AskHumanOption[];", rendered)
        self.assertIn("export interface AgentIterationSummaryItem {", rendered)
        self.assertIn("iteration_summary?: AgentIterationSummaryItem[];", rendered)
        self.assertIn("headers?: Record<string, string>;", rendered)
        self.assertIn("model_context_window_tokens?: Record<string, number>;", rendered)
        self.assertIn(
            "export type AgentSendResponse = AgentSendSuccessResponse | AgentSendAwaitingHumanResponse;",
            rendered,
        )

    def test_rust_and_kotlin_render_union_variants(self) -> None:
        rust = render_rust(self.schema)
        kotlin = render_kotlin(self.schema)

        self.assertIn("pub enum AgentPayload {", rust)
        self.assertIn("pub struct AgentStreamEvent {", rust)
        self.assertIn("AwaitingHuman(AgentSendAwaitingHumanResponse)", rust)
        self.assertIn("pub struct AgentIterationSummaryItem {", rust)
        self.assertIn("pub iteration_summary: Option<Vec<AgentIterationSummaryItem>>", rust)
        self.assertIn("pub headers: Option<BTreeMap<String, String>>", rust)
        self.assertIn("pub model_context_window_tokens: Option<BTreeMap<String, i64>>", rust)
        self.assertIn("sealed interface AgentSendResponse", kotlin)
        self.assertIn("data class SessionPushEvent(", kotlin)
        self.assertIn(") : AgentSendResponse", kotlin)
        self.assertIn("data class AgentIterationSummaryItem(", kotlin)
        self.assertIn("val iterationSummary: List<AgentIterationSummaryItem>? = null", kotlin)
        self.assertIn("val headers: Map<String, String>? = null", kotlin)
        self.assertIn("val modelContextWindowTokens: Map<String, Int>? = null", kotlin)


if __name__ == "__main__":
    unittest.main()
