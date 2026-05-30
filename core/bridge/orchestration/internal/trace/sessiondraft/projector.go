package sessiondraft

import (
	"fmt"
	"strings"
	"time"

	bridgesession "ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

const (
	draftAssistantOrderPrefix = "assistant:"
	draftThinkingOrderPrefix  = "thinking:"
	draftToolOrderPrefix      = "tool:"

	draftAssistantSegmentPrefix = "stream-segment:assistant:"
	draftThinkingSegmentPrefix  = "stream-segment:thinking:"

	draftToolPendingStatus = "pending"
	draftToolRunningStatus = "running"
	draftToolSuccessStatus = "success"
	draftToolErrorStatus   = "error"
)

func ProjectTurnDraft(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) bool {
	switch event.Type {
	case streaming.EventRunStarted:
		return projectSessionTurnDraftRunStarted(sess, event, at)
	case streaming.EventCompletionDelta:
		return projectSessionTurnDraftCompletionDelta(sess, event, at)
	case streaming.EventToolCallStarted:
		return projectSessionTurnDraftToolStarted(sess, event, at)
	case streaming.EventToolCallFinished:
		return projectSessionTurnDraftToolFinished(sess, event, at)
	case streaming.EventAwaitingHuman:
		return projectSessionTurnDraftAwaitingHuman(sess, event, at)
	case streaming.EventError:
		return projectSessionTurnDraftError(sess, event, at)
	case streaming.EventDone:
		return clearSessionTurnDraft(sess, event, at)
	default:
		return false
	}
}

func ensureSessionTurnDraft(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) (*bridgesession.TurnDraft, bool) {
	if sess == nil || strings.TrimSpace(event.TraceID) == "" {
		return nil, false
	}

	if sess.TurnDraft != nil &&
		strings.TrimSpace(sess.TurnDraft.TraceID) == strings.TrimSpace(event.TraceID) &&
		sess.TurnDraft.Turn == event.Turn {
		return sess.TurnDraft, false
	}

	sess.TurnDraft = &bridgesession.TurnDraft{
		TraceID: strings.TrimSpace(event.TraceID),
		Turn:    event.Turn,
		Status:  bridgesession.TurnDraftStatusStreaming,
		ToolTagState: &bridgesession.TurnDraftToolTagState{
			Mode:        "normal",
			NextCallSeq: 1,
		},
	}
	sess.UpdatedAt = at.UTC()
	return sess.TurnDraft, true
}

func clearSessionTurnDraft(sess *bridgesession.Session, event streaming.Event, at time.Time) bool {
	if sess == nil || sess.TurnDraft == nil {
		return false
	}
	if strings.TrimSpace(sess.TurnDraft.TraceID) != strings.TrimSpace(event.TraceID) {
		return false
	}
	if sess.TurnDraft.Turn != event.Turn {
		return false
	}
	return sess.ClearTurnDraft(at.UTC())
}

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

func projectSessionTurnDraftRunStarted(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) bool {
	draft, created := ensureSessionTurnDraft(sess, event, at)
	if draft == nil {
		return false
	}
	return updateTurnDraftTimestamp(sess, at, created || resetTurnDraftStreamingState(draft))
}

func projectSessionTurnDraftAwaitingHuman(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) bool {
	draft, created := ensureSessionTurnDraft(sess, event, at)
	if draft == nil {
		return false
	}

	changed := created || setTurnDraftStatus(draft, bridgesession.TurnDraftStatusAwaitingHuman)
	if question, ok := pendingQuestionFromPayload(event.Payload); ok {
		changed = upsertTurnDraftPendingQuestion(draft, question) || changed
	}
	if draft.Error != "" {
		draft.Error = ""
		changed = true
	}
	return updateTurnDraftTimestamp(sess, at, changed)
}

func projectSessionTurnDraftError(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) bool {
	draft, created := ensureSessionTurnDraft(sess, event, at)
	if draft == nil {
		return false
	}

	changed := created || setTurnDraftStatus(draft, bridgesession.TurnDraftStatusError)
	message := payloadString(event.Payload, "message")
	if draft.Error != message {
		draft.Error = message
		changed = true
	}
	return updateTurnDraftTimestamp(sess, at, changed)
}

func resetTurnDraftStreamingState(draft *bridgesession.TurnDraft) bool {
	if draft == nil {
		return false
	}

	changed := setTurnDraftStatus(draft, bridgesession.TurnDraftStatusStreaming)
	if draft.Error != "" {
		draft.Error = ""
		changed = true
	}
	if len(draft.PendingQuestions) > 0 {
		draft.PendingQuestions = nil
		changed = true
	}
	return changed
}

func setTurnDraftStatus(draft *bridgesession.TurnDraft, status string) bool {
	if draft == nil {
		return false
	}
	status = strings.TrimSpace(status)
	if draft.Status == status {
		return false
	}
	draft.Status = status
	return true
}

func pendingQuestionFromPayload(payload any) (bridgesession.TurnDraftPendingQuestion, bool) {
	record, ok := payload.(map[string]any)
	if !ok {
		return bridgesession.TurnDraftPendingQuestion{}, false
	}

	question := bridgesession.TurnDraftPendingQuestion{
		QuestionID:    payloadString(payload, "question_id"),
		Prompt:        payloadString(payload, "prompt"),
		SelectionMode: payloadString(payload, "selection_mode"),
		Options:       pendingQuestionOptions(record["options"]),
	}
	if question.QuestionID == "" || question.Prompt == "" {
		return bridgesession.TurnDraftPendingQuestion{}, false
	}
	return question, true
}

func pendingQuestionOptions(raw any) []bridgesession.HumanQuestionOption {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	options := make([]bridgesession.HumanQuestionOption, 0, len(items))
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		label, _ := record["label"].(string)
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		allowCustom, _ := record["allow_custom"].(bool)
		options = append(options, bridgesession.HumanQuestionOption{
			Label:       label,
			AllowCustom: allowCustom,
		})
	}
	return options
}
