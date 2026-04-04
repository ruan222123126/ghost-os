package orchestration

import (
	"context"
	"log"
	"sort"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type turnSelectorCatalogs struct {
	base      tools.ToolCatalog
	selection tools.ToolCatalog
	selector  tools.ToolCatalog
}

func (p *sessionTurnPreparer) selectToolsForTurn(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	history *agent.History,
	userMessage string,
	askHumanContinuation bool,
	traceID string,
) (tools.ToolCatalog, string, error) {
	catalogs := newTurnSelectorCatalogs(deps, sess)
	if askHumanContinuation {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=ask_human_continuation", strings.TrimSpace(traceID))
		return catalogs.base, "", nil
	}
	if !deps.cfg.ToolSelector.Enabled {
		return catalogs.base, "", nil
	}
	return p.selectScopedTools(ctx, deps, sess, history, userMessage, traceID, catalogs)
}

func newTurnSelectorCatalogs(deps agentRuntimeDependencies, sess *session.Session) turnSelectorCatalogs {
	policy := newToolSelectionPolicy(deps.cfg)
	residentStaticNames := toolCatalogNames(policy.residentCatalog(deps.registry))
	selectorStaticNames := toolCatalogNames(policy.selectorCatalog(deps.registry))
	return turnSelectorCatalogs{
		base:      newSessionTurnCatalog(deps.registry, residentStaticNames, sess, deps.cfg.ToolSearch.IdleTurns, false),
		selection: newSessionTurnCatalog(deps.registry, selectorStaticNames, sess, deps.cfg.ToolSearch.IdleTurns, false),
		selector:  newSessionTurnCatalog(deps.registry, selectorStaticNames, sess, deps.cfg.ToolSearch.IdleTurns, true),
	}
}

func (p *sessionTurnPreparer) selectScopedTools(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	history *agent.History,
	userMessage string,
	traceID string,
	catalogs turnSelectorCatalogs,
) (tools.ToolCatalog, string, error) {
	selector := p.newSelector(deps.cfg, catalogs.selector)
	if selector == nil {
		return catalogs.base, "", nil
	}
	recentMessages := getRecentMessages(history, deps.cfg.ToolSelector.RecentMsgs)
	result := selector.SelectTools(ctx, userMessage, recentMessages, "", traceID)
	if deps.cfg.ToolSelector.Shadow {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=shadow mode=%s tools=%v confidence=%.2f fallback=%t reason=%q error=%v", strings.TrimSpace(traceID), result.Mode, result.Tools, result.Confidence, result.Fallback, result.Reason, result.Error)
		return catalogs.base, "", nil
	}
	if result.Mode != "subset" || result.Fallback {
		return catalogs.base, "", nil
	}
	return buildScopedTurnCatalog(deps, sess, catalogs.selection, result.Tools)
}

func buildScopedTurnCatalog(
	deps agentRuntimeDependencies,
	sess *session.Session,
	selectionCatalog tools.ToolCatalog,
	selected []string,
) (tools.ToolCatalog, string, error) {
	policy := newToolSelectionPolicy(deps.cfg)
	scoped := newSessionTurnCatalog(
		deps.registry,
		policy.apply(toolCatalogNames(selectionCatalog), selected),
		sess,
		deps.cfg.ToolSearch.IdleTurns,
		false,
	)
	systemPrompt, err := buildSystemPromptForSession(
		deps.cfg,
		scoped,
		sess,
		deps.cfg.ToolSearch.IdleTurns,
	)
	if err != nil {
		return nil, "", err
	}
	return scoped, systemPrompt, nil
}

func (p *sessionTurnPreparer) newSelector(cfg Config, catalog tools.ToolCatalog) selectorEngine {
	if p != nil && p.selectorFactory != nil {
		return p.selectorFactory(cfg, catalog)
	}
	return newToolSelectorFromConfig(cfg, catalog)
}

func getRecentMessages(history *agent.History, limit int) []llm.Message {
	if history == nil || limit <= 0 {
		return nil
	}
	messages := history.Messages()
	filtered := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleUser, llm.RoleAssistant:
			filtered = append(filtered, msg)
		}
	}
	if len(filtered) <= limit {
		return llm.CloneMessages(filtered)
	}
	return llm.CloneMessages(filtered[len(filtered)-limit:])
}

func toolCatalogNames(catalog tools.ToolCatalog) []string {
	if catalog == nil {
		return nil
	}
	defs := catalog.ToolDefs()
	if len(defs) == 0 {
		return nil
	}
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		if name := strings.TrimSpace(def.Name); name != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
