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
