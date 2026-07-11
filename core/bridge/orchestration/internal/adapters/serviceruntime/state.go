package serviceruntime

import (
	"time"

	bridgeconfig "ghost-os/bridge/config"
	taskstoreadapter "ghost-os/bridge/orchestration/internal/adapters/taskstore"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	bridgeTasks "ghost-os/bridge/tasks"
)

type State struct {
	tasks taskRuntime
}

type taskRuntime struct {
	store     *bridgeTasks.Store
	scheduler *bridgeTasks.TaskScheduler
	initErr   error
}

type Lifecycle struct {
	runtimes    *State
	sessionPush *internaltrace.SessionPushHub
}

func NewState() *State {
	return &State{}
}

func NewLifecycle(runtimes *State, sessionPush *internaltrace.SessionPushHub) *Lifecycle {
	return &Lifecycle{
		runtimes:    runtimes,
		sessionPush: sessionPush,
	}
}

func (s *State) Start(configStore bridgeconfig.Store, executor bridgeTasks.Executor) error {
	if s == nil {
		return nil
	}
	return s.InitTaskRuntime(configStore, executor)
}

func (s *State) BootstrapSystemTasks() error {
	if s == nil {
		return nil
	}
	return s.removeDeprecatedSystemTasks()
}

func (s *State) removeDeprecatedSystemTasks() error {
	if s == nil || s.tasks.store == nil || s.tasks.scheduler == nil {
		return nil
	}
	return taskstoreadapter.RemoveDeprecatedSystemTasks(s.tasks.store, s.tasks.scheduler)
}

func (s *State) Close() {
	if s == nil {
		return
	}
	s.tasks.stop()
}

func (s *State) InitTaskRuntime(configStore bridgeconfig.Store, executor bridgeTasks.Executor) error {
	if s == nil {
		return nil
	}
	return s.tasks.start(configStore, executor)
}

func (s *State) TaskStore() *bridgeTasks.Store {
	if s == nil {
		return nil
	}
	return s.tasks.store
}

func (s *State) TaskScheduler() *bridgeTasks.TaskScheduler {
	if s == nil {
		return nil
	}
	return s.tasks.scheduler
}

func (s *State) TaskInitErr() error {
	if s == nil {
		return nil
	}
	return s.tasks.initErr
}

func (l *Lifecycle) Close() {
	if l == nil {
		return
	}
	if l.runtimes != nil {
		l.runtimes.Close()
	}
	if l.sessionPush != nil {
		l.sessionPush.Close()
	}
}

func (l *Lifecycle) SessionPushHub() *internaltrace.SessionPushHub {
	if l == nil {
		return nil
	}
	return l.sessionPush
}

func (r *taskRuntime) start(configStore bridgeconfig.Store, executor bridgeTasks.Executor) error {
	if r == nil {
		return nil
	}
	if r.store != nil && r.scheduler != nil {
		r.initErr = nil
		return nil
	}
	taskCfg, err := apptasks.LoadRuntimeConfig(configStore)
	if err != nil {
		r.initErr = err
		return err
	}
	taskStore, err := bridgeTasks.NewStore(taskCfg.TasksPath, apptasks.ValidateDefinition)
	if err != nil {
		r.initErr = err
		return err
	}
	scheduler := bridgeTasks.NewTaskSchedulerWithTimeout(
		taskStore,
		executor,
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

func (r *taskRuntime) stop() {
	if r == nil || r.scheduler == nil {
		return
	}
	r.scheduler.Stop()
}
