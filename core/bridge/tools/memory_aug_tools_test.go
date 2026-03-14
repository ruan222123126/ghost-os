package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/memorystore"
	"ghost-os/bridge/session"
)

type recallDebugStub struct {
	lastInput memoryaug.RecallInput
	items     []memoryaug.RecallItem
}

func (s *recallDebugStub) Recall(_ context.Context, input memoryaug.RecallInput) ([]memoryaug.RecallItem, error) {
	s.lastInput = input
	return s.items, nil
}

func TestMemoryLearnedListToolListsLearnedEntries(t *testing.T) {
	store, err := memorystore.NewStore(filepath.Join(t.TempDir(), "memory.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if _, err := store.CreateLearned(context.Background(), memorystore.LearnedMemoryInput{
		ScopeType:  memorystore.ScopeTypeUser,
		ScopeID:    memorystore.DefaultUserScopeID,
		MemoryType: memorystore.MemoryTypePreference,
		Summary:    "reply in Chinese",
		Content:    "The user prefers Chinese replies.",
		Confidence: 0.9,
	}, nil); err != nil {
		t.Fatalf("create learned memory: %v", err)
	}

	tool := NewMemoryLearnedListTool(store)
	output, err := tool.Execute(context.Background(), json.RawMessage(`{"scope_type":"user","status":"active"}`), "trace-learned-list")
	if err != nil {
		t.Fatalf("execute memory_learned_list: %v", err)
	}
	var result memoryLearnedListResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected learned list result: %+v", result)
	}
}

func TestMemoryRecallDebugToolUsesSessionContext(t *testing.T) {
	stub := &recallDebugStub{
		items: []memoryaug.RecallItem{{
			Entry: memorystore.MemoryEntry{
				ID:         "mem-1",
				ScopeType:  memorystore.ScopeTypeSession,
				ScopeID:    "session-1",
				SourceKind: memorystore.SourceKindLearned,
				MemoryType: memorystore.MemoryTypeWorkflow,
				Summary:    "current phase",
				Confidence: 0.91,
				Status:     memorystore.MemoryStatusActive,
			},
		}},
	}
	tool := NewMemoryRecallDebugTool(stub, memorystore.DefaultUserScopeID)
	sess := session.NewSession("system")
	sess.ID = "session-1"

	output, err := tool.Execute(
		WithSession(context.Background(), sess),
		json.RawMessage(`{"query":"current phase"}`),
		"trace-recall-debug",
	)
	if err != nil {
		t.Fatalf("execute memory_recall_debug: %v", err)
	}
	if stub.lastInput.SessionID != "session-1" {
		t.Fatalf("expected session context to be forwarded, got %+v", stub.lastInput)
	}
	var result memoryRecallDebugResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.SessionID != "session-1" || result.PromptBlock == "" {
		t.Fatalf("unexpected recall debug result: %+v", result)
	}
}
