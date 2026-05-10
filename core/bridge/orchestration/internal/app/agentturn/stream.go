package agentturn

import (
	"context"
	"errors"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/streaming"
)

func (s Service) ExecuteStream(
	ctx context.Context,
	params api.AgentParams,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	trackedSink := newEventTurnTracker(sink)
	prepared, err := s.validateStreamRequest(ctx, params, traceID, trackedSink)
	if err != nil {
		return "", "", err
	}
	if prepared.Mode == ModePlan {
		return s.executePlanStream(ctx, prepared, traceID, trackedSink)
	}
	return s.executeStandardStream(ctx, prepared, traceID, trackedSink)
}

func (s Service) validateStreamRequest(
	ctx context.Context,
	params api.AgentParams,
	traceID string,
	sink *eventTurnTracker,
) (PreparedRequest, error) {
	prepared, err := s.Prepare(params, nil, traceID)
	if err == nil {
		return prepared, nil
	}
	if emitErr := emitStreamErrorEvent(
		ctx,
		sink,
		traceID,
		0,
		"",
		params.SessionID,
		bus.StatusFromErrorKind(bus.ErrorKindOf(err)),
		err,
	); emitErr != nil {
		return PreparedRequest{}, emitErr
	}
	return PreparedRequest{}, err
}

func (s Service) executePlanStream(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
	sink *eventTurnTracker,
) (string, string, error) {
	if s.Special == nil {
		return "", prepared.SessionID, bus.WrapError(bus.ServiceErrorInternal, errors.New("special mode runner is not configured"))
	}
	return s.executeSpecialStream(ctx, prepared, traceID, sink, s.Special.RunPlan, false)
}

func (s Service) executeProStream(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
	sink *eventTurnTracker,
) (string, string, error) {
	if s.Special == nil {
		return "", prepared.SessionID, bus.WrapError(bus.ServiceErrorInternal, errors.New("special mode runner is not configured"))
	}
	return s.executeSpecialStream(ctx, prepared, traceID, sink, s.Special.RunPro, true)
}

func (s Service) executeSpecialStream(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
	sink *eventTurnTracker,
	run func(context.Context, PreparedRequest, string) (api.AgentResponse, int, error),
	pro bool,
) (string, string, error) {
	s.log(traceID, bus.ActionAgentSend, "running", nil)
	payload, code, err := run(ctx, prepared, traceID)
	if err != nil {
		return s.handleSpecialStreamError(ctx, prepared, traceID, sink, code, err, pro)
	}
	turn := FinalizedTurn{Message: payload.Message, SessionID: payload.SessionID, SessionEnd: payload.SessionEnd}
	if emitErr := emitDirectResult(ctx, sink, traceID, 0, turn); emitErr != nil {
		return "", "", emitErr
	}
	s.publishAssistant(traceID, turn)
	s.log(traceID, bus.ActionAgentSend, "success", nil)
	return turn.Message, turn.SessionID, nil
}

func (s Service) handleSpecialStreamError(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
	sink *eventTurnTracker,
	code int,
	err error,
	pro bool,
) (string, string, error) {
	kind, cancelled, normalizedErr := normalizeSpecialStreamError(s, err)
	statusCode := code
	if statusCode <= 0 {
		statusCode = bus.StatusFromErrorKind(kind)
	}
	if cancelled {
		s.log(traceID, bus.ActionAgentSend, "cancelled", normalizedErr)
		return "", prepared.SessionID, normalizedErr
	}
	if pro && kind != bus.ServiceErrorInternal {
		err = normalizedErr
	}
	s.log(traceID, bus.ActionAgentSend, "error", err)
	if emitErr := emitStreamErrorEvent(ctx, sink, traceID, 0, "", prepared.SessionID, statusCode, err); emitErr != nil {
		return "", "", emitErr
	}
	return "", prepared.SessionID, err
}

func normalizeSpecialStreamError(s Service, err error) (bus.ServiceErrorKind, bool, error) {
	_, kind, cancelled, normalizedErr := s.classify(err)
	if kind == "" {
		return bus.ServiceErrorInternal, false, err
	}
	return kind, cancelled, normalizedErr
}
