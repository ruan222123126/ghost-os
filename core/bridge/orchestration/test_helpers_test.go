package orchestration

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/session"
)

func newTestHandlerWithService(t *testing.T, executor agentExecutorFunc, streamExecutor agentStreamExecutorFunc) (http.Handler, *bridgeService, *session.Store) {
	t.Helper()

	tempDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", tempDir+"/config.toml")
	t.Setenv("GHOST_API_KEY", "test-key")
	t.Setenv("GHOST_TASKS_PATH", tempDir+"/tasks")
	t.Setenv("GHOST_PROMPTS_DIR", tempDir+"/prompts")
	t.Setenv("GHOST_RSS_POLL_ENABLED", "false")
	t.Setenv("GHOST_RSS_BRIEFING_ENABLED", "false")
	t.Setenv("GHOST_RSS_INBOX_PATH", tempDir+"/rss/inbox.json")
	t.Setenv("GHOST_RSS_FEEDS_PATH", tempDir+"/rss/feeds.json")

	if executor == nil {
		executor = func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
			return "ok", "session-test", nil
		}
	}

	store, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}

	sessionStore, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	service := newBridgeServiceWithStreamExecutor(store, sessionStore, executor, streamExecutor)
	if err := service.StartBackgroundRuntimes(); err != nil {
		t.Fatalf("start background runtimes: %v", err)
	}
	t.Cleanup(service.Close)
	return nil, service, sessionStore
}

func newTempSessionStore(t *testing.T) *session.Store {
	t.Helper()

	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(t.TempDir(), "prompts"))
	store, err := session.NewStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	return store
}
