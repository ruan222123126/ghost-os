package sessiondraft

import (
	"testing"
	"time"

	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func TestProjectTurnDraftThinkingToolAssistantOrder(t *testing.T) {
	sess := &session.Session{}
	when := time.Now().UTC()

	ProjectTurnDraft(sess, draftEvent(streaming.EventRunStarted, "trace-1", 2, nil), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventCompletionDelta, "trace-1", 2, map[string]any{
		"kind":     "thinking",
		"thinking": "before tool",
	}), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventToolCallStarted, "trace-1", 2, map[string]any{
		"tool":           "bash_exec",
		"tool_call_id":   "call-1",
		"arguments_json": `{"command":"pwd"}`,
	}), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventCompletionDelta, "trace-1", 2, map[string]any{
		"kind":     "thinking",
		"thinking": "after tool",
	}), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventCompletionDelta, "trace-1", 2, map[string]any{
		"kind": "text",
		"text": "done",
	}), when)

	if sess.TurnDraft == nil {
		t.Fatal("expected turn draft")
	}
	if sess.TurnDraft.Status != session.TurnDraftStatusStreaming {
		t.Fatalf("unexpected draft status: %q", sess.TurnDraft.Status)
	}
	if got := sess.TurnDraft.ItemOrder; len(got) != 4 {
		t.Fatalf("unexpected item order length: %+v", got)
	}
	if sess.TurnDraft.ItemOrder[0] != "thinking:stream-segment:thinking:1" ||
		sess.TurnDraft.ItemOrder[1] != "tool:stream-tool:trace-1:call-1" ||
		sess.TurnDraft.ItemOrder[2] != "thinking:stream-segment:thinking:2" ||
		sess.TurnDraft.ItemOrder[3] != "assistant:stream-segment:assistant:1" {
		t.Fatalf("unexpected item order: %+v", sess.TurnDraft.ItemOrder)
	}
}

func TestProjectTurnDraftMergesStructuredPreviewWithToolLifecycle(t *testing.T) {
	sess := &session.Session{}
	when := time.Now().UTC()

	ProjectTurnDraft(sess, draftEvent(streaming.EventRunStarted, "trace-2", 3, nil), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventCompletionDelta, "trace-2", 3, map[string]any{
		"kind":            "tool_call_start",
		"tool_call_index": 0,
		"tool_name":       "bash_exec",
	}), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventCompletionDelta, "trace-2", 3, map[string]any{
		"kind":               "tool_call_delta",
		"tool_call_index":    0,
		"arguments_fragment": `{"command":"pwd"}`,
	}), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventCompletionDelta, "trace-2", 3, map[string]any{
		"kind":            "tool_call_end",
		"tool_call_index": 0,
	}), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventToolCallStarted, "trace-2", 3, map[string]any{
		"tool":           "bash_exec",
		"tool_call_id":   "call-9",
		"arguments_json": `{"command":"pwd"}`,
	}), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventToolCallFinished, "trace-2", 3, map[string]any{
		"tool":         "bash_exec",
		"tool_call_id": "call-9",
		"status":       "success",
		"output":       "/workspace",
	}), when)

	if sess.TurnDraft == nil || len(sess.TurnDraft.Tools) != 1 {
		t.Fatalf("expected single merged tool, got %+v", sess.TurnDraft)
	}
	if sess.TurnDraft.Status != session.TurnDraftStatusStreaming {
		t.Fatalf("unexpected draft status: %q", sess.TurnDraft.Status)
	}
	tool := sess.TurnDraft.Tools[0]
	if tool.ID != "stream-tool:trace-2:preview:1:index:0" {
		t.Fatalf("unexpected merged tool id: %q", tool.ID)
	}
	if tool.ToolCallID != "call-9" {
		t.Fatalf("expected lifecycle to bind tool_call_id, got %+v", tool)
	}
	if tool.ToolStatus != "success" || tool.Content != "/workspace" {
		t.Fatalf("unexpected merged tool payload: %+v", tool)
	}
	if len(sess.TurnDraft.ItemOrder) != 1 || sess.TurnDraft.ItemOrder[0] != "tool:"+tool.ID {
		t.Fatalf("unexpected tool order: %+v", sess.TurnDraft.ItemOrder)
	}
}

func TestProjectTurnDraftClearsOnDone(t *testing.T) {
	sess := &session.Session{}
	when := time.Now().UTC()
	ProjectTurnDraft(sess, draftEvent(streaming.EventRunStarted, "trace-3", 1, nil), when)
	ProjectTurnDraft(sess, draftEvent(streaming.EventCompletionDelta, "trace-3", 1, map[string]any{
		"kind": "text",
		"text": "partial",
	}), when)

	if !ProjectTurnDraft(sess, draftEvent(streaming.EventDone, "trace-3", 1, map[string]any{}), when) {
		t.Fatal("expected done to clear draft")
	}
	if sess.TurnDraft != nil {
		t.Fatalf("expected draft to be cleared on done, got %+v", sess.TurnDraft)
	}
}

func draftEvent(
	eventType streaming.EventType,
	traceID string,
	turn int,
	payload any,
) streaming.Event {
	return streaming.Event{
		ID:      string(eventType) + ":" + traceID,
		TraceID: traceID,
		Turn:    turn,
		Type:    eventType,
		Payload: payload,
	}
}
