package orchestration

import (
	"context"

	"ghost-os/bridge/streaming"
)

func (s *bridgeService) executeAgentStreamAction(
	ctx context.Context,
	params agentParams,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	broadcastSink := newSessionStreamBroadcastSink(sink, s.sessionPushHub())
	return s.agentTurnService().ExecuteStream(ctx, params, traceID, broadcastSink)
}
