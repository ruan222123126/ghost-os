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
	SessionPushAssistantMessage SessionPushEventType = "assistant_message"
	SessionPushAwaitingHuman    SessionPushEventType = "awaiting_human"
	SessionPushRunStarted       SessionPushEventType = SessionPushEventType(streaming.EventRunStarted)
	SessionPushCompletionDelta  SessionPushEventType = SessionPushEventType(streaming.EventCompletionDelta)
	SessionPushToolCallStarted  SessionPushEventType = SessionPushEventType(streaming.EventToolCallStarted)
	SessionPushToolCallFinished SessionPushEventType = SessionPushEventType(streaming.EventToolCallFinished)
	SessionPushError            SessionPushEventType = SessionPushEventType(streaming.EventError)
	SessionPushDone             SessionPushEventType = SessionPushEventType(streaming.EventDone)
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
	mu          sync.RWMutex
	subscribers map[string]map[chan SessionPushEvent]struct{}
	sequence    uint64
}

func NewSessionPushHub() *SessionPushHub {
	return &SessionPushHub{subscribers: make(map[string]map[chan SessionPushEvent]struct{})}
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
	if h.subscribers[sessionID] == nil {
		h.subscribers[sessionID] = make(map[chan SessionPushEvent]struct{})
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

func (h *SessionPushHub) Publish(event SessionPushEvent) {
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
	chans := make([]chan SessionPushEvent, 0, len(subscribers))
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

func (h *SessionPushHub) nextEventID(sessionID string) string {
	seq := atomic.AddUint64(&h.sequence, 1)
	return fmt.Sprintf("%s:%06d", strings.TrimSpace(sessionID), seq)
}
