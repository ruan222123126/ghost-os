package tools

func WithUpdatedAllowlist(raw []string, name string, enabled bool) []string {
	allowlist := normalizeConfiguredToolNames(raw)
	if enabled {
		return normalizeConfiguredToolNames(append(allowlist, name))
	}

	filtered := make([]string, 0, len(allowlist))
	for _, item := range allowlist {
		if item != name {
			filtered = append(filtered, item)
		}
	}
	return normalizeConfiguredToolNames(filtered)
}

func WithUpdatedBlocklist(raw []string, name string, enabled bool) []string {
	blocklist := normalizeConfiguredToolNames(raw)
	if !enabled {
		return normalizeConfiguredToolNames(append(blocklist, name))
	}

	filtered := make([]string, 0, len(blocklist))
	for _, item := range blocklist {
		if item != name {
			filtered = append(filtered, item)
		}
	}
	return normalizeConfiguredToolNames(filtered)
}

func NameSetFromSlice(raw []string) map[string]bool {
	set := make(map[string]bool, len(raw))
	for _, name := range normalizeConfiguredToolNames(raw) {
		set[name] = true
	}
	return set
}

func BlocklistForStrictAllowlist(allowlist []string) []string {
	allowset := NameSetFromSlice(allowlist)
	blocklist := make([]string, 0, len(configuredToolCatalog))

	for _, raw := range configuredToolCatalog {
		if allowset[raw] {
			continue
		}
		blocklist = append(blocklist, raw)
	}
	return normalizeConfiguredToolNames(blocklist)
}
