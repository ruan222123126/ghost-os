package tasks

import (
	"context"
	"strings"
	"sync"
	"time"
)

type taskRegistration struct {
	cancel     context.CancelFunc
	loopDone   chan struct{}
	loopCtx    context.Context
	runWG      sync.WaitGroup
	task       ScheduledTask
	running    bool
	runCancel  context.CancelFunc
	runTraceID string
	mu         sync.Mutex
}

func (r *taskRegistration) snapshot() ScheduledTask {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.task
}

func (r *taskRegistration) waitIdle() {
	if r == nil {
		return
	}
	if r.loopDone != nil {
		<-r.loopDone
	}
	r.runWG.Wait()
}

func (r *taskRegistration) beginRun(
	task ScheduledTask,
	timeout time.Duration,
	traceID string,
) (ScheduledTask, context.Context, bool, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.task = task
	if r.running {
		return r.task, nil, true, "task already running"
	}
	parentCtx := r.loopCtx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	runCtx := parentCtx
	var cancel context.CancelFunc
	if timeout > 0 {
		runCtx, cancel = context.WithTimeout(parentCtx, timeout)
	} else {
		runCtx, cancel = context.WithCancel(parentCtx)
	}
	r.running = true
	r.runCancel = cancel
	r.runTraceID = strings.TrimSpace(traceID)
	r.runWG.Add(1)
	return r.task, runCtx, false, ""
}

func (r *taskRegistration) finishRun() {
	r.mu.Lock()
	cancel := r.runCancel
	r.runCancel = nil
	r.runTraceID = ""
	r.running = false
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	r.runWG.Done()
}

func (r *taskRegistration) cancelRun() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel := r.runCancel
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (r *taskRegistration) stop() {
	if r == nil {
		return
	}
	if r.cancel != nil {
		r.cancel()
	}
	r.cancelRun()
}

func (s *TaskScheduler) lookupTask(taskID string) *taskRegistration {
	if s == nil {
		return nil
	}
	id := strings.TrimSpace(taskID)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tasks[id]
}

func schedulerValidator(store *Store) DefinitionValidator {
	if store == nil {
		return nil
	}
	return store.validator
}
