package promptpreview

import (
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/tools"
)

func TestLoaderRequiresRuntimeFactory(t *testing.T) {
	_, err := (Loader{}).Load(bridgeconfig.Config{})
	if err == nil || !strings.Contains(err.Error(), "runtime factory unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoaderClosesRuntimeDependenciesWhenRegistryMissing(t *testing.T) {
	deps := &fakeRuntimeDependencies{}
	_, err := (Loader{RuntimeFactory: fakeRuntimeFactory{deps: deps}}).Load(bridgeconfig.Config{})

	if err == nil || !strings.Contains(err.Error(), "tool registry unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deps.closed {
		t.Fatal("expected runtime dependencies to close")
	}
}

type fakeRuntimeFactory struct {
	deps RuntimeDependencies
}

func (f fakeRuntimeFactory) Build(bridgeconfig.Store) (RuntimeDependencies, error) {
	return f.deps, nil
}

type fakeRuntimeDependencies struct {
	cfg      bridgeconfig.Config
	registry *tools.Registry
	closed   bool
}

func (d *fakeRuntimeDependencies) Config() bridgeconfig.Config {
	return d.cfg
}

func (d *fakeRuntimeDependencies) Registry() *tools.Registry {
	return d.registry
}

func (d *fakeRuntimeDependencies) Close() {
	d.closed = true
}
