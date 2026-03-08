package indexer

import "context"

type graphProjector struct {
	enabled bool
	handler func(context.Context, TurnEnvelope) error
	builder func(context.Context, Bucket, []TurnEnvelope) (ViewManifest, error)
}

func NewGraphProjector(enabled bool, handler func(context.Context, TurnEnvelope) error, builder ...func(context.Context, Bucket, []TurnEnvelope) (ViewManifest, error)) Projector {
	var selected func(context.Context, Bucket, []TurnEnvelope) (ViewManifest, error)
	if len(builder) > 0 {
		selected = builder[0]
	}
	return graphProjector{enabled: enabled, handler: handler, builder: selected}
}

func (p graphProjector) Name() string { return "graph" }

func (p graphProjector) Enabled() bool { return p.enabled && p.handler != nil }

func (p graphProjector) Process(ctx context.Context, envelope TurnEnvelope) error {
	return p.handler(ctx, envelope)
}

func (p graphProjector) BuildViewManifest(ctx context.Context, bucket Bucket, envelopes []TurnEnvelope) (ViewManifest, error) {
	if p.builder == nil {
		return ViewManifest{}, nil
	}
	return p.builder(ctx, bucket, envelopes)
}
