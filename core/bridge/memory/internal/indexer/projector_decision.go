package indexer

import "context"

type decisionProjector struct {
	enabled bool
	handler func(context.Context, TurnEnvelope) error
	builder func(context.Context, Bucket, []TurnEnvelope) (ViewManifest, error)
}

func NewDecisionProjector(enabled bool, handler func(context.Context, TurnEnvelope) error, builder ...func(context.Context, Bucket, []TurnEnvelope) (ViewManifest, error)) Projector {
	var selected func(context.Context, Bucket, []TurnEnvelope) (ViewManifest, error)
	if len(builder) > 0 {
		selected = builder[0]
	}
	return decisionProjector{enabled: enabled, handler: handler, builder: selected}
}

func (p decisionProjector) Name() string { return "decision" }

func (p decisionProjector) Enabled() bool { return p.enabled && p.handler != nil }

func (p decisionProjector) Process(ctx context.Context, envelope TurnEnvelope) error {
	return p.handler(ctx, envelope)
}

func (p decisionProjector) BuildViewManifest(ctx context.Context, bucket Bucket, envelopes []TurnEnvelope) (ViewManifest, error) {
	if p.builder == nil {
		return ViewManifest{}, nil
	}
	return p.builder(ctx, bucket, envelopes)
}
