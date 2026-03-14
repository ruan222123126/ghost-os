package orchestration

import bridgeconfig "ghost-os/bridge/config"

func normalizeConfiguredToolLists(allowlist []string, blocklist []string) ([]string, []string, error) {
	return bridgeconfig.NormalizeConfiguredToolLists(allowlist, blocklist)
}
