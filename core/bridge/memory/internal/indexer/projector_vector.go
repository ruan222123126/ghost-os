package indexer

import "context"

type vectorProjector struct {
	enabled bool
	handler func(context.Context, TurnEnvelope) error
}

func NewVectorProjector(enabled bool, handler func(context.Context, TurnEnvelope) error) Projector {
	return vectorProjector{enabled: enabled, handler: handler}
}

func (p vectorProjector) Name() string { return "vector" }

func (p vectorProjector) Enabled() bool { return p.enabled && p.handler != nil }

func (p vectorProjector) Process(ctx context.Context, envelope TurnEnvelope) error {
	return p.handler(ctx, envelope)
}
