package indexer

import "context"

type markdownProjector struct {
	enabled bool
	handler func(context.Context, TurnEnvelope) error
}

func NewMarkdownProjector(enabled bool, handler func(context.Context, TurnEnvelope) error) Projector {
	return markdownProjector{enabled: enabled, handler: handler}
}

func (p markdownProjector) Name() string { return "markdown" }

func (p markdownProjector) Enabled() bool { return p.enabled && p.handler != nil }

func (p markdownProjector) Process(ctx context.Context, envelope TurnEnvelope) error {
	return p.handler(ctx, envelope)
}
