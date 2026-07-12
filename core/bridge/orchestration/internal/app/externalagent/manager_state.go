package externalagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func (r *runtimeSession) registerApproval(id string) chan string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pending == nil {
		r.pending = make(map[string]chan string)
	}
	ch := make(chan string, 1)
	r.pending[strings.TrimSpace(id)] = ch
	return ch
}

func (m *Manager) persistCancelledRun(sessionID string, traceID string) error {
	if m == nil || m.SessionStore == nil {
		return fmt.Errorf("session store is not configured")
	}
	sess, err := m.SessionStore.Load(sessionID)
	if err != nil {
		return err
	}
	if !sess.SetLastRunState(session.RunStatusCancelled, traceID, time.Now().UTC()) {
		return nil
	}
	return m.SessionStore.Save(sess)
}

func (r *runtimeSession) resolveApproval(id string, decision string) error {
	r.mu.Lock()
	ch := r.pending[strings.TrimSpace(id)]
	if ch != nil {
		delete(r.pending, strings.TrimSpace(id))
	}
	r.mu.Unlock()
	if ch == nil {
		return ErrApprovalNotFound
	}
	ch <- decision
	return nil
}

func (m *Manager) handleApproval(ctx context.Context, runtime *runtimeSession, approval ApprovalRequest) (string, error) {
	active := runtime.activeSnapshot()
	if active == nil {
		return DecisionDenied, ErrExternalRunMissing
	}
	approval.ID = defaultString(approval.ID, approval.CallID)
	if approval.ID == "" {
		approval.ID = fmt.Sprintf("approval-%d", time.Now().UnixNano())
	}
	ch := runtime.registerApproval(approval.ID)
	record := session.ExternalPendingApproval{
		ID:        approval.ID,
		Provider:  ProviderCodex,
		Kind:      approval.Kind,
		Tool:      approval.Tool,
		CallID:    approval.CallID,
		Prompt:    approval.Prompt,
		Payload:   clonePayload(approval.Payload),
		CreatedAt: time.Now().UTC(),
	}
	_ = m.addPendingApproval(active.sessionID, record)
	_ = m.flushPendingAssistantMessage(runtime, active.sessionID)
	_ = m.appendApprovalToolCall(active.sessionID, approval.ID, record.Payload)
	_ = emitApprovalEvent(ctx, active, approval)
	select {
	case <-ctx.Done():
		_ = m.removePendingApproval(active.sessionID, approval.ID)
		return DecisionDenied, ctx.Err()
	case decision := <-ch:
		_ = m.removePendingApproval(active.sessionID, approval.ID)
		_ = m.appendToolResultMessage(
			active.sessionID,
			"approval:"+approval.ID,
			"codex_approval",
			active.traceID,
			decision,
			nil,
		)
		return decision, nil
	}
}

func (m *Manager) appendApprovalToolCall(sessionID string, approvalID string, payload map[string]any) error {
	return m.appendSessionMessage(sessionID, llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "approval:" + approvalID,
			Name:      "codex_approval",
			Arguments: rawJSONMap(payload),
		}},
	})
}

func emitApprovalEvent(ctx context.Context, active *activeTurn, approval ApprovalRequest) error {
	return emit(ctx, active.sink, active.traceID, active.sessionID, active.turn, toolStep(active.turn, approval.ID), streaming.EventAwaitingHuman, map[string]any{
		"tool":           "codex_approval",
		"tool_call_id":   "approval:" + approval.ID,
		"question_id":    approval.ID,
		"prompt":         approval.Prompt,
		"selection_mode": "single",
		"approval": map[string]any{
			"id":      approval.ID,
			"kind":    approval.Kind,
			"tool":    approval.Tool,
			"payload": approval.Payload,
		},
		"options": []map[string]any{
			{"label": DecisionApproved},
			{"label": DecisionApprovedForSession},
			{"label": DecisionDenied},
			{"label": DecisionAbort},
		},
	})
}

func (m *Manager) appendSessionMessage(sessionID string, msg llm.Message) error {
	sess, err := m.SessionStore.Load(sessionID)
	if err != nil {
		return err
	}
	sess.AddMessage(msg)
	return m.SessionStore.Save(sess)
}

