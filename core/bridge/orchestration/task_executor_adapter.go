package orchestration

import (
	"context"
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	orchestrationruntime "ghost-os/bridge/orchestration/internal/adapters/orchestrationruntime"
	taskexecution "ghost-os/bridge/orchestration/internal/adapters/taskexecution"
	workflowadapter "ghost-os/bridge/orchestration/internal/adapters/workflow"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

type taskExecutorAdapter struct{ service *bridgeService }

const taskRunTranscriptEventMarker = apptasks.RunTranscriptEventMarker

func (s *Service) ExecuteUserTaskGetAction(id string, traceID string) (ServiceResult, error) {
	return s.executeTaskGetInScope(id, taskListScopeUser, traceID)
}
func (s *Service) ExecuteOrchestrationTaskGetAction(id string, traceID string) (ServiceResult, error) {
	return s.executeTaskGetInScope(id, taskListScopeOrchestration, traceID)
}
func (s *Service) ExecuteUserTaskUpdateAction(id string, req TaskUpdateParams, traceID string) (ServiceResult, error) {
	return s.executeTaskUpdateInScope(id, req, taskListScopeUser, traceID)
}
func (s *Service) ExecuteOrchestrationTaskUpdateAction(id string, req TaskUpdateParams, traceID string) (ServiceResult, error) {
	return s.executeTaskUpdateInScope(id, req, taskListScopeOrchestration, traceID)
}
func (s *Service) ExecuteUserTaskDeleteAction(id string, traceID string) (ServiceResult, error) {
	return s.executeTaskDeleteInScope(id, taskListScopeUser, traceID)
}
func (s *Service) ExecuteOrchestrationTaskDeleteAction(id string, traceID string) (ServiceResult, error) {
	return s.executeTaskDeleteInScope(id, taskListScopeOrchestration, traceID)
}
func (s *Service) ExecuteUserTaskRunAction(id string, startOnly *bool, traceID string) (ServiceResult, error) {
	return s.executeTaskRunInScope(id, startOnly, taskListScopeUser, traceID)
}
func (s *Service) ExecuteOrchestrationTaskRunAction(id string, startOnly *bool, traceID string) (ServiceResult, error) {
	return s.executeTaskRunInScope(id, startOnly, taskListScopeOrchestration, traceID)
}
func (s *Service) ExecuteUserTaskStopAction(ctx context.Context, id string, req TaskStopParams, traceID string) (ServiceResult, error) {
	return s.executeTaskStopInScope(ctx, id, req, taskListScopeUser, traceID)
}
func (s *Service) ExecuteOrchestrationTaskStopAction(ctx context.Context, id string, req TaskStopParams, traceID string) (ServiceResult, error) {
	return s.executeTaskStopInScope(ctx, id, req, taskListScopeOrchestration, traceID)
}
func (s *Service) ExecuteUserTaskLogsAction(id string, limit *int, traceID string) (ServiceResult, error) {
	return s.executeTaskLogsInScope(id, limit, taskListScopeUser, traceID)
}
func (s *Service) ExecuteOrchestrationTaskLogsAction(id string, limit *int, traceID string) (ServiceResult, error) {
	return s.executeTaskLogsInScope(id, limit, taskListScopeOrchestration, traceID)
}

func (s *Service) executeTaskGetInScope(id string, scope string, traceID string) (ServiceResult, error) {
	return s.inner.executeTaskGetActionResult(TaskIDParams{ID: id, Scope: scope}, traceID)
}

func (s *Service) executeTaskUpdateInScope(id string, req TaskUpdateParams, scope string, traceID string) (ServiceResult, error) {
	req.ID = id
	req.Scope = scope
	return s.inner.executeTaskUpdateActionResult(req, traceID)
}

func (s *Service) executeTaskDeleteInScope(id string, scope string, traceID string) (ServiceResult, error) {
	return s.inner.executeTaskDeleteActionResult(TaskIDParams{ID: id, Scope: scope}, traceID)
}

func (s *Service) executeTaskRunInScope(id string, startOnly *bool, scope string, traceID string) (ServiceResult, error) {
	return s.inner.executeTaskRunNowActionResult(TaskRunNowParams{ID: id, Scope: scope, StartOnly: startOnly}, traceID)
}

func (s *Service) executeTaskStopInScope(ctx context.Context, id string, req TaskStopParams, scope string, traceID string) (ServiceResult, error) {
	req.ID = id
	req.Scope = scope
	return s.inner.executeTaskStopActionResult(ctx, req, traceID)
}

func (s *Service) executeTaskLogsInScope(id string, limit *int, scope string, traceID string) (ServiceResult, error) {
	return s.inner.executeTaskLogsActionResult(TaskLogsParams{ID: id, Limit: limit, Scope: scope}, traceID)
}

func (a taskExecutorAdapter) Execute(ctx context.Context, task ScheduledTask, traceID string) bridgeTasks.ExecutionResult {
	return a.executor().Execute(ctx, task, traceID)
}

func (a taskExecutorAdapter) PrepareRunSession(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) (bridgeTasks.RunSession, error) {
	return a.executor().PrepareRunSession(ctx, task, traceID)
}

func (a taskExecutorAdapter) executor() taskexecution.Executor {
	cfg := taskexecution.Config{
		Workflow:      apptasks.KindExecutorFunc(a.executeWorkflowTask),
		Orchestration: apptasks.KindExecutorFunc(a.executeOrchestrationTask),
		Agent: apptasks.AgentExecutor{
			StreamRunner: a,
			RelayRunner:  a,
		},
	}
	if a.service != nil {
		cfg.SessionStore = a.service.sessionStore
		cfg.SessionPushHub = a.service.sessionPushHub()
	}
	return taskexecution.Executor{Config: cfg}
}

func (a taskExecutorAdapter) ExecuteRelayTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) (apptasks.RelayTaskResult, error) {
	if a.service == nil {
		return apptasks.RelayTaskResult{}, apptasks.NewRelayPreflightError(
			errors.New("task executor service is not configured"),
		)
	}
	if task.Relay == nil {
		return apptasks.RelayTaskResult{}, apptasks.NewRelayPreflightError(
			errors.New("relay config is required"),
		)
	}
	result, err := newRelayTaskRunner(a.service).ExecuteTask(ctx, task, traceID)
	return taskexecution.RelayTaskResult(result), err
}

