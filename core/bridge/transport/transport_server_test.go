package transport

import (
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
