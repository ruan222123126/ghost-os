package orchestration

import (
	"context"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

func runPreparedTurnStream(
	ctx context.Context,
	turn *sessionTurnState,
	input llm.Message,
	sink streaming.Sink,
) (string, string, error) {
	streamSink := newStreamTerminalBuffer(ensureEventSink(sink))
	turn.agent.SetStreamLifecyclePayloadBuilder(newSessionStreamLifecyclePayloadBuilder(turn))
	runSink := newSessionDraftCheckpointSink(streamSink, turn.sessionStore, turn.sess)

	response, runErr := turn.agent.RunMessageStreamWithTraceID(turn.execCtx, input, turn.traceID, runSink)
	response, persistedSessionID, err := turn.complete(response, runErr, func(err error, awaitingHuman bool) error {
		turnNumber := turn.agent.LastTurn()
		stepID, stepErr := streaming.AssistantStepID(turnNumber)
		if stepErr != nil {
			return stepErr
		}
		if awaitingHuman {
			stepID = ""
		}
		return emitStreamErrorEvent(ctx, streamSink, turn.traceID, turnNumber, stepID, turn.currentSessionID(), 0, err)
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
