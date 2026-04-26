package orchestration

import (
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestConfigResponseFromSnapshotIncludesWebRooterFlags(t *testing.T) {
	response := configResponseFromSnapshot(bridgeconfig.Snapshot{
		GraphQLTextSanitizeEnabled: true,
		SessionHumanLogFullEnabled: true,
		AssistantMarkdownEnabled:   false,
		MemoryModeEnabled:          true,
		WebRooterEnabled:           true,
		WebRooterBaseURL:           "http://127.0.0.1:9988",
		WebRooterTimeoutMS:         120000,
		WebRooterAPITokenSet:       true,
		WebSearchTavilyURL:         "https://proxy.example/tavily",
		WebSearchExaURL:            "https://proxy.example/exa",
		WebSearchTavilyAPIKeySet:   true,
		WebSearchExaAPIKeySet:      true,
	})

	if !response.GraphqlTextSanitizeEnabled {
		t.Fatal("expected graphql_text_sanitize_enabled to be true")
	}
	if !response.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled to be true")
	}
	if response.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be false")
	}
	if !response.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be true")
	}

	if !response.WebRooterEnabled {
		t.Fatal("expected web_rooter_enabled to be true")
	}
	if response.WebRooterBaseURL != "http://127.0.0.1:9988" {
		t.Fatalf("unexpected web_rooter_base_url: got %q want %q", response.WebRooterBaseURL, "http://127.0.0.1:9988")
	}
	if response.WebRooterTimeoutMs != 120000 {
		t.Fatalf("unexpected web_rooter_timeout_ms: got %d want %d", response.WebRooterTimeoutMs, 120000)
	}
	if !response.WebRooterAPITokenSet {
		t.Fatal("expected web_rooter_api_token_set to be true")
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
}
