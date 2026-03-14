package app

import (
	"context"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	bridgeruntime "ghost-os/bridge/runtime"
)

// runAgent 组装最小可运行链路：配置 -> LLM 客户端 -> 工具目录 -> Agent。
func runAgent(ctx context.Context, userMessage string) (string, error) {
	return runAgentWithConfigStore(ctx, userMessage, nil, "")
}

// runAgentWithConfigStore 允许注入配置存储与 trace id，便于服务层复用。
func runAgentWithConfigStore(ctx context.Context, userMessage string, store *bridgeconfig.Store, traceID string) (string, error) {
	factory := bridgeruntime.NewAgentRuntimeFactory()
	deps, err := factory.Build(store)
	if err != nil {
		return "", err
	}
	defer deps.Close()

	a := agent.NewAgent(deps.Client(), deps.Registry(), deps.SystemPrompt(), deps.Config().MaxTurns)
	return a.RunWithTraceID(ctx, userMessage, traceID)
}
