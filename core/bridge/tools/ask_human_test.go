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
	if _, ok := sess.PendingQuestions[questionID]; !ok {
		t.Fatalf("question %q should exist in pending_questions", questionID)
	}
}

func TestAskHumanToolExecuteStoresSelectableQuestion(t *testing.T) {
	tool := NewAskHumanTool()
	sess := session.NewSession("system")

	ctx := WithToolCallID(WithSession(context.Background(), sess), "call-ask-2")
	output, err := tool.Execute(ctx, json.RawMessage(`{
		"prompt":"Pick a deployment window",
		"selection_mode":"multiple",
		"options":[
			{"label":"Tonight"},
			{"label":"Tomorrow morning"},
			{"label":"Other","allow_custom":true}
		]
	}`), "trace-ask-2")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var payload askHumanAwaitingPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.SelectionMode != session.HumanQuestionSelectionMultiple {
		t.Fatalf("unexpected selection mode: got %q want %q", payload.SelectionMode, session.HumanQuestionSelectionMultiple)
	}
	if len(payload.Options) != 3 || !payload.Options[2].AllowCustom {
		t.Fatalf("unexpected options: %+v", payload.Options)
	}
	stored := sess.PendingQuestions[payload.QuestionID]
	if stored.SelectionMode != session.HumanQuestionSelectionMultiple {
		t.Fatalf("unexpected stored selection mode: got %q want %q", stored.SelectionMode, session.HumanQuestionSelectionMultiple)
	}
	if len(stored.Options) != 3 || !stored.Options[2].AllowCustom {
		t.Fatalf("unexpected stored options: %+v", stored.Options)
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

func TestAskHumanToolInterpretResultReturnsSelectableSignal(t *testing.T) {
	tool := NewAskHumanTool()
	interpreter, ok := tool.(ResultInterpreter)
	if !ok {
		t.Fatal("ask_human should implement ResultInterpreter")
	}

	meta := interpreter.InterpretResult(`{
		"status":"awaiting_human",
		"question_id":"q-123",
		"prompt":"Pick a plan",
		"selection_mode":"single",
		"options":[
			{"label":"Basic"},
			{"label":"Other","allow_custom":true}
		]
	}`)
	if meta.AwaitingHuman == nil {
		t.Fatal("expected awaiting human signal")
	}
	if meta.AwaitingHuman.SelectionMode != session.HumanQuestionSelectionSingle {
		t.Fatalf("unexpected selection mode: got %q want %q", meta.AwaitingHuman.SelectionMode, session.HumanQuestionSelectionSingle)
	}
	if len(meta.AwaitingHuman.Options) != 2 || !meta.AwaitingHuman.Options[1].AllowCustom {
		t.Fatalf("unexpected options: %+v", meta.AwaitingHuman.Options)
	}
}

func TestAskHumanToolExecuteRejectsInvalidSelectableQuestion(t *testing.T) {
	tool := NewAskHumanTool()
	sess := session.NewSession("system")
	ctx := WithToolCallID(WithSession(context.Background(), sess), "call-ask-invalid")

	_, err := tool.Execute(ctx, json.RawMessage(`{
		"prompt":"Pick one",
		"selection_mode":"single",
		"options":[{"label":"Basic"}]
	}`), "trace-ask-invalid")
	if err == nil {
		t.Fatal("expected validation error when custom option is missing")
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
