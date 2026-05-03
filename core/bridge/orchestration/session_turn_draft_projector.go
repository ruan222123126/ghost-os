package orchestration

import (
	"fmt"
	"strconv"
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

func projectSessionTurnDraft(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) bool {
	switch event.Type {
	case streaming.EventRunStarted:
		return ensureSessionTurnDraft(sess, event, at) != nil
	case streaming.EventCompletionDelta:
		return projectSessionTurnDraftCompletionDelta(sess, event, at)
	case streaming.EventToolCallStarted:
		return projectSessionTurnDraftToolStarted(sess, event, at)
	case streaming.EventToolCallFinished:
		return projectSessionTurnDraftToolFinished(sess, event, at)
	case streaming.EventAwaitingHuman, streaming.EventDone, streaming.EventError:
		return clearSessionTurnDraft(sess, event, at)
	default:
		return false
	}
}

func ensureSessionTurnDraft(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) *bridgesession.TurnDraft {
	if sess == nil || strings.TrimSpace(event.TraceID) == "" {
		return nil
	}

	if sess.TurnDraft != nil &&
		strings.TrimSpace(sess.TurnDraft.TraceID) == strings.TrimSpace(event.TraceID) &&
		sess.TurnDraft.Turn == event.Turn {
		return sess.TurnDraft
	}

	sess.TurnDraft = &bridgesession.TurnDraft{
		TraceID: strings.TrimSpace(event.TraceID),
		Turn:    event.Turn,
		ToolTagState: &bridgesession.TurnDraftToolTagState{
			Mode:        "normal",
			NextCallSeq: 1,
		},
	}
	sess.UpdatedAt = at.UTC()
	return sess.TurnDraft
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
	draft := ensureSessionTurnDraft(sess, event, at)
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
	draft := ensureSessionTurnDraft(sess, event, at)
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
	draft := ensureSessionTurnDraft(sess, event, at)
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

func payloadString(payload any, key string) string {
	record, ok := payload.(map[string]any)
	if !ok {
		return ""
	}
	value, _ := record[key].(string)
	return strings.TrimSpace(value)
}

func payloadInt(payload any, key string) (int, bool) {
	record, ok := payload.(map[string]any)
	if !ok {
		return 0, false
	}
	switch typed := record[key].(type) {
	case int:
		return typed, true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func resolveTurnDraftToolStatus(payload any) string {
	status := payloadString(payload, "status")
	if status != "" {
		return status
	}
	if payloadString(payload, "error") != "" {
		return draftToolErrorStatus
	}
	return draftToolSuccessStatus
}

func draftFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func draftDisplayToolName(toolName string) string {
	if strings.TrimSpace(toolName) == "" {
		return "Tool"
	}
	return toolName
}

func appendTurnDraftAssistant(draft *bridgesession.TurnDraft, text string) bool {
	return appendTurnDraftSegment(
		draft,
		text,
		draftAssistantOrderPrefix,
		draftAssistantSegmentPrefix,
		&draft.AssistantSegments,
	)
}

func appendTurnDraftThinking(draft *bridgesession.TurnDraft, text string) bool {
	return appendTurnDraftSegment(
		draft,
		text,
		draftThinkingOrderPrefix,
		draftThinkingSegmentPrefix,
		&draft.ThinkingSegments,
	)
}

func appendTurnDraftSegment(
	draft *bridgesession.TurnDraft,
	text string,
	orderPrefix string,
	segmentPrefix string,
	target *[]bridgesession.TurnDraftSegment,
) bool {
	if draft == nil || text == "" {
		return false
	}

	activeID := activeTurnDraftSegmentID(draft.ItemOrder, orderPrefix)
	if activeID == "" {
		segmentID := nextTurnDraftSegmentID(*target, segmentPrefix)
		*target = append(*target, bridgesession.TurnDraftSegment{ID: segmentID, Content: text})
		draft.ItemOrder = appendUniqueTurnDraftOrder(draft.ItemOrder, orderPrefix+segmentID)
		return true
	}

	index := draftSegmentIndex(*target, activeID)
	if index < 0 {
		return false
	}
	(*target)[index].Content += text
	return true
}

func upsertTurnDraftTool(draft *bridgesession.TurnDraft, next bridgesession.TurnDraftTool) bool {
	if draft == nil || strings.TrimSpace(next.ID) == "" {
		return false
	}

	index := draftToolIndex(draft, next.ID)
	if index < 0 {
		draft.Tools = append(draft.Tools, next)
		draft.ItemOrder = appendUniqueTurnDraftOrder(draft.ItemOrder, draftToolOrderPrefix+next.ID)
		return true
	}

	current := draft.Tools[index]
	draft.Tools[index] = mergeTurnDraftTool(current, next)
	return true
}

func mergeTurnDraftTool(current bridgesession.TurnDraftTool, next bridgesession.TurnDraftTool) bridgesession.TurnDraftTool {
	if strings.TrimSpace(next.Content) == "" {
		next.Content = current.Content
	}
	if strings.TrimSpace(next.ToolInput) == "" {
		next.ToolInput = current.ToolInput
	}
	if strings.TrimSpace(next.ToolName) == "" {
		next.ToolName = current.ToolName
	}
	if strings.TrimSpace(next.ToolStatus) == "" {
		next.ToolStatus = current.ToolStatus
	}
	if strings.TrimSpace(next.ToolCallID) == "" {
		next.ToolCallID = current.ToolCallID
	}
	if strings.TrimSpace(next.TraceID) == "" {
		next.TraceID = current.TraceID
	}
	return next
}

func appendUniqueTurnDraftOrder(order []string, key string) []string {
	for _, existing := range order {
		if existing == key {
			return order
		}
	}
	return append(order, key)
}

func activeTurnDraftSegmentID(order []string, prefix string) string {
	if len(order) == 0 {
		return ""
	}
	last := order[len(order)-1]
	if !strings.HasPrefix(last, prefix) {
		return ""
	}
	return strings.TrimPrefix(last, prefix)
}

func nextTurnDraftSegmentID(segments []bridgesession.TurnDraftSegment, prefix string) string {
	return fmt.Sprintf("%s%d", prefix, maxTurnDraftSegmentSequence(segments)+1)
}

func maxTurnDraftSegmentSequence(segments []bridgesession.TurnDraftSegment) int {
	maxSeq := 0
	for _, segment := range segments {
		sequence, err := strconv.Atoi(segment.ID[strings.LastIndex(segment.ID, ":")+1:])
		if err == nil && sequence > maxSeq {
			maxSeq = sequence
		}
	}
	return maxSeq
}

func draftSegmentIndex(segments []bridgesession.TurnDraftSegment, id string) int {
	for index, segment := range segments {
		if segment.ID == id {
			return index
		}
	}
	return -1
}

func draftToolIndex(draft *bridgesession.TurnDraft, id string) int {
	for index, tool := range draft.Tools {
		if tool.ID == id {
			return index
		}
	}
	return -1
}

func draftToolByID(draft *bridgesession.TurnDraft, id string) *bridgesession.TurnDraftTool {
	index := draftToolIndex(draft, id)
	if index < 0 {
		return nil
	}
	return &draft.Tools[index]
}
