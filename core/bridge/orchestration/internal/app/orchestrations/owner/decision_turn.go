package owner

import (
	"context"
	"errors"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/taskdefs"
	"ghost-os/bridge/tools"
)

type DecisionRuntimeDependencies struct {
	Config               bridgeconfig.Config
	Client               agent.Completer
	Registry             *tools.Registry
	SystemPrompt         string
	SystemPromptOverride bool
	SystemPromptFiles    *bridgeconfig.SystemPromptFiles
	Cleanup              func()
}

func (d DecisionRuntimeDependencies) Close() {
	if d.Cleanup != nil {
		d.Cleanup()
	}
}

type DecisionRuntimeBuilder interface {
	BuildOwnerDecisionRuntime(
		runtimeOverrides *taskdefs.TaskRuntimeOverrides,
	) (DecisionRuntimeDependencies, error)
}

type DecisionSessionStore interface {
	LoadOwnerDecisionSession(
		sessionID string,
		deps DecisionRuntimeDependencies,
		systemPrompt string,
	) (*session.Session, error)
	SaveOwnerDecisionSession(sess *session.Session) error
}

type DecisionExecutionPreparer interface {
	PrepareOwnerDecisionExecution(
		ctx context.Context,
		sess *session.Session,
		registry *tools.Registry,
		traceID string,
	) (context.Context, func(), error)
}

type DecisionLifecycleBuilder interface {
	BuildOwnerDecisionLifecycle(sess *session.Session) agent.StreamLifecyclePayloadBuilder
}

type DecisionRunCardStarter interface {
	StartOwnerDecisionCard(
		ctx context.Context,
		req ports.OwnerDecisionTurnRequest,
		sess *session.Session,
	) (DecisionRunCard, error)
}

type DecisionRunCard interface {
	Sink() streaming.Sink
	Finish(ctx context.Context, input DecisionRunCardFinishInput) error
}

type DecisionRunCardFinishInput struct {
	SessionID string
	Response  string
	Err       error
}

type DecisionTurnRunner struct {
	Runtime          DecisionRuntimeBuilder
	Sessions         DecisionSessionStore
	Execution        DecisionExecutionPreparer
	Lifecycle        DecisionLifecycleBuilder
	Cards            DecisionRunCardStarter
	DispatchToolName string
}

func (r DecisionTurnRunner) Run(
	ctx context.Context,
	req ports.OwnerDecisionTurnRequest,
) (group.DispatchCommand, string, error) {
	if r.Runtime == nil {
		return group.DispatchCommand{}, req.SessionID, errors.New("owner dispatch runtime is not configured")
	}
	deps, err := r.Runtime.BuildOwnerDecisionRuntime(req.RuntimeOverrides)
	if err != nil {
		return group.DispatchCommand{}, req.SessionID, err
	}
	defer deps.Close()

	systemPrompt, err := BuildDecisionSystemPrompt(
		deps,
		nil,
		BuildRuntimeCatalog(decisionRuntimeCatalogRequest(deps, nil, req)),
		req,
		r.DispatchToolName,
	)
	if err != nil {
		return group.DispatchCommand{}, req.SessionID, err
	}
	sess, err := r.loadSession(req, deps, systemPrompt)
	if err != nil {
		return group.DispatchCommand{}, req.SessionID, err
	}
	card, err := r.startCard(ctx, req, sess)
	if err != nil {
		return group.DispatchCommand{}, sess.ID, err
	}

	catalog := BuildRuntimeCatalog(decisionRuntimeCatalogRequest(deps, sess, req))
	systemPrompt, err = BuildDecisionSystemPrompt(deps, sess, catalog, req, r.DispatchToolName)
	if err != nil {
		return group.DispatchCommand{}, sess.ID, err
	}
	response, runErr := r.runAgent(ctx, req, deps, sess, catalog, systemPrompt, card)
	if err := r.saveSession(sess); err != nil {
		_ = card.Finish(ctx, DecisionRunCardFinishInput{SessionID: sess.ID, Err: err})
		return group.DispatchCommand{}, sess.ID, err
	}
	if err := card.Finish(ctx, DecisionRunCardFinishInput{
		SessionID: sess.ID,
		Response:  response,
		Err:       runErr,
	}); err != nil {
		return group.DispatchCommand{}, sess.ID, err
	}
	return DecodeDecisionResponse(response, runErr, sess.ID)
}

