package orchestration

import "strings"

func (h *sessionPushHub) SubscriberCount(sessionID string) int {
	if h == nil {
		return 0
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.subscribers[strings.TrimSpace(sessionID)])
}
