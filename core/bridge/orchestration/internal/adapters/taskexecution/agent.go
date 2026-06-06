package taskexecution

import (
	"context"
	"errors"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/taskdefs"
)

type AgentDirectRunner interface {
	RunTurnStream(
		ctx context.Context,
		message string,
		sessionID string,
		traceID string,
		sink streaming.Sink,
	) (string, string, error)
}

type AgentOverrideRunner interface {
	RunTurnStreamInputWithOverrides(
		ctx context.Context,
		input llm.Message,
		sessionID string,
		traceID string,
		sink streaming.Sink,
		runtimeOverrides *taskdefs.TaskRuntimeOverrides,
	) (string, string, error)
}

type AgentStreamRunner struct {
	Direct   AgentDirectRunner
	Override AgentOverrideRunner
}

func (r AgentStreamRunner) RunAgentTaskStream(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	sessionID := strings.TrimSpace(task.SessionID)
	if r.Direct != nil && task.RuntimeOverrides == nil {
		return r.Direct.RunTurnStream(ctx, task.Message, sessionID, traceID, sink)
	}
	if r.Override == nil {
		return "", sessionID, errors.New("task agent runner is not configured")
	}
	return r.Override.RunTurnStreamInputWithOverrides(
		ctx,
		llm.Message{Role: llm.RoleUser, Text: task.Message},
		sessionID,
		traceID,
		sink,
		taskdefs.CloneTaskRuntimeOverrides(task.RuntimeOverrides),
	)
}
