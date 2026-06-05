package sessiondraft

import (
	"strings"
	"time"

	bridgesession "ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

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
