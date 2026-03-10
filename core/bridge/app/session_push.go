package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

type sessionPushEventType string

const (
	sessionPushAssistantMessage sessionPushEventType = "assistant_message"
	sessionPushAwaitingHuman    sessionPushEventType = "awaiting_human"
	sessionPushRunStarted       sessionPushEventType = sessionPushEventType(streaming.EventRunStarted)
	sessionPushCompletionDelta  sessionPushEventType = sessionPushEventType(streaming.EventCompletionDelta)
	sessionPushToolCallStarted  sessionPushEventType = sessionPushEventType(streaming.EventToolCallStarted)
	sessionPushToolCallFinished sessionPushEventType = sessionPushEventType(streaming.EventToolCallFinished)
	sessionPushError            sessionPushEventType = sessionPushEventType(streaming.EventError)
	sessionPushDone             sessionPushEventType = sessionPushEventType(streaming.EventDone)
)

type sessionPushEvent struct {
	ID        string               `json:"id"`
	Type      sessionPushEventType `json:"type"`
	TraceID   string               `json:"trace_id,omitempty"`
	SessionID string               `json:"session_id"`
	Payload   any                  `json:"payload"`
	At        time.Time            `json:"at"`
}

type assistantMessagePushPayload struct {
	Message      string                            `json:"message"`
	SessionEnded bool                              `json:"session_ended"`
	SessionEnd   *assistantSessionEndSignalPayload `json:"session_end,omitempty"`
}

type awaitingHumanPushPayload struct {
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []askHumanOption `json:"options,omitempty"`
}

// sessionPushHub 把同一 session 的完成消息和 ask_human 事件广播给长连接订阅者。
type sessionPushHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan sessionPushEvent]struct{}
	sequence    uint64
}

func newSessionPushHub() *sessionPushHub {
	return &sessionPushHub{subscribers: make(map[string]map[chan sessionPushEvent]struct{})}
}

func (h *sessionPushHub) Close() {
	if h == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for sessionID, subscribers := range h.subscribers {
		for ch := range subscribers {
			close(ch)
		}
		delete(h.subscribers, sessionID)
	}
}

func (h *sessionPushHub) Subscribe(sessionID string) (<-chan sessionPushEvent, func()) {
	if h == nil {
		closed := make(chan sessionPushEvent)
		close(closed)
		return closed, func() {}
	}

	sessionID = strings.TrimSpace(sessionID)
	ch := make(chan sessionPushEvent, 16)

	h.mu.Lock()
	if h.subscribers[sessionID] == nil {
		h.subscribers[sessionID] = make(map[chan sessionPushEvent]struct{})
	}
	h.subscribers[sessionID][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()

			subscribers, ok := h.subscribers[sessionID]
			if !ok {
				return
			}
			if _, exists := subscribers[ch]; !exists {
				return
			}
			delete(subscribers, ch)
			close(ch)
			if len(subscribers) == 0 {
				delete(h.subscribers, sessionID)
			}
		})
	}
}

func (h *sessionPushHub) Publish(event sessionPushEvent) {
	if h == nil {
		return
	}

	sessionID := strings.TrimSpace(event.SessionID)
	if sessionID == "" {
		return
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	if strings.TrimSpace(event.ID) == "" {
		event.ID = h.nextEventID(sessionID)
	}

	h.mu.RLock()
	subscribers := h.subscribers[sessionID]
	chans := make([]chan sessionPushEvent, 0, len(subscribers))
	for ch := range subscribers {
		chans = append(chans, ch)
	}
	h.mu.RUnlock()

	for _, ch := range chans {
		select {
		case ch <- event:
		default:
		}
	}
}

func (h *sessionPushHub) nextEventID(sessionID string) string {
	seq := atomic.AddUint64(&h.sequence, 1)
	return fmt.Sprintf("%s:%06d", strings.TrimSpace(sessionID), seq)
}

type sessionStreamBroadcastSink struct {
	sink streaming.Sink
	hub  *sessionPushHub
	mu   sync.Mutex
}

func newSessionStreamBroadcastSink(sink streaming.Sink, hub *sessionPushHub) streaming.Sink {
	if hub == nil {
		return ensureEventSink(sink)
	}
	return &sessionStreamBroadcastSink{
		sink: ensureEventSink(sink),
		hub:  hub,
	}
}

func (s *sessionStreamBroadcastSink) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	event, err := s.sink.Emit(ctx, event)
	if err != nil {
		return event, err
	}
	s.publish(event)
	return event, nil
}

func (s *sessionStreamBroadcastSink) publish(event streaming.Event) {
	if s == nil || s.hub == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pushEvent, ok := newSessionPushEventFromAgentEvent(event)
	if !ok {
		return
	}
	s.hub.Publish(pushEvent)
}

