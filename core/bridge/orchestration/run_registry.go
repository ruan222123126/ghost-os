package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrSessionInflight = errors.New("session is already running")
	ErrRunNotFound     = errors.New("run not found")
	ErrRunCancelled    = errors.New("agent run cancelled")
	ErrRunRegistryNil  = errors.New("run registry is nil")
)

type RunHandle struct {
	SessionID string
	TraceID   string
	Cancel    context.CancelFunc
	StartedAt time.Time
	done      chan struct{}
	doneOnce  sync.Once
}

type RunRegistry struct {
	mu          sync.RWMutex
	bySessionID map[string]*RunHandle
	byTraceID   map[string]*RunHandle
}

func NewRunRegistry() *RunRegistry {
	return &RunRegistry{
		bySessionID: make(map[string]*RunHandle),
		byTraceID:   make(map[string]*RunHandle),
	}
}

func (r *RunRegistry) Register(sessionID string, traceID string, cancel context.CancelFunc) error {
	if err := validateRegisterInputs(r, cancel); err != nil {
		return err
	}
	trimmedSessionID, trimmedTraceID, err := normalizeRunHandleIDs(sessionID, traceID)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureRegisterIDsAvailableLocked(trimmedSessionID, trimmedTraceID); err != nil {
		return err
	}
	r.registerHandleLocked(newRunHandle(trimmedSessionID, trimmedTraceID, cancel))
	return nil
}

func validateRegisterInputs(r *RunRegistry, cancel context.CancelFunc) error {
	if r == nil {
		return ErrRunRegistryNil
	}
	if cancel == nil {
		return errors.New("cancel func is required")
	}
	return nil
}

func normalizeRunHandleIDs(sessionID string, traceID string) (string, string, error) {
	trimmedSessionID := strings.TrimSpace(sessionID)
	trimmedTraceID := strings.TrimSpace(traceID)
	if trimmedSessionID == "" && trimmedTraceID == "" {
		return "", "", errors.New("session_id or trace_id is required")
	}
	return trimmedSessionID, trimmedTraceID, nil
}

func (r *RunRegistry) ensureRegisterIDsAvailableLocked(sessionID string, traceID string) error {
	if sessionID != "" {
		if _, exists := r.bySessionID[sessionID]; exists {
			return fmt.Errorf("%w: session_id=%s", ErrSessionInflight, sessionID)
		}
	}
	if traceID != "" {
		if existing, exists := r.byTraceID[traceID]; exists && existing != nil {
			return fmt.Errorf("%w: trace_id=%s", ErrSessionInflight, traceID)
		}
	}
	return nil
}

func (r *RunRegistry) registerHandleLocked(handle *RunHandle) {
	if handle == nil {
		return
	}
	if handle.SessionID != "" {
		r.bySessionID[handle.SessionID] = handle
	}
	if handle.TraceID != "" {
		r.byTraceID[handle.TraceID] = handle
	}
}

func newRunHandle(sessionID string, traceID string, cancel context.CancelFunc) *RunHandle {
	return &RunHandle{
		SessionID: sessionID,
		TraceID:   traceID,
		Cancel:    cancel,
		StartedAt: time.Now().UTC(),
		done:      make(chan struct{}),
	}
}

func (r *RunRegistry) Unregister(sessionID string) {
	if r == nil {
		return
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.unregisterBySessionIDLocked(trimmedSessionID)
}

func (r *RunRegistry) CancelBySessionID(sessionID string) error {
	return cancelRunHandle(r.lookupBySessionID, sessionID)
}

func (r *RunRegistry) CancelByTraceID(traceID string) error {
	return cancelRunHandle(r.lookupByTraceID, traceID)
}

func (r *RunRegistry) IsInflight(sessionID string) bool {
	if r == nil {
		return false
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.bySessionID[trimmedSessionID]
	return exists
}

func (r *RunRegistry) GetBySessionID(sessionID string) *RunHandle {
	if r == nil {
		return nil
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneRunHandle(r.bySessionID[trimmedSessionID])
}

func (r *RunRegistry) GetByTraceID(traceID string) *RunHandle {
	if r == nil {
		return nil
	}

	trimmedTraceID := strings.TrimSpace(traceID)
	if trimmedTraceID == "" {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneRunHandle(r.byTraceID[trimmedTraceID])
}

func (r *RunRegistry) Count() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	seen := make(map[*RunHandle]struct{}, len(r.bySessionID)+len(r.byTraceID))
	for _, handle := range r.bySessionID {
		if handle != nil {
			seen[handle] = struct{}{}
		}
	}
	for _, handle := range r.byTraceID {
		if handle != nil {
			seen[handle] = struct{}{}
		}
	}
	return len(seen)
}

func (r *RunRegistry) unregisterBySessionIDLocked(sessionID string) {
	handle, ok := r.bySessionID[sessionID]
	if !ok || handle == nil {
		return
	}
	r.unregisterHandleLocked(handle)
}

func (r *RunRegistry) unregisterHandleLocked(handle *RunHandle) {
	if handle == nil {
		return
	}
	if handle.SessionID != "" {
		delete(r.bySessionID, handle.SessionID)
	}
	if handle.TraceID != "" {
		delete(r.byTraceID, handle.TraceID)
	}
	handle.markDone()
}

func cloneRunHandle(handle *RunHandle) *RunHandle {
	if handle == nil {
		return nil
	}
	return &RunHandle{
		SessionID: handle.SessionID,
		TraceID:   handle.TraceID,
		StartedAt: handle.StartedAt,
	}
}

func (h *RunHandle) markDone() {
	if h == nil {
		return
	}
	h.doneOnce.Do(func() {
		close(h.done)
	})
}
