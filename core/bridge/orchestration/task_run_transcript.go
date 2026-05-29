package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	taskdomain "ghost-os/bridge/orchestration/internal/domain/task"
	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
)

const taskRunTranscriptEventMarker = taskdomain.RunTranscriptEventMarker

func (a taskExecutorAdapter) attachTaskRunTranscript(
	ctx context.Context,
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
	sessionID, err := saveTaskRunTranscript(a.service.sessionStore, transcript, runTranscriptSessionID(ctx))
	if err != nil {
		return taskRunTranscriptFailure(result, fmt.Errorf("save task run transcript: %w", err))
	}
	result.SessionIDOutput = sessionID
	return result
}

func runTranscriptSessionID(ctx context.Context) string {
	session, ok := bridgeTasks.RunSessionFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(session.SessionID)
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

func saveTaskRunTranscript(
	store *session.Store,
	transcript taskdomain.RunTranscript,
	sessionID string,
) (string, error) {
	sess, err := loadOrNewTranscriptSession(store, sessionID)
	if err != nil {
		return "", err
	}
	sess.Title = transcript.Title
	for _, message := range transcript.Messages {
		sess.AddMessage(message)
	}
	if err := store.Save(sess); err != nil {
		return "", err
	}
	return sess.ID, nil
}

func loadOrNewTranscriptSession(store *session.Store, sessionID string) (*session.Session, error) {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return session.NewSession(""), nil
	}
	sess, err := store.Load(id)
	if err != nil {
		return nil, err
	}
	return sess, nil
}
