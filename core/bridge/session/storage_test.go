package session

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ghost-os/bridge/llm"
)

func TestStoreSaveAndLoadSession(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	s := NewSession("system")
	s.ID = "session-roundtrip"
	s.TurnIndex = 3
	s.ConversationState = llm.ConversationState{
		Provider:           llm.ProviderCodex,
		BaseURL:            "https://api.openai.com/v1",
		Model:              "codex-mini-latest",
		PreviousResponseID: "resp_123",
	}
	s.DynamicToolLoads = map[string]DynamicToolLoad{
		"web_search": {
			ToolName:       "web_search",
			LoadedBy:       "tool_search",
			LoadedAtTurn:   1,
			LastCalledTurn: 2,
		},
	}
	s.DynamicSkillLoads = map[string]DynamicSkillLoad{
		"release_flow": {
			SkillName:      "release_flow",
			LoadedBy:       "sfind",
			LoadedAtTurn:   1,
			LastCalledTurn: 2,
		},
	}
	s.AssistantDraft = &AssistantDraft{
		Text:      "partial answer",
		TraceID:   "trace-draft",
		Turn:      4,
		UpdatedAt: s.UpdatedAt,
	}
	s.TurnDraft = &TurnDraft{
		TraceID: "trace-draft",
		Turn:    4,
		AssistantSegments: []TurnDraftSegment{
			{ID: "stream-segment:assistant:1", Content: "partial answer"},
		},
		ThinkingSegments: []TurnDraftSegment{
			{ID: "stream-segment:thinking:1", Content: "analyzing"},
		},
		Tools: []TurnDraftTool{
			{ID: "stream-tool:trace-draft:call-1", Content: `{"path":"README.md"}`, ToolName: "read_file"},
		},
		ItemOrder: []string{
			"thinking:stream-segment:thinking:1",
			"tool:stream-tool:trace-draft:call-1",
			"assistant:stream-segment:assistant:1",
		},
	}
	s.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello"})
	s.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "hi"})

	if err := store.Save(s); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded, err := store.Load(s.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.ID != s.ID {
		t.Fatalf("unexpected id: got %q want %q", loaded.ID, s.ID)
	}
	if !reflect.DeepEqual(loaded.Messages, s.Messages) {
		t.Fatalf("messages mismatch: got=%+v want=%+v", loaded.Messages, s.Messages)
	}
	if !reflect.DeepEqual(loaded.ConversationState, s.ConversationState) {
		t.Fatalf("conversation state mismatch: got=%+v want=%+v", loaded.ConversationState, s.ConversationState)
	}
	if loaded.TurnIndex != s.TurnIndex {
		t.Fatalf("turn index mismatch: got=%d want=%d", loaded.TurnIndex, s.TurnIndex)
	}
	if !reflect.DeepEqual(loaded.DynamicToolLoads, s.DynamicToolLoads) {
		t.Fatalf("dynamic tool loads mismatch: got=%+v want=%+v", loaded.DynamicToolLoads, s.DynamicToolLoads)
	}
	if !reflect.DeepEqual(loaded.DynamicSkillLoads, s.DynamicSkillLoads) {
		t.Fatalf("dynamic skill loads mismatch: got=%+v want=%+v", loaded.DynamicSkillLoads, s.DynamicSkillLoads)
	}
	if !reflect.DeepEqual(loaded.AssistantDraft, s.AssistantDraft) {
		t.Fatalf("assistant draft mismatch: got=%+v want=%+v", loaded.AssistantDraft, s.AssistantDraft)
	}
	if !reflect.DeepEqual(loaded.TurnDraft, s.TurnDraft) {
		t.Fatalf("turn draft mismatch: got=%+v want=%+v", loaded.TurnDraft, s.TurnDraft)
	}
	if loaded.TokenCount <= 0 {
		t.Fatalf("unexpected token count: got %d want > 0", loaded.TokenCount)
	}
}