func (m *Manager) appendToolResultMessage(
	sessionID string,
	toolCallID string,
	toolName string,
	traceID string,
	output string,
	toolErr error,
) error {
	return m.appendSessionMessage(sessionID, llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: toolCallID,
		Text:       llm.FormatToolResult(toolName, traceID, output, toolErr),
	})
}

func (m *Manager) updateRuntimeState(sessionID string, mutate func(*session.ExternalRuntime)) error {
	sess, err := m.SessionStore.Load(sessionID)
	if err != nil {
		return err
	}
	ext := cloneOrNewRuntime(sess.ExternalRuntime)
	mutate(ext)
	ext.UpdatedAt = time.Now().UTC()
	sess.SetExternalRuntime(ext)
	return m.SessionStore.Save(sess)
}

func (m *Manager) addPendingApproval(sessionID string, approval session.ExternalPendingApproval) error {
	return m.updateRuntimeState(sessionID, func(ext *session.ExternalRuntime) {
		ext.Status = StatusAwaitingApproval
		ext.PendingApprovals = append(replacePendingApproval(ext.PendingApprovals, approval.ID), approval)
	})
}

func (m *Manager) removePendingApproval(sessionID string, approvalID string) error {
	return m.updateRuntimeState(sessionID, func(ext *session.ExternalRuntime) {
		ext.PendingApprovals = replacePendingApproval(ext.PendingApprovals, approvalID)
		if len(ext.PendingApprovals) == 0 {
			ext.Status = StatusRunning
		}
	})
}

func (m *Manager) externalRuntime(sessionID string) (*session.ExternalRuntime, error) {
	sess, err := m.SessionStore.Load(sessionID)
	if err != nil {
		return nil, err
	}
	if sess.ExternalRuntime == nil {
		return nil, nil
	}
	return cloneOrNewRuntime(sess.ExternalRuntime), nil
}

func (m *Manager) finishWithError(
	ctx context.Context,
	sessionID string,
	traceID string,
	turn int,
	sink streaming.Sink,
	err error,
) error {
	_ = m.updateRuntimeState(sessionID, func(ext *session.ExternalRuntime) {
		ext.Status = StatusError
		ext.PendingApprovals = nil
	})
	return emit(ctx, sink, traceID, sessionID, turn, "", streaming.EventError, map[string]any{
		"message":    err.Error(),
		"session_id": sessionID,
	})
}

func replacePendingApproval(items []session.ExternalPendingApproval, approvalID string) []session.ExternalPendingApproval {
	out := make([]session.ExternalPendingApproval, 0, len(items))
	for _, item := range items {
		if item.ID != strings.TrimSpace(approvalID) {
			out = append(out, item)
		}
	}
	return out
}

func cloneOrNewRuntime(raw *session.ExternalRuntime) *session.ExternalRuntime {
	if raw == nil {
		return &session.ExternalRuntime{}
	}
	data, _ := json.Marshal(raw)
	var cloned session.ExternalRuntime
	_ = json.Unmarshal(data, &cloned)
	return &cloned
}

func emit(ctx context.Context, sink streaming.Sink, traceID string, sessionID string, turn int, stepID string, eventType streaming.EventType, payload any) error {
	event, err := streaming.NewEvent(traceID, sessionID, turn, stepID, eventType, payload)
	if err != nil {
		return err
	}
	_, err = ensureSink(sink).Emit(ctx, event)
	return err
}

func ensureSink(sink streaming.Sink) streaming.Sink {
	if sink == nil {
		return streaming.NopSink{}
	}
	return sink
}

func assistantStep(turn int) string {
	step, err := streaming.AssistantStepID(turn)
	if err != nil {
		return ""
	}
	return step
}

func toolStep(turn int, callID string) string {
	index := 0
	if strings.TrimSpace(callID) != "" {
		index = int(time.Now().UnixNano() % 100000)
	}
	step, err := streaming.ToolStepID(turn, index)
	if err != nil {
		return ""
	}
	return step
}
