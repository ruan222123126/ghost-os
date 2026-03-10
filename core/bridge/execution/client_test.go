package execution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocateNativeBinaryUsesConfiguredPath(t *testing.T) {
	tmp := t.TempDir()
	binary := filepath.Join(tmp, "native")
	if err := writeFile(binary, []byte("stub")); err != nil {
		t.Fatalf("write binary fixture: %v", err)
	}

	got, err := locateNativeBinary(nativeBinaryLocator{configuredPath: binary})
	if err != nil {
		t.Fatalf("locateNativeBinary returned error: %v", err)
	}

	want, err := filepath.Abs(binary)
	if err != nil {
		t.Fatalf("abs binary path: %v", err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("unexpected path: got %q want %q", got, filepath.Clean(want))
	}
}

func TestLocateNativeBinaryRejectsInvalidConfiguredPath(t *testing.T) {
	configured := filepath.Join(t.TempDir(), "missing-native")

	_, err := locateNativeBinary(nativeBinaryLocator{configuredPath: configured})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "native binary path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLocateNativeBinaryInRootsFindsFromSecondRoot(t *testing.T) {
	rootOne := t.TempDir()
	rootTwo := t.TempDir()
	candidate := filepath.Join("drivers", "native", "target", "debug", "native")
	binary := filepath.Join(rootTwo, candidate)

	if err := writeFile(binary, []byte("stub")); err != nil {
		t.Fatalf("write binary fixture: %v", err)
	}

	got, err := locateNativeBinaryInRoots([]string{candidate}, []string{rootOne, rootTwo})
	if err != nil {
		t.Fatalf("locateNativeBinaryInRoots returned error: %v", err)
	}

	want, err := filepath.Abs(binary)
	if err != nil {
		t.Fatalf("abs binary path: %v", err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("unexpected path: got %q want %q", got, filepath.Clean(want))
	}
}

func writeFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o755)
}
