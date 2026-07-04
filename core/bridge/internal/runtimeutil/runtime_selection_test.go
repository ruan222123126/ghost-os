package runtimeutil

import (
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

func TestBuildGhostRuntimeSelectionFromSnapshotUsesSnapshotValues(t *testing.T) {
	selection := BuildGhostRuntimeSelectionFromSnapshot(
		bridgeconfig.Snapshot{
			Provider:     "openai-main",
			ProviderType: string(llm.ProviderOpenAI),
			Model:        "gpt-5.4",
		},
		bridgeconfig.Config{
			Provider: bridgeconfig.ProviderConfig{
				Type:  llm.ProviderAnthropic,
				Model: "claude-4",
			},
		},
		session.RuntimeSelectionModeDefault,
	)

	if selection == nil {
		t.Fatal("expected runtime selection")
	}
	if selection.Runtime != session.RuntimeSelectionGhost {
		t.Fatalf("unexpected runtime: got %q", selection.Runtime)
	}
	if selection.Provider != "openai-main" {
		t.Fatalf("unexpected provider: got %q", selection.Provider)
	}
	if selection.ProviderType != string(llm.ProviderOpenAI) {
		t.Fatalf("unexpected provider_type: got %q", selection.ProviderType)
	}
	if selection.Model != "gpt-5.4" {
		t.Fatalf("unexpected model: got %q", selection.Model)
	}
	if selection.Mode != session.RuntimeSelectionModeDefault {
		t.Fatalf("unexpected mode: got %q", selection.Mode)
	}
}

func TestBuildGhostRuntimeSelectionFromSnapshotFallsBackToCustomProviderType(t *testing.T) {
	selection := BuildGhostRuntimeSelectionFromSnapshot(
		bridgeconfig.Snapshot{},
		bridgeconfig.Config{},
		"",
	)

	if selection == nil {
		t.Fatal("expected runtime selection")
	}
	if selection.ProviderType != string(llm.ProviderCustom) {
		t.Fatalf("unexpected provider_type: got %q want %q", selection.ProviderType, llm.ProviderCustom)
	}
	if selection.Provider != string(llm.ProviderCustom) {
		t.Fatalf("unexpected provider: got %q want %q", selection.Provider, llm.ProviderCustom)
	}
	if selection.Mode != session.RuntimeSelectionModeDefault {
		t.Fatalf("unexpected mode: got %q want %q", selection.Mode, session.RuntimeSelectionModeDefault)
	}
}
