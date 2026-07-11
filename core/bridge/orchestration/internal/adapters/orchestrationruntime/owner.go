package orchestrationruntime

import (
	"context"
	"errors"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	runtimeadapter "ghost-os/bridge/orchestration/internal/adapters/runtime"
	tooladapter "ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	ownerapp "ghost-os/bridge/orchestration/internal/app/orchestrations/owner"
	appsessions "ghost-os/bridge/orchestration/internal/app/sessions"
	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/orchestration/internal/trace/runcards"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/taskdefs"
	"ghost-os/bridge/tools"
)

func (r Runner) RunOwnerDecisionTurn(
	ctx context.Context,
	req ports.OwnerDecisionTurnRequest,
) (groupdomain.DispatchCommand, string, error) {
	return ownerapp.DecisionTurnRunner{
		Runtime:          r,
		Sessions:         r,
		Execution:        r,
		Lifecycle:        r,
		Cards:            r,
		DispatchToolName: tooladapter.DispatchToolName,
	}.Run(ctx, req)
}

func (r Runner) BuildOwnerDecisionRuntime(
	runtimeOverrides *taskdefs.TaskRuntimeOverrides,
) (ownerapp.DecisionRuntimeDependencies, error) {
	if r.Config.RuntimeBuilder == nil {
		return ownerapp.DecisionRuntimeDependencies{}, errors.New("owner dispatch runtime is not configured")
	}
	deps, err := r.Config.RuntimeBuilder.BuildRuntime(runtimeOverrides)
	if err != nil {
		return ownerapp.DecisionRuntimeDependencies{}, err
	}
	return runtimeadapter.ToOwnerDecisionDependencies(deps), nil
}

func (r Runner) LoadOwnerDecisionSession(
	sessionID string,
	deps ownerapp.DecisionRuntimeDependencies,
	systemPrompt string,
) (*session.Session, error) {
	sess, created, err := appsessions.NewHistoryBuilderFromConfig(
		deps.Config,
		systemPrompt,
		r.Config.SessionStore,
		"",
	).LoadOrCreateSession(sessionID)
	if err != nil {
		return nil, err
	}
	if created {
		return sess, r.SaveOwnerDecisionSession(sess)
	}
	return sess, nil
}

func (r Runner) SaveOwnerDecisionSession(sess *session.Session) error {
	if r.Config.SessionStore == nil {
		return nil
	}
	return r.Config.SessionStore.Save(sess)
}

func (r Runner) PrepareOwnerDecisionExecution(
	ctx context.Context,
	sess *session.Session,
	registry *tools.Registry,
	traceID string,
) (context.Context, func(), error) {
	if r.Config.ExecutionPreparer == nil {
		return ctx, func() {}, errors.New("owner dispatch execution context is not configured")
	}
	return r.Config.ExecutionPreparer.PrepareExecutionContext(ctx, sess, registry, traceID)
}

func (r Runner) BuildOwnerDecisionLifecycle(
	sess *session.Session,
) agent.StreamLifecyclePayloadBuilder {
	return internaltrace.NewSessionStreamLifecyclePayloadBuilder(
		func() string {
			if sess == nil {
				return ""
			}
			return sess.ID
		},
		r.Config.SessionEndParser,
	)
}

func (r Runner) StartOwnerDecisionCard(
	ctx context.Context,
	req ports.OwnerDecisionTurnRequest,
	sess *session.Session,
) (ownerapp.DecisionRunCard, error) {
	recorder := runcards.RecorderFromContext(ctx)
	if recorder == nil {
		return nil, errors.New("orchestration owner task run card recorder is not configured")
	}
	title := req.OwnerNode.ID
	if req.OwnerNode.Agent != nil && strings.TrimSpace(req.OwnerNode.Agent.Title) != "" {
		title = strings.TrimSpace(req.OwnerNode.Agent.Title)
	}
	handle, err := recorder.StartCard(ctx, runcards.StartInput{
		Kind:            taskdefs.RunCardKindOrchestrationOwner,
		Title:           title,
		NodeID:          strings.TrimSpace(req.OwnerNode.ID),
		NodeType:        "agent",
		Round:           req.Round,
		SourceSessionID: strings.TrimSpace(sess.ID),
		StartedAt:       time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	return ownerDecisionRunCard{
		handle: handle,
		sink: internaltrace.NewSessionDraftCheckpointSink(
			runcards.NewStreamSink(handle),
			r.Config.SessionStore,
			sess,
		),
	}, nil
}

type ownerDecisionRunCard struct {
	handle *runcards.Handle
	sink   streaming.Sink
}

func (c ownerDecisionRunCard) Sink() streaming.Sink {
	return c.sink
}

func (c ownerDecisionRunCard) Finish(
	ctx context.Context,
	input ownerapp.DecisionRunCardFinishInput,
) error {
	status, preview, errorText := ownerapp.RunCardStatus(input.Response, input.Err)
	return c.handle.Finish(ctx, runcards.FinishInput{
		Status:          status,
		Preview:         preview,
		ErrorText:       errorText,
		SourceSessionID: strings.TrimSpace(input.SessionID),
		FinishedAt:      time.Now().UTC(),
	})
}