func (a taskExecutorAdapter) RunAgentTaskStream(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	return taskexecution.AgentStreamRunner{
		Direct:   a.directAgentTaskRunner(),
		Override: a.sessionAgentRunner(),
	}.RunAgentTaskStream(ctx, task, traceID, sink)
}

func (a taskExecutorAdapter) directAgentTaskRunner() taskexecution.AgentDirectRunner {
	if a.service == nil || a.service.agentRunner == nil {
		return nil
	}
	return a.service.agentRunner
}

func (a taskExecutorAdapter) executeWorkflowTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) bridgeTasks.ExecutionResult {
	return workflowadapter.Runner{Config: workflowadapter.Config{
		ServiceAvailable:              a.service != nil,
		ConfigStore:                   a.workflowConfigStore(),
		RuntimeFactory:                a.workflowRuntimeFactory(),
		AgentPreparer:                 a.workflowAgentPreparer(),
		DirectAgent:                   a.workflowDirectAgentRunner(),
		OverrideAgent:                 a.workflowOverrideAgentRunner(),
		RuntimeOverridesRequireStream: a.service != nil && a.service.agentRunner != nil,
		SessionEndParser:              parseSessionEndForStream,
	}}.Execute(ctx, task.Workflow, traceID)
}

func (a taskExecutorAdapter) workflowConfigStore() bridgeconfig.Store {
	if a.service == nil {
		return nil
	}
	return a.service.configStore
}

