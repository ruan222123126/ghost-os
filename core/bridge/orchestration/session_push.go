package orchestration

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
