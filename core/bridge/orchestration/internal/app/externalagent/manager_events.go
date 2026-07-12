package externalagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func (m *Manager) handleEvent(ctx context.Context, runtime *runtimeSession, event CodexEvent) {
	active := runtime.activeSnapshot()
	if active == nil {
		return
	}
	switch event.Type {
	case "task_started":
		m.recordTaskStarted(active, event)
	case codexEventAgentMessageSnapshot:
		m.recordAgentMessageSnapshot(ctx, runtime, active, event)
	case "agent_message", "agent_message_chunk", "agent_message_delta", "agent_message_content_delta":
		m.recordAgentMessage(ctx, runtime, active, event)
	case "agent_reasoning", "agent_reasoning_delta", "agent_reasoning_content_delta", "reasoning_content_delta", "reasoning_raw_content_delta":
		m.recordReasoning(ctx, active, event)
	case "exec_command_begin":
		m.recordToolStart(ctx, runtime, active, "codex_exec", event)
	case "patch_apply_begin":
		m.recordToolStart(ctx, runtime, active, "codex_patch", event)
	case "mcp_tool_begin":
		m.recordToolStart(ctx, runtime, active, "codex_mcp", event)
	case "exec_command_end":
		m.recordToolEnd(ctx, active, "codex_exec", event)
	case "patch_apply_end":
		m.recordToolEnd(ctx, active, "codex_patch", event)
	case "mcp_tool_end":
		m.recordToolEnd(ctx, active, "codex_mcp", event)
	case "task_complete":
		m.finishTurn(ctx, runtime, active, false, event)
	case "turn_aborted":
		m.finishTurn(ctx, runtime, active, true, event)
	}
}

func (m *Manager) recordTaskStarted(active *activeTurn, event CodexEvent) {
	if turnID := firstString(event.Payload["turn_id"], event.Payload["turnId"]); turnID != "" {
		_ = m.updateRuntimeState(active.sessionID, func(ext *session.ExternalRuntime) {
			ext.TurnID = turnID
			ext.Status = StatusRunning
		})
	}
}

func (m *Manager) recordAgentMessage(ctx context.Context, runtime *runtimeSession, active *activeTurn, event CodexEvent) {
	text := firstString(event.Payload["message"], event.Payload["text"], event.Payload["delta"], event.Payload["chunk"], event.Payload["content"])
	if text == "" {
		return
	}
	runtime.appendText(text)
	_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, assistantStep(active.turn), streaming.EventCompletionDelta, map[string]any{
		"kind": "text",
		"text": text,
	})
}

func (m *Manager) recordAgentMessageSnapshot(ctx context.Context, runtime *runtimeSession, active *activeTurn, event CodexEvent) {
	text := firstString(event.Payload["message"], event.Payload["text"], event.Payload["delta"], event.Payload["chunk"], event.Payload["content"])
	if text == "" {
		return
	}
	if delta := runtime.textSnapshotDelta(text); delta != "" {
		m.recordAgentMessage(ctx, runtime, active, CodexEvent{Payload: map[string]any{"message": delta}})
	}
}

func (m *Manager) recordReasoning(ctx context.Context, active *activeTurn, event CodexEvent) {
	text := firstString(event.Payload["text"], event.Payload["delta"], event.Payload["chunk"], event.Payload["content"])
	if text == "" {
		return
	}
	_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, assistantStep(active.turn), streaming.EventCompletionDelta, map[string]any{
		"kind":     "thinking",
		"thinking": text,
	})
}

func (m *Manager) recordToolStart(ctx context.Context, runtime *runtimeSession, active *activeTurn, tool string, event CodexEvent) {
	callID := firstString(event.Payload["call_id"], event.Payload["callId"])
	if callID == "" {
		callID = fmt.Sprintf("%s-%d", tool, time.Now().UnixNano())
	}
	args := clonePayload(event.Payload)
	_ = m.flushPendingAssistantMessage(runtime, active.sessionID)
	_ = m.appendSessionMessage(active.sessionID, llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        callID,
			Name:      tool,
			Arguments: rawJSONMap(args),
		}},
	})
	_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, toolStep(active.turn, callID), streaming.EventToolCallStarted, map[string]any{
		"tool":           tool,
		"tool_call_id":   callID,
		"arguments_json": string(rawJSONMap(args)),
	})
}

func (m *Manager) recordToolEnd(ctx context.Context, active *activeTurn, tool string, event CodexEvent) {
	callID := firstString(event.Payload["call_id"], event.Payload["callId"])
	output, errorText := resolveToolEndResult(event.Payload)
	status := "success"
	var toolErr error
	if errorText != "" {
		status = "error"
		toolErr = fmt.Errorf("%s", errorText)
	}
	_ = m.appendToolResultMessage(active.sessionID, callID, tool, active.traceID, output, toolErr)
	_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, toolStep(active.turn, callID), streaming.EventToolCallFinished, map[string]any{
		"tool":         tool,
		"tool_call_id": callID,
		"status":       status,
		"output":       output,
		"error":        errorText,
	})
}

