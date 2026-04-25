package orchestration

import (
	"context"
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
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
	graphQLMode := graphQLToolRuntimeEnabled(deps.cfg)
	promptCatalog := catalog
	if graphQLMode {
		promptCatalog = tools.NewStructuredToolHiddenCatalog(catalog)
	}
	basePrompt := strings.TrimSpace(systemPrompt)
	if basePrompt == "" || graphQLMode {
		prompt, err := bridgeruntime.BuildSystemPromptForSession(
			deps.cfg,
			promptCatalog,
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
	if graphQLMode {
		basePrompt = withGraphQLTextProtocolPrompt(basePrompt, catalog)
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

func (p *sessionTurnPreparer) newGraphQLSystemPromptRefreshHook(
	deps agentRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
	systemPrompt string,
) agent.BeforeCompletionHook {
	return func(_ context.Context, _ int, history *agent.History) error {
		if history == nil {
			return nil
		}
		prompt, err := p.buildCompletionSystemPrompt(deps, sess, catalog, systemPrompt)
		if err != nil {
			return err
		}
		history.UpdateSystemPrompt(prompt)
		return nil
	}
}

func withGraphQLTextProtocolPrompt(basePrompt string, catalog tools.ToolCatalog) string {
	protocol := tools.FormatGraphQLToolRuntimePrompt(catalog)
	trimmed := strings.TrimSpace(basePrompt)
	if trimmed == "" {
		return protocol
	}
	return trimmed + "\n\n" + protocol
}

func graphQLToolRuntimeEnabled(cfg bridgeconfig.Config) bool {
	return cfg.GraphQL.ToolRuntimeEnabled
}
