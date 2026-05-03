package orchestration

import (
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestConfigResponseFromSnapshotIncludesRuntimeFlags(t *testing.T) {
	response := configResponseFromSnapshot(bridgeconfig.Snapshot{
		MaxTurns:                     9,
		LLMCompletionRetryCount:      0,
		LLMCompletionRetryIntervalMS: 150,
		SessionHumanLogFullEnabled:   true,
		SessionSystemPromptVisible:   false,
		AssistantMarkdownEnabled:     false,
		ToolCallCompactOutputEnabled: true,
		MemoryModeEnabled:            true,
		MicrocompactEnabled:          true,
		WebSearchTavilyURL:           "https://proxy.example/tavily",
		WebSearchExaURL:              "https://proxy.example/exa",
		WebSearchTavilyAPIKeySet:     true,
		WebSearchExaAPIKeySet:        true,
		ProjectRoot:                  "/tmp/ghost-os",
	})

	if !response.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled to be true")
	}
	if response.MaxTurns != 9 {
		t.Fatalf("unexpected max_turns: got %d want %d", response.MaxTurns, 9)
	}
	if response.LlmCompletionRetryCount != 0 {
		t.Fatalf("unexpected llm_completion_retry_count: got %d want %d", response.LlmCompletionRetryCount, 0)
	}
	if response.LlmCompletionRetryIntervalMs != 150 {
		t.Fatalf(
			"unexpected llm_completion_retry_interval_ms: got %d want %d",
			response.LlmCompletionRetryIntervalMs,
			150,
		)
	}
	if response.SessionSystemPromptVisibleEnabled {
		t.Fatal("expected session_system_prompt_visible_enabled to be false")
	}
	if response.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be false")
	}
	if !response.ToolCallCompactOutputEnabled {
		t.Fatal("expected tool_call_compact_output_enabled to be true")
	}
	if !response.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be true")
	}
	if !response.MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to be true")
	}
	if response.WebSearchTavilyURL != "https://proxy.example/tavily" {
		t.Fatalf("unexpected web_search_tavily_url: got %q want %q", response.WebSearchTavilyURL, "https://proxy.example/tavily")
	}
	if response.WebSearchExaURL != "https://proxy.example/exa" {
		t.Fatalf("unexpected web_search_exa_url: got %q want %q", response.WebSearchExaURL, "https://proxy.example/exa")
	}
	if !response.WebSearchTavilyAPIKeySet {
		t.Fatal("expected web_search_tavily_api_key_set to be true")
	}
	if !response.WebSearchExaAPIKeySet {
		t.Fatal("expected web_search_exa_api_key_set to be true")
	}
	if response.ProjectRoot != "/tmp/ghost-os" {
		t.Fatalf("unexpected project_root: got %q want %q", response.ProjectRoot, "/tmp/ghost-os")
	}
}
