package sessiondraft

import (
	"fmt"
	"strings"
	"time"

	bridgesession "ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func projectSessionTurnDraftCompletionDelta(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) bool {
	draft, _ := ensureSessionTurnDraft(sess, event, at)
	if draft == nil {
		return false
	}

	kind := payloadString(event.Payload, "kind")
	switch kind {
	case "thinking":
		return updateTurnDraftTimestamp(sess, at, appendTurnDraftThinking(draft, payloadString(event.Payload, "thinking")))
	case "text":
		return updateTurnDraftTimestamp(sess, at, projectTurnDraftTextDelta(draft, event.TraceID, payloadString(event.Payload, "text")))
	case "tool_call_start":
		return updateTurnDraftTimestamp(sess, at, projectTurnDraftStructuredToolStart(draft, event.TraceID, event.Payload))
	case "tool_call_delta":
		return updateTurnDraftTimestamp(sess, at, projectTurnDraftStructuredToolDelta(draft, event.TraceID, event.Payload))
	case "tool_call_end":
		return updateTurnDraftTimestamp(sess, at, projectTurnDraftStructuredToolEnd(draft, event.TraceID, event.Payload))
	default:
		return false
	}
}

func projectSessionTurnDraftToolStarted(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) bool {
	draft, _ := ensureSessionTurnDraft(sess, event, at)
	if draft == nil {
		return false
	}

	toolCallID := payloadString(event.Payload, "tool_call_id")
	messageID := resolveDraftToolMessageID(draft, event.TraceID, toolCallID, event.ID)
	existing := draftToolByID(draft, messageID)
	argsPreview := payloadString(event.Payload, "arguments_json")
	if argsPreview == "" && existing != nil {
		argsPreview = existing.Content
	}
	toolName := payloadString(event.Payload, "tool")
	if toolName == "" && existing != nil {
		toolName = existing.ToolName
	}
	tool := bridgesession.TurnDraftTool{
		ID:         messageID,
		Content:    draftFirstNonEmpty(argsPreview, fmt.Sprintf("%s running", draftDisplayToolName(toolName))),
		ToolInput:  draftResolveToolInput(toolName, argsPreview),
		ToolName:   toolName,
		ToolStatus: draftToolRunningStatus,
		ToolCallID: toolCallID,
		TraceID:    strings.TrimSpace(event.TraceID),
	}
	return updateTurnDraftTimestamp(sess, at, upsertTurnDraftTool(draft, tool))
}

func projectSessionTurnDraftToolFinished(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) bool {
	draft, _ := ensureSessionTurnDraft(sess, event, at)
	if draft == nil {
		return false
	}

	toolCallID := payloadString(event.Payload, "tool_call_id")
	messageID := resolveDraftToolMessageID(draft, event.TraceID, toolCallID, event.ID)
	existing := draftToolByID(draft, messageID)
	toolName := payloadString(event.Payload, "tool")
	if toolName == "" && existing != nil {
		toolName = existing.ToolName
	}
	argsPreview := ""
	if existing != nil {
		argsPreview = existing.Content
	}
	content := draftFirstNonEmpty(
		payloadString(event.Payload, "error"),
		payloadString(event.Payload, "output"),
		argsPreview,
		fmt.Sprintf("%s finished", draftDisplayToolName(toolName)),
	)
	tool := bridgesession.TurnDraftTool{
		ID:         messageID,
		Content:    content,
		ToolInput:  draftResolveToolInput(toolName, argsPreview),
		ToolName:   toolName,
		ToolStatus: resolveTurnDraftToolStatus(event.Payload),
		ToolCallID: toolCallID,
		TraceID:    strings.TrimSpace(event.TraceID),
	}
	return updateTurnDraftTimestamp(sess, at, upsertTurnDraftTool(draft, tool))
}

func updateTurnDraftTimestamp(sess *bridgesession.Session, at time.Time, changed bool) bool {
	if !changed || sess == nil {
		return false
	}
	sess.UpdatedAt = at.UTC()
	return true
}