func (a taskExecutorAdapter) workflowRuntimeFactory() workflowadapter.RuntimeFactory {
	if a.service == nil || a.service.runtimeFactory == nil {
		return nil
	}
	return workflowadapter.RuntimeFactoryFunc(func(store bridgeconfig.Store) (workflowadapter.RuntimeDependencies, error) {
		return a.service.runtimeFactory.Build(store)
	})
}

func (a taskExecutorAdapter) workflowAgentPreparer() workflowadapter.AgentStatePreparer {
	runner := a.sessionAgentRunner()
	if runner == nil {
		return nil
	}
	return workflowadapter.AgentStatePreparerFunc(func(
		ctx context.Context,
		req appworkflows.AgentRequest,
		input llm.Message,
	) (*sessionTurnState, error) {
		return runner.prepareTurnWithRuntimeOverrides(ctx, input, "", req.TraceID, req.RuntimeOverrides)
	})
}

func (a taskExecutorAdapter) workflowDirectAgentRunner() workflowadapter.DirectAgentRunner {
	if a.service == nil || a.service.agentRunner == nil {
		return nil
	}
	return a.service.agentRunner
}

func (a taskExecutorAdapter) workflowOverrideAgentRunner() workflowadapter.OverrideAgentRunner {
	if a.service == nil || a.service.agentRunner == nil {
		return nil
	}
	runner, ok := a.service.agentRunner.(workflowadapter.OverrideAgentRunner)
	if !ok {
		return nil
	}
	return runner
}

func (a taskExecutorAdapter) executeOrchestrationTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) bridgeTasks.ExecutionResult {
	if _, err := (groupdomain.PlanBuilder{}).Build(task.Orchestration); err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	return a.orchestrationRuntimeRunner().Execute(ctx, task.Orchestration, traceID)
}

func (a taskExecutorAdapter) orchestrationRuntimeRunner() orchestrationruntime.Runner {
	preparer := newSessionTurnPreparer(
		a.service.runtimeFactory,
		a.service.configStore,
		a.service.sessionStore,
		a.service.runRegistry,
		nil,
	)
	return orchestrationruntime.Runner{
		Config: orchestrationruntime.Config{
			RuntimeBuilder: orchestrationruntime.RuntimeBuilderFunc(func(
				runtimeOverrides *TaskRuntimeOverrides,
			) (orchestrationruntime.RuntimeDependencies, error) {
				deps, _, _, err := preparer.BuildPrepareDependencies(runtimeOverrides)
				if err != nil {
					return nil, err
				}
				return deps, nil
			}),
			ExecutionPreparer: orchestrationruntime.ExecutionPreparerFunc(preparer.PrepareExecutionContext),
			MemberPreparer: orchestrationruntime.MemberTurnPreparerFunc(func(
				ctx context.Context,
				req ports.AgentActionRequest,
				input llm.Message,
			) (*sessionTurnState, error) {
				runner := a.sessionAgentRunner()
				if runner == nil {
					return nil, errors.New("orchestration member turn preparer is not configured")
				}
				return runner.prepareTurnWithRuntimeOverrides(ctx, input, req.SessionID, req.TraceID, req.RuntimeOverrides)
			}),
			DirectMember:     a.orchestrationDirectMemberRunner(),
			SessionStore:     a.service.sessionStore,
			SessionEndParser: parseSessionEndForStream,
		},
	}
}

func (a taskExecutorAdapter) orchestrationDirectMemberRunner() orchestrationruntime.DirectMemberRunner {
	if a.service == nil || a.service.agentRunner == nil {
		return nil
	}
	return a.service.agentRunner
}

func (a taskExecutorAdapter) sessionAgentRunner() *SessionAgentRunner {
	if a.service == nil {
		return nil
	}
	return NewSessionAgentRunner(
		a.service.runtimeFactory,
		a.service.configStore,
		a.service.sessionStore,
		a.service.runRegistry,
	)
}
