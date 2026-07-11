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
	case "agent_message":
		m.recordAgentMessage(ctx, runtime, active, event)
	case "agent_reasoning", "agent_reasoning_delta":
		m.recordReasoning(ctx, active, event)
	case "exec_command_begin":
		m.recordToolStart(ctx, active, "codex_exec", event)
	case "patch_apply_begin":
		m.recordToolStart(ctx, active, "codex_patch", event)
	case "mcp_tool_begin":
		m.recordToolStart(ctx, active, "codex_mcp", event)
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
	text := firstString(event.Payload["message"], event.Payload["text"])
	if text == "" {
		return
	}
	runtime.appendText(text)
	_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, assistantStep(active.turn), streaming.EventCompletionDelta, map[string]any{
		"kind": "text",
		"text": text,
	})
}

func (m *Manager) recordReasoning(ctx context.Context, active *activeTurn, event CodexEvent) {
	text := firstString(event.Payload["text"], event.Payload["delta"])
	if text == "" {
		return
	}
	_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, assistantStep(active.turn), streaming.EventCompletionDelta, map[string]any{
		"kind":     "thinking",
		"thinking": text,
	})
}

func (m *Manager) recordToolStart(ctx context.Context, active *activeTurn, tool string, event CodexEvent) {
	callID := firstString(event.Payload["call_id"], event.Payload["callId"])
	if callID == "" {
		callID = fmt.Sprintf("%s-%d", tool, time.Now().UnixNano())
	}
	args := clonePayload(event.Payload)
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
	final := strings.TrimSpace(runtime.finalText())
	if final != "" {
		_ = m.appendSessionMessage(active.sessionID, llm.Message{Role: llm.RoleAssistant, Text: final})
		_ = emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, assistantStep(active.turn), streaming.EventMessage, map[string]any{
			"text":       final,
			"session_id": active.sessionID,
		})
	}
	_ = event
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