func newSessionPushEventFromAgentEvent(event streaming.Event) (sessionPushEvent, bool) {
	trimmedSessionID := strings.TrimSpace(event.SessionID)
	if trimmedSessionID == "" {
		return sessionPushEvent{}, false
	}

	var eventType sessionPushEventType
	switch event.Type {
	case streaming.EventRunStarted:
		eventType = sessionPushRunStarted
	case streaming.EventCompletionDelta:
		eventType = sessionPushCompletionDelta
	case streaming.EventToolCallStarted:
		eventType = sessionPushToolCallStarted
	case streaming.EventToolCallFinished:
		eventType = sessionPushToolCallFinished
	case streaming.EventError:
		eventType = sessionPushError
	case streaming.EventDone:
		eventType = sessionPushDone
	default:
		return sessionPushEvent{}, false
	}

	return sessionPushEvent{
		ID:        strings.TrimSpace(event.ID),
		Type:      eventType,
		TraceID:   strings.TrimSpace(event.TraceID),
		SessionID: trimmedSessionID,
		Payload:   event.Payload,
		At:        event.At,
	}, true
}

func (s *bridgeService) publishAssistantSessionPush(traceID string, result finalizedAgentTurn) {
	if s == nil || s.sessionPush == nil || strings.TrimSpace(result.sessionID) == "" {
		return
	}

	s.sessionPush.Publish(sessionPushEvent{
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

func (s *bridgeService) publishAwaitingHumanSessionPush(traceID string, sessionID string, awaitingErr *agent.ErrAwaitingHuman) {
	if s == nil || s.sessionPush == nil || awaitingErr == nil {
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

	s.sessionPush.Publish(sessionPushEvent{
		Type:      sessionPushAwaitingHuman,
		TraceID:   strings.TrimSpace(traceID),
		SessionID: trimmedSessionID,
		Payload:   payload,
	})
}

func (t *transport) handleSessionEvents(w http.ResponseWriter, r *http.Request, sessionID string) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported", "")
		return
	}

	traceID := resolveTraceID("", r)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Trace-ID", traceID)

	if event, ok := t.service.pendingQuestionSnapshot(sessionID); ok {
		if err := writeSessionPushEvent(w, flusher, event); err != nil {
			return
		}
	}

	ch, unsubscribe := t.service.sessionPush.Subscribe(sessionID)
	defer unsubscribe()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := writeSessionPushEvent(w, flusher, event); err != nil {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSessionPushEvent(w http.ResponseWriter, flusher http.Flusher, event sessionPushEvent) error {
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func (s *bridgeService) pendingQuestionSnapshot(sessionID string) (sessionPushEvent, bool) {
	if s == nil || s.sessionStore == nil {
		return sessionPushEvent{}, false
	}

	sess, err := s.sessionStore.Load(strings.TrimSpace(sessionID))
	if err != nil || len(sess.PendingQuestions) == 0 {
		return sessionPushEvent{}, false
	}

	questionID, question, ok := oldestPendingQuestion(sess)
	if !ok {
		return sessionPushEvent{}, false
	}

	payload := awaitingHumanPushPayload{
		QuestionID:    questionID,
		Prompt:        question.Prompt,
		SelectionMode: strings.TrimSpace(question.SelectionMode),
	}
	if len(question.Options) > 0 {
		payload.Options = make([]askHumanOption, 0, len(question.Options))
		for _, option := range question.Options {
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

	return sessionPushEvent{
		ID:        fmt.Sprintf("%s:pending", strings.TrimSpace(sessionID)),
		Type:      sessionPushAwaitingHuman,
		SessionID: strings.TrimSpace(sessionID),
		TraceID:   strings.TrimSpace(question.TraceID),
		Payload:   payload,
		At:        question.CreatedAt,
	}, true
}

func oldestPendingQuestion(sess *session.Session) (string, session.PendingHumanQuestion, bool) {
	if sess == nil || len(sess.PendingQuestions) == 0 {
		return "", session.PendingHumanQuestion{}, false
	}

	type pendingQuestion struct {
		id       string
		question session.PendingHumanQuestion
	}

	questions := make([]pendingQuestion, 0, len(sess.PendingQuestions))
	for id, question := range sess.PendingQuestions {
		questions = append(questions, pendingQuestion{id: strings.TrimSpace(id), question: question})
	}
	sort.Slice(questions, func(i, j int) bool {
		return questions[i].question.CreatedAt.Before(questions[j].question.CreatedAt)
	})
	if len(questions) == 0 || questions[0].id == "" {
		return "", session.PendingHumanQuestion{}, false
	}
	return questions[0].id, questions[0].question, true
}
