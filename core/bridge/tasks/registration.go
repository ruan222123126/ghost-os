package tasks

import (
	"context"
	"strings"
	"sync"
	"time"
)

const (
	skipRunReasonAlreadyRunning      = "task already running"
	skipRunReasonRegistrationRetired = "task registration retired"
)

type taskRegistration struct {
	cancel     context.CancelFunc
	loopDone   chan struct{}
	loopCtx    context.Context
	runWG      sync.WaitGroup
	retired    bool
	task       ScheduledTask
	running    bool
	runCancel  context.CancelFunc
	runTimeout time.Duration
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
) (ScheduledTask, context.Context, bool, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.retired {
		return task, nil, true, skipRunReasonRegistrationRetired
	}
	r.task = task
	if r.running {
		return r.task, nil, true, skipRunReasonAlreadyRunning
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
	r.runTimeout = timeout
	r.runWG.Add(1)
	return r.task, runCtx, false, ""
}

func (r *taskRegistration) executionTimeout() time.Duration {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runTimeout
}

func (r *taskRegistration) retire() (context.CancelFunc, context.CancelFunc) {
	if r == nil {
		return nil, nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.retired = true
	return r.cancel, r.runCancel
}

func (r *taskRegistration) finishRun() {
	r.mu.Lock()
	cancel := r.runCancel
	r.runCancel = nil
	r.runTimeout = 0
	r.running = false
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	r.runWG.Done()
}

func (r *taskRegistration) stop() {
	if r == nil {
		return
	}
	cancelLoop, cancelRun := r.retire()
	if cancelLoop != nil {
		cancelLoop()
	}
	if cancelRun != nil {
		cancelRun()
	}
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
