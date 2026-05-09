package tasks

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bridgeconfig "ghost-os/bridge/config"
)

type ExecutionResult struct {
	Status          string
	SessionIDOutput string
	ResponsePreview string
	NodeResults     []RunNodeResult
	Error           string
}

type Executor interface {
	Execute(ctx context.Context, task ScheduledTask, traceID string) ExecutionResult
}

type TaskScheduler struct {
	store    *Store
	executor Executor

	now              func() time.Time
	traceID          func() string
	execute          func(context.Context, ScheduledTask, string) ExecutionResult
	executionTimeout time.Duration

	mu              sync.Mutex
	running         bool
	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc
	tasks           map[string]*taskRegistration
}

const defaultTaskExecutionTimeout = time.Duration(bridgeconfig.DefaultTaskExecutionTimeoutMS) * time.Millisecond

var traceCounter uint64

func NewTaskScheduler(store *Store, executor Executor) *TaskScheduler {
	return NewTaskSchedulerWithTimeout(store, executor, defaultTaskExecutionTimeout)
}

func NewTaskSchedulerWithTimeout(
	store *Store,
	executor Executor,
	executionTimeout time.Duration,
) *TaskScheduler {
	scheduler := &TaskScheduler{
		store:    store,
		executor: executor,
		now: func() time.Time {
			return time.Now().UTC()
		},
		traceID:          nextTraceID,
		executionTimeout: normalizeExecutionTimeout(executionTimeout),
		tasks:            make(map[string]*taskRegistration),
	}
	scheduler.execute = scheduler.executeWithExecutor
	return scheduler
}

func (s *TaskScheduler) SetExecutionTimeout(timeout time.Duration) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.executionTimeout = normalizeExecutionTimeout(timeout)
	s.mu.Unlock()
}

func normalizeExecutionTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultTaskExecutionTimeout
	}
	return timeout
}

func (s *TaskScheduler) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.lifecycleCancel
	registrations := make([]*taskRegistration, 0, len(s.tasks))
	for id, reg := range s.tasks {
		registrations = append(registrations, reg)
		reg.stop()
		delete(s.tasks, id)
	}
	s.lifecycleCtx = nil
	s.lifecycleCancel = nil
	s.running = false
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	for _, reg := range registrations {
		reg.waitIdle()
	}
}

func (s *TaskScheduler) Upsert(task ScheduledTask) error {
	if err := s.requireConfigured(); err != nil {
		return err
	}
	if err := NormalizeScheduledTask(&task, schedulerValidator(s.store)); err != nil {
		return err
	}
	if !task.Enabled {
		return s.Unregister(task.ID)
	}
	return s.register(task)
}

func (s *TaskScheduler) Unregister(taskID string) error {
	if err := s.requireConfigured(); err != nil {
		return err
	}
	id := strings.TrimSpace(taskID)
	s.mu.Lock()
	if !s.running || s.lifecycleCtx == nil {
		s.mu.Unlock()
		return ErrTaskSchedulerStopped
	}
	reg := s.tasks[id]
	if reg != nil {
		reg.stop()
		delete(s.tasks, id)
	}
	s.mu.Unlock()
	if reg != nil {
		reg.waitIdle()
	}
	return nil
}

func (s *TaskScheduler) RunNow(task ScheduledTask, traceID string) (RunLog, error) {
	if err := s.requireConfigured(); err != nil {
		return RunLog{}, err
	}
	if err := NormalizeScheduledTask(&task, schedulerValidator(s.store)); err != nil {
		return RunLog{}, err
	}
	reg, lifecycleCtx, err := s.runningTask(task.ID)
	if err != nil {
		return RunLog{}, err
	}
	if reg == nil {
		reg = &taskRegistration{task: task, loopCtx: lifecycleCtx}
	}
	scheduledAt := s.now().UTC()
	runTraceID := strings.TrimSpace(traceID)
	if runTraceID == "" {
		runTraceID = s.traceID()
	}
	task, runCtx, reg, skipped, reason := s.beginManualRun(reg, task)
	if skipped {
		run := skippedTaskRunLog(task, runTraceID, scheduledAt, reason)
		return run, s.store.AppendRunLog(run)
	}
	return s.executeRun(runCtx, reg, task, scheduledAt, runTraceID)
}

func (s *TaskScheduler) register(task ScheduledTask) error {
	plan, err := buildTaskSchedulePlan(task)
	if err != nil {
		return err
	}
	lifecycleCtx, existing, err := s.prepareRegistration(task.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		existing.waitIdle()
	}

	ctx, cancel := context.WithCancel(lifecycleCtx)
	reg := &taskRegistration{
		cancel:   cancel,
		loopDone: make(chan struct{}),
		loopCtx:  ctx,
		task:     task,
	}
	if err := s.commitRegistration(task.ID, reg); err != nil {
		cancel()
		return err
	}

	go s.runTaskLoop(ctx, reg, plan)
	return nil
}

func (s *TaskScheduler) runTaskLoop(
	ctx context.Context,
	reg *taskRegistration,
	plan taskSchedulePlan,
) {
	defer close(reg.loopDone)
	next := plan.initialNext(s.now(), reg.snapshot())
	if err := s.persistNextRun(reg, next); err != nil {
		log.Printf("task scheduler persist next run failed: task_id=%s error=%v", reg.snapshot().ID, err)
	}

	for {
		wait := time.Until(next)
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}

		s.fireTask(reg, next)
		next = plan.nextAfter(next, s.now())
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := s.persistNextRun(reg, next); err != nil {
			log.Printf("task scheduler persist next run failed: task_id=%s error=%v", reg.snapshot().ID, err)
		}
	}
}

func (s *TaskScheduler) fireTask(reg *taskRegistration, scheduledAt time.Time) {
	runTraceID := s.traceID()
	task, runCtx, skipped, reason := reg.beginRun(reg.snapshot(), s.taskExecutionTimeout())
	if skipped {
		if reason == skipRunReasonRegistrationRetired {
			return
		}
		_ = s.store.AppendRunLog(skippedTaskRunLog(task, runTraceID, scheduledAt, reason))
		return
	}
	go func() {
		if _, err := s.executeRun(runCtx, reg, task, scheduledAt, runTraceID); err != nil {
			log.Printf("task scheduler execute run failed: task_id=%s error=%v", task.ID, err)
		}
	}()
}

func nextTraceID() string {
	sequence := atomic.AddUint64(&traceCounter, 1)
	return fmt.Sprintf("bridge-%d-%d", time.Now().UnixMilli(), sequence)
}
