package orchestration

import (
	"context"
	"fmt"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/streaming"
)

// parseSessionEndSignal 只识别完整会话结束信号；普通文本/普通 JSON 均按普通回复返回。
func parseSessionEndSignal(raw string) (message string, signal *assistantSessionEndSignalPayload, err error) {
	return agentturn.ParseSessionEndSignal(raw)
}

func (s *bridgeService) executeSessionSourcesAction(traceID string) (ServiceResult, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		logAction(traceID, internaltrace.ActionSessionSources, "error", err)
		return ServiceResult{}, wrapServiceError(serviceErrorKindFromStatus(code), err)
	}

	tasks, err := loadSessionSourceTasks(store)
	if err != nil {
		logAction(traceID, internaltrace.ActionSessionSources, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	resolution, err := internaltrace.BuildSessionSourceResolution(tasks)
	if err != nil {
		logAction(traceID, internaltrace.ActionSessionSources, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	logAction(traceID, internaltrace.ActionSessionSources, "success", nil)
	return serviceResultSuccess(resolution), nil
}

func loadSessionSourceTasks(store *TaskStore) ([]internaltrace.SessionSourceTask, error) {
	tasks, err := store.ListTasks()
	if err != nil {
		return nil, err
	}
	sources := make([]internaltrace.SessionSourceTask, 0, len(tasks))
	for _, task := range tasks {
		runs, err := store.ListRunLogs(task.ID, 0)
		if err != nil {
			return nil, fmt.Errorf("list run logs for %s: %w", task.ID, err)
		}
		sources = append(sources, internaltrace.BuildSessionSourceTask(task, runs))
	}
	return sources, nil
}

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
