package runtime

import (
	"sort"

	"ghost-os/bridge/tools"
)

type toolSelectionPolicy struct {
	visibility tools.VisibilityOptions
}

func newToolSelectionPolicy(cfg Config) toolSelectionPolicy {
	return toolSelectionPolicy{visibility: toolVisibilityOptions(cfg)}
}

func (p toolSelectionPolicy) scopeCatalog(catalog tools.ToolCatalog) tools.ToolCatalog {
	if catalog == nil {
		return nil
	}
	return tools.NewScopedCatalog(catalog, p.allowlistScope(tools.CatalogToolNames(catalog)))
}

func (p toolSelectionPolicy) allowlistScope(available []string) []string {
	return tools.StaticVisibleToolNames(available, p.visibility)
}

func (p toolSelectionPolicy) apply(available []string, selected []string) []string {
	if len(available) == 0 {
		return nil
	}
	if len(selected) == 0 {
		return p.allowlistScope(available)
	}

	availableSet := toolNameSet(available)
	result := make([]string, 0, len(available))
	resultSet := make(map[string]bool, len(available))
	add := func(name string) {
		if name == "" || !availableSet[name] || resultSet[name] {
			return
		}
		resultSet[name] = true
		result = append(result, name)
	}

	for _, name := range normalizeToolNames(selected) {
		if isBlockedTool(name, p.visibility) {
			continue
		}
		add(name)
	}
	for _, name := range p.requiredTools(available) {
		add(name)
	}
	result = normalizeToolNames(result)
	sort.Strings(result)
	return result
}

func (p toolSelectionPolicy) requiredTools(available []string) []string {
	if len(available) == 0 {
		return nil
	}

	availableSet := toolNameSet(available)
	result := make([]string, 0, len(available))
	add := func(name string) {
		if name == "" || !availableSet[name] || isBlockedTool(name, p.visibility) {
			return
		}
		result = append(result, name)
	}

	for _, name := range normalizeToolNames(p.visibility.Allowlist) {
		add(name)
	}
	return normalizeToolNames(result)
}

func toolVisibilityOptions(cfg Config) tools.VisibilityOptions {
	return tools.VisibilityOptions{
		ToolSearchEnabled: cfg.ToolSearch.Enabled,
		AllowlistOnly:     cfg.ToolSelector.AllowlistOnly,
		Allowlist:         append([]string(nil), cfg.ToolSelector.Allowlist...),
		Blocklist:         append([]string(nil), cfg.ToolSelector.Blocklist...),
	}
}

func toolNameSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, name := range names {
		set[name] = true
	}
	return set
}

func isBlockedTool(name string, visibility tools.VisibilityOptions) bool {
	for _, blocked := range visibility.Blocklist {
		if blocked == name {
			return true
		}
	}
	return false
}
