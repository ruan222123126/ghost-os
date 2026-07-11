package taskexecution

import (
	"context"
	"errors"

	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/orchestration/internal/trace/runcards"
	"ghost-os/bridge/session"
	"ghost-os/bridge/taskdefs"
	bridgeTasks "ghost-os/bridge/tasks"
)

type Config struct {
	SessionStore   *session.Store
	SessionPushHub *internaltrace.SessionPushHub
	Workflow       apptasks.KindExecutor
	Orchestration  apptasks.KindExecutor
	Agent          apptasks.KindExecutor
}

type Executor struct {
	Config Config
}

func (e Executor) Execute(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	traceID string,
) taskdefs.ExecutionResult {
	recorder, recordErr := e.prepareTaskRunCardRecorder(ctx, task)
	if recordErr != nil {
		result := taskdefs.ExecutionResult{
			Status: taskdefs.RunStatusError,
			Error:  recordErr.Error(),
		}
		return e.attachTaskRunTranscript(ctx, task, traceID, result)
	}
	if recorder != nil {
		ctx = runcards.WithRecorder(ctx, recorder)
	}
	result := e.executeTask(ctx, task, traceID)
	if recorder != nil {
		result.RunCards = recorder.Snapshot()
	}
	return e.attachTaskRunTranscript(ctx, task, traceID, result)
}

func (e Executor) PrepareRunSession(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	_ string,
) (bridgeTasks.RunSession, error) {
	return apptasks.PrepareRunSession(ctx, task, e)
}

func (e Executor) SaveTaskRunSession(
	task taskdefs.ScheduledTask,
	systemPrompt string,
) (bridgeTasks.RunSession, error) {
	if e.Config.SessionStore == nil {
		return bridgeTasks.RunSession{}, errors.New("task run session store is not configured")
	}
	sess := session.NewSession(systemPrompt)
	sess.Title = apptasks.RunSessionTitle(task)
	if err := e.Config.SessionStore.Save(sess); err != nil {
		return bridgeTasks.RunSession{}, err
	}
	return bridgeTasks.RunSession{SessionID: sess.ID}, nil
}

func (e Executor) attachTaskRunTranscript(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	traceID string,
	result taskdefs.ExecutionResult,
) taskdefs.ExecutionResult {
	return apptasks.AttachRunTranscript(ctx, e.Config.SessionStore, task, traceID, result)
}

func (e Executor) prepareTaskRunCardRecorder(
	ctx context.Context,
	task taskdefs.ScheduledTask,
) (*runcards.Recorder, error) {
	if !apptasks.ShouldRecordRunCards(task.TaskKind) {
		return nil, nil
	}
	return runcards.NewRecorder(ctx, e.Config.SessionPushHub)
}

func (e Executor) executeTask(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	traceID string,
) taskdefs.ExecutionResult {
	return apptasks.TaskExecutionRunner{
		Workflow:      e.Config.Workflow,
		Orchestration: e.Config.Orchestration,
		Agent:         e.Config.Agent,
	}.Execute(ctx, apptasks.TaskExecutionCommand{
		Task:    task,
		TraceID: traceID,
	})
}
