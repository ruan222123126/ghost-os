package tasks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cron "github.com/robfig/cron/v3"
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

type taskSchedulePlan struct {
	kind     string
	interval time.Duration
	cronExpr string
	cron     cron.Schedule
}

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
		log.Printf("task scheduler skipped corrupted task: kind=%s task_id=%s path=%s error=%s", issue.Kind, issue.TaskID, issue.Path, issue.Error)
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
			log.Printf("task scheduler skipped invalid task during registration: task_id=%s path=%s error=%v", task.ID, path, err)
			continue
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
	if reg, ok := s.tasks[id]; ok {
		reg.stop()
		delete(s.tasks, id)
	}
	s.mu.Unlock()
	if reg != nil {
		reg.waitIdle()
	}
	return nil
}

func (s *TaskScheduler) register(task ScheduledTask) error {
	plan, err := buildTaskSchedulePlan(task)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	reg := &taskRegistration{cancel: cancel, loopDone: make(chan struct{}), loopCtx: ctx, task: task}

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

func (s *TaskScheduler) runTaskLoop(ctx context.Context, reg *taskRegistration, plan taskSchedulePlan) {
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
	task, runCtx, skipped, _ := reg.beginRun(reg.snapshot(), s.taskExecutionTimeout(), runTraceID)
	if skipped {
		_ = s.store.AppendRunLog(skippedTaskRunLog(task, runTraceID, scheduledAt))
		return
	}
	go func() {
		if _, err := s.executeRun(runCtx, reg, task, scheduledAt, runTraceID); err != nil {
			log.Printf("task scheduler execute run failed: task_id=%s error=%v", task.ID, err)
		}
	}()
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
	task, runCtx, skipped, _ := reg.beginRun(task, s.taskExecutionTimeout(), runTraceID)
	if skipped {
		run := skippedTaskRunLog(task, runTraceID, scheduledAt)
		return run, s.store.AppendRunLog(run)
	}
	return s.executeRun(runCtx, reg, task, scheduledAt, runTraceID)
}

func (s *TaskScheduler) executeRun(ctx context.Context, reg *taskRegistration, task ScheduledTask, scheduledAt time.Time, traceID string) (RunLog, error) {
	defer reg.finishRun()

	startedAt := s.now().UTC()
	result := s.finalizeExecutionResult(ctx, s.execute(ctx, task, traceID))
	finishedAt := s.now().UTC()

	reg.mu.Lock()
	reg.task.LastRunAt = startedAt
	reg.task.LastError = strings.TrimSpace(result.Error)
	updated := reg.task
	reg.mu.Unlock()

	if err := s.store.SaveTask(&updated); err != nil {
		return RunLog{}, fmt.Errorf("save run state for %s: %w", task.ID, err)
	}
	run := RunLog{
		TaskID:          task.ID,
		RunID:           NewRunID(),
		TraceID:         traceID,
		TaskKind:        task.TaskKind,
		Action:          task.Action,
		ScheduledAt:     scheduledAt.UTC(),
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		Status:          result.Status,
		SessionIDInput:  task.SessionID,
		SessionIDOutput: result.SessionIDOutput,
		ResponsePreview: result.ResponsePreview,
		Error:           result.Error,
	}
	if err := s.store.AppendRunLog(run); err != nil {
		return run, fmt.Errorf("append run log for %s: %w", task.ID, err)
	}
	return run, nil
}

func (s *TaskScheduler) finalizeExecutionResult(ctx context.Context, result ExecutionResult) ExecutionResult {
	if strings.TrimSpace(result.Status) == "" {
		result.Status = RunStatusError
	}

	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		result.Status = RunStatusCancelled
		result.Error = "task execution cancelled"
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		result.Status = RunStatusError
		result.Error = fmt.Sprintf("task execution timed out after %s", s.taskExecutionTimeout())
	}

	return result
}

func (s *TaskScheduler) taskExecutionTimeout() time.Duration {
	if s == nil || s.executionTimeout <= 0 {
		return defaultTaskExecutionTimeout
	}
	return s.executionTimeout
}

