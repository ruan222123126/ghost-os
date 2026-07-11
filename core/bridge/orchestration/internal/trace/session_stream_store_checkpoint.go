package trace

import (
	"context"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

type sessionDraftStoreCheckpointSink struct {
	sink            streaming.Sink
	sessionStore    *session.Store
	sessionID       string
	currentTime     func() time.Time
	lastPersistedAt time.Time

	mu    sync.Mutex
	state *session.Session
	dirty bool
}

func NewSessionDraftStoreCheckpointSink(
	sink streaming.Sink,
	sessionStore *session.Store,
	sessionID string,
) streaming.Sink {
	trimmedSessionID := strings.TrimSpace(sessionID)
	if sessionStore == nil || trimmedSessionID == "" {
		return EnsureEventSink(sink)
	}
	return &sessionDraftStoreCheckpointSink{
		sink:         EnsureEventSink(sink),
		sessionStore: sessionStore,
		sessionID:    trimmedSessionID,
		currentTime:  time.Now,
	}
}

func (s *sessionDraftStoreCheckpointSink) Emit(
	ctx context.Context,
	event streaming.Event,
) (streaming.Event, error) {
	if err := s.persistDraftBeforeEvent(event); err != nil {
		return event, err
	}
	return s.sink.Emit(ctx, event)
}

func (s *sessionDraftStoreCheckpointSink) persistDraftBeforeEvent(
	event streaming.Event,
) error {
	if strings.TrimSpace(event.SessionID) != s.sessionID {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureState(); err != nil {
		return err
	}

	switch event.Type {
	case streaming.EventRunStarted,
		streaming.EventCompletionDelta,
		streaming.EventToolCallStarted,
		streaming.EventToolCallFinished:
		return s.persistTurnDraftEvent(event)
	case streaming.EventAwaitingHuman,
		streaming.EventError,
		streaming.EventDone:
		return s.persistTerminalTurnDraftEvent(event)
	default:
		return nil
	}
}

func (s *sessionDraftStoreCheckpointSink) persistTurnDraftEvent(
	event streaming.Event,
) error {
	now := s.nowUTC()
	if ProjectSessionRunState(s.state, event, now) {
		s.dirty = true
	}
	if ProjectTurnDraft(s.state, event, now) {
		s.dirty = true
	}
	if text, ok := completionDeltaText(event.Payload); ok && s.state.AppendAssistantDraft(text, event.TraceID, event.Turn, now) {
		s.dirty = true
	}
	return s.saveDraftIfNeeded(false)
}

func (s *sessionDraftStoreCheckpointSink) persistTerminalTurnDraftEvent(
	event streaming.Event,
) error {
	now := s.nowUTC()
	if ProjectSessionRunState(s.state, event, now) {
		s.dirty = true
	}
	if ProjectTurnDraft(s.state, event, now) {
		s.dirty = true
	}
	if event.Type == streaming.EventDone && s.state.ClearAssistantDraft(s.nowUTC()) {
		s.dirty = true
	}
	return s.saveDraftIfNeeded(true)
}

func (s *sessionDraftStoreCheckpointSink) ensureState() error {
	if s.state != nil {
		return nil
	}

	loaded, err := s.sessionStore.Load(s.sessionID)
	if err != nil {
		return err
	}
	s.state = &session.Session{
		ID:             loaded.ID,
		AssistantDraft: loaded.AssistantDraft,
		TurnDraft:      loaded.TurnDraft,
		LastRunState:   loaded.LastRunState,
	}
	return nil
}

func (s *sessionDraftStoreCheckpointSink) saveDraftIfNeeded(force bool) error {
	if !s.dirty {
		return nil
	}
	if !force && !s.shouldSaveNow() {
		return nil
	}

	latest, err := s.sessionStore.Load(s.sessionID)
	if err != nil {
		return err
	}
	latest.AssistantDraft = s.state.AssistantDraft
	latest.TurnDraft = s.state.TurnDraft
	latest.LastRunState = s.state.LastRunState
	if err := s.sessionStore.Save(latest); err != nil {
		return err
	}
	s.lastPersistedAt = s.nowUTC()
	s.dirty = false
	return nil
}

func (s *sessionDraftStoreCheckpointSink) shouldSaveNow() bool {
	if s.lastPersistedAt.IsZero() {
		return true
	}
	return s.nowUTC().Sub(s.lastPersistedAt) >= sessionDraftCheckpointMinInterval
}

func (s *sessionDraftStoreCheckpointSink) nowUTC() time.Time {
	return s.currentTime().UTC()
}
