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

func (p *sessionTurnPreparer) buildSystemPromptWithMemory(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	history *agent.History,
	userMessage string,
	systemPrompt string,
	catalog tools.ToolCatalog,
) (string, string, *turnMemoryContext, error) {
	basePrompt, err := p.buildCompletionSystemPrompt(deps, sess, catalog, systemPrompt)
	if err != nil {
		return "", "", nil, err
	}
	memoryCtx, memoryBlock, err := p.prepareTurnMemory(ctx, deps, sess, history, userMessage)
	if err != nil {
		return "", "", nil, err
	}
	return composeTurnSystemPrompt(basePrompt, memoryBlock), memoryBlock, memoryCtx, nil
}

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

func composeTurnSystemPrompt(basePrompt string, memoryBlock string) string {
	trimmedBase := strings.TrimSpace(basePrompt)
	trimmedMemory := strings.TrimSpace(memoryBlock)
	if trimmedMemory == "" {
		return trimmedBase
	}
	if trimmedBase == "" {
		return trimmedMemory
	}
	return trimmedBase + "\n\n" + trimmedMemory
}

func (p *sessionTurnPreparer) newGraphQLSystemPromptRefreshHook(
	deps agentRuntimeDependencies,
	sess *session.Session,
	catalog tools.ToolCatalog,
	systemPrompt string,
	memoryBlock string,
) agent.BeforeCompletionHook {
	return func(_ context.Context, _ int, history *agent.History) error {
		if history == nil {
			return nil
		}
		prompt, err := p.buildCompletionSystemPrompt(deps, sess, catalog, systemPrompt)
		if err != nil {
			return err
		}
		history.UpdateSystemPrompt(composeTurnSystemPrompt(prompt, memoryBlock))
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
