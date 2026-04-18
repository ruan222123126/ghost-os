package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestSessionHumanLogFirstWriteBackfillsExistingHistory(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("system")
	sess.ID = "session-human-backfill"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "user-1"})
	sess.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "assistant-1"})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save seed session: %v", err)
	}

	logPath := filepath.Join(baseDir, "sessions", sess.ID+".md")
	if err := os.Remove(logPath); err != nil {
		t.Fatalf("remove log file: %v", err)
	}

	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	loaded.AddMessage(llm.Message{
		Role: llm.RoleTool,
		Text: mustEncodeToolEnvelope(t, sessionHumanLogToolEnvelope{
			Status:  "success",
			Tool:    "read_file",
			TraceID: "trace-1",
			Output:  "file-content",
		}),
	})
	if err := store.Save(loaded); err != nil {
		t.Fatalf("save session with rebuilt log: %v", err)
	}

	content := readSessionHumanLogText(t, logPath)
	assertContains(t, content, "## [1] `user`")
	assertContains(t, content, "user-1")
	assertContains(t, content, "## [2] `assistant`")
	assertContains(t, content, "assistant-1")
	assertContains(t, content, "## [3] `tool`")
	assertContains(t, content, "read_file")
	assertContains(t, content, "last_exported_index: `3`")
}

func TestSessionHumanLogIncrementalAppendDoesNotDuplicateBlocks(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("system")
	sess.ID = "session-human-append"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello"})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save seed session: %v", err)
	}

	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load seed session: %v", err)
	}
	loaded.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		Text: "Working...<t:1>{\"query\":\"OpenAI\"}</t>",
	})
	if err := store.Save(loaded); err != nil {
		t.Fatalf("save appended session: %v", err)
	}

	logPath := filepath.Join(store.baseDir, "sessions", sess.ID+".md")
	content := readSessionHumanLogText(t, logPath)
	if got := strings.Count(content, "## [1] `user`"); got != 1 {
		t.Fatalf("unexpected user block count: got %d want 1", got)
	}
	if got := strings.Count(content, "## [2] `assistant`"); got != 1 {
		t.Fatalf("unexpected assistant block count: got %d want 1", got)
	}
	assertContains(t, content, "tag:1")
	assertContains(t, content, "\"query\": \"OpenAI\"")
	assertContains(t, content, "last_exported_index: `2`")
}

func TestSessionHumanLogModeSwitchRewritesFullDocument(t *testing.T) {
	fullEnabled := false
	store, err := NewStore(t.TempDir(), StoreOptions{
		HumanLogFullEnabled: func() bool { return fullEnabled },
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	toolOutput := strings.Repeat("x", sessionHumanLogSummaryMaxRunes+120)
	sess := NewSession("system")
	sess.ID = "session-human-mode-switch"
	sess.AddMessage(llm.Message{
		Role: llm.RoleTool,
		Text: mustEncodeToolEnvelope(t, sessionHumanLogToolEnvelope{
			Status:  "success",
			Tool:    "script_exec",
			TraceID: "trace-tool",
			Output:  toolOutput,
		}),
	})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save summary mode session: %v", err)
	}

	logPath := filepath.Join(store.baseDir, "sessions", sess.ID+".md")
	summaryContent := readSessionHumanLogText(t, logPath)
	assertContains(t, summaryContent, "export_mode: `summary`")
	assertContains(t, summaryContent, "result_summary")
	assertContains(t, summaryContent, "truncated")

	fullEnabled = true
	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if err := store.Save(loaded); err != nil {
		t.Fatalf("save full mode session: %v", err)
	}

	fullContent := readSessionHumanLogText(t, logPath)
	assertContains(t, fullContent, "export_mode: `full`")
	assertContains(t, fullContent, "### output")
	assertContains(t, fullContent, toolOutput)
	if strings.Contains(fullContent, "result_summary") {
		t.Fatalf("expected full mode rewrite without summary section, got:\n%s", fullContent)
	}
}

func TestSessionHumanLogWriteFailureReturnsSaveError(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("system")
	sess.ID = "session-human-write-error"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "first"})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save seed session: %v", err)
	}

	logDir := filepath.Join(baseDir, "sessions")
	blockSessionHumanLogPath(t, logDir)

	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	loaded.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "second"})
	err = store.Save(loaded)
	if err == nil {
		t.Fatal("expected save error when session human log write fails")
	}
	if !strings.Contains(err.Error(), "sync session human log") {
		t.Fatalf("unexpected error: %v", err)
	}

	verified, loadErr := store.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load persisted session after write failure: %v", loadErr)
	}
	if verified.MessageCount != 3 {
		t.Fatalf("unexpected persisted message_count: got %d want 3", verified.MessageCount)
	}
}

func TestSessionHumanLogRecoversAfterPreviousWriteFailure(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("system")
	sess.ID = "session-human-recover-after-failure"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "first"})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save seed session: %v", err)
	}

	logDir := filepath.Join(baseDir, "sessions")
	blockSessionHumanLogPath(t, logDir)

	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load seed session: %v", err)
	}
	loaded.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "second"})
	if err := store.Save(loaded); err == nil {
		t.Fatal("expected save to fail when human log path is blocked")
	}

	restoreSessionHumanLogDir(t, logDir)

	reloaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session after failed save: %v", err)
	}
	if err := store.Save(reloaded); err != nil {
		t.Fatalf("save after restoring human log dir: %v", err)
	}

	logPath := filepath.Join(logDir, sess.ID+".md")
	content := readSessionHumanLogText(t, logPath)
	assertContains(t, content, "## [1] `user`")
	assertContains(t, content, "first")
	assertContains(t, content, "## [2] `assistant`")
	assertContains(t, content, "second")
	assertContains(t, content, "last_exported_index: `2`")
}

func blockSessionHumanLogPath(t *testing.T, logDir string) {
	t.Helper()
	if err := os.RemoveAll(logDir); err != nil {
		t.Fatalf("remove human log dir: %v", err)
	}
	if err := os.WriteFile(logDir, []byte("blocked"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
}

func restoreSessionHumanLogDir(t *testing.T, logDir string) {
	t.Helper()
	if err := os.Remove(logDir); err != nil {
		t.Fatalf("remove blocking file: %v", err)
	}
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatalf("recreate log dir: %v", err)
	}
}

func mustEncodeToolEnvelope(t *testing.T, envelope sessionHumanLogToolEnvelope) string {
	t.Helper()
	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("encode tool envelope: %v", err)
	}
	return string(encoded)
}

func readSessionHumanLogText(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read session human log file: %v", err)
	}
	return string(raw)
}

func assertContains(t *testing.T, content string, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("expected content to contain %q, got:\n%s", want, content)
	}
}
