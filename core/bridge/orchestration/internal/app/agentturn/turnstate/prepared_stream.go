package turnstate

import (
	"context"
	"errors"

	"ghost-os/bridge/llm"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/streaming"
)

type SessionEndParser = internaltrace.ParseSessionEndFunc

func RunPreparedTurnStream(
	ctx context.Context,
	turn *State,
	input llm.Message,
	sink streaming.Sink,
	parseSessionEnd SessionEndParser,
) (string, string, error) {
	if turn == nil || turn.Agent == nil {
		return "", "", errors.New("prepared turn is not configured")
	}
	if parseSessionEnd == nil {
		return "", turn.CurrentSessionID(), errors.New("session end parser is not configured")
	}

	streamSink := internaltrace.NewStreamTerminalBuffer(internaltrace.EnsureEventSink(sink))
	turn.Agent.SetStreamLifecyclePayloadBuilder(
		internaltrace.NewSessionStreamLifecyclePayloadBuilder(turn.CurrentSessionID, parseSessionEnd),
	)
	runSink := internaltrace.NewSessionDraftCheckpointSink(streamSink, turn.SessionStore, turn.Session)

	response, runErr := turn.Agent.RunMessageStreamWithTraceID(turn.ExecCtx, input, turn.TraceID, runSink)
	response, persistedSessionID, err := turn.Complete(response, runErr, func(err error, awaitingHuman bool) error {
		turnNumber := turn.Agent.LastTurn()
		stepID, stepErr := streaming.AssistantStepID(turnNumber)
		if stepErr != nil {
			return stepErr
		}
		if awaitingHuman {
			stepID = ""
		}
		return internaltrace.EmitStreamErrorEvent(ctx, streamSink, turn.TraceID, turnNumber, stepID, turn.CurrentSessionID(), 0, err)
	})
	if err != nil {
		streamSink.Discard()
		return "", persistedSessionID, err
	}
	if emitErr := streamSink.Flush(ctx); emitErr != nil {
		return "", "", emitErr
	}
	return response, persistedSessionID, nil
}
