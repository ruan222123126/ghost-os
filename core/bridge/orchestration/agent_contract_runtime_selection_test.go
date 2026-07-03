package orchestration

import (
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

func TestNewRuntimeDependenciesProvidesRuntimeSelection(t *testing.T) {
	deps := NewRuntimeDependencies(
		bridgeconfig.Config{
			Provider: bridgeconfig.ProviderConfig{
				Type:  llm.ProviderAnthropic,
				Model: "claude-4",
			},
		},
		nil,
		nil,
		"",
		nil,
	)

	first := deps.RuntimeSelection()
	if first == nil {
		t.Fatal("expected runtime selection")
	}
	if first.Runtime != session.RuntimeSelectionGhost {
		t.Fatalf("unexpected runtime: got %q", first.Runtime)
	}
	if first.ProviderType != string(llm.ProviderAnthropic) {
		t.Fatalf("unexpected provider_type: got %q", first.ProviderType)
	}
	if first.Model != "claude-4" {
		t.Fatalf("unexpected model: got %q", first.Model)
	}
	if first.Mode != session.RuntimeSelectionModeDefault {
		t.Fatalf("unexpected mode: got %q", first.Mode)
	}

	first.Model = "mutated"
	second := deps.RuntimeSelection()
	if second == nil {
		t.Fatal("expected runtime selection on second read")
	}
	if second.Model != "claude-4" {
		t.Fatalf("expected cloned runtime selection, got model=%q", second.Model)
	}
}
