package config

import (
	"fmt"
	"sort"
	"strings"
)

func envGraphQLSettings() (GraphQLConfig, error) {
	return graphQLSettingsFromEnv(currentEnv())
}

func graphQLSettingsFromEnv(env envSnapshot) (GraphQLConfig, error) {
	if err := validateNoLegacyGraphQLEnv(env); err != nil {
		return GraphQLConfig{}, err
	}
	toolRuntimeEnabled, err := parseBoolValue(
		env.value("GHOST_GRAPHQL_TOOL_RUNTIME_ENABLED"),
		"GHOST_GRAPHQL_TOOL_RUNTIME_ENABLED",
		false,
	)
	if err != nil {
		return GraphQLConfig{}, err
	}
	textSanitizeEnabled, err := parseBoolValue(
		env.value("GHOST_GRAPHQL_TEXT_SANITIZE_ENABLED"),
		"GHOST_GRAPHQL_TEXT_SANITIZE_ENABLED",
		defaultGraphQLTextSanitizeEnabled,
	)
	if err != nil {
		return GraphQLConfig{}, err
	}
	return finalizeGraphQLConfig(GraphQLConfig{
		ToolRuntimeEnabled:  toolRuntimeEnabled,
		TextSanitizeEnabled: textSanitizeEnabled,
	})
}

func fileGraphQLSettings(fileCfg bridgeFileConfig, fallback GraphQLConfig) (GraphQLConfig, error) {
	settings := normalizeGraphQLConfig(fallback)
	if !hasGraphQLSourceLayout(fileCfg) {
		if fileCfg.GraphQLToolRuntimeEnabled != nil {
			settings.ToolRuntimeEnabled = *fileCfg.GraphQLToolRuntimeEnabled
		}
		if fileCfg.GraphQLTextSanitizeEnabled != nil {
			settings.TextSanitizeEnabled = *fileCfg.GraphQLTextSanitizeEnabled
		}
		return finalizeGraphQLConfig(settings)
	}
	return finalizeGraphQLConfig(GraphQLConfig{
		ToolRuntimeEnabled:  resolveGraphQLToolRuntimeEnabled(fileCfg, fallback),
		TextSanitizeEnabled: resolveGraphQLTextSanitizeEnabled(fileCfg, fallback),
		DefaultSource:       stringValue(fileCfg.GraphQLDefaultSource),
		Sources:             graphQLSourcesFromFile(fileCfg.GraphQLSources),
		MutationPolicies:    graphQLMutationPoliciesFromFile(fileCfg.GraphQLMutationPolicies),
	})
}

func finalizeGraphQLConfig(cfg GraphQLConfig) (GraphQLConfig, error) {
	normalized := normalizeGraphQLConfig(cfg)
	if err := validateGraphQLConfig(normalized); err != nil {
		return GraphQLConfig{}, err
	}
	return normalized, nil
}

func normalizeGraphQLConfig(cfg GraphQLConfig) GraphQLConfig {
	return GraphQLConfig{
		ToolRuntimeEnabled:  cfg.ToolRuntimeEnabled,
		TextSanitizeEnabled: normalizeGraphQLTextSanitizeEnabled(cfg.TextSanitizeEnabled),
		DefaultSource:       normalizeOptionalString(cfg.DefaultSource),
		Sources:             normalizeGraphQLSources(cfg.Sources),
		MutationPolicies:    normalizeGraphQLMutationPolicies(cfg.MutationPolicies),
	}
}

func resolveGraphQLToolRuntimeEnabled(
	fileCfg bridgeFileConfig,
	fallback GraphQLConfig,
) bool {
	if fileCfg.GraphQLToolRuntimeEnabled != nil {
		return *fileCfg.GraphQLToolRuntimeEnabled
	}
	return fallback.ToolRuntimeEnabled
}

func resolveGraphQLTextSanitizeEnabled(
	fileCfg bridgeFileConfig,
	fallback GraphQLConfig,
) bool {
	if fileCfg.GraphQLTextSanitizeEnabled != nil {
		return *fileCfg.GraphQLTextSanitizeEnabled
	}
	return normalizeGraphQLTextSanitizeEnabled(fallback.TextSanitizeEnabled)
}

func normalizeGraphQLTextSanitizeEnabled(value bool) bool {
	if !value {
		return false
	}
	return defaultGraphQLTextSanitizeEnabled
}

