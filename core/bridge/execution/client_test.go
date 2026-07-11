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

func TestLocateNativeBinaryDefaultsIgnoreRepositoryRelativeCandidates(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "drivers", "native", "target", "debug", "native")

	if err := writeFile(project, []byte("project")); err != nil {
		t.Fatalf("write project binary fixture: %v", err)
	}

	_, err := locateNativeBinary(nativeBinaryLocator{roots: []string{root}})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "native binary not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLocateNativeBinaryUsesRepositoryCandidateWhenExplicitlyConfigured(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join("drivers", "native", "target", "debug", "native")
	project := filepath.Join(root, candidate)

	if err := writeFile(project, []byte("project")); err != nil {
		t.Fatalf("write project binary fixture: %v", err)
	}

	got, err := locateNativeBinary(nativeBinaryLocator{
		roots:      []string{root},
		candidates: []string{candidate, "native"},
	})
	if err != nil {
		t.Fatalf("locateNativeBinary returned error: %v", err)
	}

	want, err := filepath.Abs(project)
	if err != nil {
		t.Fatalf("abs project binary path: %v", err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("unexpected path: got %q want %q", got, filepath.Clean(want))
	}
}

func TestNewClientWithOptionsExposesPersistentSessionCapability(t *testing.T) {
	oneShot := NewClientWithOptions(ClientOptions{Persistent: false})
	persistent := NewClientWithOptions(ClientOptions{Persistent: true})

	type persistentSupport interface {
		SupportsPersistentSessions() bool
	}

	oneShotSupport, ok := oneShot.(persistentSupport)
	if !ok || oneShotSupport.SupportsPersistentSessions() {
		t.Fatalf("unexpected one-shot persistent support: %T", oneShot)
	}

	persistentSupportClient, ok := persistent.(persistentSupport)
	if !ok || !persistentSupportClient.SupportsPersistentSessions() {
		t.Fatalf("unexpected persistent support: %T", persistent)
	}
}

func writeFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o755)
}
