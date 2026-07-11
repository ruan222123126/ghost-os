package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigProjectRootUpdatePersistsNormalizedDirectory(t *testing.T) {
	store, _ := newProjectRootTestStore(t)
	projectRoot := filepath.Join(t.TempDir(), "project-root")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", projectRoot, err)
	}

	if err := store.Update(configUpdateRequest{ProjectRoot: &projectRoot}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got := store.RuntimeConfig().ProjectRoot; got != projectRoot {
		t.Fatalf("unexpected runtime project_root: got %q want %q", got, projectRoot)
	}
	snapshot, err := store.PublicSnapshot()
	if err != nil {
		t.Fatalf("PublicSnapshot: %v", err)
	}
	if snapshot.ProjectRoot != projectRoot {
		t.Fatalf("unexpected public project_root: got %q want %q", snapshot.ProjectRoot, projectRoot)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if got := stringValue(fileCfg.ProjectRoot); got != projectRoot {
		t.Fatalf("unexpected persisted project_root: got %q want %q", got, projectRoot)
	}
}

func TestConfigProjectRootUpdateExpandsHomeDirectory(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	store, _ := newProjectRootTestStore(t)
	projectRoot := filepath.Join(homeDir, "repo")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", projectRoot, err)
	}
	value := "~/repo"

	if err := store.Update(configUpdateRequest{ProjectRoot: &value}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if got := stringValue(fileCfg.ProjectRoot); got != projectRoot {
		t.Fatalf("unexpected persisted project_root: got %q want %q", got, projectRoot)
	}
}

func TestConfigProjectRootUpdateRejectsInvalidPaths(t *testing.T) {
	store, _ := newProjectRootTestStore(t)
	filePath := filepath.Join(t.TempDir(), "plain-file")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", filePath, err)
	}

	cases := []struct {
		name string
		path string
		want error
	}{
		{name: "relative", path: "relative/path", want: errProjectRootAbsolute},
		{name: "missing", path: filepath.Join(t.TempDir(), "missing"), want: errProjectRootMissing},
		{name: "file", path: filePath, want: errProjectRootDir},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := store.Update(configUpdateRequest{ProjectRoot: &tc.path})
			if !errors.Is(err, tc.want) {
				t.Fatalf("unexpected error: got %v want %v", err, tc.want)
			}
		})
	}
}

func TestConfigProjectRootUpdateClearsToEnvFallback(t *testing.T) {
	store, envRoot := newProjectRootTestStore(t)
	projectRoot := filepath.Join(t.TempDir(), "project-root")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", projectRoot, err)
	}
	if err := store.Update(configUpdateRequest{ProjectRoot: &projectRoot}); err != nil {
		t.Fatalf("set project_root: %v", err)
	}

	empty := ""
	if err := store.Update(configUpdateRequest{ProjectRoot: &empty}); err != nil {
		t.Fatalf("clear project_root: %v", err)
	}

	if got := store.RuntimeConfig().ProjectRoot; got != envRoot {
		t.Fatalf("unexpected runtime project_root after clear: got %q want %q", got, envRoot)
	}
	snapshot, err := store.PublicSnapshot()
	if err != nil {
		t.Fatalf("PublicSnapshot: %v", err)
	}
	if snapshot.ProjectRoot != "" {
		t.Fatalf("expected empty public project_root after clear, got %q", snapshot.ProjectRoot)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.ProjectRoot != nil {
		t.Fatalf("expected project_root to be removed from file config, got %#v", fileCfg.ProjectRoot)
	}
}

func TestPublicSnapshotProjectRootOnlyShowsExplicitConfig(t *testing.T) {
	store, _ := newProjectRootTestStore(t)

	snapshot, err := store.PublicSnapshot()
	if err != nil {
		t.Fatalf("PublicSnapshot: %v", err)
	}
	if snapshot.ProjectRoot != "" {
		t.Fatalf("expected empty public project_root when unset, got %q", snapshot.ProjectRoot)
	}
}

func TestWithProjectRootOverrideOnlyAffectsResolvedConfig(t *testing.T) {
	store, _ := newProjectRootTestStore(t)
	overrideRoot := filepath.Join(t.TempDir(), "override-root")
	if err := os.MkdirAll(overrideRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", overrideRoot, err)
	}

	overrideStore := WithProjectRootOverride(store, overrideRoot)
	cfg, err := overrideStore.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if cfg.ProjectRoot != overrideRoot {
		t.Fatalf("unexpected overridden project_root: got %q want %q", cfg.ProjectRoot, overrideRoot)
	}

	snapshot, err := overrideStore.PublicSnapshot()
	if err != nil {
		t.Fatalf("PublicSnapshot: %v", err)
	}
	if snapshot.ProjectRoot != "" {
		t.Fatalf("expected public snapshot to stay unchanged, got %q", snapshot.ProjectRoot)
	}
}

func newProjectRootTestStore(t *testing.T) (*store, string) {
	t.Helper()

	tempDir := t.TempDir()
	envRoot := filepath.Join(tempDir, "env-root")
	if err := os.MkdirAll(envRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", envRoot, err)
	}
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(tempDir, "config.toml"))
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_PROJECT_ROOT", envRoot)

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}
	return store, envRoot
}
