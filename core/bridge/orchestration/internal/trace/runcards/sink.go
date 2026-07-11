package runcards

import (
	"context"

	"ghost-os/bridge/streaming"
)

type StreamSink struct {
	handle *Handle
}

func NewStreamSink(handle *Handle) streaming.Sink {
	return StreamSink{handle: handle}
}

func (s StreamSink) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	if s.handle == nil || s.handle.recorder == nil {
		return event, nil
	}
	return event, s.handle.recorder.EmitCardEvent(ctx, s.handle.cardID, event)
}
