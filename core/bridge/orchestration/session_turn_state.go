package orchestration

import (
	"context"
	"errors"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

type sessionTurnSetupError struct {
	sessionID  string
	statusCode int
	err        error
}

func (e *sessionTurnSetupError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *sessionTurnSetupError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

type sessionTurnState struct {
	sessionStore    *session.Store
	deps            agentRuntimeDependencies
	persistence     *SessionTurnCommitter
	sess            *session.Session
	agent           *agent.Agent
	execCtx         context.Context
	traceID         string
	userMessage     string
	preTurnMessages []llm.Message
	turnStartedAt   time.Time
	cleanup         func()
}

func (s *sessionTurnState) close() {
	if s == nil {
		return
	}
	if s.cleanup != nil {
		s.cleanup()
	}
	s.deps.Close()
}

func (s *sessionTurnState) currentSessionID() string {
	if s == nil || s.sess == nil {
		return ""
	}
	return strings.TrimSpace(s.sess.ID)
}

func (s *sessionTurnState) persistedSessionID() string {
	if s == nil || s.sessionStore == nil {
		return ""
	}
	return s.currentSessionID()
}

func (s *sessionTurnState) persistNewMessages(newMessages []llm.Message, completed bool) error {
	if s == nil || s.persistence == nil || s.sess == nil || s.agent == nil {
		return nil
	}
	newMessages = s.messagesForPersistence(newMessages, completed)
	s.sess.ConversationState = s.agent.GetConversationState()
	return s.persistence.CommitTurn(s.execCtx, s.sess, newMessages, s.traceID, completed)
}

func (s *sessionTurnState) complete(
	response string,
	runErr error,
	onPersistErr func(err error, awaitingHuman bool) error,
) (string, string, error) {
	awaitingHuman, err := s.resolveAwaitingHumanState(runErr)
	if err != nil {
		return "", "", err
	}

	newMessages := s.newMessagesForCommit()
	s.clearAssistantDraftBeforeCommit(newMessages, runErr, awaitingHuman)
	if err := s.persistTurnCompletion(newMessages, awaitingHuman, onPersistErr); err != nil {
		return "", "", err
	}
	s.resetCommittedMessages()
	return s.finalizeCompletedTurn(response, runErr, awaitingHuman)
}

func (s *sessionTurnState) resolveAwaitingHumanState(runErr error) (bool, error) {
	if runErr == nil {
		return false, nil
	}
	var awaitingErr *agent.ErrAwaitingHuman
	if errors.As(runErr, &awaitingErr) {
		return true, nil
	}
	if !s.hasTurnStateToPersist() {
		return false, runErr
	}
	return false, nil
}

func (s *sessionTurnState) newMessagesForCommit() []llm.Message {
	if s == nil || s.agent == nil {
		return nil
	}
	return s.agent.GetNewMessages()
}

func (s *sessionTurnState) persistTurnCompletion(
	newMessages []llm.Message,
	awaitingHuman bool,
	onPersistErr func(err error, awaitingHuman bool) error,
) error {
	saveErr := s.persistNewMessages(newMessages, !awaitingHuman)
	if saveErr == nil {
		return nil
	}
	if onPersistErr != nil {
		if emitErr := onPersistErr(saveErr, awaitingHuman); emitErr != nil {
			return emitErr
		}
	}
	return saveErr
}

func (s *sessionTurnState) resetCommittedMessages() {
	if s != nil && s.agent != nil {
		s.agent.ResetNewMessages()
	}
}

func (s *sessionTurnState) finalizeCompletedTurn(response string, runErr error, awaitingHuman bool) (string, string, error) {
	sessionID := s.persistedSessionID()
	if awaitingHuman {
		return "", sessionID, runErr
	}
	if runErr != nil {
		return "", sessionID, runErr
	}
	return response, sessionID, nil
}

func (s *sessionTurnState) hasCommittedMessages() bool {
	if s == nil || s.agent == nil {
		return false
	}
	return len(s.agent.GetNewMessages()) > 0
}

func (s *sessionTurnState) hasTurnStateToPersist() bool {
	return s.hasCommittedMessages() || s.hasCurrentAssistantDraft()
}

func (s *sessionTurnState) hasCurrentAssistantDraft() bool {
	if s == nil || s.sess == nil || s.sess.AssistantDraft == nil {
		return false
	}
	return strings.TrimSpace(s.sess.AssistantDraft.TraceID) == strings.TrimSpace(s.traceID)
}

func (s *sessionTurnState) clearAssistantDraftBeforeCommit(
	newMessages []llm.Message,
	runErr error,
	awaitingHuman bool,
) {
	if s == nil || s.sess == nil || (!s.hasCurrentAssistantDraft() && len(newMessages) == 0) {
		return
	}
	if runErr != nil && !awaitingHuman {
		return
	}
	s.sess.ClearAssistantDraft(time.Now().UTC())
}

func (s *sessionTurnState) messagesForPersistence(
	newMessages []llm.Message,
	completed bool,
) []llm.Message {
	if s == nil {
		return newMessages
	}
	if len(newMessages) > 0 {
		return newMessages
	}
	if !s.hasCurrentAssistantDraft() {
		return newMessages
	}
	userMessage, ok := normalizePersistenceUserMessage(llm.Message{
		Role: llm.RoleUser,
		Text: s.userMessage,
	})
	if !ok {
		return newMessages
	}
	return []llm.Message{userMessage}
}

func normalizePersistenceUserMessage(message llm.Message) (llm.Message, bool) {
	cloned := llm.CloneMessages([]llm.Message{message})
	if len(cloned) == 0 {
		return llm.Message{}, false
	}
	cloned[0].Role = llm.RoleUser
	cloned[0].Text = strings.TrimSpace(cloned[0].Text)
	if cloned[0].Text == "" && len(cloned[0].Content) == 0 {
		return llm.Message{}, false
	}
	return cloned[0], true
}
