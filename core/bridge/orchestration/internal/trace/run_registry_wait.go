package trace

import (
	"context"
	"strings"
)

func (r *RunRegistry) CancelAndWaitBySessionID(ctx context.Context, sessionID string) (*RunHandle, error) {
	handle, err := r.lookupBySessionID(sessionID)
	if err != nil {
		return nil, err
	}
	handle.Cancel()
	if err := waitForRunHandle(ctx, handle); err != nil {
		return nil, err
	}
	return cloneRunHandle(handle), nil
}

func (r *RunRegistry) CancelAndWaitByTraceID(ctx context.Context, traceID string) (*RunHandle, error) {
	handle, err := r.lookupByTraceID(traceID)
	if err != nil {
		return nil, err
	}
	handle.Cancel()
	if err := waitForRunHandle(ctx, handle); err != nil {
		return nil, err
	}
	return cloneRunHandle(handle), nil
}

func (r *RunRegistry) lookupBySessionID(sessionID string) (*RunHandle, error) {
	if r == nil {
		return nil, ErrRunNotFound
	}
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return nil, ErrRunNotFound
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	handle, ok := r.bySessionID[trimmedSessionID]
	if !ok || handle == nil {
		return nil, ErrRunNotFound
	}
	return handle, nil
}

func (r *RunRegistry) lookupByTraceID(traceID string) (*RunHandle, error) {
	if r == nil {
		return nil, ErrRunNotFound
	}
	trimmedTraceID := strings.TrimSpace(traceID)
	if trimmedTraceID == "" {
		return nil, ErrRunNotFound
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	handle, ok := r.byTraceID[trimmedTraceID]
	if !ok || handle == nil {
		return nil, ErrRunNotFound
	}
	return handle, nil
}

func cancelRunHandle(lookup func(string) (*RunHandle, error), id string) error {
	handle, err := lookup(id)
	if err != nil {
		return err
	}
	handle.Cancel()
	return nil
}

func waitForRunHandle(ctx context.Context, handle *RunHandle) error {
	if handle == nil {
		return ErrRunNotFound
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if handle.done == nil {
		return nil
	}
	select {
	case <-handle.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
