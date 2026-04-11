package app

import (
	"context"

	bridgeorchestration "ghost-os/bridge/orchestration"
)

// runAgent 通过正式 session turn runner 执行一次无持久化的单轮请求。
func runAgent(ctx context.Context, userMessage string) (string, error) {
	runner := bridgeorchestration.NewSessionAgentRunner(
		nil,
		nil,
		nil,
		nil,
	)
	response, _, err := runner.RunTurn(ctx, userMessage, "", "")
	return response, err
}
