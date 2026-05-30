package orchestration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
)

type serviceRuntimeState struct {
	tasks serviceTaskRuntime
}

type serviceTaskRuntime struct {
	store     *TaskStore
	scheduler *TaskScheduler
	initErr   error
}

func newServiceRuntimeState() *serviceRuntimeState {
	return &serviceRuntimeState{}
}

func (s *serviceRuntimeState) start(configStore bridgeconfig.Store, schedulerService *bridgeService) error {
	if s == nil {
		return nil
	}
	return s.initTaskRuntime(configStore, schedulerService)
}

func (s *serviceRuntimeState) bootstrapSystemTasks(_ bridgeconfig.Store) error {
	if s == nil {
		return nil
	}
	return s.removeDeprecatedSystemTasks()
}

func (s *serviceRuntimeState) removeDeprecatedSystemTasks() error {
	if s == nil || s.tasks.store == nil || s.tasks.scheduler == nil {
		return nil
	}
	tasksDir := strings.TrimSpace(s.tasks.store.TasksDir())
	if tasksDir == "" {
		return nil
	}
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		taskID, ok := parseSystemTaskEntryID(entry.Name(), entry.IsDir())
		if !ok {
			continue
		}
		path := filepath.Join(tasksDir, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !isSystemTaskDocument(data) {
			continue
		}
		_ = s.tasks.scheduler.Unregister(taskID)
		if deleteErr := s.tasks.store.DeleteTask(taskID); deleteErr != nil && !errors.Is(deleteErr, ErrTaskNotFound) {
			return deleteErr
		}
	}
	return nil
}

func parseSystemTaskEntryID(name string, isDir bool) (string, bool) {
	if isDir || filepath.Ext(name) != ".json" {
		return "", false
	}
	id := strings.TrimSpace(strings.TrimSuffix(name, ".json"))
	if id == "" {
		return "", false
	}
	return id, true
}

func isSystemTaskDocument(data []byte) bool {
	var task struct {
		TaskKind string `json:"task_kind"`
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return false
	}
	return normalizeTaskKind(task.TaskKind) == taskKindSystemAction
}

func (s *serviceRuntimeState) close() {
	if s == nil {
		return
	}
	s.tasks.stop()
}

func (s *serviceRuntimeState) initTaskRuntime(configStore bridgeconfig.Store, schedulerService *bridgeService) error {
	if s == nil {
		return nil
	}
	return s.tasks.start(configStore, schedulerService)
}

func (s *serviceRuntimeState) taskStore() *TaskStore {
	if s == nil {
		return nil
	}
	return s.tasks.store
}

func (s *serviceRuntimeState) taskScheduler() *TaskScheduler {
	if s == nil {
		return nil
	}
	return s.tasks.scheduler
}

func (s *serviceRuntimeState) taskInitErr() error {
	if s == nil {
		return nil
	}
	return s.tasks.initErr
}

type serviceLifecycle struct {
	runtimes    *serviceRuntimeState
	sessionPush *sessionPushHub
}

func newServiceLifecycle(runtimes *serviceRuntimeState, sessionPush *sessionPushHub) *serviceLifecycle {
	return &serviceLifecycle{
		runtimes:    runtimes,
		sessionPush: sessionPush,
	}
}

func (l *serviceLifecycle) close() {
	if l == nil {
		return
	}
	if l.runtimes != nil {
		l.runtimes.close()
	}
	if l.sessionPush != nil {
		l.sessionPush.Close()
	}
}

func (l *serviceLifecycle) sessionPushHub() *sessionPushHub {
	if l == nil {
		return nil
	}
	return l.sessionPush
}

func (r *serviceTaskRuntime) start(configStore bridgeconfig.Store, schedulerService *bridgeService) error {
	if r == nil {
		return nil
	}
	if r.store != nil && r.scheduler != nil {
		r.initErr = nil
		return nil
	}
	taskCfg, err := loadTaskRuntimeConfig(configStore)
	if err != nil {
		r.initErr = err
		return err
	}
	taskStore, err := NewTaskStore(taskCfg.TasksPath)
	if err != nil {
		r.initErr = err
		return err
	}
	scheduler := NewTaskSchedulerWithTimeout(
		taskStore,
		schedulerService,
		time.Duration(taskCfg.ExecutionTimeoutMS)*time.Millisecond,
	)
	if err := scheduler.Start(); err != nil {
		r.initErr = err
		return err
	}
	r.store = taskStore
	r.scheduler = scheduler
	r.initErr = nil
	return nil
}

func (r *serviceTaskRuntime) stop() {
	if r == nil || r.scheduler == nil {
		return
	}
	r.scheduler.Stop()
}
