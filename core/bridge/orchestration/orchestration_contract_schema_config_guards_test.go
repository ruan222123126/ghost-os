package orchestration

import (
	"context"
	"encoding/json"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestSchemaActionEnumMatchesRegisteredBusActions(t *testing.T) {
	schemaActions := loadSchemaActionEnum(t)
	runtimeActions := registeredBusActions()

	missingInRuntime := actionDiff(schemaActions, runtimeActions)
	if len(missingInRuntime) != 0 {
		t.Fatalf("schema declares unsupported bus actions: %v", missingInRuntime)
	}

	missingInSchema := actionDiff(runtimeActions, schemaActions)
	if len(missingInSchema) != 0 {
		t.Fatalf("runtime bus actions missing from schema enum: %v", missingInSchema)
	}
}

func loadSchemaActionEnum(t *testing.T) map[string]struct{} {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "shared", "schema", "defs", "base.json"))

	type actionSchema struct {
		Defs struct {
			Action struct {
				Enum []string `json:"enum"`
			} `json:"action"`
		} `json:"$defs"`
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read base schema: %v", err)
	}

	var parsed actionSchema
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("decode base schema: %v", err)
	}

	actions := make(map[string]struct{}, len(parsed.Defs.Action.Enum))
	for _, action := range parsed.Defs.Action.Enum {
		normalized := strings.TrimSpace(action)
		if normalized != "" {
			actions[normalized] = struct{}{}
		}
	}
	return actions
}

func registeredBusActions() map[string]struct{} {
	service := newBridgeServiceState(nil, nil)
	registerDefaultActions(service)

	names := service.registeredActionNames()
	actions := make(map[string]struct{}, len(names))
	for _, action := range names {
		actions[action] = struct{}{}
	}
	return actions
}

func actionDiff(left map[string]struct{}, right map[string]struct{}) []string {
	missing := make([]string, 0, len(left))
	for action := range left {
		if _, ok := right[action]; !ok {
			missing = append(missing, action)
		}
	}
	sort.Strings(missing)
	return missing
}

