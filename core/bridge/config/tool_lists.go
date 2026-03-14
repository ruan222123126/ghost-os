package config

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"ghost-os/bridge/tools"
)

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
		if name == tools.AskHumanToolName || name == tools.ToolSearchToolName {
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

func normalizeToolNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}

	result := make([]string, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	return result
}