func (r DecisionTurnRunner) loadSession(
	req ports.OwnerDecisionTurnRequest,
	deps DecisionRuntimeDependencies,
	systemPrompt string,
) (*session.Session, error) {
	if r.Sessions == nil {
		return nil, errors.New("owner dispatch session store is not configured")
	}
	return r.Sessions.LoadOwnerDecisionSession(req.SessionID, deps, systemPrompt)
}

func (r DecisionTurnRunner) saveSession(sess *session.Session) error {
	if r.Sessions == nil {
		return errors.New("owner dispatch session store is not configured")
	}
	return r.Sessions.SaveOwnerDecisionSession(sess)
}

func (r DecisionTurnRunner) startCard(
	ctx context.Context,
	req ports.OwnerDecisionTurnRequest,
	sess *session.Session,
) (DecisionRunCard, error) {
	if r.Cards == nil {
		return nil, errors.New("orchestration owner task run card recorder is not configured")
	}
	return r.Cards.StartOwnerDecisionCard(ctx, req, sess)
}

func (r DecisionTurnRunner) runAgent(
	ctx context.Context,
	req ports.OwnerDecisionTurnRequest,
	deps DecisionRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
	systemPrompt string,
	card DecisionRunCard,
) (string, error) {
	if r.Execution == nil {
		return "", errors.New("owner dispatch execution context is not configured")
	}
	runAgent, err := BuildDispatchAgent(DispatchAgentConfig{
		Config:       deps.Config,
		Client:       deps.Client,
		Catalog:      catalog,
		Session:      sess,
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		return "", err
	}
	if r.Lifecycle != nil {
		runAgent.SetStreamLifecyclePayloadBuilder(r.Lifecycle.BuildOwnerDecisionLifecycle(sess))
	}
	agentturn.AttachDynamicPromptRefresh(runAgent, deps.Config, sess, func() (string, error) {
		return systemPrompt, nil
	})
	execCtx, cleanup, err := r.Execution.PrepareOwnerDecisionExecution(ctx, sess, deps.Registry, req.TraceID)
	if err != nil {
		return "", err
	}
	defer cleanup()
	response, runErr := RunDispatchWithRepair(execCtx, runAgent, DispatchRunRequest{
		UserPrompt: req.UserPrompt,
		TraceID:    req.TraceID,
	}, card.Sink())
	PersistAgentMessages(sess, runAgent)
	return response, runErr
}

func BuildDecisionSystemPrompt(
	deps DecisionRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
	req ports.OwnerDecisionTurnRequest,
	dispatchToolName string,
) (string, error) {
	prompt, err := agentturn.BuildCompletionSystemPrompt(agentturn.CompletionPromptRequest{
		Config:               deps.Config,
		Catalog:              catalog,
		Session:              sess,
		SystemPrompt:         deps.SystemPrompt,
		FallbackSystemPrompt: deps.SystemPrompt,
		SystemPromptOverride: deps.SystemPromptOverride,
		SystemPromptFiles:    deps.SystemPromptFiles,
	})
	if err != nil {
		return "", err
	}
	return BuildControlPrompt(ControlPromptRequest{
		BasePrompt:       prompt,
		OwnerNode:        req.OwnerNode,
		GroupNode:        req.GroupNode,
		MemberNodes:      req.MemberNodes,
		MemberOrder:      append([]string(nil), req.MemberOrder...),
		PublicTranscript: req.PublicTranscript,
		LastDispatch:     req.LastDispatch,
		Round:            req.Round,
		DispatchToolName: dispatchToolName,
	}), nil
}

func decisionRuntimeCatalogRequest(
	deps DecisionRuntimeDependencies,
	sess *session.Session,
	req ports.OwnerDecisionTurnRequest,
) RuntimeCatalogRequest {
	return RuntimeCatalogRequest{
		Config:      deps.Config,
		Registry:    deps.Registry,
		Session:     sess,
		GroupNode:   req.GroupNode,
		MemberOrder: req.MemberOrder,
	}
}
