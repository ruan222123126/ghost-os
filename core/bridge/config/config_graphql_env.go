package config

import (
	"fmt"
	"strings"
)

func resolveGraphQLSettingsFromEnv(env envSnapshot) (GraphQLConfig, error) {
	if err := validateNoLegacyGraphQLEnvFromEnv(env); err != nil {
		return GraphQLConfig{}, err
	}
	return finalizeGraphQLConfig(GraphQLConfig{})
}

func validateNoLegacyGraphQLEnvFromEnv(env envSnapshot) error {
	legacyEnv := configuredLegacyGraphQLEnvFromEnv(env)
	if len(legacyEnv) == 0 {
		return nil
	}
	return fmt.Errorf(
		"legacy graphql env vars are no longer supported: %s; configure graphql_sources/graphql_mutation_policies in the bridge config file",
		strings.Join(legacyEnv, ", "),
	)
}

func configuredLegacyGraphQLEnvFromEnv(env envSnapshot) []string {
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
