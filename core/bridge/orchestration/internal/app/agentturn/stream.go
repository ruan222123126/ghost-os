package agentturn

import (
	"context"

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
	return s.executePreparedStream(ctx, prepared, traceID, trackedSink)
}

func (s Service) ExecutePreparedStream(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	return s.executePreparedStream(ctx, prepared, traceID, newEventTurnTracker(sink))
}

func (s Service) executePreparedStream(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
	sink *eventTurnTracker,
) (string, string, error) {
	return s.executeStandardStream(ctx, prepared, traceID, sink)
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
