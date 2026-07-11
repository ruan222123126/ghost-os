package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var ErrTaskRunNotRunning = errors.New("task run is not running")

type activeRunRegistry struct {
	mu       sync.RWMutex
	byTaskID map[string]*activeRunHandle
}

type activeRunHandle struct {
	TaskID string
	RunID  string
	cancel context.CancelFunc
	done   chan struct{}

	doneOnce sync.Once
	mu       sync.Mutex
	final    RunLog
	finalSet bool
}

func newActiveRunRegistry() *activeRunRegistry {
	return &activeRunRegistry{
		byTaskID: make(map[string]*activeRunHandle),
	}
}

func (r *activeRunRegistry) register(
	taskID string,
	runID string,
	cancel context.CancelFunc,
) (*activeRunHandle, error) {
	if r == nil {
		return nil, errors.New("active run registry is not configured")
	}
	if cancel == nil {
		return nil, errors.New("task run cancel func is not available")
	}
	normalizedTaskID := strings.TrimSpace(taskID)
	normalizedRunID := strings.TrimSpace(runID)
	if normalizedTaskID == "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidTaskID, taskID)
	}
	if normalizedRunID == "" {
		return nil, fmt.Errorf("%w: run_id is required", ErrInvalidTaskConfig)
	}

	handle := &activeRunHandle{
		TaskID: normalizedTaskID,
		RunID:  normalizedRunID,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.byTaskID[normalizedTaskID]; existing != nil {
		return nil, fmt.Errorf("task %q already has an active run", normalizedTaskID)
	}
	r.byTaskID[normalizedTaskID] = handle
	return handle, nil
}

func (r *activeRunRegistry) discard(handle *activeRunHandle) {
	r.finish(handle, RunLog{})
}

func (r *activeRunRegistry) finish(handle *activeRunHandle, run RunLog) {
	if r == nil || handle == nil {
		return
	}
	if handle.RunID != "" {
		handle.setFinal(run)
	}
	r.mu.Lock()
	if current := r.byTaskID[handle.TaskID]; current == handle {
		delete(r.byTaskID, handle.TaskID)
	}
	r.mu.Unlock()
	handle.markDone()
}

func (r *activeRunRegistry) cancelAndWait(
	ctx context.Context,
	taskID string,
	runID string,
) (RunLog, error) {
	handle, err := r.lookup(taskID, runID)
	if err != nil {
		return RunLog{}, err
	}
	handle.cancel()
	if err := handle.wait(ctx); err != nil {
		return RunLog{}, err
	}
	run, ok := handle.finalRun()
	if !ok {
		return RunLog{}, ErrTaskRunNotRunning
	}
	return run, nil
}

func (r *activeRunRegistry) lookup(taskID string, runID string) (*activeRunHandle, error) {
	if r == nil {
		return nil, ErrTaskRunNotRunning
	}
	normalizedTaskID := strings.TrimSpace(taskID)
	normalizedRunID := strings.TrimSpace(runID)
	if normalizedTaskID == "" || normalizedRunID == "" {
		return nil, ErrTaskRunNotRunning
	}
	r.mu.RLock()
	handle := r.byTaskID[normalizedTaskID]
	r.mu.RUnlock()
	if handle == nil || handle.RunID != normalizedRunID {
		return nil, ErrTaskRunNotRunning
	}
	return handle, nil
}

func (h *activeRunHandle) setFinal(run RunLog) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.final = cloneRunLog(run)
	h.finalSet = strings.TrimSpace(run.RunID) != ""
	h.mu.Unlock()
}

func (h *activeRunHandle) finalRun() (RunLog, bool) {
	if h == nil {
		return RunLog{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return cloneRunLog(h.final), h.finalSet
}

func (h *activeRunHandle) wait(ctx context.Context) error {
	if h == nil {
		return ErrTaskRunNotRunning
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *activeRunHandle) markDone() {
	if h == nil {
		return
	}
	h.doneOnce.Do(func() {
		close(h.done)
	})
}

func cloneRunLog(run RunLog) RunLog {
	cloned := run
	cloned.NodeResults = CloneRunNodeResults(run.NodeResults)
	cloned.RunCards = CloneRunCards(run.RunCards)
	return cloned
}
