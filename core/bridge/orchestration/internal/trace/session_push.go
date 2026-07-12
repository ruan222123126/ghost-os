package trace

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ghost-os/bridge/streaming"
)

type SessionPushEventType string

const (
	SessionPushAssistantMessage    SessionPushEventType = "assistant_message"
	SessionPushAwaitingHuman       SessionPushEventType = "awaiting_human"
	SessionPushRunStarted          SessionPushEventType = SessionPushEventType(streaming.EventRunStarted)
	SessionPushCompletionDelta     SessionPushEventType = SessionPushEventType(streaming.EventCompletionDelta)
	SessionPushToolCallStarted     SessionPushEventType = SessionPushEventType(streaming.EventToolCallStarted)
	SessionPushToolCallFinished    SessionPushEventType = SessionPushEventType(streaming.EventToolCallFinished)
	SessionPushError               SessionPushEventType = SessionPushEventType(streaming.EventError)
	SessionPushDone                SessionPushEventType = SessionPushEventType(streaming.EventDone)
	SessionPushTaskRunCardStarted  SessionPushEventType = "task_run_card_started"
	SessionPushTaskRunCardEvent    SessionPushEventType = "task_run_card_event"
	SessionPushTaskRunCardFinished SessionPushEventType = "task_run_card_finished"
)

type SessionPushEvent struct {
	ID        string               `json:"id"`
	Type      SessionPushEventType `json:"type"`
	TraceID   string               `json:"trace_id,omitempty"`
	SessionID string               `json:"session_id"`
	Payload   any                  `json:"payload"`
	At        time.Time            `json:"at"`
}

// SessionPushHub 把同一 session 的完成消息和 ask_human 事件广播给长连接订阅者。
type SessionPushHub struct {
	mu             sync.RWMutex
	subscribers    map[string]map[chan SessionPushEvent]struct{}
	replaysByTrace map[string]sessionPushReplay
	traceBySession map[string]string
	sequence       uint64
}

func NewSessionPushHub() *SessionPushHub {
	return &SessionPushHub{
		subscribers:    make(map[string]map[chan SessionPushEvent]struct{}),
		replaysByTrace: make(map[string]sessionPushReplay),
		traceBySession: make(map[string]string),
	}
}

type sessionPushReplay struct {
	sessionID string
	events    []SessionPushEvent
}

func (h *SessionPushHub) Close() {
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
	clear(h.replaysByTrace)
	clear(h.traceBySession)
}

func (h *SessionPushHub) Subscribe(sessionID string) (<-chan SessionPushEvent, func()) {
	if h == nil {
		closed := make(chan SessionPushEvent)
		close(closed)
		return closed, func() {}
	}

	sessionID = strings.TrimSpace(sessionID)
	ch := make(chan SessionPushEvent, 16)

	h.mu.Lock()
	h.subscribeLocked(sessionID, ch)
	h.mu.Unlock()

	return ch, h.unsubscribeFunc(sessionID, ch)
}

// SubscribeTrace atomically subscribes to the session owning traceID and returns
// events emitted after lastEventID. This closes the replay/live-event race during reconnects.
func (h *SessionPushHub) SubscribeTrace(
	traceID string,
	lastEventID string,
) (string, []SessionPushEvent, <-chan SessionPushEvent, func(), bool) {
	if h == nil {
		return "", nil, nil, func() {}, false
	}

	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return "", nil, nil, func() {}, false
	}

	h.mu.Lock()
	replay, ok := h.replaysByTrace[traceID]
	if !ok || replay.sessionID == "" {
		h.mu.Unlock()
		return "", nil, nil, func() {}, false
	}
	ch := make(chan SessionPushEvent, 16)
	h.subscribeLocked(replay.sessionID, ch)
	events := replayEventsAfter(replay.events, lastEventID)
	h.mu.Unlock()

	return replay.sessionID, events, ch, h.unsubscribeFunc(replay.sessionID, ch), true
}

func (h *SessionPushHub) ReplayTraceAfter(traceID string, lastEventID string) []SessionPushEvent {
	if h == nil {
		return nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	replay, ok := h.replaysByTrace[strings.TrimSpace(traceID)]
	if !ok {
		return nil
	}
	return replayEventsAfter(replay.events, lastEventID)
}

func (h *SessionPushHub) subscribeLocked(sessionID string, ch chan SessionPushEvent) {
	if h.subscribers[sessionID] == nil {
		h.subscribers[sessionID] = make(map[chan SessionPushEvent]struct{})
	}
	h.subscribers[sessionID][ch] = struct{}{}
}

func (h *SessionPushHub) unsubscribeFunc(sessionID string, ch chan SessionPushEvent) func() {
	var once sync.Once
	return func() {
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

func (h *SessionPushHub) Publish(event SessionPushEvent) {
	if h == nil {
		return
	}

	sessionID := strings.TrimSpace(event.SessionID)
	if sessionID == "" {
		return
	}
	event.SessionID = sessionID
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	if strings.TrimSpace(event.ID) == "" {
		event.ID = h.nextEventID(sessionID)
	}

	h.mu.Lock()
	h.recordReplayLocked(event)
	subscribers := h.subscribers[sessionID]
	chans := make([]chan SessionPushEvent, 0, len(subscribers))
	for ch := range subscribers {
		chans = append(chans, ch)
	}
	h.mu.Unlock()

	for _, ch := range chans {
		select {
		case ch <- event:
		default:
		}
	}
}

func (h *SessionPushHub) recordReplayLocked(event SessionPushEvent) {
	traceID := strings.TrimSpace(event.TraceID)
	if traceID == "" {
		return
	}

	previousTraceID := h.traceBySession[event.SessionID]
	if previousTraceID != traceID {
		delete(h.replaysByTrace, previousTraceID)
		h.traceBySession[event.SessionID] = traceID
		h.replaysByTrace[traceID] = sessionPushReplay{sessionID: event.SessionID}
	}

	replay := h.replaysByTrace[traceID]
	replay.sessionID = event.SessionID
	replay.events = append(replay.events, event)
	h.replaysByTrace[traceID] = replay
}

func replayEventsAfter(events []SessionPushEvent, lastEventID string) []SessionPushEvent {
	lastEventID = strings.TrimSpace(lastEventID)
	start := 0
	if lastEventID != "" {
		for index, event := range events {
			if event.ID == lastEventID {
				start = index + 1
				break
			}
		}
	}
	return append([]SessionPushEvent(nil), events[start:]...)
}

func (h *SessionPushHub) nextEventID(sessionID string) string {
	seq := atomic.AddUint64(&h.sequence, 1)
	return fmt.Sprintf("%s:%06d", strings.TrimSpace(sessionID), seq)
}