func (m *Manager) finishTurn(ctx context.Context, runtime *runtimeSession, active *activeTurn, aborted bool, event CodexEvent) {
	if aborted {
		m.finishAbortedTurn(ctx, runtime, active)
		return
	}
	if err := taskCompleteError(event.Payload); err != nil {
		m.finishErroredTurn(ctx, runtime, active, err)
		return
	}
	m.recordFinalMessageSnapshot(ctx, runtime, active, event)
	final := strings.TrimSpace(runtime.finalText())
	if final != "" {
		_ = m.flushPendingAssistantMessage(runtime, active.sessionID)
		_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, assistantStep(active.turn), streaming.EventMessage, map[string]any{
			"text":       final,
			"session_id": active.sessionID,
		})
	}
	_ = m.updateRuntimeState(active.sessionID, func(ext *session.ExternalRuntime) {
		ext.Status = StatusIdle
		ext.TurnID = ""
		ext.PendingApprovals = nil
	})
	_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, "", streaming.EventDone, map[string]any{
		"session_id":    active.sessionID,
		"session_ended": false,
	})
	runtime.finish(turnDone{})
}

func taskCompleteError(payload map[string]any) error {
	if errorText := firstString(payload["error"], payload["error_message"], payload["errorMessage"]); errorText != "" {
		return fmt.Errorf("%s", errorText)
	}
	if statusText := firstString(payload["status"]); failedStatus(statusText) {
		return fmt.Errorf("codex turn failed with status %q", statusText)
	}
	return nil
}

func (m *Manager) recordFinalMessageSnapshot(ctx context.Context, runtime *runtimeSession, active *activeTurn, event CodexEvent) {
	text := firstString(event.Payload["last_agent_message"], event.Payload["lastAgentMessage"])
	if text == "" {
		return
	}
	current := strings.TrimSpace(runtime.finalText())
	if current == "" {
		m.recordAgentMessage(ctx, runtime, active, CodexEvent{Payload: map[string]any{"message": text}})
		return
	}
	if strings.HasPrefix(text, current) && text != current {
		m.recordAgentMessage(ctx, runtime, active, CodexEvent{Payload: map[string]any{"message": strings.TrimPrefix(text, current)}})
	}
}

func (m *Manager) finishAbortedTurn(ctx context.Context, runtime *runtimeSession, active *activeTurn) {
	_ = m.updateRuntimeState(active.sessionID, func(ext *session.ExternalRuntime) {
		ext.Status = StatusIdle
		ext.TurnID = ""
		ext.PendingApprovals = nil
	})
	runtime.finish(turnDone{aborted: true})
	_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, "", streaming.EventDone, map[string]any{
		"session_id": active.sessionID,
		"aborted":    true,
	})
}

func (m *Manager) finishErroredTurn(ctx context.Context, runtime *runtimeSession, active *activeTurn, err error) {
	_ = m.updateRuntimeState(active.sessionID, func(ext *session.ExternalRuntime) {
		ext.Status = StatusError
		ext.TurnID = ""
		ext.PendingApprovals = nil
	})
	_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, "", streaming.EventError, map[string]any{
		"message":    err.Error(),
		"session_id": active.sessionID,
	})
	runtime.finish(turnDone{err: err})
}

func (m *Manager) flushPendingAssistantMessage(runtime *runtimeSession, sessionID string) error {
	text := runtime.pendingText()
	if strings.TrimSpace(text) == "" {
		runtime.clearPendingText()
		return nil
	}
	if err := m.appendSessionMessage(sessionID, llm.Message{Role: llm.RoleAssistant, Text: text}); err != nil {
		return err
	}
	runtime.clearPendingText()
	return nil
}

func failedStatus(value any) bool {
	status := strings.ToLower(stringValue(value))
	return status == "failed" || status == "error" || status == "declined"
}

func resolveToolEndResult(payload map[string]any) (string, string) {
	statusText := firstString(payload["status"])
	errorText := firstString(payload["error"], payload["stderr"])
	if errorText == "" && failedStatus(payload["status"]) {
		errorText = statusText
	}
	output := firstString(payload["output"], payload["stdout"])
	if output == "" && errorText == "" {
		output = statusText
	}
	return output, errorText
}

func (r *runtimeSession) textSnapshotDelta(snapshot string) string {
	text := strings.TrimSpace(snapshot)
	if text == "" {
		return ""
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil {
		return ""
	}

	pending := strings.TrimSpace(r.active.pendingText.String())
	if delta, ok := resolveSnapshotDelta(text, pending); ok {
		return delta
	}

	final := strings.TrimSpace(r.active.text.String())
	if delta, ok := resolveSnapshotDelta(text, final); ok {
		return delta
	}
	if pending == "" && final != "" && strings.HasSuffix(final, text) {
		return ""
	}
	return text
}

func resolveSnapshotDelta(snapshot string, current string) (string, bool) {
	if current == "" {
		return "", false
	}
	if snapshot == current || strings.HasPrefix(current, snapshot) {
		return "", true
	}
	if strings.HasPrefix(snapshot, current) {
		return strings.TrimSpace(strings.TrimPrefix(snapshot, current)), true
	}
	return "", false
}
