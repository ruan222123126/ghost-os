package session

import (
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestNewSessionIncludesSystemPrompt(t *testing.T) {
	s := NewSession("  system prompt  ")
	if strings.TrimSpace(s.ID) == "" {
		t.Fatal("session id should not be empty")
	}
	if len(s.Messages) != 1 {
		t.Fatalf("unexpected message count: got %d want %d", len(s.Messages), 1)
	}
	if s.Messages[0].Role != llm.RoleSystem {
		t.Fatalf("unexpected role: got %q want %q", s.Messages[0].Role, llm.RoleSystem)
	}
	if s.Messages[0].Text != "system prompt" {
		t.Fatalf("unexpected system prompt: got %q want %q", s.Messages[0].Text, "system prompt")
	}
	if s.TokenCount <= 0 {
		t.Fatalf("unexpected token count: got %d want > 0", s.TokenCount)
	}
	if s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() {
		t.Fatal("timestamps should not be zero")
	}
}

func TestAddMessageUpdatesTokenCountAndTimestamp(t *testing.T) {
	s := NewSession("")
	beforeTokens := s.TokenCount
	beforeUpdatedAt := s.UpdatedAt

	time.Sleep(time.Millisecond)
	s.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello session"})

	if len(s.Messages) != 1 {
		t.Fatalf("unexpected message count: got %d want %d", len(s.Messages), 1)
	}
	if s.TokenCount <= beforeTokens {
		t.Fatalf("token count should increase: before=%d after=%d", beforeTokens, s.TokenCount)
	}
	if !s.UpdatedAt.After(beforeUpdatedAt) {
		t.Fatalf("updated_at should move forward: before=%v after=%v", beforeUpdatedAt, s.UpdatedAt)
	}
}

func TestGetMessagesPrunesWhenLimitReached(t *testing.T) {
	s := NewSession("system prompt")
	for i := 0; i < 30; i++ {
		s.AddMessage(llm.Message{
			Role: llm.RoleUser,
			Text: strings.Repeat("long message ", 30),
		})
	}

	pruned := s.GetMessages(120)
	if len(pruned) >= len(s.Messages) {
		t.Fatalf("expected pruning to shrink message count: before=%d after=%d", len(s.Messages), len(pruned))
	}
	if len(pruned) == 0 || pruned[0].Role != llm.RoleSystem {
		t.Fatal("system message should be preserved after pruning")
	}
}

func TestPendingQuestionsLifecycle(t *testing.T) {
	s := NewSession("")
	s.AddPendingQuestion("q-1", PendingHumanQuestion{
		Prompt:     "Which database?",
		ToolCallID: "call-1",
		TraceID:    "trace-1",
	})

	if !s.HasPendingQuestion("q-1") {
		t.Fatal("expected q-1 to be pending")
	}
	if ok := s.SetHumanAnswer("q-1", "postgres"); !ok {
		t.Fatal("expected SetHumanAnswer to succeed")
	}

	resolved := s.PopAnsweredQuestions()
	if len(resolved) != 1 {
		t.Fatalf("unexpected resolved question count: got %d want %d", len(resolved), 1)
	}
	if resolved[0].QuestionID != "q-1" || resolved[0].Answer != "postgres" {
		t.Fatalf("unexpected resolved payload: %+v", resolved[0])
	}
	if s.HasPendingQuestion("q-1") {
		t.Fatal("pending question should be cleared after pop")
	}
}

func TestMemoryMetadataLifecycle(t *testing.T) {
	s := NewSession("")
	if !s.MemoryMetadata.IsZero() {
		t.Fatalf("memory metadata should be zero value at init: %+v", s.MemoryMetadata)
	}

	archiveTime := time.Now().UTC().Add(-time.Minute)
	s.MarkMemoryArchived(archiveTime)
	if s.MemoryMetadata.ArchivedAt.IsZero() {
		t.Fatal("archived_at should be set")
	}

	accessTime := time.Now().UTC()
	s.MarkMemoryAccess(accessTime)
	if s.MemoryMetadata.AccessCount != 1 {
		t.Fatalf("unexpected access count: got %d want %d", s.MemoryMetadata.AccessCount, 1)
	}
	if s.MemoryMetadata.LastAccessAt.IsZero() {
		t.Fatal("last_access_at should be set")
	}
}
