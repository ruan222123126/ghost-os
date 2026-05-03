package orchestration

import (
	"context"
	"strings"
	"time"

	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

const sessionDraftCheckpointMinInterval = 300 * time.Millisecond

type sessionDraftCheckpointSink struct {
	sink            streaming.Sink
	sessionStore    *session.Store
	sess            *session.Session
	currentTime     func() time.Time
	lastPersistedAt time.Time
	dirty           bool
}

func newSessionDraftCheckpointSink(
	sink streaming.Sink,
	sessionStore *session.Store,
	sess *session.Session,
) streaming.Sink {
	if sessionStore == nil || sess == nil {
		return sink
	}
	return &sessionDraftCheckpointSink{
		sink:         ensureEventSink(sink),
		sessionStore: sessionStore,
		sess:         sess,
		currentTime:  time.Now,
	}
}

func (s *sessionDraftCheckpointSink) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	if err := s.persistDraftBeforeEvent(ctx, event); err != nil {
		return event, err
	}
	return s.sink.Emit(ctx, event)
}

func (s *sessionDraftCheckpointSink) persistDraftBeforeEvent(ctx context.Context, event streaming.Event) error {
	switch event.Type {
	case streaming.EventRunStarted,
		streaming.EventCompletionDelta,
		streaming.EventToolCallStarted,
		streaming.EventToolCallFinished:
		return s.persistTurnDraftEvent(ctx, event)
	case streaming.EventAwaitingHuman, streaming.EventDone, streaming.EventError:
		return s.persistTerminalTurnDraftEvent(ctx, event)
	default:
		return nil
	}
}

func (s *sessionDraftCheckpointSink) persistTurnDraftEvent(ctx context.Context, event streaming.Event) error {
	draftChanged := projectSessionTurnDraft(s.sess, event, s.nowUTC())
	if draftChanged {
		s.dirty = true
	}

	text, ok := completionDeltaText(event.Payload)
	if ok && s.sess.AppendAssistantDraft(text, event.TraceID, event.Turn, s.nowUTC()) {
		s.dirty = true
	}
	return s.saveDraftIfNeeded(ctx, false)
}

func (s *sessionDraftCheckpointSink) persistTerminalTurnDraftEvent(ctx context.Context, event streaming.Event) error {
	if projectSessionTurnDraft(s.sess, event, s.nowUTC()) {
		s.dirty = true
	}
	return s.saveDraftIfNeeded(ctx, true)
}

func (s *sessionDraftCheckpointSink) saveDraftIfNeeded(ctx context.Context, force bool) error {
	if !s.dirty {
		return nil
	}
	if !force && !s.shouldSaveNow() {
		return nil
	}
	if err := s.sessionStore.Save(s.sess); err != nil {
		return err
	}
	s.lastPersistedAt = s.nowUTC()
	s.dirty = false
	return nil
}

func (s *sessionDraftCheckpointSink) shouldSaveNow() bool {
	if s.lastPersistedAt.IsZero() {
		return true
	}
	return s.nowUTC().Sub(s.lastPersistedAt) >= sessionDraftCheckpointMinInterval
}

func (s *sessionDraftCheckpointSink) nowUTC() time.Time {
	return s.currentTime().UTC()
}

func completionDeltaText(payload any) (string, bool) {
	record, ok := payload.(map[string]any)
	if !ok {
		return "", false
	}
	kind, _ := record["kind"].(string)
	if strings.TrimSpace(kind) != "text" {
		return "", false
	}
	text, _ := record["text"].(string)
	if text == "" {
		return "", false
	}
	return text, true
}
