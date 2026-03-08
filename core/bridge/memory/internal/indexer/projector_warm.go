package indexer

import "context"

type warmProjector struct {
	enabled bool
	handler func(context.Context, TurnEnvelope) error
}

func NewWarmProjector(enabled bool, handler func(context.Context, TurnEnvelope) error) Projector {
	return warmProjector{enabled: enabled, handler: handler}
}

func (p warmProjector) Name() string { return "warm" }

func (p warmProjector) Enabled() bool { return p.enabled && p.handler != nil }

func (p warmProjector) Process(ctx context.Context, envelope TurnEnvelope) error {
	return p.handler(ctx, envelope)
}
