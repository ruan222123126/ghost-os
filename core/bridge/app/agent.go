package app

import (
	"context"

	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
)

type sessionTurnRunner interface {
	RunTurn(ctx context.Context, message string, sessionID string, traceID string) (string, string, error)
}

var newAgentTurnRunner = func(store *bridgeconfig.Store) sessionTurnRunner {
	return bridgeorchestration.NewSessionAgentRunner(
		nil,
		bridgeorchestration.WrapConfigStore(store),
		nil,
		nil,
	)
}

// runAgent 通过正式 session turn runner 执行一次无持久化的单轮请求。
func runAgent(ctx context.Context, userMessage string) (string, error) {
	return runAgentWithConfigStore(ctx, userMessage, nil, "")
}

// runAgentWithConfigStore 复用正式单轮编排入口；CLI one-shot 不持久化 session。
func runAgentWithConfigStore(ctx context.Context, userMessage string, store *bridgeconfig.Store, traceID string) (string, error) {
	response, _, err := newAgentTurnRunner(store).RunTurn(ctx, userMessage, "", traceID)
	return response, err
}
