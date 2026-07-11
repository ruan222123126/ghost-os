package agentturnservice

import (
	"context"
	"errors"
	"net/http"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	runtimeadapter "ghost-os/bridge/orchestration/internal/adapters/runtime"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	agentturnplan "ghost-os/bridge/orchestration/internal/app/agentturn/plan"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

var (
	errRunnerRequired    = errors.New("agent turn runner is not configured")
	errFinalizerRequired = errors.New("agent turn finalizer is not configured")
	errStopperRequired   = errors.New("agent turn stopper is not configured")
)

type RuntimeDependencies = runtimeadapter.Dependencies

type RuntimeBuilder func(bridgeconfig.Store) (RuntimeDependencies, error)

type RunTurnFunc func(context.Context, agentturn.PreparedRequest, string) (string, string, error)

type RunTurnStreamFunc func(
	context.Context,
	agentturn.PreparedRequest,
	string,
	streaming.Sink,
) (string, string, error)

type FinalizeFunc func(string, string) (agentturn.FinalizedTurn, error)

type NewResponsePayloadFunc func(agentturn.FinalizedTurn, agentturn.ResponseMeta) (api.AgentResponse, error)

type PublishAssistantFunc func(string, agentturn.FinalizedTurn)

type PublishAwaitingHumanFunc func(string, string, *agent.ErrAwaitingHuman)

type ClassifyFunc func(error) (*agent.ErrAwaitingHuman, bus.ServiceErrorKind, bool, error)

type LogFunc func(string, string, string, error)

type CancelFunc func(context.Context, string) (agentturn.StopHandle, error)

type Config struct {
	EnsureSessionNotInflight func(string) error
	EnsureSessionActive      func(string) error
	RunTurn                  RunTurnFunc
	RunTurnStream            RunTurnStreamFunc
	Plan                     PlanConfig
	Finalize                 FinalizeFunc
	NewResponsePayload       NewResponsePayloadFunc
	PublishAssistant         PublishAssistantFunc
	PublishAwaitingHuman     PublishAwaitingHumanFunc
	Classify                 ClassifyFunc
	Log                      LogFunc
	Stop                     StopConfig
}

type PlanConfig struct {
	RuntimeBuilder RuntimeBuilder
	ConfigStore    bridgeconfig.Store
	SessionStore   *session.Store
	RunRegistry    agentturnplan.RunRegistry
}

type StopConfig struct {
	CancelAndWaitBySessionID CancelFunc
	CancelAndWaitByTraceID   CancelFunc
}

func New(config Config) agentturn.Service {
	service := agentturn.Service{
		Guards:     guards{config: config},
		Special:    planRunner{config: config.Plan},
		Finalizer:  finalizer{config: config},
		Publisher:  publisher{config: config},
		Classifier: classifier{classify: config.Classify},
		Logger:     logger{log: config.Log},
	}
	if config.RunTurn != nil || config.RunTurnStream != nil {
		service.Runner = runner{config: config}
	}
	if config.Stop.CancelAndWaitBySessionID != nil || config.Stop.CancelAndWaitByTraceID != nil {
		service.Stopper = stopper{config: config.Stop}
	}
	return service
}

type guards struct {
	config Config
}

func (g guards) EnsureSessionNotInflight(sessionID string) error {
	if g.config.EnsureSessionNotInflight == nil {
		return nil
	}
	return g.config.EnsureSessionNotInflight(sessionID)
}

func (g guards) EnsureSessionActive(sessionID string) error {
	if g.config.EnsureSessionActive == nil {
		return nil
	}
	return g.config.EnsureSessionActive(sessionID)
}

type runner struct {
	config Config
}

func (r runner) RunTurn(
	ctx context.Context,
	req agentturn.PreparedRequest,
	traceID string,
) (string, string, error) {
	if r.config.RunTurn == nil {
		return "", "", errRunnerRequired
	}
	return r.config.RunTurn(ctx, req, traceID)
}

func (r runner) RunTurnStream(
	ctx context.Context,
	req agentturn.PreparedRequest,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	if r.config.RunTurnStream != nil {
		return r.config.RunTurnStream(ctx, req, traceID, sink)
	}
	return r.RunTurn(ctx, req, traceID)
}

type planRunner struct {
	config PlanConfig
}

func (r planRunner) RunPlan(
	ctx context.Context,
	req agentturn.PreparedRequest,
	traceID string,
) (api.AgentResponse, int, error) {
	result, err := r.runner().Execute(ctx, agentturnplan.Request{
		UserInput:      req.UserInput,
		SessionID:      req.SessionID,
		RequestRuntime: req.RequestRuntime,
	}, traceID)
	if err != nil {
		return api.AgentResponse{}, agentturnplan.ErrorStatus(err), err
	}
	payload, err := agentturn.NewResponsePayload(result.Message, result.SessionID, nil, agentturn.ResponseMeta{
		Mode: agentturn.ModePlan,
	})
	if err != nil {
		return api.AgentResponse{}, http.StatusInternalServerError, err
	}
	return payload, http.StatusOK, nil
}

func (r planRunner) runner() agentturnplan.Runner {
	return agentturnplan.Runner{
		RuntimeFactory: planRuntimeFactory{build: r.config.RuntimeBuilder},
		ConfigStore:    r.config.ConfigStore,
		SessionStore:   r.config.SessionStore,
		RunRegistry:    r.config.RunRegistry,
	}
}

type planRuntimeFactory struct {
	build RuntimeBuilder
}

func (f planRuntimeFactory) Build(store bridgeconfig.Store) (agentturnplan.RuntimeDependencies, error) {
	if f.build == nil {
		return agentturnplan.RuntimeDependencies{}, agentturnplan.ErrRuntimeFactoryRequired
	}
	deps, err := f.build(store)
	if err != nil {
		return agentturnplan.RuntimeDependencies{}, err
	}
	return runtimeadapter.ToPlanDependencies(deps), nil
}

type finalizer struct {
	config Config
}

func (f finalizer) Finalize(response string, sessionID string) (agentturn.FinalizedTurn, error) {
	if f.config.Finalize == nil {
		return agentturn.FinalizedTurn{}, errFinalizerRequired
	}
	return f.config.Finalize(response, sessionID)
}

func (f finalizer) NewResponsePayload(
	turn agentturn.FinalizedTurn,
	meta agentturn.ResponseMeta,
) (api.AgentResponse, error) {
	if f.config.NewResponsePayload != nil {
		return f.config.NewResponsePayload(turn, meta)
	}
	return agentturn.NewResponsePayload(turn.Message, turn.SessionID, turn.SessionEnd, meta)
}

type publisher struct {
	config Config
}

func (p publisher) PublishAssistant(traceID string, turn agentturn.FinalizedTurn) {
	if p.config.PublishAssistant != nil {
		p.config.PublishAssistant(traceID, turn)
	}
}

func (p publisher) PublishAwaitingHuman(
	traceID string,
	sessionID string,
	awaitingErr *agent.ErrAwaitingHuman,
) {
	if p.config.PublishAwaitingHuman != nil {
		p.config.PublishAwaitingHuman(traceID, sessionID, awaitingErr)
	}
}

type classifier struct {
	classify ClassifyFunc
}

func (c classifier) Classify(err error) (*agent.ErrAwaitingHuman, bus.ServiceErrorKind, bool, error) {
	if c.classify == nil {
		return nil, bus.ServiceErrorInternal, false, err
	}
	return c.classify(err)
}

type logger struct {
	log LogFunc
}

func (l logger) Log(traceID string, action string, status string, err error) {
	if l.log != nil {
		l.log(traceID, action, status, err)
	}
}

type stopper struct {
	config StopConfig
}

func (s stopper) CancelAndWaitBySessionID(ctx context.Context, sessionID string) (agentturn.StopHandle, error) {
	if s.config.CancelAndWaitBySessionID == nil {
		return agentturn.StopHandle{}, errStopperRequired
	}
	return s.config.CancelAndWaitBySessionID(ctx, sessionID)
}

func (s stopper) CancelAndWaitByTraceID(ctx context.Context, traceID string) (agentturn.StopHandle, error) {
	if s.config.CancelAndWaitByTraceID == nil {
		return agentturn.StopHandle{}, errStopperRequired
	}
	return s.config.CancelAndWaitByTraceID(ctx, traceID)
}
