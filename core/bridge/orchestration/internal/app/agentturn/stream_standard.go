package agentturn

import (
	"context"

	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/streaming"
)

func (s Service) executeStandardStream(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
	sink *eventTurnTracker,
) (string, string, error) {
	s.log(traceID, bus.ActionAgentSend, "running", nil)
	response, sessionID, err := s.Runner.RunTurnStream(ctx, prepared, traceID, sink)
	if err != nil {
		return s.handleStandardStreamError(traceID, sessionID, err)
	}
	return s.completeStandardStream(ctx, traceID, response, sessionID, sink)
}

func (s Service) handleStandardStreamError(
	traceID string,
	sessionID string,
	err error,
) (string, string, error) {
	awaitingErr, _, cancelled, normalizedErr := s.classify(err)
	if awaitingErr != nil {
		s.log(traceID, bus.ActionAgentSend, "awaiting_human", nil)
		s.publishAwaiting(traceID, sessionID, awaitingErr)
		return "", sessionID, err
	}
	if cancelled {
		s.log(traceID, bus.ActionAgentSend, "cancelled", normalizedErr)
		return "", sessionID, normalizedErr
	}
	s.log(traceID, bus.ActionAgentSend, "error", normalizedErr)
	return "", sessionID, normalizedErr
}

func (s Service) completeStandardStream(
	ctx context.Context,
	traceID string,
	response string,
	sessionID string,
	sink *eventTurnTracker,
) (string, string, error) {
	result, err := s.Finalizer.Finalize(response, sessionID)
	if err != nil {
		return s.emitFinalizeStreamError(ctx, traceID, sessionID, sink, err)
	}
	s.publishAssistant(traceID, result)
	s.log(traceID, bus.ActionAgentSend, "success", nil)
	return result.Message, result.SessionID, nil
}

func (s Service) emitFinalizeStreamError(
	ctx context.Context,
	traceID string,
	sessionID string,
	sink *eventTurnTracker,
	err error,
) (string, string, error) {
	s.log(traceID, bus.ActionAgentSend, "error", err)
	stepID, stepErr := streaming.AssistantStepID(sink.finalAssistantTurn())
	if stepErr != nil {
		return "", "", stepErr
	}
	code := bus.StatusFromErrorKind(bus.ErrorKindOf(err))
	if emitErr := emitStreamErrorEvent(ctx, sink, traceID, sink.finalAssistantTurn(), stepID, sessionID, code, err); emitErr != nil {
		return "", "", emitErr
	}
	return "", "", err
}

func emitDirectResult(
	ctx context.Context,
	sink streaming.Sink,
	traceID string,
	turn int,
	result FinalizedTurn,
) error {
	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		return err
	}
	messageEvent, err := streaming.NewEvent(traceID, result.SessionID, turn, stepID, streaming.EventMessage, map[string]any{
		"text":       result.Message,
		"session_id": result.SessionID,
	})
	if err != nil {
		return err
	}
	if err := emitStreamEvent(ctx, sink, messageEvent); err != nil {
		return err
	}
	return emitDirectDone(ctx, sink, traceID, turn, result)
}

func EmitDirectResult(
	ctx context.Context,
	sink streaming.Sink,
	traceID string,
	turn int,
	result FinalizedTurn,
) error {
	return emitDirectResult(ctx, sink, traceID, turn, result)
}

func emitDirectDone(
	ctx context.Context,
	sink streaming.Sink,
	traceID string,
	turn int,
	result FinalizedTurn,
) error {
	doneEvent, err := streaming.NewEvent(traceID, result.SessionID, turn, "", streaming.EventDone, map[string]any{
		"session_id":    result.SessionID,
		"session_ended": result.SessionEnd != nil,
	})
	if err != nil {
		return err
	}
	return emitStreamEvent(ctx, sink, doneEvent)
}
