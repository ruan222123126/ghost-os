from __future__ import annotations

import importlib.util
import subprocess
import tempfile
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).with_name("remind_git_commit_stop.py")
SPEC = importlib.util.spec_from_file_location("remind_git_commit_stop", SCRIPT_PATH)
assert SPEC is not None and SPEC.loader is not None
HOOK = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(HOOK)


class CommitReminderHookTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary_directory.name)
        self.repo = self.root / "repo"
        self.state_dir = self.root / "state"
        self.repo.mkdir()
        self.git("init", "-q")
        self.git("config", "user.email", "codex@example.com")
        self.git("config", "user.name", "Codex Test")
        (self.repo / "tracked.txt").write_text("initial\n", encoding="utf-8")
        self.git("add", "tracked.txt")
        self.git("commit", "-qm", "initial")

    def tearDown(self) -> None:
        self.temporary_directory.cleanup()

    def git(self, *args: str) -> None:
        subprocess.run(["git", "-C", str(self.repo), *args], check=True)

    def event(self, name: str, *, stop_hook_active: bool = False) -> dict[str, object]:
        return {
            "cwd": str(self.repo),
            "hook_event_name": name,
            "session_id": "session-1",
            "turn_id": "turn-1",
            "stop_hook_active": stop_hook_active,
        }

    def process(self, event: dict[str, object]) -> dict[str, object]:
        return HOOK.process_event(event, self.repo, self.state_dir)

    def test_existing_dirty_file_without_turn_change_does_not_block(self) -> None:
        (self.repo / "tracked.txt").write_text("user change\n", encoding="utf-8")

        self.process(self.event("UserPromptSubmit"))

        self.assertEqual({"continue": True}, self.process(self.event("Stop")))

    def test_changing_an_existing_dirty_file_blocks(self) -> None:
        (self.repo / "tracked.txt").write_text("user change\n", encoding="utf-8")
        self.process(self.event("UserPromptSubmit"))

        (self.repo / "tracked.txt").write_text("user and codex change\n", encoding="utf-8")

        result = self.process(self.event("Stop"))
        self.assertEqual("block", result.get("decision"))

    def test_new_untracked_file_blocks(self) -> None:
        self.process(self.event("UserPromptSubmit"))

        (self.repo / "new.txt").write_text("new\n", encoding="utf-8")

        result = self.process(self.event("Stop"))
        self.assertEqual("block", result.get("decision"))

    def test_reverted_turn_change_does_not_block(self) -> None:
        self.process(self.event("UserPromptSubmit"))
        (self.repo / "tracked.txt").write_text("temporary\n", encoding="utf-8")
        (self.repo / "tracked.txt").write_text("initial\n", encoding="utf-8")

        self.assertEqual({"continue": True}, self.process(self.event("Stop")))


if __name__ == "__main__":
    unittest.main()
