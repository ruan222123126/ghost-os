package app

import (
	"context"

	bridgeorchestration "ghost-os/bridge/orchestration"
)

// runAgent 通过正式 session turn runner 执行一次无持久化的单轮请求。
func runAgent(ctx context.Context, userMessage string) (string, error) {
	runner := bridgeorchestration.NewSessionAgentRunner(
		nil, // runtimeFactory: nil => 使用 orchestration 默认 runtime 组装
		nil, // configStore: nil => 从环境加载配置，不绑定外部 store
		nil, // sessionStore: nil => 单轮临时会话，不做持久化
		nil, // runRegistry: nil => 不注册 in-flight run
	)
	response, _, err := runner.RunTurn(ctx, userMessage, "", "")
	return response, err
}
