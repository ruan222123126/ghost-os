package orchestration

import (
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestConfigResponseFromSnapshotIncludesWebRooterFlags(t *testing.T) {
	response := configResponseFromSnapshot(bridgeconfig.Snapshot{
		GraphQLTextSanitizeEnabled: true,
		WebRooterEnabled:           true,
		WebRooterAPITokenSet:       true,
		WebSearchTavilyURL:         "https://proxy.example/tavily",
		WebSearchExaURL:            "https://proxy.example/exa",
		WebSearchTavilyAPIKeySet:   true,
		WebSearchExaAPIKeySet:      true,
	})

	if !response.GraphqlTextSanitizeEnabled {
		t.Fatal("expected graphql_text_sanitize_enabled to be true")
	}

	if !response.WebRooterEnabled {
		t.Fatal("expected web_rooter_enabled to be true")
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
