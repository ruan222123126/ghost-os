package runtime

import "testing"

func TestAgentRuntimeFactoryWebRooterDisabledKeepsToolHidden(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.registry.Get("web_rooter") != nil {
		t.Fatal("expected web_rooter to stay hidden when disabled")
	}
}

func TestAgentRuntimeFactoryWebRooterEnabledRegistersAlongsideWebSearch(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	t.Setenv("GHOST_WEB_ROOTER_ENABLED", "true")
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.registry.Get("web_rooter") == nil {
		t.Fatal("expected web_rooter to be registered")
	}
	if deps.registry.Get("web_search") == nil {
		t.Fatal("expected web_search to stay registered alongside web_rooter")
	}
}
