package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func envGraphQLSettings() (GraphQLConfig, error) {
	source, ok, err := legacyGraphQLSourceFromEnv()
	if err != nil {
		return GraphQLConfig{}, err
	}
	if !ok {
		return finalizeGraphQLConfig(GraphQLConfig{})
	}
	return finalizeGraphQLConfig(GraphQLConfig{
		DefaultSource: defaultGraphQLLegacySourceName,
		Sources:       []GraphQLSourceConfig{source},
	})
}

func fileGraphQLSettings(fileCfg bridgeFileConfig, fallback GraphQLConfig) (GraphQLConfig, error) {
	settings := normalizeGraphQLConfig(fallback)
	switch {
	case hasGraphQLSourceLayout(fileCfg):
		return finalizeGraphQLConfig(GraphQLConfig{
			DefaultSource: stringValue(fileCfg.GraphQLDefaultSource),
			Sources:       graphQLSourcesFromFile(fileCfg.GraphQLSources),
		})
	case hasLegacyGraphQLConfig(fileCfg):
		source, ok, err := legacyGraphQLSourceFromFile(fileCfg, legacyGraphQLFallbackSource(settings))
		if err != nil {
			return GraphQLConfig{}, err
		}
		if !ok {
			return finalizeGraphQLConfig(GraphQLConfig{})
		}
		return finalizeGraphQLConfig(GraphQLConfig{
			DefaultSource: defaultGraphQLLegacySourceName,
			Sources:       []GraphQLSourceConfig{source},
		})
	default:
		return finalizeGraphQLConfig(settings)
	}
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
		DefaultSource: normalizeOptionalString(cfg.DefaultSource),
		Sources:       normalizeGraphQLSources(cfg.Sources),
	}
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

func legacyGraphQLSourceFromEnv() (GraphQLSourceConfig, bool, error) {
	if !parseBoolEnv("GHOST_GRAPHQL_ENABLED", false) {
		return GraphQLSourceConfig{}, false, nil
	}
	headers, err := parseGraphQLHeaders(os.Getenv("GHOST_GRAPHQL_HEADERS"))
	if err != nil {
		return GraphQLSourceConfig{}, false, err
	}
	return normalizeGraphQLSourceConfig(GraphQLSourceConfig{
		Name:             defaultGraphQLLegacySourceName,
		Endpoint:         getenvDefault("GHOST_GRAPHQL_ENDPOINT", ""),
		APIKey:           firstNonEmptyEnv("GHOST_GRAPHQL_API_KEY"),
		SchemaPath:       getenvDefault("GHOST_GRAPHQL_SCHEMA_PATH", ""),
		TimeoutMS:        parsePositiveIntEnv("GHOST_GRAPHQL_TIMEOUT_MS", defaultGraphQLTimeoutMS),
		MaxResponseBytes: parsePositiveIntEnv("GHOST_GRAPHQL_MAX_RESPONSE_BYTES", defaultGraphQLMaxResponseBytes),
		Headers:          headers,
	}), true, nil
}

func legacyGraphQLSourceFromFile(
	fileCfg bridgeFileConfig,
	fallback GraphQLSourceConfig,
) (GraphQLSourceConfig, bool, error) {
	enabled := fallback.Name != ""
	if fileCfg.GraphQLEnabled != nil {
		enabled = *fileCfg.GraphQLEnabled
	}
	if !enabled {
		return GraphQLSourceConfig{}, false, nil
	}

	source := normalizeGraphQLSourceConfig(fallback)
	source.Name = defaultGraphQLLegacySourceName
	if fileCfg.GraphQLEndpoint != nil {
		source.Endpoint = normalizeOptionalString(*fileCfg.GraphQLEndpoint)
	}
	if fileCfg.GraphQLAPIKey != nil {
		source.APIKey = normalizeOptionalString(*fileCfg.GraphQLAPIKey)
	}
	if fileCfg.GraphQLSchemaPath != nil {
		source.SchemaPath = normalizeOptionalString(*fileCfg.GraphQLSchemaPath)
	}
	if fileCfg.GraphQLTimeoutMS != nil {
		source.TimeoutMS = *fileCfg.GraphQLTimeoutMS
	}
	if fileCfg.GraphQLMaxResponseBytes != nil {
		source.MaxResponseBytes = *fileCfg.GraphQLMaxResponseBytes
	}
	if fileCfg.GraphQLHeaders != nil {
		headers, err := normalizeGraphQLHeaders(fileCfg.GraphQLHeaders)
		if err != nil {
			return GraphQLSourceConfig{}, false, err
		}
		source.Headers = headers
	}
	return normalizeGraphQLSourceConfig(source), true, nil
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

func legacyGraphQLFallbackSource(cfg GraphQLConfig) GraphQLSourceConfig {
	for _, source := range cfg.Sources {
		if source.Name == defaultGraphQLLegacySourceName {
			return source
		}
	}
	return GraphQLSourceConfig{}
}

func hasLegacyGraphQLConfig(fileCfg bridgeFileConfig) bool {
	return fileCfg.GraphQLEnabled != nil ||
		fileCfg.GraphQLEndpoint != nil ||
		fileCfg.GraphQLAPIKey != nil ||
		fileCfg.GraphQLSchemaPath != nil ||
		fileCfg.GraphQLTimeoutMS != nil ||
		fileCfg.GraphQLMaxResponseBytes != nil ||
		fileCfg.GraphQLHeaders != nil
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
