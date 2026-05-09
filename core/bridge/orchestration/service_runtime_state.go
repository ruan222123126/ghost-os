package orchestration

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	bridgerss "ghost-os/bridge/rss"
)

type serviceRuntimeState struct {
	tasks serviceTaskRuntime
	rss   serviceRSSRuntime
}

type serviceTaskRuntime struct {
	store     *TaskStore
	scheduler *TaskScheduler
	initErr   error
}

type serviceRSSRuntime struct {
	handler *bridgerss.ActionHandler
}

func newServiceRuntimeState() *serviceRuntimeState {
	return &serviceRuntimeState{}
}

func (s *serviceRuntimeState) start(configStore bridgeconfig.Store, schedulerService *bridgeService, rssLog bridgerss.LogFunc) error {
	if s == nil {
		return nil
	}
	if err := s.initRSSInbox(configStore, rssLog); err != nil {
		return err
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

func (s *serviceRuntimeState) initRSSInbox(configStore bridgeconfig.Store, logFunc bridgerss.LogFunc) error {
	if s == nil {
		return nil
	}
	return s.reloadRSSInbox(configStore, logFunc)
}

func (s *serviceRuntimeState) reloadRSSInbox(configStore bridgeconfig.Store, logFunc bridgerss.LogFunc) error {
	if s == nil {
		return nil
	}
	service, err := newRSSInboxServiceFromConfig(configStore)
	if err != nil {
		s.rss.setHandler(nil, err, logFunc)
		return err
	}
	s.rss.setHandler(service, nil, logFunc)
	return nil
}

func (s *serviceRuntimeState) setRSSHandler(service *bridgerss.RSSInboxService, initErr error, logFunc bridgerss.LogFunc) {
	if s == nil {
		return
	}
	s.rss.setHandler(service, initErr, logFunc)
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

func (s *serviceRuntimeState) rssHandler() *bridgerss.ActionHandler {
	if s == nil {
		return nil
	}
	return s.rss.handler
}

func (s *serviceRuntimeState) rssInitErr() error {
	if s == nil {
		return nil
	}
	return s.rss.initErr()
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

func (r *serviceRSSRuntime) setHandler(service *bridgerss.RSSInboxService, initErr error, logFunc bridgerss.LogFunc) {
	if r == nil {
		return
	}
	if initErr != nil {
		r.handler = bridgerss.NewActionHandler(nil, initErr, logFunc)
		return
	}
	r.handler = bridgerss.NewActionHandler(service, nil, logFunc)
}

func (r *serviceRSSRuntime) initErr() error {
	if r == nil || r.handler == nil {
		return nil
	}
	return r.handler.InitErr()
}
