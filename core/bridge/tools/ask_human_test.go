package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"ghost-os/bridge/session"
)

func TestAskHumanToolExecuteStoresPendingQuestion(t *testing.T) {
	tool := NewAskHumanTool()
	sess := session.NewSession("system")

	ctx := WithToolCallID(WithSession(context.Background(), sess), "call-ask-1")
	output, err := tool.Execute(ctx, json.RawMessage(`{"prompt":"Which database should we use?"}`), "trace-ask-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["status"] != "awaiting_human" {
		t.Fatalf("unexpected status: got %v", payload["status"])
	}
	questionID, _ := payload["question_id"].(string)
	if strings.TrimSpace(questionID) == "" {
		t.Fatal("question_id should not be empty")
	}
	if !sess.HasPendingQuestion(questionID) {
		t.Fatalf("question %q should exist in pending_questions", questionID)
	}
}

func TestAskHumanToolExecuteRequiresSession(t *testing.T) {
	tool := NewAskHumanTool()
	ctx := WithToolCallID(context.Background(), "call-ask-1")

	_, err := tool.Execute(ctx, json.RawMessage(`{"prompt":"confirm?"}`), "trace-ask-1")
	if err == nil {
		t.Fatal("expected error when session is missing")
	}
}

func TestAskHumanToolInterpretResultReturnsAwaitingSignal(t *testing.T) {
	tool := NewAskHumanTool()
	interpreter, ok := tool.(ResultInterpreter)
	if !ok {
		t.Fatal("ask_human should implement ResultInterpreter")
	}

	meta := interpreter.InterpretResult(`{"status":"awaiting_human","question_id":"q-123","prompt":"Approve deploy?"}`)
	if meta.AwaitingHuman == nil {
		t.Fatal("expected awaiting human signal")
	}
	if meta.AwaitingHuman.QuestionID != "q-123" {
		t.Fatalf("unexpected question id: got %q want %q", meta.AwaitingHuman.QuestionID, "q-123")
	}
	if meta.AwaitingHuman.Prompt != "Approve deploy?" {
		t.Fatalf("unexpected prompt: got %q want %q", meta.AwaitingHuman.Prompt, "Approve deploy?")
	}
}

func TestAskHumanToolInterpretResultIgnoresInvalidPayload(t *testing.T) {
	tool := NewAskHumanTool()
	interpreter, ok := tool.(ResultInterpreter)
	if !ok {
		t.Fatal("ask_human should implement ResultInterpreter")
	}

	meta := interpreter.InterpretResult(`{"status":"ok"}`)
	if meta.AwaitingHuman != nil {
		t.Fatalf("expected no awaiting signal, got %+v", meta.AwaitingHuman)
	}
}
