from __future__ import annotations

import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
BASE_SCHEMA_PATH = ROOT / "schema" / "defs" / "base.json"

REQUIRED_RUNTIME_ACTIONS = {
    "PING",
    "AGENT_SEND",
    "AGENT_STOP",
    "HUMAN_RESPONSE",
    "CONFIG_GET",
    "CONFIG_UPDATE",
    "TASK_CREATE",
    "TASK_LIST",
    "TASK_SYSTEM_LIST",
    "TASK_GET",
    "TASK_UPDATE",
    "TASK_RUN_NOW",
    "TASK_LOGS",
    "TASK_DELETE",
    "RSS_INBOX_POLL",
    "RSS_INBOX_LIST",
    "RSS_INBOX_GET",
    "RSS_INBOX_GROUPS",
    "RSS_BRIEFING_BUILD",
    "RSS_BRIEFING_GET",
    "LIST_FILES",
    "READ_FILE",
    "SEARCH_FILES",
    "APPLY_DIFF",
    "EXPORT_FILE",
    "BASH_EXEC",
    "SCRIPT_EXEC",
    "SCREEN_CAPTURE",
    "IMAGE_CROP",
    "OCR_IMAGE",
    "TEMPLATE_MATCH_IMAGE",
    "TEXT_INPUT",
    "KEY_HOTKEY",
    "MOUSE_CLICK",
    "MOUSE_DOUBLE_CLICK",
    "MOUSE_RIGHT_CLICK",
    "MOUSE_DRAG",
    "MOUSE_SCROLL",
    "ACTIVE_WINDOW_INFO",
    "CODEX_CLI_START",
    "CODEX_CLI_STATUS",
}

REMOVED_ACTIONS = {"BROWSER_LAUNCH"}


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


if __name__ == "__main__":
    unittest.main()
