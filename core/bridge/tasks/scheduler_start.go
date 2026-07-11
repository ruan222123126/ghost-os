package tasks

import (
	"context"
	"log"
)

func (s *TaskScheduler) Start() error {
	if err := s.requireConfigured(); err != nil {
		return err
	}
	rollback, alreadyRunning := s.beginStart()
	if alreadyRunning {
		return nil
	}
	started := false
	defer func() {
		if started || rollback == nil {
			return
		}
		rollback()
	}()
	if err := s.loadAndRegisterEnabledTasks(); err != nil {
		return err
	}
	started = true
	return nil
}

func (s *TaskScheduler) beginStart() (func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil, true
	}
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())
	s.lifecycleCtx = lifecycleCtx
	s.lifecycleCancel = lifecycleCancel
	s.running = true
	return func() {
		s.mu.Lock()
		s.lifecycleCtx = nil
		s.lifecycleCancel = nil
		s.running = false
		s.mu.Unlock()
		lifecycleCancel()
	}, false
}

func (s *TaskScheduler) loadAndRegisterEnabledTasks() error {
	tasks, issues, err := s.store.ListTasksTolerant()
	if err != nil {
		return err
	}
	s.logTaskLoadIssues(issues)
	s.registerEnabledTasks(tasks)
	return nil
}

func (s *TaskScheduler) logTaskLoadIssues(issues []LoadIssue) {
	for _, issue := range issues {
		log.Printf(
			"task scheduler skipped corrupted task: kind=%s task_id=%s path=%s error=%s",
			issue.Kind,
			issue.TaskID,
			issue.Path,
			issue.Error,
		)
	}
}

func (s *TaskScheduler) registerEnabledTasks(tasks []ScheduledTask) {
	for _, task := range tasks {
		if !task.Enabled {
			continue
		}
		if err := s.register(task); err != nil {
			s.logInvalidRegistration(task, err)
		}
	}
}

func (s *TaskScheduler) logInvalidRegistration(task ScheduledTask, err error) {
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
