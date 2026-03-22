package tools

import (
	"sort"
	"strings"
)

type VisibilityOptions struct {
	ToolSearchEnabled bool
	AllowlistOnly     bool
	Allowlist         []string
	Blocklist         []string
}

func ToolMetadataByName(name string) (ToolMetadata, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ToolMetadata{}, false
	}
	for _, item := range GetToolMetadata() {
		if item.Name == trimmed {
			return item, true
		}
	}
	return ToolMetadata{}, false
}

func selectorVisibleMetadata(catalog ToolCatalog) []ToolMetadata {
	return visibleMetadataForCatalog(catalog, func(item ToolMetadata) bool {
		if item.Name == ToolSearchToolName {
			return false
		}
		return catalog != nil || !item.OnDemand
	})
}

func FormatPromptToolsForCatalog(catalog ToolCatalog) string {
	if catalog == nil {
		return "- (none)"
	}

	defs := catalog.ToolDefs()
	if len(defs) == 0 {
		return "- (none)"
	}

	lines := make([]string, 0, len(defs))
	for _, def := range defs {
		name := strings.TrimSpace(def.Name)
		if name == "" {
			continue
		}
		lines = append(lines, "- "+name+": "+toolShortDescription(name, def.Description))
	}
	if len(lines) == 0 {
		return "- (none)"
	}
	return strings.Join(lines, "\n")
}

func visibleMetadataForCatalog(catalog ToolCatalog, keep func(ToolMetadata) bool) []ToolMetadata {
	items := GetToolMetadata()
	if catalog == nil {
		return filterMetadata(items, keep, nil)
	}

	allowed := toolNameSet(CatalogToolNames(catalog))
	if len(allowed) == 0 {
		return nil
	}
	return filterMetadata(items, keep, allowed)
}

func filterMetadata(items []ToolMetadata, keep func(ToolMetadata) bool, allowed map[string]bool) []ToolMetadata {
	filtered := make([]ToolMetadata, 0, len(items))
	for _, item := range items {
		if allowed != nil && !allowed[item.Name] {
			continue
		}
		if keep != nil && !keep(item) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func CatalogToolNames(catalog ToolCatalog) []string {
	if catalog == nil {
		return nil
	}
	return toolDefNames(catalog.ToolDefs())
}

func StaticVisibleToolNames(available []string, opts VisibilityOptions) []string {
	availableSet := toolNameSet(available)
	if len(availableSet) == 0 {
		return nil
	}

	resultSet := make(map[string]bool, len(availableSet))
	result := make([]string, 0, len(availableSet))
	add := func(name string) {
		if !availableSet[name] || resultSet[name] {
			return
		}
		resultSet[name] = true
		result = append(result, name)
	}

	for _, name := range normalizeVisibleToolNames(opts.Allowlist) {
		if isBlockedVisibleTool(name, opts) {
			continue
		}
		add(name)
	}

	sort.Strings(result)
	return result
}

func SearchCandidateToolNames(available []string, loaded []string, opts VisibilityOptions) []string {
	staticVisible := toolNameSet(StaticVisibleToolNames(available, opts))
	loadedSet := toolNameSet(loaded)
	out := make([]string, 0, len(available))
	for _, name := range normalizeVisibleToolNames(available) {
		switch {
		case name == AskHumanToolName, name == ToolSearchToolName:
			continue
		case isBlockedVisibleTool(name, opts):
			continue
		case staticVisible[name], loadedSet[name]:
			continue
		default:
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func SelectorVisibleToolNames(names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range normalizeVisibleToolNames(names) {
		if name == ToolSearchToolName {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func toolShortDescription(name string, fallback string) string {
	metadata, ok := ToolMetadataByName(name)
	if ok && strings.TrimSpace(metadata.ShortDesc) != "" {
		return metadata.ShortDesc
	}
	text := strings.TrimSpace(fallback)
	if text == "" {
		return "Available tool."
	}
	if index := strings.Index(text, "."); index > 0 {
		return strings.TrimSpace(text[:index+1])
	}
	return text
}

func normalizeVisibleToolNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func toolNameSet(names []string) map[string]bool {
	normalized := normalizeVisibleToolNames(names)
	if len(normalized) == 0 {
		return nil
	}
	set := make(map[string]bool, len(normalized))
	for _, name := range normalized {
		set[name] = true
	}
	return set
}

func isBlockedVisibleTool(name string, opts VisibilityOptions) bool {
	for _, blocked := range normalizeVisibleToolNames(opts.Blocklist) {
		if blocked == name {
			return true
		}
	}
	return false
}
