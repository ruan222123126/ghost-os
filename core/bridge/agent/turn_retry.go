package agent

import (
	"context"

	"ghost-os/bridge/llm"
)

func shouldRetryTurnProcessing(
	ctx context.Context,
	err error,
	attempt int,
	maxAttempts int,
	completionDeltaEmitted bool,
) bool {
	if attempt+1 >= maxAttempts {
		return false
	}
	if completionDeltaEmitted {
		return false
	}
	if ctx.Err() != nil {
		return false
	}
	return llm.IsRetryableCompletionProtocolError(err)
}

func finalizeTurnProcessingError(
	ctx context.Context,
	turn int,
	state *agentRunState,
	err error,
) error {
	if !llm.IsRetryableCompletionProtocolError(err) {
		return err
	}
	return state.terminalRunError(ctx, turn, err)
}