func normalizeGraphQLSources(raw []GraphQLSourceConfig) []GraphQLSourceConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLSourceConfig, 0, len(raw))
	for _, source := range raw {
		out = append(out, normalizeGraphQLSourceConfig(source))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func normalizeGraphQLSourceConfig(cfg GraphQLSourceConfig) GraphQLSourceConfig {
	return GraphQLSourceConfig{
		Name:             normalizeOptionalString(cfg.Name),
		Description:      normalizeOptionalString(cfg.Description),
		Endpoint:         normalizeOptionalString(cfg.Endpoint),
		APIKey:           normalizeOptionalString(cfg.APIKey),
		SchemaPath:       normalizeOptionalString(cfg.SchemaPath),
		TimeoutMS:        normalizeGraphQLSourceInt(cfg.TimeoutMS, defaultGraphQLTimeoutMS),
		MaxResponseBytes: normalizeGraphQLSourceInt(cfg.MaxResponseBytes, defaultGraphQLMaxResponseBytes),
		Headers:          normalizeGraphQLSourceHeaders(cfg.Headers),
		MaxDepth:         normalizeGraphQLSourceInt(cfg.MaxDepth, defaultGraphQLMaxDepth),
		MaxFields:        normalizeGraphQLSourceInt(cfg.MaxFields, defaultGraphQLMaxFields),
		MaxRootFields:    normalizeGraphQLSourceInt(cfg.MaxRootFields, defaultGraphQLMaxRootFields),
		MaxFragments:     normalizeGraphQLSourceInt(cfg.MaxFragments, defaultGraphQLMaxFragments),
		Domains:          normalizeGraphQLDomains(cfg.Domains),
	}
}

func normalizeGraphQLDomains(raw []GraphQLDomainConfig) []GraphQLDomainConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLDomainConfig, 0, len(raw))
	for _, domain := range raw {
		out = append(out, GraphQLDomainConfig{
			Name:          normalizeOptionalString(domain.Name),
			Description:   normalizeOptionalString(domain.Description),
			RootQueries:   normalizeGraphQLNames(domain.RootQueries),
			Types:         normalizeGraphQLNames(domain.Types),
			MaxDepth:      domain.MaxDepth,
			MaxFields:     domain.MaxFields,
			MaxRootFields: domain.MaxRootFields,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func normalizeGraphQLNames(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		name := strings.TrimSpace(item)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func graphQLSourcesFromFile(raw []graphQLSourceFileConfig) []GraphQLSourceConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLSourceConfig, 0, len(raw))
	for _, source := range raw {
		out = append(out, GraphQLSourceConfig{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			APIKey:           stringValue(source.APIKey),
			SchemaPath:       source.SchemaPath,
			TimeoutMS:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			Headers:          cloneStringMap(source.Headers),
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Domains:          graphQLDomainsFromFile(source.Domains),
		})
	}
	return out
}

func graphQLDomainsFromFile(raw []graphQLDomainFileConfig) []GraphQLDomainConfig {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLDomainConfig, 0, len(raw))
	for _, domain := range raw {
		out = append(out, GraphQLDomainConfig{
			Name:          domain.Name,
			Description:   domain.Description,
			RootQueries:   append([]string(nil), domain.RootQueries...),
			Types:         append([]string(nil), domain.Types...),
			MaxDepth:      domain.MaxDepth,
			MaxFields:     domain.MaxFields,
			MaxRootFields: domain.MaxRootFields,
		})
	}
	return out
}

func normalizeGraphQLSourceInt(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func normalizeGraphQLSourceHeaders(raw map[string]string) map[string]string {
	headers, _ := normalizeGraphQLHeaders(raw)
	return headers
}

func validateGraphQLBudgetValue(value int, source string, field string) error {
	if value > 0 {
		return nil
	}
	return fmt.Errorf("graphql source %q %s must be > 0", source, field)
}

func validateGraphQLOverrideValue(value int, source string, domain string, field string) error {
	if value >= 0 {
		return nil
	}
	return fmt.Errorf("graphql source %q domain %q %s must be >= 0", source, domain, field)
}

func normalizeOptionalString(raw string) string {
	return strings.TrimSpace(raw)
}

func validateNoLegacyGraphQLEnv(env envSnapshot) error {
	legacyEnv := configuredLegacyGraphQLEnv(env)
	if len(legacyEnv) == 0 {
		return nil
	}
	return fmt.Errorf(
		"legacy graphql env vars are no longer supported: %s; configure graphql_sources/graphql_mutation_policies in the bridge config file",
		strings.Join(legacyEnv, ", "),
	)
}

func configuredLegacyGraphQLEnv(env envSnapshot) []string {
	names := []string{
		"GHOST_GRAPHQL_ENABLED",
		"GHOST_GRAPHQL_ENDPOINT",
		"GHOST_GRAPHQL_API_KEY",
		"GHOST_GRAPHQL_SCHEMA_PATH",
		"GHOST_GRAPHQL_TIMEOUT_MS",
		"GHOST_GRAPHQL_MAX_RESPONSE_BYTES",
		"GHOST_GRAPHQL_HEADERS",
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		if env.value(name) == "" {
			continue
		}
		out = append(out, name)
	}
	return out
}
