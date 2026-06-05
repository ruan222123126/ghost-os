package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/streaming"
)

// parseSessionEndSignal 只识别完整会话结束信号；普通文本/普通 JSON 均按普通回复返回。
func parseSessionEndSignal(raw string) (message string, signal *assistantSessionEndSignalPayload, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil, nil
	}
	probe, matched := probeSessionEndSignal(trimmed)
	if !matched {
		return trimmed, nil, nil
	}
	if len(probe) != 2 {
		return "", nil, errors.New("session end signal only allows signal and message fields")
	}
	parsed, err := decodeSessionEndSignalPayload(trimmed)
	if err != nil {
		return "", nil, err
	}
	return parsed.Message, &parsed, nil
}

func probeSessionEndSignal(raw string) (map[string]json.RawMessage, bool) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, false
	}
	signalValueRaw, hasSignal := probe["signal"]
	if !hasSignal {
		return nil, false
	}
	var signalText string
	if err := json.Unmarshal(signalValueRaw, &signalText); err != nil {
		return nil, false
	}
	if strings.TrimSpace(signalText) != busAssistantSessionEndSignal {
		return nil, false
	}
	return probe, true
}

func decodeSessionEndSignalPayload(raw string) (assistantSessionEndSignalPayload, error) {
	var parsed assistantSessionEndSignalPayload
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return assistantSessionEndSignalPayload{}, fmt.Errorf("decode session end signal: %w", err)
	}
	parsed.Signal = strings.TrimSpace(parsed.Signal)
	parsed.Message = strings.TrimSpace(parsed.Message)
	if parsed.Signal != busAssistantSessionEndSignal {
		return assistantSessionEndSignalPayload{}, fmt.Errorf("session end signal must set signal=%q", busAssistantSessionEndSignal)
	}
	if parsed.Message == "" {
		return assistantSessionEndSignalPayload{}, errors.New("session end signal message is empty")
	}
	return parsed, nil
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
