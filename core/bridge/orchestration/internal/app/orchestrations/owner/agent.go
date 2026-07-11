package owner

import (
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/runtimeopts"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type DispatchAgentConfig struct {
	Config       bridgeconfig.Config
	Client       agent.Completer
	Catalog      tools.ToolCatalog
	Session      *session.Session
	SystemPrompt string
}

func BuildDispatchAgent(cfg DispatchAgentConfig) (*agent.Agent, error) {
	history := agent.NewHistory("")
	history.SetConversationState(cfg.Session.ConversationState)
	for _, message := range cfg.Session.Messages {
		history.Append(message)
	}
	history.UpdateSystemPrompt(cfg.SystemPrompt)

	runAgent := agent.NewAgentWithHistory(cfg.Client, cfg.Catalog, history, cfg.Config.MaxTurns)
	runAgent.SetResponseOptions(llm.CloneResponseOptions(cfg.Config.ResponseOptions))
	retry, err := runtimeopts.NormalizeRetryPolicy(
		cfg.Config.LLMCompletionRetryCount,
		cfg.Config.LLMCompletionRetryIntervalMS,
	)
	if err != nil {
		return nil, err
	}
	runAgent.SetCompletionRetryPolicy(agent.NewCompletionRetryPolicy(
		retry.Count,
		time.Duration(retry.IntervalMS)*time.Millisecond,
	))
	return runAgent, nil
}

func PersistAgentMessages(sess *session.Session, runAgent *agent.Agent) {
	sess.ConversationState = runAgent.GetConversationState()
	for _, message := range runAgent.GetNewMessages() {
		sess.AddMessage(message)
	}
}
