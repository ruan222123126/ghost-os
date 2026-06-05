package orchestration

import (
	"context"
	"errors"
	"time"

	"ghost-os/bridge/orchestration/internal/trace/runcards"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

type taskRunCardRecorder struct {
	inner *runcards.Recorder
}

type taskRunCardStartInput struct {
	kind            string
	title           string
	nodeID          string
	nodeType        string
	round           int
	iteration       int
	branchID        string
	sourceSessionID string
	startedAt       time.Time
}

type taskRunCardFinishInput struct {
	status          string
	preview         string
	errorText       string
	finalText       string
	sourceSessionID string
	finishedAt      time.Time
}

type taskRunCardHandle struct {
	inner *runcards.Handle
}

func newTaskRunCardRecorder(
	ctx context.Context,
	hub *sessionPushHub,
) (*taskRunCardRecorder, error) {
	recorder, err := runcards.NewRecorder(ctx, hub)
	if err != nil {
		return nil, err
	}
	return &taskRunCardRecorder{inner: recorder}, nil
}

func (r *taskRunCardRecorder) StartCard(
	ctx context.Context,
	input taskRunCardStartInput,
) (*taskRunCardHandle, error) {
	handle, err := r.inner.StartCard(ctx, toRunCardStartInput(input))
	if err != nil {
		return nil, err
	}
	return &taskRunCardHandle{inner: handle}, nil
}

func (r *taskRunCardRecorder) Snapshot() []bridgeTasks.RunCard {
	if r == nil || r.inner == nil {
		return nil
	}
	return r.inner.Snapshot()
}

func (h *taskRunCardHandle) Finish(
	ctx context.Context,
	input taskRunCardFinishInput,
) error {
	if h == nil || h.inner == nil {
		return errors.New("task run card recorder is not configured")
	}
	return h.inner.Finish(ctx, toRunCardFinishInput(input))
}

func withTaskRunCardRecorder(
	ctx context.Context,
	recorder *taskRunCardRecorder,
) context.Context {
	if recorder == nil {
		return ctx
	}
	return runcards.WithRecorder(ctx, recorder.inner)
}

func taskRunCardRecorderFromContext(
	ctx context.Context,
) *taskRunCardRecorder {
	recorder := runcards.RecorderFromContext(ctx)
	if recorder == nil {
		return nil
	}
	return &taskRunCardRecorder{inner: recorder}
}

func newTaskRunCardStreamSink(handle *taskRunCardHandle) streaming.Sink {
	if handle == nil {
		return runcards.NewStreamSink(nil)
	}
	return runcards.NewStreamSink(handle.inner)
}

func toRunCardStartInput(input taskRunCardStartInput) runcards.StartInput {
	return runcards.StartInput{
		Kind:            input.kind,
		Title:           input.title,
		NodeID:          input.nodeID,
		NodeType:        input.nodeType,
		Round:           input.round,
		Iteration:       input.iteration,
		BranchID:        input.branchID,
		SourceSessionID: input.sourceSessionID,
		StartedAt:       input.startedAt,
	}
}

func toRunCardFinishInput(input taskRunCardFinishInput) runcards.FinishInput {
	return runcards.FinishInput{
		Status:          input.status,
		Preview:         input.preview,
		ErrorText:       input.errorText,
		FinalText:       input.finalText,
		SourceSessionID: input.sourceSessionID,
		FinishedAt:      input.finishedAt,
	}
}
