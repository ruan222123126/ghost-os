package app

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"ghost-os/bridge/tools"
)

type toolSelectionPolicy struct {
	enabled   bool
	allowlist []string
	blocked   map[string]bool
}

func newToolSelectionPolicy(cfg ToolSelectorConfig) toolSelectionPolicy {
	blocked := make(map[string]bool, len(cfg.Blocklist))
	for _, name := range cfg.Blocklist {
		blocked[name] = true
	}
	return toolSelectionPolicy{
		enabled:   len(cfg.Allowlist) > 0 || len(cfg.Blocklist) > 0,
		allowlist: append([]string(nil), cfg.Allowlist...),
		blocked:   blocked,
	}
}

func (p toolSelectionPolicy) scopeCatalog(catalog tools.ToolCatalog) tools.ToolCatalog {
	if !p.enabled || catalog == nil {
		return catalog
	}
	return tools.NewScopedCatalog(catalog, p.apply(toolCatalogNames(catalog), nil))
}

func (p toolSelectionPolicy) apply(available []string, selected []string) []string {
	if len(available) == 0 {
		return nil
	}

	availableSet := make(map[string]bool, len(available))
	for _, name := range available {
		availableSet[name] = true
	}

	resultSet := make(map[string]bool, len(available))
	result := make([]string, 0, len(available))
	if len(selected) == 0 {
		for _, name := range available {
			if p.blocked[name] {
				continue
			}
			resultSet[name] = true
			result = append(result, name)
		}
	} else {
		for _, name := range normalizeToolNames(selected) {
			if !availableSet[name] || p.blocked[name] || resultSet[name] {
				continue
			}
			resultSet[name] = true
			result = append(result, name)
		}
	}

	for _, name := range p.allowlist {
		if !availableSet[name] || p.blocked[name] || resultSet[name] {
			continue
		}
		resultSet[name] = true
		result = append(result, name)
	}

	if availableSet["ask_human"] && !resultSet["ask_human"] {
		result = append(result, "ask_human")
	}

	sort.Strings(result)
	return result
}

func toolNameListOrEnv(raw []string, envName string) []string {
	if raw != nil {
		return normalizeConfiguredToolNames(raw)
	}
	return normalizeConfiguredToolNames(parseStringCSV(getenvDefault(envName, "")))
}

func normalizeConfiguredToolLists(allowlist []string, blocklist []string) ([]string, []string, error) {
	normalizedAllowlist := normalizeConfiguredToolNames(allowlist)
	normalizedBlocklist := normalizeConfiguredToolNames(blocklist)
	valid := validConfiguredToolNames()

	for _, name := range normalizedAllowlist {
		if !valid[name] {
			return nil, nil, fmt.Errorf("unknown tool in tool_allowlist: %s", name)
		}
	}

	filteredBlocklist := make([]string, 0, len(normalizedBlocklist))
	for _, name := range normalizedBlocklist {
		if !valid[name] {
			return nil, nil, fmt.Errorf("unknown tool in tool_blocklist: %s", name)
		}
		if name == "ask_human" {
			log.Printf("action=TOOL_POLICY status=ignore_blocked_tool tool=%q reason=%q", name, "always_on")
			continue
		}
		filteredBlocklist = append(filteredBlocklist, name)
	}

	allowSet := make(map[string]bool, len(normalizedAllowlist))
	for _, name := range normalizedAllowlist {
		allowSet[name] = true
	}
	for _, name := range filteredBlocklist {
		if allowSet[name] {
			return nil, nil, fmt.Errorf("tool %q cannot appear in both tool_allowlist and tool_blocklist", name)
		}
	}

	return normalizedAllowlist, filteredBlocklist, nil
}

func normalizeConfiguredToolNames(names []string) []string {
	normalized := normalizeToolNames(names)
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)
	return normalized
}

func validConfiguredToolNames() map[string]bool {
	valid := make(map[string]bool, len(tools.GetToolMetadata()))
	for _, item := range tools.GetToolMetadata() {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			valid[name] = true
		}
	}
	return valid
}
