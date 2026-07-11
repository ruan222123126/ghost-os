package toolselect

import (
	"context"
	"log"
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	appagentturn "ghost-os/bridge/orchestration/internal/app/agentturn"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type SelectorFactory func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine

type Request struct {
	Config               bridgeconfig.Config
	Registry             *tools.Registry
	Session              *session.Session
	History              *agent.History
	UserMessage          string
	AskHumanContinuation bool
	TraceID              string
	SelectorFactory      SelectorFactory
}

type catalogs struct {
	base      tools.ToolCatalog
	selection tools.ToolCatalog
	selector  tools.ToolCatalog
}

func SelectForTurn(ctx context.Context, req Request) (tools.ToolCatalog, string, error) {
	catalogs := newCatalogs(req.Config, req.Registry, req.Session)
	if req.AskHumanContinuation {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=ask_human_continuation", strings.TrimSpace(req.TraceID))
		return catalogs.base, "", nil
	}
	if !req.Config.ToolSelector.Enabled {
		return catalogs.base, "", nil
	}
	return selectScopedTools(ctx, req, catalogs)
}

func newCatalogs(cfg bridgeconfig.Config, registry *tools.Registry, sess *session.Session) catalogs {
	policy := bridgeruntime.NewToolSelectionPolicy(cfg)
	baseCatalog := tools.NewPromptOverrideCatalog(registry, cfg.ToolSelector.PromptOverrides)
	residentStaticNames := appagentturn.ToolCatalogNames(policy.ResidentCatalog(baseCatalog))
	selectorStaticNames := appagentturn.ToolCatalogNames(policy.SelectorCatalog(baseCatalog))
	return catalogs{
		base:      bridgeruntime.NewSessionTurnCatalog(baseCatalog, residentStaticNames, sess, cfg.ToolSearch.IdleTurns, false),
		selection: bridgeruntime.NewSessionTurnCatalog(baseCatalog, selectorStaticNames, sess, cfg.ToolSearch.IdleTurns, false),
		selector:  bridgeruntime.NewSessionTurnCatalog(baseCatalog, selectorStaticNames, sess, cfg.ToolSearch.IdleTurns, true),
	}
}

func selectScopedTools(
	ctx context.Context,
	req Request,
	catalogs catalogs,
) (tools.ToolCatalog, string, error) {
	selector := newSelector(req.Config, catalogs.selector, req.SelectorFactory)
	if selector == nil {
		return catalogs.base, "", nil
	}
	recentMessages := appagentturn.RecentMessages(req.History, req.Config.ToolSelector.RecentMsgs)
	result := selector.SelectTools(ctx, req.UserMessage, recentMessages, "", req.TraceID)
	if req.Config.ToolSelector.Shadow {
		log.Printf("trace_id=%s action=TOOL_SELECTOR status=shadow mode=%s tools=%v confidence=%.2f fallback=%t reason=%q error=%v", strings.TrimSpace(req.TraceID), result.Mode, result.Tools, result.Confidence, result.Fallback, result.Reason, result.Error)
		return catalogs.base, "", nil
	}
	if result.Mode != "subset" || result.Fallback {
		return catalogs.base, "", nil
	}
	return buildScopedCatalog(req.Config, req.Registry, req.Session, catalogs.selection, result.Tools)
}

func newSelector(
	cfg bridgeconfig.Config,
	catalog tools.ToolCatalog,
	factory SelectorFactory,
) bridgeruntime.SelectorEngine {
	if factory != nil {
		return factory(cfg, catalog)
	}
	return bridgeruntime.NewSelectorFromConfig(cfg, catalog)
}

func buildScopedCatalog(
	cfg bridgeconfig.Config,
	registry *tools.Registry,
	sess *session.Session,
	selectionCatalog tools.ToolCatalog,
	selected []string,
) (tools.ToolCatalog, string, error) {
	policy := bridgeruntime.NewToolSelectionPolicy(cfg)
	scoped := bridgeruntime.NewSessionTurnCatalog(
		registry,
		policy.Apply(appagentturn.ToolCatalogNames(selectionCatalog), selected),
		sess,
		cfg.ToolSearch.IdleTurns,
		false,
	)
	systemPrompt, err := bridgeruntime.BuildSystemPromptForSession(
		cfg,
		scoped,
		sess,
		cfg.ToolSearch.IdleTurns,
	)
	if err != nil {
		return nil, "", err
	}
	return scoped, systemPrompt, nil
}
