package config

import (
	"os"
	"strings"
)

func envGraphQLSettings() (GraphQLConfig, error) {
	headers, err := parseGraphQLHeaders(os.Getenv("GHOST_GRAPHQL_HEADERS"))
	if err != nil {
		return GraphQLConfig{}, err
	}
	return normalizeGraphQLConfig(GraphQLConfig{
		Enabled:          parseBoolEnv("GHOST_GRAPHQL_ENABLED", false),
		Endpoint:         getenvDefault("GHOST_GRAPHQL_ENDPOINT", ""),
		APIKey:           firstNonEmptyEnv("GHOST_GRAPHQL_API_KEY"),
		SchemaPath:       getenvDefault("GHOST_GRAPHQL_SCHEMA_PATH", ""),
		TimeoutMS:        parsePositiveIntEnv("GHOST_GRAPHQL_TIMEOUT_MS", defaultGraphQLTimeoutMS),
		MaxResponseBytes: parsePositiveIntEnv("GHOST_GRAPHQL_MAX_RESPONSE_BYTES", defaultGraphQLMaxResponseBytes),
		Headers:          headers,
	}), nil
}

func fileGraphQLSettings(fileCfg bridgeFileConfig, fallback GraphQLConfig) (GraphQLConfig, error) {
	settings := normalizeGraphQLConfig(fallback)
	if fileCfg.GraphQLEnabled != nil {
		settings.Enabled = *fileCfg.GraphQLEnabled
	}
	if fileCfg.GraphQLEndpoint != nil {
		settings.Endpoint = *fileCfg.GraphQLEndpoint
	}
	if fileCfg.GraphQLAPIKey != nil {
		settings.APIKey = *fileCfg.GraphQLAPIKey
	}
	if fileCfg.GraphQLSchemaPath != nil {
		settings.SchemaPath = *fileCfg.GraphQLSchemaPath
	}
	if fileCfg.GraphQLTimeoutMS != nil {
		settings.TimeoutMS = normalizePositiveInt(*fileCfg.GraphQLTimeoutMS)
	}
	if fileCfg.GraphQLMaxResponseBytes != nil {
		settings.MaxResponseBytes = normalizePositiveInt(*fileCfg.GraphQLMaxResponseBytes)
	}
	if fileCfg.GraphQLHeaders != nil {
		headers, err := normalizeGraphQLHeaders(fileCfg.GraphQLHeaders)
		if err != nil {
			return GraphQLConfig{}, err
		}
		settings.Headers = headers
	}
	return normalizeGraphQLConfig(settings), nil
}

func normalizeGraphQLConfig(cfg GraphQLConfig) GraphQLConfig {
	out := cfg
	out.Endpoint = normalizeOptionalString(out.Endpoint)
	out.APIKey = normalizeOptionalString(out.APIKey)
	out.SchemaPath = normalizeOptionalString(out.SchemaPath)
	out.TimeoutMS = normalizeWithDefault(out.TimeoutMS, defaultGraphQLTimeoutMS)
	out.MaxResponseBytes = normalizeWithDefault(out.MaxResponseBytes, defaultGraphQLMaxResponseBytes)
	out.Headers, _ = normalizeGraphQLHeaders(out.Headers)
	return out
}

func normalizeOptionalString(raw string) string {
	return strings.TrimSpace(raw)
}

func normalizeWithDefault(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
