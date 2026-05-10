package orchestration

import (
	"errors"
	"fmt"
	"strings"

	taskdomain "ghost-os/bridge/orchestration/internal/domain/task"
	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
)

const taskRunTranscriptEventMarker = taskdomain.RunTranscriptEventMarker

func (a taskExecutorAdapter) attachTaskRunTranscript(
	task ScheduledTask,
	traceID string,
	result bridgeTasks.ExecutionResult,
) bridgeTasks.ExecutionResult {
	if !taskdomain.ShouldCreateRunTranscript(task.TaskKind) {
		return result
	}
	if a.service == nil || a.service.sessionStore == nil {
		return taskRunTranscriptFailure(result, errors.New("task run transcript session store is not configured"))
	}
	sessionSources := loadTaskRunTranscriptSessions(a.service.sessionStore, result)
	transcript := taskdomain.BuildRunTranscriptMessages(taskdomain.RunTranscriptOptions{
		Task:     task,
		TraceID:  traceID,
		Result:   result,
		Sessions: sessionSources,
	})
	sessionID, err := saveTaskRunTranscript(a.service.sessionStore, transcript)
	if err != nil {
		return taskRunTranscriptFailure(result, fmt.Errorf("save task run transcript: %w", err))
	}
	result.SessionIDOutput = sessionID
	return result
}

func taskRunTranscriptFailure(result bridgeTasks.ExecutionResult, err error) bridgeTasks.ExecutionResult {
	result.Status = taskRunStatusError
	errText := strings.TrimSpace(err.Error())
	if strings.TrimSpace(result.Error) != "" {
		result.Error = strings.TrimSpace(result.Error) + "; " + errText
		return result
	}
	result.Error = errText
	return result
}

func loadTaskRunTranscriptSessions(
	store *session.Store,
	result bridgeTasks.ExecutionResult,
) map[string]taskdomain.SessionSource {
	sessionIDs := taskdomain.CollectRunTranscriptSessionIDs(result)
	sources := make(map[string]taskdomain.SessionSource, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		loaded, err := store.Load(sessionID)
		if err != nil {
			sources[sessionID] = taskdomain.SessionSource{Err: err}
			continue
		}
		sources[sessionID] = taskdomain.SessionSource{Messages: loaded.Messages}
	}
	return sources
}

func saveTaskRunTranscript(store *session.Store, transcript taskdomain.RunTranscript) (string, error) {
	sess := session.NewSession("")
	sess.Title = transcript.Title
	for _, message := range transcript.Messages {
		sess.AddMessage(message)
	}
	if err := store.Save(sess); err != nil {
		return "", err
	}
	return sess.ID, nil
}
