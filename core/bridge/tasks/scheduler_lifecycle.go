package tasks

import (
	"context"
	"errors"
	"strings"
)

var ErrTaskSchedulerStopped = errors.New("task scheduler is not running")
var ErrTaskSchedulerNotConfigured = errors.New("task scheduler is not configured")

func (s *TaskScheduler) requireConfigured() error {
	if s == nil || s.store == nil {
		return ErrTaskSchedulerNotConfigured
	}
	return nil
}

func (s *TaskScheduler) prepareRegistration(taskID string) (context.Context, *taskRegistration, error) {
	id := strings.TrimSpace(taskID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running || s.lifecycleCtx == nil {
		return nil, nil, ErrTaskSchedulerStopped
	}
	current := s.tasks[id]
	if current == nil {
		return s.lifecycleCtx, nil, nil
	}
	current.stop()
	delete(s.tasks, id)
	return s.lifecycleCtx, current, nil
}

func (s *TaskScheduler) commitRegistration(taskID string, reg *taskRegistration) error {
	id := strings.TrimSpace(taskID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running || s.lifecycleCtx == nil {
		return ErrTaskSchedulerStopped
	}
	s.tasks[id] = reg
	return nil
}

func (s *TaskScheduler) runningTask(taskID string) (*taskRegistration, context.Context, error) {
	id := strings.TrimSpace(taskID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running || s.lifecycleCtx == nil {
		return nil, nil, ErrTaskSchedulerStopped
	}
	return s.tasks[id], s.lifecycleCtx, nil
}
