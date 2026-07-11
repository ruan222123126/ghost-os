package sessiondraft

import (
	"fmt"
	"strconv"
	"strings"

	bridgesession "ghost-os/bridge/session"
)

type turnDraftSegmentAppend struct {
	draft         *bridgesession.TurnDraft
	text          string
	orderPrefix   string
	segmentPrefix string
	target        *[]bridgesession.TurnDraftSegment
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
	if draft == nil {
		return false
	}
	return appendTurnDraftSegment(turnDraftSegmentAppend{
		draft:         draft,
		text:          text,
		orderPrefix:   draftAssistantOrderPrefix,
		segmentPrefix: draftAssistantSegmentPrefix,
		target:        &draft.AssistantSegments,
	})
}

func appendTurnDraftThinking(draft *bridgesession.TurnDraft, text string) bool {
	if draft == nil {
		return false
	}
	return appendTurnDraftSegment(turnDraftSegmentAppend{
		draft:         draft,
		text:          text,
		orderPrefix:   draftThinkingOrderPrefix,
		segmentPrefix: draftThinkingSegmentPrefix,
		target:        &draft.ThinkingSegments,
	})
}

func appendTurnDraftSegment(input turnDraftSegmentAppend) bool {
	if input.draft == nil || input.text == "" {
		return false
	}

	activeID := activeTurnDraftSegmentID(input.draft.ItemOrder, input.orderPrefix)
	if activeID == "" {
		return appendNewTurnDraftSegment(input)
	}
	return appendActiveTurnDraftSegment(input.text, input.target, activeID)
}

func appendNewTurnDraftSegment(input turnDraftSegmentAppend) bool {
	segmentID := nextTurnDraftSegmentID(*input.target, input.segmentPrefix)
	*input.target = append(*input.target, bridgesession.TurnDraftSegment{ID: segmentID, Content: input.text})
	input.draft.ItemOrder = appendUniqueTurnDraftOrder(input.draft.ItemOrder, input.orderPrefix+segmentID)
	return true
}

func appendActiveTurnDraftSegment(
	text string,
	target *[]bridgesession.TurnDraftSegment,
	activeID string,
) bool {
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

func upsertTurnDraftPendingQuestion(
	draft *bridgesession.TurnDraft,
	next bridgesession.TurnDraftPendingQuestion,
) bool {
	if draft == nil {
		return false
	}

	questionID := strings.TrimSpace(next.QuestionID)
	prompt := strings.TrimSpace(next.Prompt)
	if questionID == "" || prompt == "" {
		return false
	}

	next.QuestionID = questionID
	next.Prompt = prompt
	next.SelectionMode = strings.TrimSpace(next.SelectionMode)
	next.Options = bridgesession.CloneHumanQuestionOptionsForDraft(next.Options)

	for index, current := range draft.PendingQuestions {
		if strings.TrimSpace(current.QuestionID) != questionID {
			continue
		}
		draft.PendingQuestions[index] = next
		return true
	}

	draft.PendingQuestions = append(draft.PendingQuestions, next)
	draft.ItemOrder = appendUniqueTurnDraftOrder(draft.ItemOrder, draftQuestionOrderKey(questionID))
	return true
}

func draftQuestionOrderKey(questionID string) string {
	return "question:" + strings.TrimSpace(questionID)
}
