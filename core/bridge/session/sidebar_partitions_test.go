package session

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadSidebarPartitionStateReturnsEmptyState(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	state, err := store.LoadSidebarPartitionState()
	if err != nil {
		t.Fatalf("load sidebar partition state: %v", err)
	}

	expectSidebarStateEqual(t, state, emptySidebarPartitionState())
}

func TestSaveSidebarPartitionStateRoundTrips(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	saveSessionForSidebarTest(t, store, "session-a")
	saveSessionForSidebarTest(t, store, "session-b")

	input := SessionSidebarPartitionState{
		Version: sessionSidebarPartitionVersion,
		Partitions: []SessionSidebarPartition{
			{ID: "work", Name: "Work"},
			{ID: "personal", Name: "Personal"},
		},
		Assignments: map[string]string{
			"session-a": "work",
			"session-b": "personal",
		},
	}

	saved, err := store.SaveSidebarPartitionState(input)
	if err != nil {
		t.Fatalf("save sidebar partition state: %v", err)
	}
	loaded, err := store.LoadSidebarPartitionState()
	if err != nil {
		t.Fatalf("load sidebar partition state: %v", err)
	}

	expectSidebarStateEqual(t, saved, input)
	expectSidebarStateEqual(t, loaded, input)
}

func TestLoadSidebarPartitionStateNormalizesDirtyFile(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	saveSessionForSidebarTest(t, store, "session-keep")

	dirty := `{
		"version": 9,
		"partitions": [
			{"id":"work","name":"Work"},
			{"id":"work","name":"Duplicate"},
			{"id":"__unclassified__","name":"Reserved"},
			{"id":" ","name":"Empty"},
			{"id":"personal","name":" Personal "}
		],
		"assignments": {
			"session-keep":"work",
			"session-missing":"work",
			"session-keep-2":"missing",
			"session-blank":" "
		}
	}`
	path := store.sidebarPartitionStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir ui dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(dirty), 0o600); err != nil {
		t.Fatalf("write dirty sidebar state: %v", err)
	}

	loaded, err := store.LoadSidebarPartitionState()
	if err != nil {
		t.Fatalf("load sidebar partition state: %v", err)
	}

	expected := SessionSidebarPartitionState{
		Version: sessionSidebarPartitionVersion,
		Partitions: []SessionSidebarPartition{
			{ID: "work", Name: "Work"},
			{ID: "personal", Name: "Personal"},
		},
		Assignments: map[string]string{
			"session-keep": "work",
		},
	}
	expectSidebarStateEqual(t, loaded, expected)

	rewritten, exists, err := readSidebarPartitionStateFile(path)
	if err != nil {
		t.Fatalf("read rewritten sidebar state: %v", err)
	}
	if !exists {
		t.Fatal("expected rewritten sidebar state file to exist")
	}
	expectSidebarStateEqual(t, rewritten, expected)
}

func TestSaveSidebarPartitionStateUsesAtomicReplacement(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	saveSessionForSidebarTest(t, store, "session-a")
	saveSessionForSidebarTest(t, store, "session-b")

	first, err := store.SaveSidebarPartitionState(SessionSidebarPartitionState{
		Version: sessionSidebarPartitionVersion,
		Partitions: []SessionSidebarPartition{
			{ID: "work", Name: "Work"},
		},
		Assignments: map[string]string{
			"session-a": "work",
		},
	})
	if err != nil {
		t.Fatalf("save first sidebar state: %v", err)
	}

	path := store.sidebarPartitionStatePath()
	opened, err := os.Open(path)
	if err != nil {
		t.Fatalf("open sidebar state file: %v", err)
	}
	defer opened.Close()

	initialBytes, err := io.ReadAll(opened)
	if err != nil {
		t.Fatalf("read initial bytes: %v", err)
	}
	if len(initialBytes) == 0 {
		t.Fatal("expected initial sidebar state content")
	}

	second, err := store.SaveSidebarPartitionState(SessionSidebarPartitionState{
		Version: sessionSidebarPartitionVersion,
		Partitions: []SessionSidebarPartition{
			{ID: "work", Name: "Work"},
			{ID: "personal", Name: "Personal"},
		},
		Assignments: map[string]string{
			"session-a": "work",
			"session-b": "personal",
		},
	})
	if err != nil {
		t.Fatalf("save second sidebar state: %v", err)
	}

	if _, err := opened.Seek(0, 0); err != nil {
		t.Fatalf("seek old file: %v", err)
	}
	oldHandleBytes, err := io.ReadAll(opened)
	if err != nil {
		t.Fatalf("read old file handle: %v", err)
	}
	if string(oldHandleBytes) != string(initialBytes) {
		t.Fatalf("expected old handle to retain pre-rename content: got=%s want=%s", oldHandleBytes, initialBytes)
	}

	currentBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read current sidebar state file: %v", err)
	}
	if string(currentBytes) == string(initialBytes) {
		t.Fatal("expected current sidebar state file to be replaced")
	}

	glob, err := filepath.Glob(filepath.Join(baseDir, "ui", ".session-sidebar-partitions-*.tmp"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(glob) != 0 {
		t.Fatalf("expected temp files cleaned up, got %v", glob)
	}

	expectSidebarStateEqual(t, first, SessionSidebarPartitionState{
		Version: sessionSidebarPartitionVersion,
		Partitions: []SessionSidebarPartition{
			{ID: "work", Name: "Work"},
		},
		Assignments: map[string]string{
			"session-a": "work",
		},
	})
	expectSidebarStateEqual(t, second, SessionSidebarPartitionState{
		Version: sessionSidebarPartitionVersion,
		Partitions: []SessionSidebarPartition{
			{ID: "work", Name: "Work"},
			{ID: "personal", Name: "Personal"},
		},
		Assignments: map[string]string{
			"session-a": "work",
			"session-b": "personal",
		},
	})
}

func TestDeleteSessionCleansSidebarAssignments(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	saveSessionForSidebarTest(t, store, "session-a")
	saveSessionForSidebarTest(t, store, "session-b")

	if _, err := store.SaveSidebarPartitionState(SessionSidebarPartitionState{
		Version: sessionSidebarPartitionVersion,
		Partitions: []SessionSidebarPartition{
			{ID: "work", Name: "Work"},
		},
		Assignments: map[string]string{
			"session-a": "work",
			"session-b": "work",
		},
	}); err != nil {
		t.Fatalf("save sidebar state: %v", err)
	}

	if err := store.Delete("session-a"); err != nil {
		t.Fatalf("delete session: %v", err)
	}

	loaded, err := store.LoadSidebarPartitionState()
	if err != nil {
		t.Fatalf("load sidebar partition state: %v", err)
	}
	expectSidebarStateEqual(t, loaded, SessionSidebarPartitionState{
		Version: sessionSidebarPartitionVersion,
		Partitions: []SessionSidebarPartition{
			{ID: "work", Name: "Work"},
		},
		Assignments: map[string]string{
			"session-b": "work",
		},
	})
}

func saveSessionForSidebarTest(t *testing.T, store *Store, id string) {
	t.Helper()
	sess := NewSession("system")
	sess.ID = id
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session %s: %v", id, err)
	}
}

func expectSidebarStateEqual(
	t *testing.T,
	got SessionSidebarPartitionState,
	want SessionSidebarPartitionState,
) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected sidebar state:\n got=%+v\nwant=%+v", got, want)
	}
}
