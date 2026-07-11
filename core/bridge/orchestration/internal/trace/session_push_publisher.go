package trace

import (
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/session"
	bridgetools "ghost-os/bridge/tools"
)

func PublishAssistantMessage(
	hub *SessionPushHub,
	traceID string,
	sessionID string,
	payload api.AssistantMessagePushPayload,
) {
	if hub == nil || strings.TrimSpace(sessionID) == "" {
		return
	}
	hub.Publish(SessionPushEvent{
		Type:      SessionPushAssistantMessage,
		TraceID:   strings.TrimSpace(traceID),
		SessionID: strings.TrimSpace(sessionID),
		Payload:   payload,
	})
}

func PublishAwaitingHuman(
	hub *SessionPushHub,
	traceID string,
	sessionID string,
	awaitingErr *agent.ErrAwaitingHuman,
) {
	if hub == nil || awaitingErr == nil {
		return
	}
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return
	}
	hub.Publish(SessionPushEvent{
		Type:      SessionPushAwaitingHuman,
		TraceID:   strings.TrimSpace(traceID),
		SessionID: trimmedSessionID,
		Payload: api.AwaitingHumanPushPayload{
			QuestionID:    awaitingErr.QuestionID,
			Prompt:        awaitingErr.Prompt,
			SelectionMode: strings.TrimSpace(awaitingErr.SelectionMode),
			Options:       awaitingHumanOptions(awaitingErr.Options),
		},
	})
}

func PendingQuestionSnapshot(
	store *session.Store,
	sessionID string,
) (SessionPushEvent, bool) {
	if store == nil {
		return SessionPushEvent{}, false
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return SessionPushEvent{}, false
	}
	sess, err := store.Load(id)
	questionID, question, ok := latestPendingQuestion(sess)
	if err != nil || !ok {
		return SessionPushEvent{}, false
	}
	return SessionPushEvent{
		Type:      SessionPushAwaitingHuman,
		TraceID:   strings.TrimSpace(question.TraceID),
		SessionID: id,
		Payload: api.AwaitingHumanPushPayload{
			QuestionID:    questionID,
			Prompt:        question.Prompt,
			SelectionMode: strings.TrimSpace(question.SelectionMode),
			Options:       sessionHumanOptions(question.Options),
		},
		At: time.Now().UTC(),
	}, true
}

func latestPendingQuestion(sess *session.Session) (string, session.PendingHumanQuestion, bool) {
	if sess == nil || len(sess.PendingQuestions) == 0 {
		return "", session.PendingHumanQuestion{}, false
	}
	var (
		latestID       string
		latestQuestion session.PendingHumanQuestion
		found          bool
	)
	for id, question := range sess.PendingQuestions {
		if !found || question.CreatedAt.After(latestQuestion.CreatedAt) {
			latestID = id
			latestQuestion = question
			found = true
		}
	}
	return latestID, latestQuestion, found
}

func awaitingHumanOptions(options []bridgetools.AskHumanOption) []api.AskHumanOption {
	if len(options) == 0 {
		return nil
	}
	out := make([]api.AskHumanOption, 0, len(options))
	for _, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			continue
		}
		out = append(out, api.AskHumanOption{
			Label:       label,
			AllowCustom: option.AllowCustom,
		})
	}
	return out
}

func sessionHumanOptions(options []session.HumanQuestionOption) []api.AskHumanOption {
	if len(options) == 0 {
		return nil
	}
	out := make([]api.AskHumanOption, 0, len(options))
	for _, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			continue
		}
		out = append(out, api.AskHumanOption{
			Label:       label,
			AllowCustom: option.AllowCustom,
		})
	}
	return out
}
