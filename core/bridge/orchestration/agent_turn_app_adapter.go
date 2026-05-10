package orchestration

import (
	"context"
	"errors"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/streaming"
)

func (s *bridgeService) agentTurnService() agentturn.Service {
	return agentturn.Service{
		Guards:     s.agentTurnGuards(),
		Runner:     agentTurnRunnerAdapter{service: s},
		Special:    agentTurnSpecialAdapter{service: s},
		Finalizer:  agentTurnFinalizer{service: s},
		Publisher:  agentTurnPublisher{service: s},
		Classifier: agentTurnClassifier{},
		Logger:     agentTurnLogger{},
		Stopper:    s.agentTurnStopper(),
	}
}

func (s *bridgeService) agentTurnStopper() agentturn.RunStopper {
	if s == nil || s.runRegistry == nil {
		return nil
	}
	return agentTurnStopper{registry: s.runRegistry}
}

func (s *bridgeService) agentTurnGuards() agentturn.SessionGuards {
	return agentTurnGuards{service: s}
}

type agentTurnGuards struct {
	service *bridgeService
}

func (g agentTurnGuards) EnsureSessionNotInflight(sessionID string) error {
	if g.service == nil {
		return nil
	}
	return g.service.ensureSessionNotInflight(sessionID)
}

func (g agentTurnGuards) EnsureSessionActive(sessionID string) error {
	if g.service == nil {
		return nil
	}
	return g.service.ensureSessionActive(sessionID)
}

type agentTurnRunnerAdapter struct {
	service *bridgeService
}

func (r agentTurnRunnerAdapter) RunTurn(
	ctx context.Context,
	req agentturn.PreparedRequest,
	traceID string,
) (string, string, error) {
	return r.service.runPreparedAgentTurn(ctx, req, traceID)
}

func (r agentTurnRunnerAdapter) RunTurnStream(
	ctx context.Context,
	req agentturn.PreparedRequest,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	return r.service.runPreparedAgentTurnStream(ctx, req, traceID, sink)
}

type agentTurnSpecialAdapter struct {
	service *bridgeService
}

func (r agentTurnSpecialAdapter) RunPlan(
	ctx context.Context,
	req agentturn.PreparedRequest,
	traceID string,
) (api.AgentResponse, int, error) {
	return r.service.executePlanModeAction(ctx, req, traceID)
}

func (r agentTurnSpecialAdapter) RunPro(
	ctx context.Context,
	req agentturn.PreparedRequest,
	traceID string,
) (api.AgentResponse, int, error) {
	return r.service.executeProModeAction(ctx, req, traceID)
}

type agentTurnFinalizer struct {
	service *bridgeService
}

func (f agentTurnFinalizer) Finalize(response string, sessionID string) (agentturn.FinalizedTurn, error) {
	result, err := f.service.finalizeAgentTurn(response, sessionID)
	return finalizedTurnToApp(result), err
}

func (f agentTurnFinalizer) NewResponsePayload(
	turn agentturn.FinalizedTurn,
	meta agentturn.ResponseMeta,
) (api.AgentResponse, error) {
	return newAgentResponsePayload(turn.Message, turn.SessionID, turn.SessionEnd, agentResponseMeta{
		Mode:             meta.Mode,
		IterationCount:   meta.IterationCount,
		StoppedBy:        meta.StoppedBy,
		FinalChangeLog:   meta.FinalChangeLog,
		IterationSummary: meta.IterationSummary,
	})
}

type agentTurnPublisher struct {
	service *bridgeService
}

func (p agentTurnPublisher) PublishAssistant(traceID string, turn agentturn.FinalizedTurn) {
	p.service.publishAssistantSessionPush(traceID, finalizedTurnFromApp(turn))
}

func (p agentTurnPublisher) PublishAwaitingHuman(
	traceID string,
	sessionID string,
	awaitingErr *agent.ErrAwaitingHuman,
) {
	p.service.publishAwaitingHumanSessionPush(traceID, sessionID, awaitingErr)
}

type agentTurnClassifier struct{}

func (agentTurnClassifier) Classify(err error) (*agent.ErrAwaitingHuman, ServiceErrorKind, bool, error) {
	return classifyAgentTurnError(err)
}

type agentTurnLogger struct{}

func (agentTurnLogger) Log(traceID string, action string, status string, err error) {
	logAction(traceID, action, status, err)
}

type agentTurnStopper struct {
	registry *RunRegistry
}

func (s agentTurnStopper) CancelAndWaitBySessionID(ctx context.Context, sessionID string) (agentturn.StopHandle, error) {
	handle, err := s.registry.CancelAndWaitBySessionID(ctx, sessionID)
	return stopHandleFromRun(handle), mapStopError(err)
}

func (s agentTurnStopper) CancelAndWaitByTraceID(ctx context.Context, traceID string) (agentturn.StopHandle, error) {
	handle, err := s.registry.CancelAndWaitByTraceID(ctx, traceID)
	return stopHandleFromRun(handle), mapStopError(err)
}

func mapStopError(err error) error {
	if errors.Is(err, ErrRunNotFound) {
		return agentturn.ErrRunNotFound
	}
	return err
}

func stopHandleFromRun(handle *RunHandle) agentturn.StopHandle {
	if handle == nil {
		return agentturn.StopHandle{}
	}
	return agentturn.StopHandle{SessionID: handle.SessionID}
}

func finalizedTurnToApp(result finalizedAgentTurn) agentturn.FinalizedTurn {
	return agentturn.FinalizedTurn{
		Message:    result.message,
		SessionID:  result.sessionID,
		SessionEnd: result.sessionEnd,
	}
}

func finalizedTurnFromApp(result agentturn.FinalizedTurn) finalizedAgentTurn {
	return finalizedAgentTurn{
		message:    result.Message,
		sessionID:  result.SessionID,
		sessionEnd: result.SessionEnd,
	}
}
