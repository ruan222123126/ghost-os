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

	if _, ok := s.PendingQuestions["q-1"]; !ok {
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
	if _, ok := s.PendingQuestions["q-1"]; ok {
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

func TestDynamicToolLoadLifecycle(t *testing.T) {
	s := NewSession("")
	s.AdvanceToolTurn(3)

	loaded := s.EnsureDynamicToolLoaded("web_search", "tfind")
	if loaded.AlreadyLoaded {
		t.Fatal("newly loaded tool should not report already_loaded")
	}
	if got := s.VisibleDynamicToolNames(3); len(got) != 1 || got[0] != "web_search" {
		t.Fatalf("loaded tool should be visible immediately in the current turn, got %v", got)
	}

	visible := s.VisibleDynamicToolNames(3)
	if len(visible) != 1 || visible[0] != "web_search" {
		t.Fatalf("unexpected visible tools: %v", visible)
	}
	loads := s.DynamicToolLoadsSnapshot()
	if len(loads) != 1 {
		t.Fatal("expected dynamic tool snapshot")
	}
	snapshot := loads[0]
	if !snapshot.VisibleForTurn(s.TurnIndex) {
		t.Fatal("expected tool to be visible in the current turn")
	}
	if snapshot.RemainingIdleTurns(s.TurnIndex, 3) != 3 {
		t.Fatalf("unexpected remaining idle turns: %d", snapshot.RemainingIdleTurns(s.TurnIndex, 3))
	}

	if !s.NoteDynamicToolCall("web_search") {
		t.Fatal("expected NoteDynamicToolCall to succeed")
	}
	s.AdvanceToolTurn(3)
	s.AdvanceToolTurn(3)
	s.AdvanceToolTurn(3)
	expired := s.AdvanceToolTurn(3)
	if len(expired) != 1 || expired[0] != "web_search" {
		t.Fatalf("expected web_search to expire after idle turns, got %v", expired)
	}
}
