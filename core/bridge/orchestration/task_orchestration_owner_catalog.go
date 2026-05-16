package orchestration

import (
	"sort"
	"strings"

	"ghost-os/bridge/llm"
	tooladapter "ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type ownerRuntimeCatalog struct {
	base     tools.ToolCatalog
	dispatch tools.Tool
}

func buildOwnerRuntimeCatalog(
	deps agentRuntimeDependencies,
	sess *session.Session,
	req ports.OwnerDecisionTurnRequest,
) tools.ToolCatalog {
	base := tools.NewPromptOverrideCatalog(deps.registry, deps.cfg.ToolSelector.PromptOverrides)
	catalog := ownerRuntimeCatalog{
		base: base,
		dispatch: tooladapter.DispatchTool{
			GroupNode:   req.GroupNode,
			MemberOrder: append([]string(nil), req.MemberOrder...),
		},
	}
	staticNames := ownerRuntimeStaticTools(deps, base)
	return bridgeruntime.NewSessionTurnCatalog(
		catalog,
		appendOwnerRuntimeTool(staticNames, tooladapter.DispatchToolName),
		sess,
		deps.cfg.ToolSearch.IdleTurns,
		false,
	)
}

func ownerRuntimeStaticTools(
	deps agentRuntimeDependencies,
	base tools.ToolCatalog,
) []string {
	policy := bridgeruntime.NewToolSelectionPolicy(deps.cfg)
	return toolCatalogNames(policy.ResidentCatalog(base))
}

func appendOwnerRuntimeTool(names []string, toolName string) []string {
	trimmed := strings.TrimSpace(toolName)
	if trimmed == "" {
		return append([]string(nil), names...)
	}
	for _, name := range names {
		if strings.TrimSpace(name) == trimmed {
			return append([]string(nil), names...)
		}
	}
	return append(append([]string(nil), names...), trimmed)
}

func (c ownerRuntimeCatalog) Get(name string) tools.Tool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil
	}
	if c.dispatch != nil && trimmed == c.dispatch.Name() {
		return c.dispatch
	}
	if c.base == nil {
		return nil
	}
	return c.base.Get(trimmed)
}

func (c ownerRuntimeCatalog) ToolDefs() []llm.ToolDef {
	defs := ownerCatalogDefs(c.base, c.dispatch)
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}

func ownerCatalogDefs(base tools.ToolCatalog, dispatch tools.Tool) []llm.ToolDef {
	defs := make([]llm.ToolDef, 0, 8)
	seen := make(map[string]bool, 8)
	appendDef := func(def llm.ToolDef) {
		name := strings.TrimSpace(def.Name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		defs = append(defs, def)
	}
	if base != nil {
		for _, def := range base.ToolDefs() {
			appendDef(def)
		}
	}
	if dispatch != nil {
		appendDef(tools.ToolDefFromTool(dispatch))
	}
	return defs
}
