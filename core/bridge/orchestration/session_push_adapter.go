package orchestration

import (
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/session"
)

type sessionPushAdapter struct {
	sessionStore *session.Store
	hub          *sessionPushHub
}

func newSessionPushAdapter(store *session.Store, hub *sessionPushHub) sessionPushAdapter {
	return sessionPushAdapter{
		sessionStore: store,
		hub:          hub,
	}
}

func (a sessionPushAdapter) publishAssistant(traceID string, result finalizedAgentTurn) {
	if a.hub == nil || strings.TrimSpace(result.sessionID) == "" {
		return
	}

	a.hub.Publish(sessionPushEvent{
		Type:      sessionPushAssistantMessage,
		TraceID:   strings.TrimSpace(traceID),
		SessionID: strings.TrimSpace(result.sessionID),
		Payload: assistantMessagePushPayload{
			Message:      result.message,
			SessionEnded: result.sessionEnd != nil,
			SessionEnd:   result.sessionEnd,
		},
	})
}

func (a sessionPushAdapter) publishAwaitingHuman(traceID string, sessionID string, awaitingErr *agent.ErrAwaitingHuman) {
	if a.hub == nil || awaitingErr == nil {
		return
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return
	}

	payload := awaitingHumanPushPayload{
		QuestionID:    awaitingErr.QuestionID,
		Prompt:        awaitingErr.Prompt,
		SelectionMode: strings.TrimSpace(awaitingErr.SelectionMode),
	}
	if len(awaitingErr.Options) > 0 {
		payload.Options = make([]askHumanOption, 0, len(awaitingErr.Options))
		for _, option := range awaitingErr.Options {
			label := strings.TrimSpace(option.Label)
			if label == "" {
				continue
			}
			payload.Options = append(payload.Options, askHumanOption{
				Label:       label,
				AllowCustom: option.AllowCustom,
			})
		}
	}

	a.hub.Publish(sessionPushEvent{
		Type:      sessionPushAwaitingHuman,
		TraceID:   strings.TrimSpace(traceID),
		SessionID: trimmedSessionID,
		Payload:   payload,
	})
}

func (a sessionPushAdapter) pendingQuestionSnapshot(sessionID string) (sessionPushEvent, bool) {
	if a.sessionStore == nil {
		return sessionPushEvent{}, false
	}

	id := strings.TrimSpace(sessionID)
	if id == "" {
		return sessionPushEvent{}, false
	}

	sess, err := a.sessionStore.Load(id)
	questionID, question, ok := latestPendingQuestion(sess)
	if err != nil || !ok {
		return sessionPushEvent{}, false
	}
	options := make([]askHumanOption, 0, len(question.Options))
	for _, option := range question.Options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			continue
		}
		options = append(options, askHumanOption{
			Label:       label,
			AllowCustom: option.AllowCustom,
		})
	}

	return sessionPushEvent{
		Type:      sessionPushAwaitingHuman,
		TraceID:   strings.TrimSpace(question.TraceID),
		SessionID: id,
		Payload: awaitingHumanPushPayload{
			QuestionID:    questionID,
			Prompt:        question.Prompt,
			SelectionMode: strings.TrimSpace(question.SelectionMode),
			Options:       options,
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

func (s *bridgeService) sessionPushAdapter() sessionPushAdapter {
	if s == nil {
		return newSessionPushAdapter(nil, nil)
	}
	return newSessionPushAdapter(s.sessionStore, s.sessionPushHub())
}

func (s *bridgeService) publishAssistantSessionPush(traceID string, result finalizedAgentTurn) {
	s.sessionPushAdapter().publishAssistant(traceID, result)
}

func (s *bridgeService) publishAwaitingHumanSessionPush(traceID string, sessionID string, awaitingErr *agent.ErrAwaitingHuman) {
	s.sessionPushAdapter().publishAwaitingHuman(traceID, sessionID, awaitingErr)
}

func (s *bridgeService) pendingQuestionSnapshot(sessionID string) (sessionPushEvent, bool) {
	return s.sessionPushAdapter().pendingQuestionSnapshot(sessionID)
}
