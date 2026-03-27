package tasks

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ExecutionResult struct {
	Status          string
	SessionIDOutput string
	ResponsePreview string
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

	mu      sync.Mutex
	running bool
	tasks   map[string]*taskRegistration
}

const defaultTaskExecutionTimeout = 2 * time.Minute

var traceCounter uint64

func NewTaskScheduler(store *Store, executor Executor) *TaskScheduler {
	scheduler := &TaskScheduler{
		store:    store,
		executor: executor,
		now: func() time.Time {
			return time.Now().UTC()
		},
		traceID:          nextTraceID,
		executionTimeout: defaultTaskExecutionTimeout,
		tasks:            make(map[string]*taskRegistration),
	}
	scheduler.execute = scheduler.executeWithExecutor
	return scheduler
}

func (s *TaskScheduler) Start() error {
	if s == nil || s.store == nil {
		return nil
	}
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()
	started := false
	defer func() {
		if started {
			return
		}
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	tasks, issues, err := s.store.ListTasksTolerant()
	if err != nil {
		return err
	}
	for _, issue := range issues {
		log.Printf(
			"task scheduler skipped corrupted task: kind=%s task_id=%s path=%s error=%s",
			issue.Kind,
			issue.TaskID,
			issue.Path,
			issue.Error,
		)
	}
	for _, task := range tasks {
		if !task.Enabled {
			continue
		}
		if err := s.register(task); err != nil {
			path, pathErr := s.store.PathForTask(task.ID)
			if pathErr != nil {
				path = ""
			}
			log.Printf(
				"task scheduler skipped invalid task during registration: task_id=%s path=%s error=%v",
				task.ID,
				path,
				err,
			)
		}
	}
	started = true
	return nil
}

func (s *TaskScheduler) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	registrations := make([]*taskRegistration, 0, len(s.tasks))
	for id, reg := range s.tasks {
		registrations = append(registrations, reg)
		reg.stop()
		delete(s.tasks, id)
	}
	s.running = false
	s.mu.Unlock()

	for _, reg := range registrations {
		reg.waitIdle()
	}
}

func (s *TaskScheduler) Upsert(task ScheduledTask) error {
	if s == nil || s.store == nil {
		return nil
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
	if s == nil {
		return nil
	}
	id := strings.TrimSpace(taskID)
	s.mu.Lock()
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
	if s == nil || s.store == nil {
		return RunLog{}, fmt.Errorf("task scheduler is not configured")
	}
	if err := NormalizeScheduledTask(&task, schedulerValidator(s.store)); err != nil {
		return RunLog{}, err
	}
	reg := s.lookupTask(task.ID)
	if reg == nil {
		reg = &taskRegistration{task: task}
	}
	scheduledAt := s.now().UTC()
	runTraceID := strings.TrimSpace(traceID)
	if runTraceID == "" {
		runTraceID = s.traceID()
	}
	task, runCtx, reg, skipped, reason := s.beginManualRun(reg, task, runTraceID)
	if skipped {
		run := skippedTaskRunLog(task, runTraceID, scheduledAt, reason)
		return run, s.store.AppendRunLog(run)
	}
	return s.executeRun(runCtx, reg, task, scheduledAt, runTraceID)
}

func (s *TaskScheduler) beginManualRun(
	reg *taskRegistration,
	task ScheduledTask,
	traceID string,
) (ScheduledTask, context.Context, *taskRegistration, bool, string) {
	if reg == nil {
		reg = &taskRegistration{task: task}
	}
	task, runCtx, skipped, reason := reg.beginRun(task, s.taskExecutionTimeout(), traceID)
	if reason != skipRunReasonRegistrationRetired {
		return task, runCtx, reg, skipped, reason
	}
	reg = &taskRegistration{task: task}
	task, runCtx, skipped, reason = reg.beginRun(task, s.taskExecutionTimeout(), traceID)
	return task, runCtx, reg, skipped, reason
}

func (s *TaskScheduler) register(task ScheduledTask) error {
	plan, err := buildTaskSchedulePlan(task)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	reg := &taskRegistration{
		cancel:   cancel,
		loopDone: make(chan struct{}),
		loopCtx:  ctx,
		task:     task,
	}

	var existing *taskRegistration
	s.mu.Lock()
	if current, ok := s.tasks[task.ID]; ok {
		existing = current
		existing.stop()
		delete(s.tasks, task.ID)
	}
	s.mu.Unlock()
	if existing != nil {
		existing.waitIdle()
	}

	s.mu.Lock()
	s.tasks[task.ID] = reg
	s.mu.Unlock()

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
	task, runCtx, skipped, reason := reg.beginRun(reg.snapshot(), s.taskExecutionTimeout(), runTraceID)
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
