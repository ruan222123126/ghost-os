package turnstate

import (
	"context"
	"errors"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

type Persistence interface {
	CommitTurn(ctx context.Context, sess *session.Session, messages []llm.Message, traceID string, completed bool) error
}

type CompleteRequest struct {
	Context          context.Context
	Session          *session.Session
	Agent            *agent.Agent
	Persistence      Persistence
	SessionPersisted bool
	TraceID          string
	UserMessage      string
	Response         string
	RunErr           error
	OnPersistErr     func(err error, awaitingHuman bool) error
}

type State struct {
	SessionStore    *session.Store
	RuntimeCleanup  func()
	Persistence     Persistence
	Session         *session.Session
	Agent           *agent.Agent
	ExecCtx         context.Context
	TraceID         string
	UserMessage     string
	PreTurnMessages []llm.Message
	TurnStartedAt   time.Time
	Cleanup         func()
}

func (s *State) Close() {
	if s == nil {
		return
	}
	if s.Cleanup != nil {
		s.Cleanup()
	}
	if s.RuntimeCleanup != nil {
		s.RuntimeCleanup()
	}
}

func (s *State) CurrentSessionID() string {
	if s == nil || s.Session == nil {
		return ""
	}
	return strings.TrimSpace(s.Session.ID)
}

func (s *State) PersistedSessionID() string {
	if s == nil || s.SessionStore == nil {
		return ""
	}
	return s.CurrentSessionID()
}

func (s *State) Complete(
	response string,
	runErr error,
	onPersistErr func(err error, awaitingHuman bool) error,
) (string, string, error) {
	req := CompleteRequest{
		Response:     response,
		RunErr:       runErr,
		OnPersistErr: onPersistErr,
	}
	if s != nil {
		req.Context = s.ExecCtx
		req.Session = s.Session
		req.Agent = s.Agent
		req.Persistence = s.Persistence
		req.SessionPersisted = s.SessionStore != nil
		req.TraceID = s.TraceID
		req.UserMessage = s.UserMessage
	}
	return Complete(req)
}

func Complete(req CompleteRequest) (string, string, error) {
	awaitingHuman, err := resolveAwaitingHumanState(req)
	if err != nil {
		return "", "", err
	}
	newMessages := newMessagesForCommit(req.Agent)
	clearAssistantDraftBeforeCommit(req.Session, req.TraceID, newMessages, req.RunErr, awaitingHuman)
	clearTurnDraftBeforeCommit(req.Session, req.RunErr)
	if err := persistTurnCompletion(req, newMessages, awaitingHuman); err != nil {
		return "", "", err
	}
	resetCommittedMessages(req.Agent)
	return finalizeCompletedTurn(req.Response, req.RunErr, awaitingHuman, persistedSessionID(req))
}

func resolveAwaitingHumanState(req CompleteRequest) (bool, error) {
	if req.RunErr == nil {
		return false, nil
	}
	var awaitingErr *agent.ErrAwaitingHuman
	if errors.As(req.RunErr, &awaitingErr) {
		return true, nil
	}
	if !hasTurnStateToPersist(req) {
		return false, req.RunErr
	}
	return false, nil
}

func hasTurnStateToPersist(req CompleteRequest) bool {
	return len(newMessagesForCommit(req.Agent)) > 0 || hasCurrentAssistantDraft(req.Session, req.TraceID)
}

func newMessagesForCommit(runAgent *agent.Agent) []llm.Message {
	if runAgent == nil {
		return nil
	}
	return runAgent.GetNewMessages()
}

func persistTurnCompletion(
	req CompleteRequest,
	newMessages []llm.Message,
	awaitingHuman bool,
) error {
	saveErr := persistNewMessages(req, newMessages, !awaitingHuman)
	if saveErr == nil {
		return nil
	}
	if req.OnPersistErr != nil {
		if emitErr := req.OnPersistErr(saveErr, awaitingHuman); emitErr != nil {
			return emitErr
		}
	}
	return saveErr
}

func persistNewMessages(
	req CompleteRequest,
	newMessages []llm.Message,
	completed bool,
) error {
	if req.Persistence == nil || req.Session == nil || req.Agent == nil {
		return nil
	}
	newMessages = messagesForPersistence(newMessages, req.Session, req.TraceID, req.UserMessage)
	req.Session.ConversationState = req.Agent.GetConversationState()
	return req.Persistence.CommitTurn(req.Context, req.Session, newMessages, req.TraceID, completed)
}

func messagesForPersistence(
	newMessages []llm.Message,
	sess *session.Session,
	traceID string,
	userMessage string,
) []llm.Message {
	if len(newMessages) > 0 {
		return newMessages
	}
	if !hasCurrentAssistantDraft(sess, traceID) {
		return newMessages
	}
	message, ok := normalizePersistenceUserMessage(llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	})
	if !ok {
		return newMessages
	}
	return []llm.Message{message}
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

func hasCurrentAssistantDraft(sess *session.Session, traceID string) bool {
	if sess == nil || sess.AssistantDraft == nil {
		return false
	}
	return strings.TrimSpace(sess.AssistantDraft.TraceID) == strings.TrimSpace(traceID)
}

func clearAssistantDraftBeforeCommit(
	sess *session.Session,
	traceID string,
	newMessages []llm.Message,
	runErr error,
	awaitingHuman bool,
) {
	if sess == nil || (!hasCurrentAssistantDraft(sess, traceID) && len(newMessages) == 0) {
		return
	}
	if runErr != nil && !awaitingHuman {
		return
	}
	sess.ClearAssistantDraft(time.Now().UTC())
}

func clearTurnDraftBeforeCommit(sess *session.Session, runErr error) {
	if sess == nil || sess.TurnDraft == nil {
		return
	}
	if runErr != nil {
		return
	}
	sess.ClearTurnDraft(time.Now().UTC())
}

func resetCommittedMessages(runAgent *agent.Agent) {
	if runAgent != nil {
		runAgent.ResetNewMessages()
	}
}

func persistedSessionID(req CompleteRequest) string {
	if !req.SessionPersisted || req.Session == nil {
		return ""
	}
	return strings.TrimSpace(req.Session.ID)
}

func finalizeCompletedTurn(response string, runErr error, awaitingHuman bool, sessionID string) (string, string, error) {
	if awaitingHuman {
		return "", sessionID, runErr
	}
	if runErr != nil {
		return "", sessionID, runErr
	}
	return response, sessionID, nil
}

type Committer struct {
	sessionStore *session.Store
}

func NewCommitter(sessionStore *session.Store) *Committer {
	return &Committer{sessionStore: sessionStore}
}

func (c *Committer) CommitTurn(
	ctx context.Context,
	sess *session.Session,
	messages []llm.Message,
	traceID string,
	completed bool,
) error {
	_ = ctx
	_ = traceID
	_ = completed
	if c == nil || c.sessionStore == nil {
		return nil
	}
	for _, msg := range messages {
		sess.AddMessage(msg)
	}
	return c.sessionStore.Save(sess)
}
