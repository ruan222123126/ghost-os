package config

import (
	"fmt"
	"strings"
)

func validateNoRemovedGraphQLEnv(env envSnapshot) error {
	names := configuredRemovedGraphQLEnv(env)
	if len(names) == 0 {
		return nil
	}
	return fmt.Errorf("unsupported GraphQL env vars: %s", strings.Join(names, ", "))
}

func configuredRemovedGraphQLEnv(env envSnapshot) []string {
	names := []string{
		"GHOST_GRAPHQL_TOOL_RUNTIME_ENABLED",
		"GHOST_GRAPHQL_TEXT_SANITIZE_ENABLED",
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

func validateNoRemovedProEnv(env envSnapshot) error {
	if env.value("GHOST_PRO_MAX_ITERATIONS") == "" {
		return nil
	}
	return fmt.Errorf("unsupported pro env vars: %s", "GHOST_PRO_MAX_ITERATIONS")
}
