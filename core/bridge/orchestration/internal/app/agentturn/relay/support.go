package relay

import (
	"context"
	"errors"
	"time"

	"ghost-os/bridge/orchestration/internal/trace/runcards"
	"ghost-os/bridge/streaming"
)

type runcardRecorder struct {
	inner *runcards.Recorder
}

type runcardStartInput struct {
	kind            string
	title           string
	nodeID          string
	nodeType        string
	round           int
	sourceSessionID string
	startedAt       time.Time
}

type runcardFinishInput struct {
	status          string
	preview         string
	errorText       string
	finalText       string
	sourceSessionID string
	finishedAt      time.Time
}

type runcardHandle struct {
	inner *runcards.Handle
}

func (r *runcardRecorder) StartCard(ctx context.Context, input runcardStartInput) (*runcardHandle, error) {
	handle, err := r.inner.StartCard(ctx, runcards.StartInput{
		Kind:            input.kind,
		Title:           input.title,
		NodeID:          input.nodeID,
		NodeType:        input.nodeType,
		Round:           input.round,
		SourceSessionID: input.sourceSessionID,
		StartedAt:       input.startedAt,
	})
	if err != nil {
		return nil, err
	}
	return &runcardHandle{inner: handle}, nil
}

func (h *runcardHandle) Finish(ctx context.Context, input runcardFinishInput) error {
	if h == nil || h.inner == nil {
		return errors.New("task run card recorder is not configured")
	}
	return h.inner.Finish(ctx, runcards.FinishInput{
		Status:          input.status,
		Preview:         input.preview,
		ErrorText:       input.errorText,
		FinalText:       input.finalText,
		SourceSessionID: input.sourceSessionID,
		FinishedAt:      input.finishedAt,
	})
}

func runcardRecorderFromContext(ctx context.Context) *runcardRecorder {
	recorder := runcards.RecorderFromContext(ctx)
	if recorder == nil {
		return nil
	}
	return &runcardRecorder{inner: recorder}
}

func newRuncardStreamSink(handle *runcardHandle) streaming.Sink {
	if handle == nil {
		return runcards.NewStreamSink(nil)
	}
	return runcards.NewStreamSink(handle.inner)
}
