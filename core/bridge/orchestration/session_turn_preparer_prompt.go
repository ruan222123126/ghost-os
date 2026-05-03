package orchestration

import (
	"strings"

	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func (p *sessionTurnPreparer) buildCompletionSystemPrompt(
	deps agentRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
	systemPrompt string,
) (string, error) {
	if deps.systemPromptOverride {
		return strings.TrimSpace(deps.systemPrompt), nil
	}
	basePrompt := strings.TrimSpace(systemPrompt)
	if basePrompt == "" {
		prompt, err := bridgeruntime.BuildSystemPromptForSession(
			deps.cfg,
			catalog,
			sess,
			deps.cfg.ToolSearch.IdleTurns,
		)
		if err != nil {
			return "", err
		}
		basePrompt = strings.TrimSpace(prompt)
	}
	if basePrompt == "" {
		basePrompt = strings.TrimSpace(deps.systemPrompt)
	}
	return basePrompt, nil
}

func (p *sessionTurnPreparer) buildTurnSystemPrompt(
	deps agentRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
	systemPrompt string,
) (string, error) {
	return p.buildCompletionSystemPrompt(deps, sess, catalog, systemPrompt)
}
