package workflow

import (
	"context"
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	runtimeadapter "ghost-os/bridge/orchestration/internal/adapters/runtime"
	"ghost-os/bridge/orchestration/internal/app/agentturn/turnstate"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	workflowagent "ghost-os/bridge/orchestration/internal/app/workflows/agentnode"
	workflowtask "ghost-os/bridge/orchestration/internal/app/workflows/taskrun"
	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/taskdefs"
)

type RuntimeDependencies = runtimeadapter.Dependencies

type RuntimeFactory interface {
	Build(store bridgeconfig.Store) (RuntimeDependencies, error)
}

type RuntimeFactoryFunc func(bridgeconfig.Store) (RuntimeDependencies, error)

func (f RuntimeFactoryFunc) Build(store bridgeconfig.Store) (RuntimeDependencies, error) {
	return f(store)
}

type AgentStatePreparer interface {
	PrepareWorkflowAgentState(
		ctx context.Context,
		req appworkflows.AgentRequest,
		input llm.Message,
	) (*turnstate.State, error)
}

type AgentStatePreparerFunc func(
	ctx context.Context,
	req appworkflows.AgentRequest,
	input llm.Message,
) (*turnstate.State, error)

func (f AgentStatePreparerFunc) PrepareWorkflowAgentState(
	ctx context.Context,
	req appworkflows.AgentRequest,
	input llm.Message,
) (*turnstate.State, error) {
	return f(ctx, req, input)
}

type DirectAgentRunner interface {
	RunTurnStream(
		ctx context.Context,
		message string,
		sessionID string,
		traceID string,
		sink streaming.Sink,
	) (string, string, error)
}

type OverrideAgentRunner interface {
	RunTurnStreamWithOverrides(
		ctx context.Context,
		message string,
		sessionID string,
		traceID string,
		sink streaming.Sink,
		runtimeOverrides *taskdefs.TaskRuntimeOverrides,
	) (string, string, error)
}

type Config struct {
	ServiceAvailable              bool
	ConfigStore                   bridgeconfig.Store
	RuntimeFactory                RuntimeFactory
	AgentPreparer                 AgentStatePreparer
	DirectAgent                   DirectAgentRunner
	OverrideAgent                 OverrideAgentRunner
	RuntimeOverridesRequireStream bool
	SessionEndParser              internaltrace.ParseSessionEndFunc
}

type Runner struct {
	Config Config
}

func (r Runner) Execute(
	ctx context.Context,
	definition *taskdefs.WorkflowDefinition,
	traceID string,
) taskdefs.ExecutionResult {
	return workflowtask.Runner{}.Execute(ctx, workflowtask.Command{
		Definition:       definition,
		TraceID:          traceID,
		ServiceAvailable: r.Config.ServiceAvailable,
		Planner:          workflowdomain.PlanBuilder{},
		Runtime:          r,
		Validator:        r,
		Agent:            r.executeAgent,
		Cards:            appworkflows.NewRunCardObserverFromContext(ctx),
		TemplateUploader: TemplateUploader{},
	})
}

func (r Runner) ValidateWorkflowRuntime(definition *taskdefs.WorkflowDefinition) error {
	if !r.Config.ServiceAvailable {
		return nil
	}
	cfg, err := apptasks.LoadRuntimeConfig(r.Config.ConfigStore)
	if err != nil {
		return err
	}
	if err := apptasks.ValidateWorkflowRuntime(definition, cfg); err != nil {
		return err
	}
	return apptasks.ValidateWorkflowAgentRuntime(definition, r.Config.ConfigStore)
}

func (r Runner) LoadWorkflowRuntime(
	plan workflowdomain.Plan,
) (appworkflows.RuntimeDependencies, error) {
	if !plan.NeedsRuntimeDependencies() {
		return appworkflows.RuntimeDependencies{}, nil
	}
	if r.Config.RuntimeFactory == nil {
		return appworkflows.RuntimeDependencies{}, errors.New("workflow runtime service is not configured")
	}
	deps, err := r.Config.RuntimeFactory.Build(r.Config.ConfigStore)
	if err != nil {
		return appworkflows.RuntimeDependencies{}, err
	}
	return runtimeadapter.ToWorkflowDependencies(deps), nil
}

func (r Runner) executeAgent(
	ctx context.Context,
	req appworkflows.AgentRequest,
) taskdefs.ExecutionResult {
	return workflowagent.Runner{
		Preparer:                      r,
		Direct:                        r.directRunner(),
		Override:                      r.overrideRunner(),
		Cards:                         workflowagent.RunCardRecorder{},
		Results:                       workflowAgentResultMapper{},
		RuntimeOverridesRequireStream: r.Config.RuntimeOverridesRequireStream,
	}.Execute(ctx, req)
}

func (r Runner) directRunner() workflowagent.DirectRunner {
	if r.Config.DirectAgent == nil {
		return nil
	}
	return r
}

func (r Runner) overrideRunner() workflowagent.OverrideRunner {
	if r.Config.OverrideAgent == nil {
		return nil
	}
	return r
}

func (r Runner) PrepareWorkflowAgentTurn(
	ctx context.Context,
	req appworkflows.AgentRequest,
	input llm.Message,
) (workflowagent.PreparedTurn, error) {
	if r.Config.AgentPreparer == nil {
		return nil, errors.New("workflow agent runner is not configured")
	}
	state, err := r.Config.AgentPreparer.PrepareWorkflowAgentState(ctx, req, input)
	if err != nil {
		return nil, err
	}
	return preparedWorkflowAgentTurn{
		state:  state,
		parser: r.Config.SessionEndParser,
	}, nil
}

func (r Runner) RunDirectWorkflowAgent(
	ctx context.Context,
	req appworkflows.AgentRequest,
	sink streaming.Sink,
) (string, string, error) {
	return r.Config.DirectAgent.RunTurnStream(ctx, req.Message, "", req.TraceID, sink)
}

func (r Runner) RunWorkflowAgentWithOverrides(
	ctx context.Context,
	req appworkflows.AgentRequest,
	sink streaming.Sink,
) (string, string, error) {
	return r.Config.OverrideAgent.RunTurnStreamWithOverrides(
		ctx,
		req.Message,
		"",
		req.TraceID,
		sink,
		req.RuntimeOverrides,
	)
}

type preparedWorkflowAgentTurn struct {
	state  *turnstate.State
	parser internaltrace.ParseSessionEndFunc
}

func (t preparedWorkflowAgentTurn) CurrentSessionID() string {
	return t.state.CurrentSessionID()
}

func (t preparedWorkflowAgentTurn) Run(
	ctx context.Context,
	input llm.Message,
	sink streaming.Sink,
) (string, string, error) {
	return turnstate.RunPreparedTurnStream(ctx, t.state, input, sink, t.parser)
}

func (t preparedWorkflowAgentTurn) Close() {
	t.state.Close()
}

type workflowAgentResultMapper struct{}

func (workflowAgentResultMapper) FromWorkflowAgentStream(
	ctx context.Context,
	response string,
	sessionID string,
	runErr error,
) taskdefs.ExecutionResult {
	return apptasks.ExecutionResultFromStreamOutcome(ctx, response, sessionID, runErr)
}