func TestStoreLoadMissingSession(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, err = store.Load("missing-session")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestStoreLoadCorruptedSession(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	path := filepath.Join(dir, "broken-session.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0o600); err != nil {
		t.Fatalf("write corrupted session: %v", err)
	}

	_, err = store.Load("broken-session")
	if !errors.Is(err, ErrSessionCorrupted) {
		t.Fatalf("expected ErrSessionCorrupted, got: %v", err)
	}
}

func TestStoreListSessions(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	s1 := NewSession("system")
	s1.ID = "session-a"
	s2 := NewSession("system")
	s2.ID = "session-b"

	if err := store.Save(s2); err != nil {
		t.Fatalf("save s2: %v", err)
	}
	if err := store.Save(s1); err != nil {
		t.Fatalf("save s1: %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	want := []string{"session-a", "session-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected list result: got=%v want=%v", got, want)
	}
}

func TestStoreListMetadata(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	first := NewSession("system")
	first.ID = "session-a"
	first.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello"})
	if err := store.Save(first); err != nil {
		t.Fatalf("save first: %v", err)
	}

	second := NewSession("system")
	second.ID = "session-b"
	second.AddMessage(llm.Message{Role: llm.RoleUser, Text: "task"})
	second.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "done"})
	if err := store.Save(second); err != nil {
		t.Fatalf("save second: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "session-c.json"), []byte("{invalid json"), 0o600); err != nil {
		t.Fatalf("write corrupted session: %v", err)
	}

	got, err := store.ListMetadata()
	if err != nil {
		t.Fatalf("list metadata: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected metadata count: got=%d want=2", len(got))
	}

	if got[0].ID != "session-a" || got[1].ID != "session-b" {
		t.Fatalf("unexpected metadata order: %+v", got)
	}
	if got[0].MessageCount != len(first.Messages) {
		t.Fatalf("unexpected session-a message count: got=%d want=%d", got[0].MessageCount, len(first.Messages))
	}
	if got[1].MessageCount != len(second.Messages) {
		t.Fatalf("unexpected session-b message count: got=%d want=%d", got[1].MessageCount, len(second.Messages))
	}
	if got[0].TokenCount <= 0 || got[1].TokenCount <= 0 {
		t.Fatalf("token_count should be positive: got=%+v", got)
	}
	if got[0].CreatedAt.IsZero() || got[0].UpdatedAt.IsZero() || got[1].CreatedAt.IsZero() || got[1].UpdatedAt.IsZero() {
		t.Fatalf("timestamps should not be zero: got=%+v", got)
	}
}

func TestStoreRejectsInvalidSessionID(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, err = store.Load("../etc/passwd")
	if !errors.Is(err, ErrInvalidSessionID) {
		t.Fatalf("expected ErrInvalidSessionID, got: %v", err)
	}
}

func TestStoreDeleteRemovesSessionHumanLog(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("system")
	sess.ID = "session-delete-with-human-log"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "cleanup"})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	logPath := filepath.Join(baseDir, "sessions", sess.ID+".md")
	if _, statErr := os.Stat(logPath); statErr != nil {
		t.Fatalf("stat human log before delete: %v", statErr)
	}

	if err := store.Delete(sess.ID); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	if _, statErr := os.Stat(logPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected human log removed, got stat error: %v", statErr)
	}
	if _, loadErr := store.Load(sess.ID); !errors.Is(loadErr, ErrSessionNotFound) {
		t.Fatalf("expected session removed, got load error: %v", loadErr)
	}
}

func TestStoreDeleteRemovesOrphanSessionHumanLog(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sessionID := "session-orphan-human-log"
	logPath := filepath.Join(baseDir, "sessions", sessionID+".md")
	if err := os.WriteFile(logPath, []byte("orphan"), 0o600); err != nil {
		t.Fatalf("write orphan human log: %v", err)
	}

	if err := store.Delete(sessionID); err != nil {
		t.Fatalf("delete orphan human log: %v", err)
	}
	if _, statErr := os.Stat(logPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected orphan human log removed, got stat error: %v", statErr)
	}
}
