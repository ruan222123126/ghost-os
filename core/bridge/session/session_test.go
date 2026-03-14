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

func TestSessionMarkEnded(t *testing.T) {
	s := NewSession("")
	if s.IsEnded() {
		t.Fatal("new session should not be ended")
	}

	endAt := time.Now().UTC()
	s.MarkEnded(endAt)
	if !s.IsEnded() {
		t.Fatal("session should be ended after MarkEnded")
	}
	if s.EndedAt.IsZero() {
		t.Fatal("ended_at should be set")
	}
}

func TestIterationRuntimeLifecycle(t *testing.T) {
	s := NewSession("")
	s.StartIterationRuntime("pro", "fix config", 2, false)
	if s.IterationRuntime == nil {
		t.Fatal("expected iteration runtime")
	}
	if s.IterationRuntime.Mode != "pro" || s.IterationRuntime.OriginalTask != "fix config" {
		t.Fatalf("unexpected iteration runtime: %+v", s.IterationRuntime)
	}

	s.AppendIterationRecord(IterationRecord{
		Iteration: 1,
		Did:       "inspected config",
		Remaining: "apply patch",
	})
	if len(s.IterationRuntime.Records) != 1 {
		t.Fatalf("unexpected record count: %d", len(s.IterationRuntime.Records))
	}

	s.FinishIterationRuntime("completed", "pro_complete", "done", "updated config")
	if s.IterationRuntime.Status != "completed" {
		t.Fatalf("unexpected status: %q", s.IterationRuntime.Status)
	}
	if s.IterationRuntime.StoppedBy != "pro_complete" {
		t.Fatalf("unexpected stopped_by: %q", s.IterationRuntime.StoppedBy)
	}
	if s.IterationRuntime.FinalChangeLog != "updated config" {
		t.Fatalf("unexpected final change log: %q", s.IterationRuntime.FinalChangeLog)
	}
}
