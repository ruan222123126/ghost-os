package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveAuxConfigUsesProvidedEnvSnapshot(t *testing.T) {
	t.Setenv("GHOST_SESSIONS_PATH", "/process/sessions")
	t.Setenv("GHOST_RSS_FEEDS_PATH", "/process/feeds")
	t.Setenv("GHOST_WEB_SEARCH_TAVILY_API_KEY", "process-tavily")
	t.Setenv("GHOST_NATIVE_BINARY_PATH", "/process/native")

	env := envSnapshot{
		"GHOST_SESSIONS_PATH":              "/snapshot/sessions",
		"GHOST_RSS_FEEDS_PATH":             "/snapshot/feeds",
		"GHOST_RSS_POLL_ENABLED":           "false",
		"GHOST_RSS_POLL_INTERVAL":          "15m",
		"GHOST_WEB_SEARCH_TAVILY_API_KEY":  "snapshot-tavily",
		"GHOST_NATIVE_PERSISTENT":          "true",
		"GHOST_NATIVE_BINARY_PATH":         "/snapshot/native",
		"GHOST_NATIVE_ALLOWED_READ_PATHS":  "/snapshot/read-a,/snapshot/read-b",
		"GHOST_NATIVE_ALLOWED_WRITE_PATHS": "/snapshot/write-a,/snapshot/write-b",
		"GHOST_PROJECT_ROOT":               "/snapshot/project",
	}

	aux, err := resolveAuxConfig(bridgeFileConfig{}, env)
	if err != nil {
		t.Fatalf("resolveAuxConfig: %v", err)
	}
	if aux.SessionsPath != "/snapshot/sessions" {
		t.Fatalf("unexpected sessions path: got %q want %q", aux.SessionsPath, "/snapshot/sessions")
	}
	if aux.RSS.FeedsPath != "/snapshot/feeds" {
		t.Fatalf("unexpected rss feeds path: got %q want %q", aux.RSS.FeedsPath, "/snapshot/feeds")
	}
	if aux.RSS.PollEnabled {
		t.Fatalf("expected rss poll to use snapshot env value, got enabled")
	}
	if aux.RSS.PollInterval != 15*time.Minute {
		t.Fatalf("unexpected rss poll interval: got %s want %s", aux.RSS.PollInterval, 15*time.Minute)
	}
	if aux.WebSearchTavilyAPIKey != "snapshot-tavily" {
		t.Fatalf("unexpected tavily api key: got %q want %q", aux.WebSearchTavilyAPIKey, "snapshot-tavily")
	}
	if !aux.Execution.Persistent {
		t.Fatalf("expected native persistent to use snapshot env value")
	}
	if aux.Execution.NativeBinaryPath != "/snapshot/native" {
		t.Fatalf("unexpected native binary path: got %q want %q", aux.Execution.NativeBinaryPath, "/snapshot/native")
	}
	if aux.Execution.ProjectRoot != "/snapshot/project" {
		t.Fatalf("unexpected project root: got %q want %q", aux.Execution.ProjectRoot, "/snapshot/project")
	}
	if len(aux.Execution.AllowedReadPaths) != 2 {
		t.Fatalf("unexpected read path count: got %d want %d", len(aux.Execution.AllowedReadPaths), 2)
	}
	if len(aux.Execution.AllowedWritePaths) != 2 {
		t.Fatalf("unexpected write path count: got %d want %d", len(aux.Execution.AllowedWritePaths), 2)
	}
}

func TestResolveAuxConfigFailsFastOnInvalidEnvValues(t *testing.T) {
	cases := []struct {
		name string
		env  envSnapshot
		want string
	}{
		{
			name: "native persistent",
			env:  envSnapshot{"GHOST_NATIVE_PERSISTENT": "maybe"},
			want: "invalid GHOST_NATIVE_PERSISTENT",
		},
		{
			name: "rss poll interval",
			env:  envSnapshot{"GHOST_RSS_POLL_INTERVAL": "later"},
			want: "invalid GHOST_RSS_POLL_INTERVAL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveAuxConfig(bridgeFileConfig{}, tc.env)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestLoadExecutionConfigFailsFastOnInvalidEnv(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.toml"))
	t.Setenv("GHOST_NATIVE_PERSISTENT", "maybe")

	_, err := LoadExecutionConfig()
	if err == nil || !strings.Contains(err.Error(), "invalid GHOST_NATIVE_PERSISTENT") {
		t.Fatalf("expected invalid native persistent error, got %v", err)
	}
}
