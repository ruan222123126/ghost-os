package runtime

import (
	"strings"
	"testing"

	"ghost-os/bridge/config/internal/storage"
)

func TestFallbackFromEnvRejectsRemovedGraphQLEnvVars(t *testing.T) {
	env := storage.EnvSnapshot{
		"GHOST_GRAPHQL_TOOL_RUNTIME_ENABLED":  "true",
		"GHOST_GRAPHQL_TEXT_SANITIZE_ENABLED": "false",
	}

	if _, err := FallbackFromEnv(env); err == nil {
		t.Fatal("expected removed graphql env config error")
	} else if !strings.Contains(err.Error(), "GHOST_GRAPHQL_TOOL_RUNTIME_ENABLED") ||
		!strings.Contains(err.Error(), "GHOST_GRAPHQL_TEXT_SANITIZE_ENABLED") {
		t.Fatalf("unexpected error: %v", err)
	}
}
