package app

import (
	"context"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

// runAgent 组装最小可运行链路：配置 -> LLM 客户端 -> 工具目录 -> Agent。
func runAgent(ctx context.Context, userMessage string) (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}

	client := llm.NewClientWithOptions(llm.ClientOptions{
		Provider:           cfg.Provider,
		BaseURL:            cfg.BaseURL,
		APIKey:             cfg.APIKey,
		Model:              cfg.Model,
		ChatPath:           cfg.ChatPath,
		Headers:            cfg.ProviderHeaders,
		AnthropicVersion:   cfg.AnthropicVersion,
		AnthropicMaxTokens: cfg.AnthropicMaxTokens,
	})

	registry := tools.NewRegistry()
	registry.Register(tools.NewBashExecTool())
	registry.Register(tools.NewListFilesTool())

	// 这里维持现有系统提示词行为，不做策略层升级。
	systemPrompt := "You are Ghost-OS bridge agent. Use tools when needed and keep answers concise."
	a := agent.NewAgent(client, registry, systemPrompt, cfg.MaxTurns)
	return a.Run(ctx, userMessage)
}
