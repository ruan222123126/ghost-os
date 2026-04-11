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
	memoryCtx       *turnMemoryContext
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
	s.sess.ConversationState = s.agent.GetConversationState()
	return s.persistence.CommitTurn(s.execCtx, s.sess, newMessages, s.traceID, completed, s.memoryCtx)
}

func (s *sessionTurnState) complete(
	response string,
	runErr error,
	onPersistErr func(err error, awaitingHuman bool) error,
) (string, string, error) {
	awaitingHuman := false
	if runErr != nil {
		var awaitingErr *agent.ErrAwaitingHuman
		if errors.As(runErr, &awaitingErr) {
			awaitingHuman = true
		} else {
			if !s.hasCommittedMessages() {
				return "", "", runErr
			}
		}
	}

	newMessages := []llm.Message(nil)
	if s != nil && s.agent != nil {
		newMessages = s.agent.GetNewMessages()
	}
	s.clearAssistantDraftBeforeCommit(newMessages)

	if saveErr := s.persistNewMessages(newMessages, !awaitingHuman); saveErr != nil {
		if onPersistErr != nil {
			if emitErr := onPersistErr(saveErr, awaitingHuman); emitErr != nil {
				return "", "", emitErr
			}
		}
		return "", "", saveErr
	}
	if s != nil && s.agent != nil {
		s.agent.ResetNewMessages()
	}
	if awaitingHuman {
		return "", s.persistedSessionID(), runErr
	}
	if runErr != nil {
		return "", s.persistedSessionID(), runErr
	}
	return response, s.persistedSessionID(), nil
}

func (s *sessionTurnState) hasCommittedMessages() bool {
	if s == nil || s.agent == nil {
		return false
	}
	return len(s.agent.GetNewMessages()) > 0
}

func (s *sessionTurnState) clearAssistantDraftBeforeCommit(newMessages []llm.Message) {
	if s == nil || s.sess == nil || len(newMessages) == 0 {
		return
	}
	s.sess.ClearAssistantDraft(time.Now().UTC())
}
