package orchestrationruntime

import (
	"context"
	"errors"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	agentadapter "ghost-os/bridge/orchestration/internal/adapters/agent"
	runtimeadapter "ghost-os/bridge/orchestration/internal/adapters/runtime"
	"ghost-os/bridge/orchestration/internal/app/agentturn/turnstate"
	apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"
	memberapp "ghost-os/bridge/orchestration/internal/app/orchestrations/member"
	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/orchestration/internal/trace/runcards"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/taskdefs"
	"ghost-os/bridge/tools"
)

type RuntimeDependencies = runtimeadapter.Dependencies

type RuntimeBuilder interface {
	BuildRuntime(runtimeOverrides *taskdefs.TaskRuntimeOverrides) (RuntimeDependencies, error)
}

type RuntimeBuilderFunc func(*taskdefs.TaskRuntimeOverrides) (RuntimeDependencies, error)

func (f RuntimeBuilderFunc) BuildRuntime(
	runtimeOverrides *taskdefs.TaskRuntimeOverrides,
) (RuntimeDependencies, error) {
	return f(runtimeOverrides)
}

type ExecutionPreparer interface {
	PrepareExecutionContext(
		ctx context.Context,
		sess *session.Session,
		registry *tools.Registry,
		traceID string,
	) (context.Context, func(), error)
}

type ExecutionPreparerFunc func(
	ctx context.Context,
	sess *session.Session,
	registry *tools.Registry,
	traceID string,
) (context.Context, func(), error)

func (f ExecutionPreparerFunc) PrepareExecutionContext(
	ctx context.Context,
	sess *session.Session,
	registry *tools.Registry,
	traceID string,
) (context.Context, func(), error) {
	return f(ctx, sess, registry, traceID)
}

type MemberTurnPreparer interface {
	PrepareMemberActionState(
		ctx context.Context,
		req ports.AgentActionRequest,
		input llm.Message,
	) (*turnstate.State, error)
}

type MemberTurnPreparerFunc func(
	ctx context.Context,
	req ports.AgentActionRequest,
	input llm.Message,
) (*turnstate.State, error)

func (f MemberTurnPreparerFunc) PrepareMemberActionState(
	ctx context.Context,
	req ports.AgentActionRequest,
	input llm.Message,
) (*turnstate.State, error) {
	return f(ctx, req, input)
}

type DirectMemberRunner interface {
	RunTurnStream(
		ctx context.Context,
		message string,
		sessionID string,
		traceID string,
		sink streaming.Sink,
	) (string, string, error)
}

type Config struct {
	RuntimeBuilder    RuntimeBuilder
	ExecutionPreparer ExecutionPreparer
	MemberPreparer    MemberTurnPreparer
	DirectMember      DirectMemberRunner
	SessionStore      *session.Store
	SessionEndParser  internaltrace.ParseSessionEndFunc
}

type Runner struct {
	Config Config
}

func (r Runner) Execute(
	ctx context.Context,
	definition *taskdefs.OrchestrationDefinition,
	traceID string,
) taskdefs.ExecutionResult {
	runner := apporchestrations.NewRunner(apporchestrations.RunnerConfig{
		Planner: groupdomain.PlanBuilder{},
		Members: r.memberRunner(),
		Owners:  r.ownerDecisionRunner(),
	})
	return runner.Execute(ctx, apporchestrations.ExecuteCommand{
		Definition: definition,
		TraceID:    traceID,
	})
}

func (r Runner) memberRunner() ports.MemberAgentRunner {
	return apporchestrations.MemberRunner{
		Executor: agentadapter.MemberAgentRunner{
			Invoker: r,
		},
	}
}

func (r Runner) ownerDecisionRunner() ports.OwnerDecisionRunner {
	return agentadapter.OwnerDecisionRunner{Executor: r}
}

func (r Runner) ExecuteAgentAction(
	ctx context.Context,
	req ports.AgentActionRequest,
) (ports.AgentActionPayload, error) {
	return memberapp.ActionRunner{
		Preparer: r,
		Direct:   r.directMemberActionRunner(),
		Cards:    r,
	}.Run(ctx, req)
}

func (r Runner) directMemberActionRunner() memberapp.DirectActionRunner {
	if r.Config.DirectMember == nil {
		return nil
	}
	return r
}

func (r Runner) PrepareMemberActionTurn(
	ctx context.Context,
	req ports.AgentActionRequest,
	input llm.Message,
) (memberapp.PreparedActionTurn, error) {
	if r.Config.MemberPreparer == nil {
		return nil, errors.New("orchestration member turn preparer is not configured")
	}
	state, err := r.Config.MemberPreparer.PrepareMemberActionState(ctx, req, input)
	if err != nil {
		return nil, err
	}
	return preparedMemberActionTurn{
		state:  state,
		parser: r.Config.SessionEndParser,
	}, nil
}

func (r Runner) RunDirectMemberAction(
	ctx context.Context,
	req ports.AgentActionRequest,
	sink streaming.Sink,
) (string, string, error) {
	if r.Config.DirectMember == nil {
		return "", strings.TrimSpace(req.SessionID), errors.New("orchestration member direct runner is not configured")
	}
	return r.Config.DirectMember.RunTurnStream(ctx, req.Message, req.SessionID, req.TraceID, sink)
}

func (r Runner) StartMemberActionCard(
	ctx context.Context,
	req ports.AgentActionRequest,
	sourceSessionID string,
) (memberapp.ActionRunCard, error) {
	recorder := runcards.RecorderFromContext(ctx)
	if recorder == nil {
		return nil, errors.New("orchestration task run card recorder is not configured")
	}
	handle, err := recorder.StartCard(ctx, runcards.StartInput{
		Kind:            taskdefs.RunCardKindOrchestrationMember,
		Title:           req.Title,
		NodeID:          req.AgentID,
		NodeType:        "agent",
		Round:           req.Round,
		SourceSessionID: sourceSessionID,
		StartedAt:       time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	return memberActionRunCard{handle: handle}, nil
}

type preparedMemberActionTurn struct {
	state  *turnstate.State
	parser internaltrace.ParseSessionEndFunc
}

func (t preparedMemberActionTurn) CurrentSessionID() string {
	return t.state.CurrentSessionID()
}

func (t preparedMemberActionTurn) Run(
	ctx context.Context,
	input llm.Message,
	sink streaming.Sink,
) (string, string, error) {
	return turnstate.RunPreparedTurnStream(ctx, t.state, input, sink, t.parser)
}

func (t preparedMemberActionTurn) Close() {
	t.state.Close()
}

type memberActionRunCard struct {
	handle *runcards.Handle
}

func (c memberActionRunCard) Sink() streaming.Sink {
	return runcards.NewStreamSink(c.handle)
}

func (c memberActionRunCard) Finish(
	ctx context.Context,
	input memberapp.ActionRunCardFinishInput,
) error {
	return c.handle.Finish(ctx, runcards.FinishInput{
		Status:          input.Status,
		Preview:         input.Preview,
		ErrorText:       input.ErrorText,
		SourceSessionID: input.SourceSessionID,
		FinishedAt:      time.Now().UTC(),
	})
}