func TestConfigUpdatePropagatesRuntimeFlagsThroughOrchestration(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	raw := json.RawMessage(`{
		"max_turns": 9,
		"task_execution_timeout_ms": 600000,
		"llm_completion_retry_count": 0,
		"llm_completion_retry_interval_ms": 150,
		"session_human_log_full_enabled": true,
		"session_system_prompt_visible_enabled": false,
		"assistant_markdown_enabled": false,
		"tool_call_compact_output_enabled": true,
		"memory_mode_enabled": true,
		"microcompact_enabled": true,
		"session_title_mode": "first_message"
	}`)

	result, err := service.dispatchAction(
		context.Background(),
		busActionConfigUpdate,
		raw,
		"trace-web-rooter-update",
	)
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}
	if result.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}

	snapshot, ok := result.Payload.(configResponse)
	if !ok {
		t.Fatalf("unexpected payload type: %T", result.Payload)
	}
	if !snapshot.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled in snapshot")
	}
	if snapshot.MaxTurns != 9 {
		t.Fatalf("expected max_turns to be 9 in snapshot, got %d", snapshot.MaxTurns)
	}
	if snapshot.TaskExecutionTimeoutMs != 600000 {
		t.Fatalf(
			"expected task_execution_timeout_ms to be 600000 in snapshot, got %d",
			snapshot.TaskExecutionTimeoutMs,
		)
	}
	if snapshot.LlmCompletionRetryCount != 0 {
		t.Fatalf(
			"expected llm_completion_retry_count to be 0 in snapshot, got %d",
			snapshot.LlmCompletionRetryCount,
		)
	}
	if snapshot.LlmCompletionRetryIntervalMs != 150 {
		t.Fatalf(
			"expected llm_completion_retry_interval_ms to be 150 in snapshot, got %d",
			snapshot.LlmCompletionRetryIntervalMs,
		)
	}
	if snapshot.SessionSystemPromptVisibleEnabled {
		t.Fatal("expected session_system_prompt_visible_enabled to be false in snapshot")
	}
	if snapshot.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be false in snapshot")
	}
	if !snapshot.ToolCallCompactOutputEnabled {
		t.Fatal("expected tool_call_compact_output_enabled to be true in snapshot")
	}
	if !snapshot.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be true in snapshot")
	}
	if !snapshot.MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to be true in snapshot")
	}
	if snapshot.SessionTitleMode != "first_message" {
		t.Fatalf("expected session_title_mode in snapshot, got %q", snapshot.SessionTitleMode)
	}

	cfg, err := service.configStore.Config()
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if !cfg.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled in runtime config")
	}
	if cfg.MaxTurns != 9 {
		t.Fatalf("expected max_turns to be 9 in runtime config, got %d", cfg.MaxTurns)
	}
	if cfg.TaskExecutionTimeoutMS != 600000 {
		t.Fatalf(
			"expected task_execution_timeout_ms to be 600000 in runtime config, got %d",
			cfg.TaskExecutionTimeoutMS,
		)
	}
	if cfg.LLMCompletionRetryCount != 0 {
		t.Fatalf(
			"expected llm_completion_retry_count to be 0 in runtime config, got %d",
			cfg.LLMCompletionRetryCount,
		)
	}
	if cfg.LLMCompletionRetryIntervalMS != 150 {
		t.Fatalf(
			"expected llm_completion_retry_interval_ms to be 150 in runtime config, got %d",
			cfg.LLMCompletionRetryIntervalMS,
		)
	}
	if cfg.SessionSystemPromptVisible {
		t.Fatal("expected session_system_prompt_visible_enabled to be false in runtime config")
	}
	if cfg.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be false in runtime config")
	}
	if !cfg.ToolCallCompactOutputEnabled {
		t.Fatal("expected tool_call_compact_output_enabled to be true in runtime config")
	}
	if !cfg.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be true in runtime config")
	}
	if !cfg.MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to be true in runtime config")
	}
	if cfg.SessionTitleMode != "first_message" {
		t.Fatalf("expected session_title_mode in runtime config, got %q", cfg.SessionTitleMode)
	}
	scheduler := service.taskScheduler()
	if scheduler == nil {
		t.Fatal("expected task scheduler to be initialized")
	}
	if got := scheduler.ExecutionTimeout(); got != 10*time.Minute {
		t.Fatalf("expected scheduler timeout to be 10m, got %s", got)
	}
}

func TestEnsureSessionActiveRequiresSessionStoreForExistingSession(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionActive("session-1")
	if err == nil {
		t.Fatal("expected error when session store is missing")
	}
	if kind := ServiceErrorKindOf(err); kind != ServiceErrorInternal {
		t.Fatalf("unexpected error kind: got=%s want=%s", kind, ServiceErrorInternal)
	}
	if code := bus.StatusFromError(err); code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got=%d want=%d", code, http.StatusInternalServerError)
	}
	if !strings.Contains(err.Error(), "session store is not configured") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnsureSessionActiveSkipsValidationWhenSessionIDEmpty(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionActive("   ")
	if err != nil {
		t.Fatalf("expected nil error for empty session id, got: %v", err)
	}
}

func TestEnsureSessionNotInflightRequiresRunRegistryForExistingSession(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionNotInflight("session-1")
	if err == nil {
		t.Fatal("expected error when run registry is missing")
	}
	if kind := ServiceErrorKindOf(err); kind != ServiceErrorInternal {
		t.Fatalf("unexpected error kind: got=%s want=%s", kind, ServiceErrorInternal)
	}
	if code := bus.StatusFromError(err); code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got=%d want=%d", code, http.StatusInternalServerError)
	}
	if !strings.Contains(err.Error(), "run registry is not configured") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnsureSessionNotInflightSkipsValidationWhenSessionIDEmpty(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionNotInflight("")
	if err != nil {
		t.Fatalf("expected nil error for empty session id, got: %v", err)
	}
}
