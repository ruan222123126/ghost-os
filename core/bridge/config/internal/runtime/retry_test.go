package runtime

import (
	"strings"
	"testing"

	"ghost-os/bridge/config/internal/storage"
)

func TestResolveLoadsDefaultLLMCompletionRetrySettings(t *testing.T) {
	cfg, err := Resolve(storage.FileConfig{}, storage.EnvSnapshot{"GHOST_PROVIDER": "custom"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.LLMCompletionRetryCount != DefaultLLMCompletionRetryCount {
		t.Fatalf(
			"unexpected llm_completion_retry_count: got %d want %d",
			cfg.LLMCompletionRetryCount,
			DefaultLLMCompletionRetryCount,
		)
	}
	if cfg.LLMCompletionRetryIntervalMS != DefaultLLMCompletionRetryIntervalMS {
		t.Fatalf(
			"unexpected llm_completion_retry_interval_ms: got %d want %d",
			cfg.LLMCompletionRetryIntervalMS,
			DefaultLLMCompletionRetryIntervalMS,
		)
	}
}

func TestResolveAllowsZeroLLMCompletionRetrySettings(t *testing.T) {
	cfg, err := Resolve(storage.FileConfig{
		LLMCompletionRetryCount:      intPointer(0),
		LLMCompletionRetryIntervalMS: intPointer(0),
	}, storage.EnvSnapshot{
		"GHOST_PROVIDER":                         "custom",
		"GHOST_LLM_COMPLETION_RETRY_COUNT":       "5",
		"GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS": "500",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.LLMCompletionRetryCount != 0 {
		t.Fatalf("expected llm_completion_retry_count to stay 0, got %d", cfg.LLMCompletionRetryCount)
	}
	if cfg.LLMCompletionRetryIntervalMS != 0 {
		t.Fatalf("expected llm_completion_retry_interval_ms to stay 0, got %d", cfg.LLMCompletionRetryIntervalMS)
	}
}

func TestResolveRejectsInvalidLLMCompletionRetrySettings(t *testing.T) {
	cases := []struct {
		name    string
		fileCfg storage.FileConfig
		env     storage.EnvSnapshot
		want    string
	}{
		{
			name: "env negative retry count",
			env: storage.EnvSnapshot{
				"GHOST_PROVIDER":                   "custom",
				"GHOST_LLM_COMPLETION_RETRY_COUNT": "-1",
			},
			want: "invalid GHOST_LLM_COMPLETION_RETRY_COUNT",
		},
		{
			name: "env invalid retry interval",
			env: storage.EnvSnapshot{
				"GHOST_PROVIDER":                         "custom",
				"GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS": "fast",
			},
			want: "invalid GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS",
		},
		{
			name: "file negative retry interval",
			fileCfg: storage.FileConfig{
				LLMCompletionRetryIntervalMS: intPointer(-1),
			},
			env:  storage.EnvSnapshot{"GHOST_PROVIDER": "custom"},
			want: "invalid llm_completion_retry_interval_ms",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Resolve(tc.fileCfg, tc.env)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func intPointer(value int) *int {
	return &value
}
