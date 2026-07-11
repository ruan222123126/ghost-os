package agent

import (
	"time"

	bridgeagent "ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/app/agentturn/sessionprep"
	"ghost-os/bridge/orchestration/internal/domain/runtimeopts"
)

type TurnAgentBuilder struct{}

func (TurnAgentBuilder) BuildTurnAgent(req sessionprep.TurnAgentRequest) (*bridgeagent.Agent, error) {
	runAgent := bridgeagent.NewAgentWithHistory(req.Client, req.Catalog, req.History, req.Config.MaxTurns)
	runAgent.SetResponseOptions(llm.CloneResponseOptions(req.Config.ResponseOptions))
	retry, err := runtimeopts.NormalizeRetryPolicy(
		req.Config.LLMCompletionRetryCount,
		req.Config.LLMCompletionRetryIntervalMS,
	)
	if err != nil {
		return nil, err
	}
	runAgent.SetCompletionRetryPolicy(bridgeagent.NewCompletionRetryPolicy(
		retry.Count,
		time.Duration(retry.IntervalMS)*time.Millisecond,
	))
	agentturn.AttachDynamicPromptRefresh(runAgent, req.Config, req.Session, func() (string, error) {
		return agentturn.BuildCompletionSystemPrompt(agentturn.CompletionPromptRequest{
			Config:               req.Config,
			Catalog:              req.Catalog,
			Session:              req.Session,
			FallbackSystemPrompt: req.SystemPrompt,
			SystemPromptOverride: req.SystemPromptOverride,
			SystemPromptFiles:    req.SystemPromptFiles,
		})
	})
	return runAgent, nil
}
