package transport

import (
	"errors"
	"strings"
	"testing"
)

func TestResolveBindAddrDefaultsToLocalhost(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_BIND_ADDR", "")
	got, err := resolveBindAddr(8080)
	if err != nil {
		t.Fatalf("resolve bind addr: %v", err)
	}
	if want := "127.0.0.1:8080"; got != want {
		t.Fatalf("unexpected bind addr: got %q want %q", got, want)
	}
}

func TestResolveBindAddrUsesOverride(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_BIND_ADDR", "0.0.0.0:9090")
	got, err := resolveBindAddr(8080)
	if err != nil {
		t.Fatalf("resolve bind addr: %v", err)
	}
	if want := "0.0.0.0:9090"; got != want {
		t.Fatalf("unexpected bind addr: got %q want %q", got, want)
	}
}

func TestServeStartupErrorIncludesStageName(t *testing.T) {
	err := newServeStartupError(startupStageConfig, errors.New("config load failed"))
	if err == nil {
		t.Fatal("expected startup error")
	}
	if !strings.Contains(err.Error(), "stage=config") {
		t.Fatalf("expected stage in error, got %v", err)
	}
}

func TestServeStartupErrorSupportsUnwrap(t *testing.T) {
	root := errors.New("listen failed")
	err := newServeStartupError(startupStageListen, root)
	if !errors.Is(err, root) {
		t.Fatalf("expected wrapped error to support errors.Is, got %v", err)
	}
}
