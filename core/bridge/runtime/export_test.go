package runtime

import (
	"testing"

	"ghost-os/bridge/session"
)

func TestAgentRuntimeDependenciesRuntimeSelectionClonesValue(t *testing.T) {
	deps := agentRuntimeDependencies{
		runtimeSelection: &session.RuntimeSelection{
			Runtime:      session.RuntimeSelectionGhost,
			Provider:     "openai-main",
			ProviderType: "openai",
			Model:        "gpt-5.4",
			Mode:         session.RuntimeSelectionModeDefault,
		},
	}

	first := deps.RuntimeSelection()
	if first == nil {
		t.Fatal("expected runtime selection")
	}
	first.Model = "mutated"

	second := deps.RuntimeSelection()
	if second == nil {
		t.Fatal("expected runtime selection on second read")
	}
	if second.Model != "gpt-5.4" {
		t.Fatalf("expected cloned runtime selection, got model=%q", second.Model)
	}
}
