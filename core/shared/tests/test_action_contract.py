from __future__ import annotations

import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
BASE_SCHEMA_PATH = ROOT / "schema" / "defs" / "base.json"
TASKS_SCHEMA_PATH = ROOT / "schema" / "defs" / "tasks.json"

REQUIRED_RUNTIME_ACTIONS = {
    "AGENT_SEND",
    "AGENT_STOP",
    "HUMAN_RESPONSE",
    "CONFIG_GET",
    "CONFIG_UPDATE",
    "TASK_CREATE",
    "TASK_LIST",
    "TASK_GET",
    "TASK_UPDATE",
    "TASK_RUN_NOW",
    "TASK_STOP",
    "TASK_LOGS",
    "TASK_DELETE",
}

REMOVED_ACTIONS = {"BROWSER_LAUNCH"}
TASK_SCOPES = ["user", "system", "orchestration"]


class ActionContractTest(unittest.TestCase):
    def test_action_enum_covers_runtime_actions(self) -> None:
        schema = json.loads(BASE_SCHEMA_PATH.read_text(encoding="utf-8"))
        actions = schema["$defs"]["action"]["enum"]
        action_set = set(actions)

        missing = sorted(REQUIRED_RUNTIME_ACTIONS - action_set)
        self.assertEqual([], missing, f"missing runtime actions in schema enum: {missing}")

    def test_action_enum_keeps_entries_unique(self) -> None:
        schema = json.loads(BASE_SCHEMA_PATH.read_text(encoding="utf-8"))
        actions = schema["$defs"]["action"]["enum"]
        self.assertEqual(len(actions), len(set(actions)), "action enum contains duplicated entries")

    def test_removed_actions_are_not_present(self) -> None:
        schema = json.loads(BASE_SCHEMA_PATH.read_text(encoding="utf-8"))
        actions = set(schema["$defs"]["action"]["enum"])
        stale = sorted(REMOVED_ACTIONS & actions)
        self.assertEqual([], stale, f"stale actions must stay removed: {stale}")

    def test_task_run_now_has_dedicated_params(self) -> None:
        schema = json.loads(TASKS_SCHEMA_PATH.read_text(encoding="utf-8"))
        task_id_props = schema["$defs"]["taskIDRequest"]["properties"]
        run_now = schema["$defs"]["taskRunNowRequest"]

        self.assertNotIn("start_only", task_id_props)
        self.assertEqual(["id"], run_now["required"])
        self.assertEqual("boolean", run_now["properties"]["start_only"]["type"])
        self.assertEqual(TASK_SCOPES, run_now["properties"]["scope"]["enum"])

    def test_task_list_params_define_scope(self) -> None:
        schema = json.loads(TASKS_SCHEMA_PATH.read_text(encoding="utf-8"))
        task_list = schema["$defs"]["taskListRequest"]

        self.assertEqual([], task_list.get("required", []))
        self.assertEqual(TASK_SCOPES, task_list["properties"]["scope"]["enum"])

    def test_task_create_and_update_params_define_scope(self) -> None:
        schema = json.loads(TASKS_SCHEMA_PATH.read_text(encoding="utf-8"))
        create_defs = [
            "agentMessageTaskCreateRequest",
            "workflowTaskCreateRequest",
            "orchestrationTaskCreateRequest",
        ]

        for def_name in create_defs:
            with self.subTest(def_name=def_name):
                props = schema["$defs"][def_name]["properties"]
                self.assertEqual(TASK_SCOPES, props["scope"]["enum"])

        update_props = schema["$defs"]["taskUpdateRequest"]["properties"]
        self.assertEqual(TASK_SCOPES, update_props["scope"]["enum"])
        self.assertNotIn("scope", schema["$defs"]["taskPatchRequest"]["properties"])

    def test_task_logs_params_match_runtime_contract(self) -> None:
        schema = json.loads(TASKS_SCHEMA_PATH.read_text(encoding="utf-8"))
        task_logs = schema["$defs"]["taskLogsRequest"]

        self.assertEqual(["id"], task_logs["required"])
        self.assertEqual(0, task_logs["properties"]["limit"]["minimum"])
        self.assertEqual(TASK_SCOPES, task_logs["properties"]["scope"]["enum"])

    def test_task_stop_params_match_runtime_contract(self) -> None:
        schema = json.loads(TASKS_SCHEMA_PATH.read_text(encoding="utf-8"))
        task_stop = schema["$defs"]["taskStopRequest"]

        self.assertEqual(["id", "run_id"], task_stop["required"])
        self.assertEqual(TASK_SCOPES, task_stop["properties"]["scope"]["enum"])


if __name__ == "__main__":
    unittest.main()
