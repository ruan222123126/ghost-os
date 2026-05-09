package orchestration

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

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
		"microcompact_enabled": true
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
	scheduler := service.taskScheduler()
	if scheduler == nil {
		t.Fatal("expected task scheduler to be initialized")
	}
	if got := scheduler.ExecutionTimeout(); got != 10*time.Minute {
		t.Fatalf("expected scheduler timeout to be 10m, got %s", got)
	}
}