func (s *TaskScheduler) persistNextRun(reg *taskRegistration, next time.Time) error {
	reg.mu.Lock()
	reg.task.NextRunAt = next.UTC()
	updated := reg.task
	reg.mu.Unlock()
	return s.store.SaveTask(&updated)
}

func (s *TaskScheduler) executeWithExecutor(ctx context.Context, task ScheduledTask, traceID string) ExecutionResult {
	if s == nil || s.executor == nil {
		return ExecutionResult{Status: RunStatusError, Error: "task executor is not configured"}
	}
	return s.executor.Execute(ctx, task, traceID)
}

func buildTaskSchedulePlan(task ScheduledTask) (taskSchedulePlan, error) {
	switch task.ScheduleType {
	case ScheduleTypeInterval:
		if task.IntervalSeconds <= 0 {
			return taskSchedulePlan{}, fmt.Errorf("%w: interval_seconds must be > 0", ErrInvalidTaskConfig)
		}
		return taskSchedulePlan{kind: ScheduleTypeInterval, interval: time.Duration(task.IntervalSeconds) * time.Second}, nil
	case ScheduleTypeCron:
		schedule, err := cron.ParseStandard(task.CronExpr)
		if err != nil {
			return taskSchedulePlan{}, fmt.Errorf("%w: invalid cron_expr: %w", ErrInvalidTaskConfig, err)
		}
		return taskSchedulePlan{kind: ScheduleTypeCron, cronExpr: task.CronExpr, cron: schedule}, nil
	default:
		return taskSchedulePlan{}, fmt.Errorf("%w: unsupported schedule type %q", ErrInvalidTaskConfig, task.ScheduleType)
	}
}

func (p taskSchedulePlan) initialNext(now time.Time, task ScheduledTask) time.Time {
	if !task.NextRunAt.IsZero() && task.NextRunAt.After(now) {
		return task.NextRunAt.UTC()
	}
	switch p.kind {
	case ScheduleTypeInterval:
		return now.UTC().Add(p.interval)
	case ScheduleTypeCron:
		return p.cron.Next(now.UTC()).UTC()
	default:
		return now.UTC()
	}
}

func (p taskSchedulePlan) nextAfter(previous time.Time, now time.Time) time.Time {
	switch p.kind {
	case ScheduleTypeInterval:
		next := previous.UTC().Add(p.interval)
		if next.After(now.UTC()) {
			return next
		}
		return now.UTC().Add(p.interval)
	case ScheduleTypeCron:
		next := p.cron.Next(previous.UTC()).UTC()
		if next.After(now.UTC()) {
			return next
		}
		return p.cron.Next(now.UTC()).UTC()
	default:
		return now.UTC()
	}
}

func nextTaskRunAt(task ScheduledTask, now time.Time) (time.Time, error) {
	plan, err := buildTaskSchedulePlan(task)
	if err != nil {
		return time.Time{}, err
	}
	return plan.initialNext(now.UTC(), task), nil
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

func (r *taskRegistration) beginRun(task ScheduledTask, timeout time.Duration, traceID string) (ScheduledTask, context.Context, bool, string) {
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

func skippedTaskRunLog(task ScheduledTask, traceID string, scheduledAt time.Time) RunLog {
	return RunLog{
		TaskID:         task.ID,
		RunID:          NewRunID(),
		TraceID:        traceID,
		TaskKind:       task.TaskKind,
		Action:         task.Action,
		ScheduledAt:    scheduledAt.UTC(),
		Status:         RunStatusSkipped,
		SessionIDInput: task.SessionID,
		Error:          "task already running",
	}
}

func schedulerValidator(store *Store) DefinitionValidator {
	if store == nil {
		return nil
	}
	return store.validator
}

func nextTraceID() string {
	sequence := atomic.AddUint64(&traceCounter, 1)
	return fmt.Sprintf("bridge-%d-%d", time.Now().UnixMilli(), sequence)
}
