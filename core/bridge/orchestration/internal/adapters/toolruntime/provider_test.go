package toolruntime

import (
	"errors"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	bridgetools "ghost-os/bridge/tools"
)

func TestProviderListSchemasAllowsMissingRuntimeFactory(t *testing.T) {
	schemas, err := (Provider{}).ListToolInputSchemas()
	if err != nil {
		t.Fatalf("list schemas: %v", err)
	}
	if schemas != nil {
		t.Fatalf("expected nil schemas, got %#v", schemas)
	}
}

func TestProviderToolRequiresRuntimeFactory(t *testing.T) {
	_, cleanup, err := (Provider{}).Tool("screen_control")

	if cleanup == nil {
		t.Fatal("expected cleanup function")
	}
	if bus.ErrorKindOf(err) != bus.ServiceErrorUnavailable {
		t.Fatalf("unexpected error kind: %v", err)
	}
}

func TestProviderToolClosesDependenciesWhenRegistryMissing(t *testing.T) {
	deps := &fakeDeps{}
	_, _, err := (Provider{RuntimeFactory: fakeFactory{deps: deps}}).Tool("screen_control")

	if bus.ErrorKindOf(err) != bus.ServiceErrorUnavailable {
		t.Fatalf("unexpected error kind: %v", err)
	}
	if !deps.closed {
		t.Fatal("expected runtime dependencies to close")
	}
}

func TestProviderListSchemasPropagatesBuildError(t *testing.T) {
	wantErr := errors.New("build failed")
	_, err := (Provider{RuntimeFactory: fakeFactory{err: wantErr}}).ListToolInputSchemas()
	if !errors.Is(err, wantErr) {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeFactory struct {
	deps RuntimeDependencies
	err  error
}

func (f fakeFactory) Build(bridgeconfig.Store) (RuntimeDependencies, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.deps, nil
}

type fakeDeps struct {
	registry *bridgetools.Registry
	closed   bool
}

func (d *fakeDeps) Registry() *bridgetools.Registry {
	return d.registry
}

func (d *fakeDeps) Close() {
	d.closed = true
}
