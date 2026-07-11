package tools

import (
	"strings"

	"ghost-os/bridge/orchestration/internal/domain/runtimeopts"
	bridgetools "ghost-os/bridge/tools"
)

func NormalizeConfiguredToolLists(allowlist []string, blocklist []string) ([]string, []string, error) {
	return runtimeopts.NormalizeToolLists(allowlist, blocklist, validConfiguredToolNames())
}

func validConfiguredToolNames() []string {
	valid := make([]string, 0, len(bridgetools.GetToolMetadata()))
	for _, item := range bridgetools.GetToolMetadata() {
		name := strings.TrimSpace(item.Name)
		if name != "" {
			valid = append(valid, name)
		}
	}
	return valid
}
